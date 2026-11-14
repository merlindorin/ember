package net

import (
	"fmt"
	stdnet "net"
	"os"
	"reflect"
	"time"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/internal/netside"
	"github.com/lantervale/ember/world"
)

// Predicted is implemented by entities a client simulates ahead of the
// server when it owns them, so its own inputs show at once. The client
// applies each intent it sends with PredictIntent and advances the entity
// with PredictStep at the server's tick rate. Whenever the server's state
// for it arrives, the client rewinds to it, replays its recent acknowledged
// intents to rebuild the prediction's own state (a destination, say), steps
// forward by the round trip applying each unacknowledged intent when the
// server will, and blends the difference away. PredictIntent must therefore
// only record intent, never move the entity.
type Predicted interface {
	Replicated
	PredictIntent(msg any)
	PredictStep(dt float32)
}

// Settled is implemented by Predicted entities with one-shot intents (a
// dash, say): PredictSettled is called once the acknowledged intents are
// replayed, whose one-shot effects the server's state already holds.
type Settled interface {
	PredictSettled()
}

// maxPending bounds the unacknowledged intents kept for replay, and
// maxHistory the acknowledged ones replayed to rebuild prediction state.
const (
	maxPending = 256
	maxHistory = 32
)

// NewClient returns the replica side of a game over conn, a plugin the game
// adds to its World and keeps to send intents and handle messages. With
// WithEndpoint, conn is nil.
func NewClient(conn stdnet.Conn, opts ...ClientOption) *Client {
	c := &Client{
		now:       time.Now,
		handlers:  map[reflect.Type][]func(any){},
		mirrors:   map[ID]*mirror{},
		connected: true,
		missing:   map[string]bool{},
		step:      1.0 / 30,
		blend:     20,
		snap:      1,
	}
	for _, opt := range opts {
		opt.client(c)
	}
	if c.peer == nil {
		c.peer = newPeer(conn, c.codec)
	}
	if c.hello != nil {
		c.handshake(c.hello)
	}
	return c
}

func (c *Client) Build(a *world.World) {
	c.world = a
	netside.SetClient(a, c)
	a.On(world.PreUpdate, "net.client.receive", c.update)
}

// Client is the replica side.
type Client struct {
	world     *world.World
	hello     any
	peer      link
	now       func() time.Time
	id        ClientID
	handlers  map[reflect.Type][]func(any)
	mirrors   map[ID]*mirror
	placed    map[ID]Replicated
	unplaced  map[ID]unplacedSpawn
	connected bool
	missing   map[string]bool
	seq       uint32
	ack       uint32
	tick      uint64

	codec      Codec
	resuming   bool
	held       []any
	stale      map[ID]bool
	self       *mirror
	history    []any
	pending    []sentIntent
	rtt        time.Duration
	step       float32
	snap       float32
	acc        float32
	blend      float32
	correction geom.Vec3
	// The predicted entity is stepped at the server's rate and shown
	// between its last two steps: prev is the pose before the last step,
	// simRot its simulated rotation, and offset what is shown minus where
	// it is simulated.
	prevPos geom.Vec3
	prevRot geom.Quat
	simRot  geom.Quat
	offset  geom.Vec3
	clock   timeline
	// selfTick is the server tick of the predicted entity's newest state.
	selfTick uint64
	// dgLast and dgMask are the Poses datagrams received (see PosesAck);
	// ackDue asks for them to be acknowledged.
	dgLast  uint32
	dgMask  uint64
	ackDue  bool
	dgCount int
	// early holds poses of entities whose spawn has not arrived yet.
	early map[ID]sample
}

type sentIntent struct {
	seq uint32
	msg any
	at  time.Time
}

type mirror struct {
	e       Replicated
	target  Pose
	samples []sample
}

// HandleClient registers a handler for server messages of type T. Every
// handler registered for a type receives its messages, in order.
func HandleClient[T any](c *Client, fn func(msg T)) {
	var zero T
	Register(zero)
	t := reflect.TypeFor[T]()
	c.handlers[t] = append(c.handlers[t], func(m any) {
		if v, ok := m.(T); ok {
			fn(v)
		}
	})
}

