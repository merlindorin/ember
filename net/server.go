package net

import (
	"cmp"
	"context"
	"fmt"
	stdnet "net"
	"reflect"
	"slices"
	"sort"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/internal/netside"
	"github.com/merlindorin/ember/spatial"
	"github.com/merlindorin/ember/world"
)

// NewServer returns the authoritative side of a game, a plugin the game adds
// to its World and keeps to accept connections, handle intents and send
// messages.
func NewServer(opts ...ServerOption) *Server {
	s := &Server{
		pending:    make(chan stdnet.Conn, 64),
		links:      make(chan link, 64),
		clients:    map[ClientID]*remote{},
		handlers:   map[reflect.Type]func(ClientID, any){},
		interval:   1.0 / 20,
		radius:     60,
		margin:     8,
		maxIntents: 256,
		budget:     256,
		byID:       map[ID]Replicated{},
		owned:      map[ClientID]ID{},
	}
	for _, opt := range opts {
		opt.server(s)
	}
	s.grid = spatial.NewGrid(s.radius / 4)
	s.entities.s = s
	return s
}

func (s *Server) Build(a *world.World) {
	s.world = a
	s.time = a.Time()
	if s.listener != nil {
		go func() {
			for c := range s.listener.Conns() {
				s.Connect(c)
			}
		}()
	}
	netside.SetServer(a, s)
	a.AddView(&s.entities)
	a.On(world.PreUpdate, "net.server.receive", s.receive)
	a.AddSystems(&intents{s}, &replicator{s})
}

// Server is the authoritative side.
type Server struct {
	world    *world.World
	listener Listener
	pending  chan stdnet.Conn
	links    chan link
	clients  map[ClientID]*remote
	handlers map[reflect.Type]func(ClientID, any)
	entities replicas
	time     *world.Time

	nextClient ClientID
	region     uint16
	needFocus  bool
	codec      Codec
	budget     int
	candidates []candidate
	counter    uint64
	interval   float32
	acc        float32
	radius     float32
	margin     float32
	maxIntents int

	grid  *spatial.Grid
	byID  map[ID]Replicated
	owned map[ClientID]ID
}

type remote struct {
	id     ClientID
	peer   link
	focus  ID
	known  map[ID]*sent
	inbox  []Intent
	ack    uint32
	acked  uint32
	joined bool
	gone   bool
	quiet  int
	// seq numbers the Poses datagrams; ackLast and ackMask are the ones
	// the client acknowledged (see PosesAck).
	seq     uint32
	ackLast uint32
	ackMask uint64
}

// delivered reports whether the datagram seq reached the client; 0 stands
// for the reliable stream.
func (r *remote) delivered(seq uint32) bool {
	switch {
	case seq == 0 || seq == r.ackLast:
		return true
	case seq > r.ackLast || r.ackLast-seq > 64:
		return false
	}
	return r.ackMask&(1<<(r.ackLast-seq-1)) != 0
}

// acknowledge merges an acknowledgement, which may arrive out of order.
func (r *remote) acknowledge(a PosesAck) {
	switch {
	case a.Last > r.ackLast:
		d := a.Last - r.ackLast
		mask := a.Mask
		if d <= 64 && r.ackLast != 0 {
			mask |= 1 << (d - 1)
			if d < 64 {
				mask |= r.ackMask << d
			}
		}
		r.ackLast, r.ackMask = a.Last, mask
	case a.Last == r.ackLast:
		r.ackMask |= a.Mask
	default:
		if d := r.ackLast - a.Last; d <= 64 {
			r.ackMask |= 1 << (d - 1)
			if d < 64 {
				r.ackMask |= a.Mask << d
			}
		}
	}
}

// PosesPerDatagram keeps a Poses datagram under a QUIC packet: codecs must
// fit that many poses in about 1100 bytes.
const PosesPerDatagram = 12

// replicas assigns IDs as entities join, so entities spawned during a
// fixed step can be targeted by the next one.
type replicas struct {
	world.Group[Replicated]
	s *Server
}

func (v *replicas) Remove(o world.Identifier) {
	v.Group.Remove(o)
	if e, ok := o.(Replicated); ok {
		v.s.grid.Remove(uint64(e.GetNetwork().NetID))
	}
}

func (v *replicas) AddByInterface(o world.Identifier) {
	v.Group.AddByInterface(o)
	if e, ok := o.(Replicated); ok {
		if n := e.GetNetwork(); n.NetID == 0 {
			v.s.counter++
			n.NetID = NewID(v.s.region, v.s.counter)
		}
	}
}

type sent struct {
	pose    Pose
	owner   ClientID
	state   any
	private any
	// poseSeq is the datagram that last carried pose, 0 when the stream
	// did.
	poseSeq uint32
}

// Connect adds a connection (from Pipe or a custom transport). It is safe
// to call from any goroutine.
func (s *Server) Connect(c stdnet.Conn) { s.pending <- c }

// ConnectEndpoint adds the server end of a Lag pair.
func (s *Server) ConnectEndpoint(e Endpoint) { s.links <- e.l }

// Handle registers a handler for client messages of type T.
func Handle[T any](s *Server, fn func(c ClientID, msg T)) {
	var zero T
	Register(zero)
	s.handlers[reflect.TypeFor[T]()] = func(c ClientID, m any) {
		if v, ok := m.(T); ok {
			fn(c, v)
		}
	}
}

// Inject queues msg as if client c had sent it, applied at the next fixed
// step like any received intent. Bots, tools and tests use it.
func (s *Server) Inject(c ClientID, msg any) {
	if r, ok := s.clients[c]; ok {
		r.inbox = append(r.inbox, Intent{Msg: msg})
	}
}

// Counter is the last ID counter assigned. A shard saves it and passes
// it to Resume after a restart, so no ID is ever reused.
func (s *Server) Counter() uint64 { return s.counter }

// Resume continues assigning IDs after counter.
func (s *Server) Resume(counter uint64) { s.counter = max(s.counter, counter) }

// Ack returns the last intent sequence number applied for a client.
func (s *Server) Ack(c ClientID) uint32 {
	if r, ok := s.clients[c]; ok {
		return r.ack
	}
	return 0
}

// Send queues a message for one client.
func (s *Server) Send(c ClientID, msg any) {
	if r, ok := s.clients[c]; ok {
		r.peer.send(msg)
	}
}

// Broadcast queues a message for every client.
func (s *Server) Broadcast(msg any) {
	for _, r := range s.clients {
		r.peer.send(msg)
	}
}