// Send queues an intent for the server and returns its sequence number.
// The owned Predicted entity applies it at once. While a reconnection is
// being resumed, intents are held back, in order, and Send returns 0.
func (c *Client) Send(msg any) uint32 {
	Register(msg)
	if p := c.predicted(); p != nil {
		p.PredictIntent(msg)
	}
	if c.resuming {
		c.held = append(c.held, msg)
		return 0
	}
	return c.transmit(msg)
}

func (c *Client) transmit(msg any) uint32 {
	c.seq++
	c.peer.send(Intent{Seq: c.seq, Msg: msg})
	if len(c.pending) == maxPending {
		c.pending = c.pending[1:]
	}
	c.pending = append(c.pending, sentIntent{c.seq, msg, c.now()})
	return c.seq
}

func (c *Client) handshake(hello any) {
	Register(hello)
	c.peer.send(Intent{Msg: hello})
}

// Reconnect carries on over a new connection after ServerDisconnected, opening it
// with hello. Intents are then held back until the server either resumes
// the session (Resumed), which resends those it had not applied, or starts
// a new one (it spawns an entity owned by the new connection), which drops
// them. Mirrors are kept until then; those the new connection has not
// replicated again by the next snapshot are removed.
func (c *Client) Reconnect(conn stdnet.Conn, hello any) {
	c.peer.close()
	c.peer = newPeer(conn, c.codec)
	c.connected, c.resuming, c.id = true, true, 0
	c.stale = make(map[ID]bool, len(c.mirrors))
	for id := range c.mirrors {
		c.stale[id] = true
	}
	c.handshake(hello)
}

// resume resends, in the new connection's sequence, the intents the
// previous connection sent after last and the ones held since.
func (c *Client) resume(last uint32) {
	old := c.pending
	c.pending = nil
	for _, in := range old {
		if in.seq > last {
			c.transmit(in.msg)
		} else {
			c.history = append(c.history, in.msg)
		}
	}
	c.release()
}

// release sends what was held while resuming.
func (c *Client) release() {
	for _, msg := range c.held {
		c.transmit(msg)
	}
	c.held, c.resuming = nil, false
}

// RTT is the smoothed time between sending an intent and seeing it
// acknowledged.
func (c *Client) RTT() time.Duration { return c.rtt }

func (c *Client) predicted() Predicted {
	if c.self == nil {
		return nil
	}
	p, _ := c.self.e.(Predicted)
	return p
}

// claim tracks the first owned Predicted mirror as the one to predict.
func (c *Client) claim(m *mirror) {
	_, predicted := m.e.(Predicted)
	owned := predicted && c.IsOwned(m.e)
	switch {
	case owned && c.self == nil:
		c.self, c.correction, c.acc, c.offset = m, geom.Vec3{}, 0, geom.Vec3{}
		p := poseOf(m.e)
		c.prevPos, c.prevRot, c.simRot = p.Position, p.Rotation, p.Rotation
	case !owned && c.self == m:
		c.self = nil
	}
}

func (c *Client) acked(ack uint32) {
	now := c.now()
	i := 0
	for ; i < len(c.pending) && c.pending[i].seq <= ack; i++ {
		c.history = append(c.history, c.pending[i].msg)
		if c.pending[i].seq == ack {
			sample := now.Sub(c.pending[i].at)
			if c.rtt == 0 {
				c.rtt = sample
			} else {
				c.rtt += (sample - c.rtt) / 8
			}
		}
	}
	c.pending = c.pending[i:]
	if n := len(c.history) - maxHistory; n > 0 {
		c.history = append(c.history[:0], c.history[n:]...)
	}
}

// reconcile rewinds the predicted entity to the server's state, which is
// half a round trip old, and replays the round trip up to where the server
// will be once it has the intents sent so far: each pending intent applies
// at the step the server will receive it. What is displayed stays put, as
// a correction that decays.
func (c *Client) reconcile(p Predicted, u Update) {
	shown := poseOf(p).Position
	setPose(p, u.Pose)
	c.apply(p, u)
	for _, msg := range c.history {
		p.PredictIntent(msg)
	}
	if s, ok := p.(Settled); ok {
		s.PredictSettled()
	}
	now := c.now()
	window := c.rtt
	if len(c.pending) > 0 {
		window = max(window, now.Sub(c.pending[0].at))
	}
	// The replay's steps fall at the live steps' times, the last one acc
	// ago, each after the intents sent before it, as live prediction does.
	steps := max(int((window.Seconds()-float64(c.acc))/float64(c.step)+0.5), 0)
	last := now.Add(-time.Duration(float64(c.acc) * float64(time.Second)))
	if len(c.pending) > 0 && !c.pending[0].at.After(last) {
		steps = max(steps, int(last.Sub(c.pending[0].at).Seconds()/float64(c.step))+1)
	}
	next := 0
	start := poseOf(p)
	c.prevPos, c.prevRot = start.Position, start.Rotation
	for i := range steps {
		at := last.Add(-time.Duration(float64(steps-1-i) * float64(c.step) * float64(time.Second)))
		for ; next < len(c.pending) && !c.pending[next].at.After(at); next++ {
			p.PredictIntent(c.pending[next].msg)
		}
		c.stepPredicted(p)
	}
	for ; next < len(c.pending); next++ {
		p.PredictIntent(c.pending[next].msg)
	}
	t := poseOf(p)
	c.simRot = t.Rotation
	alpha := c.acc / c.step
	c.correction = shown.Sub(c.prevPos.Lerp(t.Position, alpha))
	c.offset = shown.Sub(t.Position)
	t.Position = shown
	t.Rotation = c.prevRot.Slerp(c.simRot, alpha)
	setPose(p, t)
}

// predict advances the predicted entity at the server's step, shows it
// between its last two steps so it moves every frame, and decays its
// correction.
func (c *Client) predict(dt float32) {
	p := c.predicted()
	if p == nil {
		return
	}
	t := poseOf(p)
	t.Position, t.Rotation = t.Position.Sub(c.offset), c.simRot
	setPose(p, t)
	for c.acc += dt; c.acc >= c.step; c.acc -= c.step {
		c.stepPredicted(p)
	}
	t = poseOf(p)
	c.simRot = t.Rotation
	c.correction = c.correction.Scale(geom.Exp(-c.blend * dt))
	alpha := c.acc / c.step
	shown := c.prevPos.Lerp(t.Position, alpha).Add(c.correction)
	c.offset = shown.Sub(t.Position)
	t.Position = shown
	t.Rotation = c.prevRot.Slerp(c.simRot, alpha)
	setPose(p, t)
}

// stepPredicted advances the predicted entity one step. A step longer
// than the snap distance is a jump (a dash, say): it is shown at once
// instead of blended over the step, so live and replayed predictions show
// it alike whichever step it falls in.
func (c *Client) stepPredicted(p Predicted) {
	before := poseOf(p)
	c.prevPos, c.prevRot = before.Position, before.Rotation
	p.PredictStep(c.step)
	if after := poseOf(p); after.Position.Dist(c.prevPos) > c.snap {
		c.prevPos, c.prevRot = after.Position, after.Rotation
	}
}

// Acked is the last intent sequence number the server has applied.
func (c *Client) Acked() uint32 { return c.ack }

// Datagrams counts the Poses datagrams received, 0 while poses come on the
// reliable stream.
func (c *Client) Datagrams() int { return c.dgCount }

// Tick is the server fixed step of the latest snapshot.
func (c *Client) Tick() uint64 { return c.tick }

// ID is the ClientID assigned by the server (0 until welcomed).
func (c *Client) ID() ClientID { return c.id }

func (c *Client) Connected() bool { return c.connected }

// Lookup finds the local mirror of a replicated entity.
func (c *Client) Lookup(id ID) (Replicated, bool) {
	m, ok := c.mirrors[id]
	if !ok {
		return nil, false
	}
	return m.e, true
}

// IsOwned reports whether the entity belongs to this client.
func (c *Client) IsOwned(e Replicated) bool {
	return c.id != 0 && e.GetNetwork().Owner == c.id
}

// Owned returns the first mirrored entity owned by this client.
func (c *Client) Owned() (Replicated, bool) {
	for _, m := range c.mirrors {
		if c.IsOwned(m.e) {
			return m.e, true
		}
	}
	return nil, false
}

// Close drops the connection.
func (c *Client) Close() { c.peer.close() }