// Clients returns connected client IDs in ascending order.
func (s *Server) Clients() []ClientID {
	ids := make([]ClientID, 0, len(s.clients))
	for id := range s.clients {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

// SetFocus centers a client's interest area on an entity. By default the
// first entity the client owns is used.
func (s *Server) SetFocus(c ClientID, id ID) {
	if r, ok := s.clients[c]; ok {
		r.focus = id
	}
}

// Kick closes a client connection once what was sent to it is written.
func (s *Server) Kick(c ClientID) {
	if r, ok := s.clients[c]; ok {
		r.peer.closeAfterFlush()
	}
}

// CloseAll closes every client once what was sent to it is written, waiting
// for that up to ctx, so a last message reaches clients before the process
// exits.
func (s *Server) CloseAll(ctx context.Context) {
	waits := make([]<-chan struct{}, 0, len(s.clients))
	for _, r := range s.clients {
		r.peer.closeAfterFlush()
		waits = append(waits, r.peer.closed())
	}
	for _, w := range waits {
		select {
		case <-w:
		case <-ctx.Done():
			return
		}
	}
}

func (s *Server) accept(l link) {
	s.nextClient++
	r := &remote{id: s.nextClient, peer: l, known: map[ID]*sent{}}
	s.clients[r.id] = r
	r.peer.send(Welcome{ID: r.id})
}

func (s *Server) receive(world.Frame) {
	for {
		select {
		case c := <-s.pending:
			s.accept(newPeer(c, s.codec))
			continue
		case l := <-s.links:
			s.accept(l)
			continue
		default:
		}
		break
	}

	for _, r := range s.clients {
		if r.gone {
			continue
		}
		msgs, alive := r.peer.poll()
		for _, m := range msgs {
			switch m := m.(type) {
			case Intent:
				r.inbox = append(r.inbox, m)
			case PosesAck:
				r.acknowledge(m)
			}
		}
		if len(r.inbox) > s.maxIntents {
			r.inbox = r.inbox[:s.maxIntents]
			r.peer.close()
			alive = false
		}
		r.gone = !alive
	}
}

// IntentPriority is the FixedUpdate priority at which intents are applied
// and ClientConnected/ClientDisconnected are emitted. Systems reading those
// events in FixedUpdate run just below it, so they see them in the same
// step: events expire after two frames, which at high frame rates can pass
// without any fixed step.
const IntentPriority = 1 << 20

// apply runs queued intents at the start of a fixed step, in client order,
// skipping sequence numbers already applied, then drops clients whose
// connection closed once their last intents ran. A client's intents wait
// one step after it joins, so its join can be handled first.
func (s *Server) apply(float32) {
	for _, id := range s.Clients() {
		r := s.clients[id]
		if !r.joined {
			r.joined = true
			world.Emit(s.world, ClientConnected{ID: id})
			continue
		}
		for _, in := range r.inbox {
			if in.Seq != 0 && in.Seq <= r.ack {
				continue
			}
			if h, ok := s.handlers[reflect.TypeOf(in.Msg)]; ok {
				h(id, in.Msg)
			}
			if in.Seq != 0 {
				r.ack = in.Seq
			}
		}
		clear(r.inbox)
		r.inbox = r.inbox[:0]
		if r.gone {
			r.peer.close()
			delete(s.clients, id)
			world.Emit(s.world, ClientDisconnected{ID: id, Ack: r.ack})
		}
	}
}

type intents struct{ s *Server }

func (i *intents) Update(f world.Frame)    { i.s.apply(f.DT) }
func (i *intents) Remove(world.Identifier) {}
func (i *intents) Phase() world.Phase      { return world.FixedUpdate }
func (i *intents) Priority() int           { return IntentPriority }
func (i *intents) Name() string            { return "net.server.intents" }

type replicator struct{ s *Server }

func (r *replicator) Update(f world.Frame)    { r.s.replicate(f.DT) }
func (r *replicator) Remove(world.Identifier) {}
func (r *replicator) Phase() world.Phase      { return world.PostUpdate }
func (r *replicator) Priority() int           { return 500 }
func (r *replicator) Name() string            { return "net.server.replicate" }

func (s *Server) replicate(dt float32) {
	s.acc += dt
	if s.acc < s.interval {
		return
	}
	s.acc = min(s.acc-s.interval, s.interval)

	clear(s.byID)
	clear(s.owned)
	for _, e := range s.entities.Slice() {
		n := e.GetNetwork()
		s.byID[n.NetID] = e
		s.grid.Move(uint64(n.NetID), e.GetTransform().WorldPosition())
		if cur, ok := s.owned[n.Owner]; n.Owner != 0 && (!ok || n.NetID < cur) {
			s.owned[n.Owner] = n.NetID
		}
	}
	for _, id := range s.Clients() {
		s.sendSnapshot(s.clients[id])
	}
}

// focusOf is the client's focus entity, or else the first entity it owns.
func (s *Server) focusOf(r *remote) (geom.Vec3, bool) {
	e, ok := s.byID[r.focus]
	if !ok {
		e, ok = s.byID[s.owned[r.id]]
	}
	if !ok {
		return geom.Vec3{}, false
	}
	return e.GetTransform().WorldPosition(), true
}

// sendSnapshot replicates what lies in the client's interest area: entities
// enter within radius, found through the grid, and leave beyond
// radius+margin, so only known and nearby entities are ever visited.
func (s *Server) sendSnapshot(r *remote) {
	snap := Snapshot{Tick: s.time.FixedFrame, Ack: r.ack}
	focus, filtered := s.focusOf(r)
	blind := !filtered && s.needFocus
	leave := s.radius + s.margin
	for id := range r.known {
		e, alive := s.byID[id]
		if !alive || blind || filtered && flatDist(e.GetTransform().WorldPosition(), focus) > leave {
			snap.Despawns = append(snap.Despawns, id)
			delete(r.known, id)
		}
	}
	s.candidates = s.candidates[:0]
	switch {
	case blind:
	case filtered:
		s.grid.Query(focus, leave, func(id uint64, p geom.Vec3) bool {
			_, known := r.known[ID(id)]
			if d := flatDist(p, focus); known || d <= s.radius {
				s.candidates = append(s.candidates, candidate{s.byID[ID(id)], d})
			}
			return true
		})
	default:
		for _, e := range s.entities.Slice() {
			s.candidates = append(s.candidates, candidate{e, 0})
		}
	}
	slices.SortFunc(s.candidates, func(a, b candidate) int {
		ownA, ownB := a.e.GetNetwork().Owner == r.id, b.e.GetNetwork().Owner == r.id
		switch {
		case ownA != ownB:
			if ownA {
				return -1
			}
			return 1
		case a.dist != b.dist:
			return cmp.Compare(a.dist, b.dist)
		}
		return cmp.Compare(a.e.GetNetwork().NetID, b.e.GetNetwork().NetID)
	})
	var poses []Update
	for _, c := range s.candidates {
		if len(snap.Spawns)+len(snap.Updates)+len(poses) >= s.budget {
			break
		}
		s.replicateTo(r, c.e, &snap, &poses)
	}
	for len(poses) > 0 {
		n := min(len(poses), PosesPerDatagram)
		r.seq++
		for _, u := range poses[:n] {
			r.known[u.ID].poseSeq = r.seq
		}
		r.peer.sendDatagram(Poses{Seq: r.seq, Tick: snap.Tick, Ack: r.ack, Updates: poses[:n]})
		poses = poses[n:]
	}
	r.quiet++
	if len(snap.Spawns)+len(snap.Updates)+len(snap.Despawns) > 0 || r.ack != r.acked ||
		float32(r.quiet)*s.interval >= 1 {
		r.acked, r.quiet = r.ack, 0
		r.peer.send(snap)
	}
}

type candidate struct {
	e    Replicated
	dist float32
}

// replicateTo adds what changed of e to the snapshot. On links with
// datagrams a change of pose alone goes to poses instead, and a pose is
// sent again until the client acknowledges a datagram carrying it.
func (s *Server) replicateTo(r *remote, e Replicated, snap *Snapshot, poses *[]Update) {
	n := e.GetNetwork()
	p := poseOf(e.GetTransform())
	var state, private any
	if st, ok := e.(Stateful); ok {
		state = st.NetState()
	}
	if st, ok := e.(OwnerStateful); ok && n.Owner == r.id {
		private = st.OwnerState()
	}
	known, ok := r.known[n.NetID]
	if !ok {
		mustCompare(n.Kind, state)
		mustCompare(n.Kind, private)
		snap.Spawns = append(
			snap.Spawns,
			Spawn{ID: n.NetID, Kind: n.Kind, Owner: n.Owner, Pose: p, State: state, Private: private},
		)
		r.known[n.NetID] = &sent{pose: p, owner: n.Owner, state: state, private: private}
		return
	}
	u := Update{ID: n.NetID, Pose: p}
	if state != known.state {
		u.State, u.HasState, known.state = state, true, state
	}
	if private != known.private {
		u.Private, u.HasPrivate, known.private = private, true, private
	}
	if n.Owner != known.owner {
		u.Owner, u.HasOwner, known.owner = n.Owner, true, n.Owner
	}
	if u.HasState || u.HasPrivate || u.HasOwner || !r.peer.datagrams() {
		if p == known.pose && !u.HasState && !u.HasPrivate && !u.HasOwner && r.delivered(known.poseSeq) {
			return
		}
		known.pose, known.poseSeq = p, 0
		snap.Updates = append(snap.Updates, u)
		return
	}
	if p == known.pose && r.delivered(known.poseSeq) {
		return
	}
	known.pose = p
	*poses = append(*poses, Update{ID: n.NetID, Pose: p})
}

func mustCompare(kind string, v any) {
	if v != nil && !reflect.TypeOf(v).Comparable() {
		panic(fmt.Sprintf("net: %q replicates %T, which is not comparable with ==", kind, v))
	}
}

func flatDist(a, b geom.Vec3) float32 { return a.WithY(0).Dist(b.WithY(0)) }