func (c *Client) update(frame world.Frame) {
	dt := frame.DT
	if c.connected {
		msgs, alive := c.peer.poll()
		for _, m := range msgs {
			c.dispatch(m)
		}
		if !alive {
			c.connected = false
			world.Emit(c.world, ServerDisconnected{})
		}
	}
	if c.ackDue && c.connected {
		c.ackDue = false
		c.peer.sendDatagram(PosesAck{Last: c.dgLast, Mask: c.dgMask})
	}
	c.predict(dt)
	if !c.clock.has {
		return
	}
	at := c.clock.advance(c.now(), dt, c.step)
	for _, m := range c.mirrors {
		if m == c.self {
			continue
		}
		if p, ok := m.at(at); ok {
			setPose(m.e, p)
		}
	}
}

func (c *Client) dispatch(m any) {
	switch msg := m.(type) {
	case Welcome:
		c.id = msg.ID
	case Resumed:
		if c.resuming {
			c.resume(msg.LastSeq)
		}
		for _, h := range c.handlers[reflect.TypeOf(m)] {
			h(m)
		}
	case Poses:
		c.poses(msg)
	case Snapshot:
		c.snapshot(msg)
	default:
		for _, h := range c.handlers[reflect.TypeOf(m)] {
			h(m)
		}
	}
}

func (c *Client) snapshot(msg Snapshot) {
	if msg.Ack > c.ack {
		c.acked(msg.Ack)
	}
	c.clock.observe(msg.Tick, c.now(), c.step)
	c.ack, c.tick = max(c.ack, msg.Ack), max(c.tick, msg.Tick)
	settled := !c.resuming
	for _, s := range msg.Spawns {
		if c.resuming && c.id != 0 && s.Owner == c.id {
			c.pending, c.history = nil, nil
			c.release()
		}
		c.spawn(s, msg.Tick)
		delete(c.stale, s.ID)
	}
	if settled && c.stale != nil {
		for id := range c.stale {
			c.despawn(id)
		}
		c.stale = nil
	}
	if !c.updates(msg) && c.self != nil {
		c.reconcile(c.predicted(), Update{Pose: c.self.target})
	}
	for _, id := range msg.Despawns {
		c.despawn(id)
	}
}

// updates applies msg's updates to their mirrors and reports whether one
// of them reconciled the owned entity.
func (c *Client) updates(msg Snapshot) bool {
	reconciled := false
	for _, u := range msg.Updates {
		m, ok := c.mirrors[u.ID]
		if !ok {
			continue
		}
		m.target = u.Pose
		m.record(msg.Tick, u.Pose)
		if u.HasOwner {
			m.e.GetNetwork().Owner = u.Owner
			c.claim(m)
		}
		if m == c.self {
			c.selfTick = max(c.selfTick, msg.Tick)
			c.reconcile(c.predicted(), u)
			reconciled = true
			continue
		}
		c.apply(m.e, u)
	}
	return reconciled
}

func (c *Client) despawn(id ID) {
	if m, ok := c.mirrors[id]; ok {
		if m == c.self {
			c.self = nil
		}
		c.drop(m)
		delete(c.mirrors, id)
	}
}

func (c *Client) apply(e Replicated, u Update) {
	if st, ok := e.(Stateful); ok && u.HasState {
		st.ApplyNetState(u.State)
	}
	if st, ok := e.(OwnerStateful); ok && u.HasPrivate && u.Private != nil {
		st.ApplyOwnerState(u.Private)
	}
}

func (c *Client) spawn(s Spawn, tick uint64) {
	if m, ok := c.mirrors[s.ID]; ok {
		m.target = s.Pose
		m.record(tick, s.Pose)
		m.e.GetNetwork().Owner = s.Owner
		c.claim(m)
		u := Update{
			ID:         s.ID,
			Pose:       s.Pose,
			State:      s.State,
			HasState:   s.State != nil,
			Private:    s.Private,
			HasPrivate: s.Private != nil,
		}
		if m == c.self {
			c.reconcile(c.predicted(), u)
		} else {
			c.apply(m.e, u)
		}
		return
	}
	e, placed := c.placed[s.ID]
	if !placed && s.ID.Region() == PlacedRegion {
		if c.unplaced == nil {
			c.unplaced = map[ID]unplacedSpawn{}
		}
		c.unplaced[s.ID] = unplacedSpawn{s, tick}
		return
	}
	if placed {
		c.adopt(e, s, tick)
		return
	}
	factory, ok := kindFactory(s.Kind)
	if !ok {
		if !c.missing[s.Kind] {
			c.missing[s.Kind] = true
			fmt.Fprintf(os.Stderr, "net: no factory registered for kind %q\n", s.Kind)
		}
		return
	}
	e = factory()
	c.adopt(e, s, tick)
	c.world.AddEntity(e)
}

type unplacedSpawn struct {
	s    Spawn
	tick uint64
}

// Place registers an entity the client already has, placed in a level the
// server also loaded, under its ID (see PlacedID): the server's spawn
// for that ID updates it instead of making a mirror, and net never removes
// it, a despawn only stopping the updates.
func (c *Client) Place(e Replicated) {
	id := e.GetNetwork().NetID
	if c.placed == nil {
		c.placed = map[ID]Replicated{}
	}
	c.placed[id] = e
	if u, ok := c.unplaced[id]; ok {
		delete(c.unplaced, id)
		c.adopt(e, u.s, u.tick)
	}
}

func (c *Client) adopt(e Replicated, s Spawn, tick uint64) {
	*e.GetNetwork() = Network{NetID: s.ID, Kind: s.Kind, Owner: s.Owner}
	setPose(e, s.Pose)
	if st, ok := e.(Stateful); ok && s.State != nil {
		st.ApplyNetState(s.State)
	}
	if st, ok := e.(OwnerStateful); ok && s.Private != nil {
		st.ApplyOwnerState(s.Private)
	}
	m := &mirror{e: e, target: s.Pose, samples: []sample{{tick, s.Pose}}}
	if early, ok := c.early[s.ID]; ok {
		if early.tick > tick {
			m.target = early.pose
			m.record(early.tick, early.pose)
		}
		delete(c.early, s.ID)
	}
	c.mirrors[s.ID] = m
	c.claim(m)
}

// poses applies a Poses datagram: it may be late, duplicated or out of
// order, so each pose only counts when newer than what the mirror holds.
func (c *Client) poses(msg Poses) {
	c.dgCount++
	c.receivedDatagram(msg.Seq)
	c.clock.observe(msg.Tick, c.now(), c.step)
	if msg.Ack > c.ack {
		c.acked(msg.Ack)
		c.ack = msg.Ack
	}
	c.tick = max(c.tick, msg.Tick)
	for _, u := range msg.Updates {
		m, ok := c.mirrors[u.ID]
		switch {
		case !ok:
			if c.early == nil {
				c.early = map[ID]sample{}
			}
			if e, seen := c.early[u.ID]; !seen || e.tick < msg.Tick {
				c.early[u.ID] = sample{msg.Tick, u.Pose}
			}
		case m == c.self:
			if msg.Tick > c.selfTick {
				c.selfTick, m.target = msg.Tick, u.Pose
				c.reconcile(c.predicted(), Update{ID: u.ID, Pose: u.Pose})
			}
		default:
			if n := len(m.samples); n == 0 || msg.Tick > m.samples[n-1].tick {
				m.target = u.Pose
			}
			m.record(msg.Tick, u.Pose)
		}
	}
	for id, e := range c.early {
		if e.tick+maxEarly < c.tick {
			delete(c.early, id)
		}
	}
}

// maxEarly is how many ticks a pose waits for its entity's spawn.
const maxEarly = 60

func (c *Client) receivedDatagram(seq uint32) {
	switch {
	case seq > c.dgLast:
		d := seq - c.dgLast
		mask := uint64(0)
		if d <= 64 && c.dgLast != 0 {
			mask = 1 << (d - 1)
			if d < 64 {
				mask |= c.dgMask << d
			}
		}
		c.dgLast, c.dgMask = seq, mask
	case seq < c.dgLast && c.dgLast-seq <= 64:
		c.dgMask |= 1 << (c.dgLast - seq - 1)
	}
	c.ackDue = true
}

// drop removes a mirror's entity, unless it is placed.
func (c *Client) drop(m *mirror) {
	if _, placed := c.placed[m.e.GetNetwork().NetID]; !placed {
		c.world.RemoveEntity(m.e)
	}
}
