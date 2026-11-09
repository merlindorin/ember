package net //nolint:testpackage // white-box: Client.mirrors, the server's clients and outbox

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/gob"
	"io"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/internal/netside"
	"github.com/merlindorin/ember/scene"
	"github.com/merlindorin/ember/world"
)

type Vitals struct{ HP int }

type unit struct {
	*world.BasicEntity
	scene.Transform
	Network
	Vitals Vitals
}

func (u *unit) NetState() any { return u.Vitals }

func (u *unit) ApplyNetState(s any) {
	if v, ok := s.(Vitals); ok {
		u.Vitals = v
	}
}

func newUnit(pos geom.Vec3, owner ClientID) *unit {
	return &unit{
		BasicEntity: world.NewBasic(),
		Transform:   scene.AtVec(pos),
		Network:     Network{Kind: "unit", Owner: owner},
		Vitals:      Vitals{HP: 10},
	}
}

type Ping struct{ N int }
type Pong struct{ N int }

func TestMain(m *testing.M) {
	registerUnit()
	registerChest()
	registerWalker()
	registerMover()
	os.Exit(m.Run())
}

func registerUnit() {
	Register(Vitals{})
	RegisterKind("unit", func() Replicated {
		return &unit{BasicEntity: world.NewBasic(), Transform: scene.At(0, 0, 0)}
	})
}

type rig struct {
	t      *testing.T
	server *world.World
	client *world.World
}

func newRig(t *testing.T, sp *Server) *rig {
	srvConn, cliConn := Pipe()
	r := &rig{t: t, server: world.New(), client: world.New()}
	r.server.AddPlugins(sp)
	serverOf(r.server).Connect(srvConn)
	r.client.AddPlugins(NewClient(cliConn))
	t.Cleanup(func() {
		clientOf(r.client).Close()
	})
	return r
}

func (r *rig) srv() *Server { return serverOf(r.server) }
func (r *rig) cli() *Client { return clientOf(r.client) }

// until steps both apps until cond holds or a timeout elapses.
func (r *rig) until(what string, cond func() bool) {
	r.t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			r.t.Fatalf("timed out waiting for %s", what)
		}
		r.server.RunFrames(1)
		time.Sleep(time.Millisecond)
		r.client.RunFrames(1)
	}
}

func TestSpawnStateUpdateAndDespawn(t *testing.T) {
	r := newRig(t, NewServer())
	u := newUnit(geom.V3(1, 0, 2), 0)
	r.server.AddEntities(u)

	r.until("spawn", func() bool { return len(r.cli().mirrors) == 1 })
	m, ok := r.cli().Lookup(u.NetID)
	if !ok {
		t.Fatal("mirror missing")
	}
	mirror, ok := m.(*unit)
	if !ok {
		t.Fatalf("mirror %T", m)
	}
	if mirror.Position != geom.V3(1, 0, 2) || mirror.Vitals.HP != 10 || mirror.Kind != "unit" {
		t.Fatalf("mirror %+v", mirror)
	}

	u.Position = geom.V3(5, 0, 2)
	u.Vitals.HP = 3
	r.until("smoothed transform and state", func() bool {
		return mirror.Position.ApproxEq(geom.V3(5, 0, 2), 1e-2) && mirror.Vitals.HP == 3
	})

	r.server.RemoveEntity(u)
	r.until("despawn", func() bool { return r.client.Len() == 0 })
}

func TestTypedMessagesBothWays(t *testing.T) {
	r := newRig(t, NewServer())
	var gotPing Ping
	var from ClientID
	Handle(r.srv(), func(c ClientID, p Ping) {
		gotPing, from = p, c
		r.srv().Send(c, Pong{N: p.N * 2})
	})
	var gotPong Pong
	HandleClient(r.cli(), func(p Pong) { gotPong = p })

	r.until("welcome", func() bool { return r.cli().ID() != 0 })
	r.cli().Send(Ping{N: 21})
	r.until("pong", func() bool { return gotPong.N == 42 })
	if gotPing.N != 21 || from != r.cli().ID() {
		t.Fatalf("ping %v from %v", gotPing, from)
	}
}

func TestInterestRadius(t *testing.T) {
	r := newRig(t, NewServer(Interest(10, 0)))
	r.until("welcome", func() bool { return r.cli().ID() != 0 })
	hero := newUnit(geom.V3(0, 0, 0), r.cli().ID())
	far := newUnit(geom.V3(50, 0, 0), 0)
	r.server.AddEntities(hero, far)

	r.until("hero spawn", func() bool { _, ok := r.cli().Lookup(hero.NetID); return ok })
	r.server.RunFrames(6)
	r.client.RunFrames(2)
	if _, ok := r.cli().Lookup(far.NetID); ok {
		t.Fatal("far entity must not be replicated")
	}
	if owned, ok := r.cli().Owned(); !ok || owned.GetNetwork().NetID != hero.NetID {
		t.Fatal("client should own the hero")
	}

	hero.Position = geom.V3(45, 0, 0)
	r.until("far spawn", func() bool { _, ok := r.cli().Lookup(far.NetID); return ok })

	hero.Position = geom.V3(0, 0, 0)
	r.until("far despawn", func() bool { _, ok := r.cli().Lookup(far.NetID); return !ok })
}

func TestConnectionEvents(t *testing.T) {
	r := newRig(t, NewServer())
	var events world.Reader[ClientConnected]
	var gone world.Reader[ClientDisconnected]
	connected, disconnected := 0, 0
	r.server.On(world.Update, "count", func(world.Frame) {
		for range events.Read(r.server) {
			connected++
		}
		for range gone.Read(r.server) {
			disconnected++
		}
	})
	r.until("connect", func() bool { return connected == 1 })
	if ids := r.srv().Clients(); len(ids) != 1 {
		t.Fatalf("clients %v", ids)
	}
	r.cli().Close()
	r.until("disconnect", func() bool { return disconnected == 1 && len(r.srv().Clients()) == 0 })
	if r.cli().Connected() {
		r.client.RunFrames(1)
	}
	if r.cli().Connected() {
		t.Fatal("client should notice the closed connection")
	}
}

func TestTCP(t *testing.T) {
	ln, err := Listen("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	server := world.New().AddPlugins(NewServer(WithListener(ln)))
	conn, err := Dial(ln.Addr())
	if err != nil {
		t.Fatal(err)
	}
	client := world.New().AddPlugins(NewClient(conn))
	defer clientOf(client).Close()
	server.AddEntities(newUnit(geom.V3(1, 2, 3), 0))

	r := &rig{t: t, server: server, client: client}
	r.until("tcp spawn", func() bool { return client.Len() == 1 })
}

type Purse struct{ Coins int }

type Bag struct{ Items []int }

type chest struct {
	*world.BasicEntity
	scene.Transform
	Network
	Lid   Vitals
	Purse Purse
	Bag   Bag
}

func (c *chest) NetState() any { return c.Lid }
func (c *chest) ApplyNetState(s any) {
	if v, ok := s.(Vitals); ok {
		c.Lid = v
	}
}

func (c *chest) OwnerState() any { return c.Purse }

func (c *chest) ApplyOwnerState(s any) {
	if p, ok := s.(Purse); ok {
		c.Purse = p
	}
}

type sack struct{ chest }

func (s *sack) NetState() any { return s.Bag }

func registerChest() {
	Register(Purse{})
	Register(Bag{})
	RegisterKind("chest", func() Replicated {
		return &chest{BasicEntity: world.NewBasic(), Transform: scene.At(0, 0, 0)}
	})
}

func TestOwnerStateOnlyReachesTheOwner(t *testing.T) {
	sp := NewServer()
	a, b := newRig(t, sp), &rig{t: t}
	srvB, cliB := Pipe()
	a.srv().Connect(srvB)
	b.server = a.server
	b.client = world.New().AddPlugins(NewClient(cliB))
	t.Cleanup(clientOf(b.client).Close)
	a.until("both welcomed", func() bool {
		b.client.RunFrames(1)
		return a.cli().ID() != 0 && b.cli().ID() != 0
	})
	mine := &chest{
		BasicEntity: world.NewBasic(),
		Transform:   scene.At(0, 0, 0),
		Network:     Network{Kind: "chest", Owner: a.cli().ID()},
		Purse:       Purse{Coins: 5},
	}
	a.server.AddEntities(mine)
	a.until("both see the chest", func() bool {
		b.client.RunFrames(1)
		_, okA := a.cli().Lookup(mine.NetID)
		_, okB := b.cli().Lookup(mine.NetID)
		return okA && okB
	})
	mine.Purse.Coins, mine.Lid.HP = 9, 4
	ma, _ := a.cli().Lookup(mine.NetID)
	mb, _ := b.cli().Lookup(mine.NetID)
	ca, _ := ma.(*chest)
	cb, _ := mb.(*chest)
	a.until("owner and public updates", func() bool {
		b.client.RunFrames(1)
		return ca.Purse.Coins == 9 && cb.Lid.HP == 4
	})
	if cb.Purse.Coins != 0 {
		t.Fatalf("non-owner received the purse: %+v", cb.Purse)
	}
}

func TestNonComparableStateIsRejected(t *testing.T) {
	r := newRig(t, NewServer())
	r.until("welcome", func() bool { return r.cli().ID() != 0 })
	defer func() {
		want := `net: "chest" replicates net.Bag, which is not comparable with ==`
		if got := recover(); got != want {
			t.Fatalf("a slice state must be rejected as it first replicates: %v", got)
		}
	}()
	r.server.AddEntities(
		&sack{chest{BasicEntity: world.NewBasic(), Transform: scene.At(0, 0, 0), Network: Network{Kind: "chest"}}},
	)
	r.server.RunFrames(3)
}

func TestInterestHysteresis(t *testing.T) {
	r := newRig(t, NewServer(Interest(10, 4)))
	r.until("welcome", func() bool { return r.cli().ID() != 0 })
	hero := newUnit(geom.V3(0, 0, 0), r.cli().ID())
	other := newUnit(geom.V3(9, 0, 0), 0)
	r.server.AddEntities(hero, other)
	r.until("spawn inside radius", func() bool { _, ok := r.cli().Lookup(other.NetID); return ok })
	other.Position = geom.V3(13, 50, 0)
	r.server.RunFrames(6)
	r.client.RunFrames(2)
	if _, ok := r.cli().Lookup(other.NetID); !ok {
		t.Fatal("entity inside the margin, at any height, must stay replicated")
	}
	other.Position = geom.V3(15, 0, 0)
	r.until("despawn beyond the margin", func() bool { _, ok := r.cli().Lookup(other.NetID); return !ok })
	other.Position = geom.V3(12, 0, 0)
	r.server.RunFrames(6)
	r.client.RunFrames(2)
	if _, ok := r.cli().Lookup(other.NetID); ok {
		t.Fatal("entity in the margin must not re-enter until within the radius")
	}
}

func TestFloodingClientIsDisconnected(t *testing.T) {
	r := newRig(t, NewServer(MaxIntents(4)))
	r.until("welcome", func() bool { return r.cli().ID() != 0 })
	for i := 0; i < 50; i++ {
		r.cli().Send(Ping{N: i})
	}
	time.Sleep(50 * time.Millisecond)
	r.until("kick", func() bool { return len(r.srv().Clients()) == 0 })
}

func TestSlowConnectionIsDroppedWhenOutboxIsFull(t *testing.T) {
	srv, cli := Pipe()
	defer cli.Close()
	p := newPeer(srv, nil)
	p.limit = 8
	for i := 0; i < 20; i++ {
		p.send(Ping{N: i})
	}
	select {
	case <-p.done:
	case <-time.After(time.Second):
		t.Fatal("peer should close once its outbox is full")
	}
	p.mu.Lock()
	n := len(p.outbox)
	p.mu.Unlock()
	if n > 8 {
		t.Fatalf("outbox grew to %d", n)
	}
}

func TestIntentsAreAppliedInOrderAndAcked(t *testing.T) {
	r := newRig(t, NewServer())
	var got []int
	Handle(r.srv(), func(_ ClientID, p Ping) { got = append(got, p.N) })
	r.until("welcome", func() bool { return r.cli().ID() != 0 })
	var last uint32
	for n := 1; n <= 3; n++ {
		last = r.cli().Send(Ping{N: n})
	}
	r.until("ack", func() bool { return r.cli().Acked() == last })
	if last != 3 || len(got) != 3 || got[0] != 1 || got[2] != 3 {
		t.Fatalf("seq %d, applied %v", last, got)
	}
	if r.srv().Ack(r.cli().ID()) != 3 || r.cli().Tick() == 0 {
		t.Fatalf("server ack %d, tick %d", r.srv().Ack(r.cli().ID()), r.cli().Tick())
	}
}

func TestIntentsWaitForTheFixedStep(t *testing.T) {
	r := newRig(t, NewServer())
	clock := r.server.Time()
	applied := false
	Handle(r.srv(), func(ClientID, Ping) {
		applied = true
		if clock.Delta != clock.FixedDelta {
			t.Errorf("intent applied outside FixedUpdate, dt = %v", clock.Delta)
		}
	})
	r.until("welcome", func() bool { return r.cli().ID() != 0 })
	r.cli().Send(Ping{N: 1})
	for deadline := time.Now().Add(3 * time.Second); !applied; {
		if time.Now().After(deadline) {
			t.Fatal("intent never applied")
		}
		r.server.Frame(clock.FixedDelta / 7)
		time.Sleep(time.Millisecond)
	}
}

func TestIDsCarryTheRegionAndResumeAfterACounter(t *testing.T) {
	id := NewID(7, 42)
	if id.Region() != 7 || id.Counter() != 42 {
		t.Fatalf("%x: region %d counter %d", uint64(id), id.Region(), id.Counter())
	}
	a := world.New().AddPlugins(NewServer(Region(3)))
	s := serverOf(a)
	s.Resume(1000)
	s.Resume(10)
	u := newUnit(geom.V3(0, 0, 0), 0)
	a.AddEntities(u)
	if u.NetID != NewID(3, 1001) || s.Counter() != 1001 {
		t.Fatalf("assigned %x, counter %d", uint64(u.NetID), s.Counter())
	}
	preset := newUnit(geom.V3(0, 0, 0), 0)
	preset.NetID = NewID(3, 1<<47)
	a.AddEntities(preset)
	if preset.NetID != NewID(3, 1<<47) {
		t.Fatal("a preset ID must be kept")
	}
}

func TestOwnershipChangesReachClientsThatAlreadyMirrorTheEntity(t *testing.T) {
	r := newRig(t, NewServer())
	r.until("welcome", func() bool { return r.cli().ID() != 0 })
	c := &chest{
		BasicEntity: world.NewBasic(),
		Transform:   scene.At(0, 0, 0),
		Network:     Network{Kind: "chest"},
		Purse:       Purse{Coins: 3},
	}
	r.server.AddEntities(c)
	r.until("spawn", func() bool { _, ok := r.cli().Lookup(c.NetID); return ok })
	if _, owned := r.cli().Owned(); owned {
		t.Fatal("nobody owns the chest yet")
	}
	c.Owner = r.cli().ID()
	r.until("ownership and private state", func() bool {
		m, ok := r.cli().Owned()
		ch, isChest := m.(*chest)
		return ok && isChest && ch.Purse.Coins == 3
	})
}

// gobFrames carries every message in its own frame, as a protobuf codec
// would.
func gobFrames() Frames {
	return Frames{
		Marshal: func(m any) ([]byte, error) {
			var b bytes.Buffer
			err := gob.NewEncoder(&b).Encode(&envelope{m})
			return b.Bytes(), err
		},
		Unmarshal: func(b []byte) (any, error) {
			var e envelope
			err := gob.NewDecoder(bytes.NewReader(b)).Decode(&e)
			return e.Msg, err
		},
	}
}

func TestFramesCodecCarriesTheWholeProtocol(t *testing.T) {
	srvConn, cliConn := Pipe()
	r := &rig{t: t, server: world.New(), client: world.New()}
	r.server.AddPlugins(NewServer(WithCodec(gobFrames())))
	r.srv().Connect(srvConn)
	r.client.AddPlugins(NewClient(cliConn, WithCodec(gobFrames())))
	t.Cleanup(r.cli().Close)
	var got Ping
	Handle(r.srv(), func(_ ClientID, p Ping) { got = p })
	u := newUnit(geom.V3(1, 0, 2), 0)
	r.server.AddEntities(u)
	r.until("spawn", func() bool { _, ok := r.cli().Lookup(u.NetID); return ok })
	seq := r.cli().Send(Ping{N: 5})
	r.until("intent and ack", func() bool { return got.N == 5 && r.cli().Acked() == seq })
}

func TestOversizedFramesCloseTheConnection(t *testing.T) {
	small := gobFrames()
	small.MaxSize = 256
	var w bytes.Buffer
	if err := small.NewEncoder(&w).Encode(Ping{N: 1}); err != nil {
		t.Fatal(err)
	}
	big := Snapshot{Despawns: make([]ID, 100)}
	for i := range big.Despawns {
		big.Despawns[i] = NewID(1, 1<<40+uint64(i))
	}
	if err := small.NewEncoder(io.Discard).Encode(big); err == nil {
		t.Fatal("an oversized frame must not be sent")
	}
	var huge bytes.Buffer
	huge.Write(binary.AppendUvarint(nil, 1<<30))
	if _, err := small.NewDecoder(&huge).Decode(); err == nil {
		t.Fatal("an oversized frame header must be refused before allocating")
	}
	if m, err := small.NewDecoder(&w).Decode(); err != nil || m != (Ping{N: 1}) {
		t.Fatalf("round trip: %v, %v", m, err)
	}
}

func TestDisconnectReportsTheLastAppliedIntent(t *testing.T) {
	r := newRig(t, NewServer())
	var left world.Reader[ClientDisconnected]
	var gone *ClientDisconnected
	r.server.On(world.Update, "left", func(world.Frame) {
		for ev := range left.Read(r.server) {
			gone = &ev
		}
	})
	r.until("welcome", func() bool { return r.cli().ID() != 0 })
	r.cli().Send(Ping{N: 1})
	seq := r.cli().Send(Ping{N: 2})
	r.until("ack", func() bool { return r.cli().Acked() == seq })
	r.cli().Close()
	r.until("disconnect", func() bool { return gone != nil })
	if gone.Ack != seq {
		t.Fatalf("disconnect acked %d, want %d", gone.Ack, seq)
	}
}

func TestSnapshotBudgetSendsOwnedThenNearestFirst(t *testing.T) {
	a := world.New().AddPlugins(NewServer(SnapshotBudget(3), Interest(100, 0)))
	s := serverOf(a)
	srvConn, cliConn := Pipe()
	defer cliConn.Close()
	r := &remote{id: 1, peer: newPeer(srvConn, nil), known: map[ID]*sent{}, joined: true}
	s.clients[r.id] = r
	hero := newUnit(geom.V3(0, 0, 0), 1)
	a.AddEntities(hero)
	var units []*unit
	for i := 8; i >= 1; i-- {
		u := newUnit(geom.V3(float32(i*5), 0, 0), 0)
		units = append([]*unit{u}, units...)
		a.AddEntities(u)
	}
	dec := gob.NewDecoder(cliConn)
	next := func() Snapshot {
		t.Helper()
		done := make(chan Snapshot)
		go func() {
			var e envelope
			if err := dec.Decode(&e); err != nil {
				t.Error(err)
			}
			snap, _ := e.Msg.(Snapshot)
			done <- snap
		}()
		s.replicate(1)
		select {
		case snap := <-done:
			return snap
		case <-time.After(time.Second):
			t.Fatal("no snapshot")
		}
		return Snapshot{}
	}
	ids := func(spawns []Spawn) []ID {
		var out []ID
		for _, sp := range spawns {
			out = append(out, sp.ID)
		}
		return out
	}
	first := next()
	if got, want := ids(first.Spawns), []ID{hero.NetID, units[0].NetID, units[1].NetID}; !slices.Equal(got, want) {
		t.Fatalf("first snapshot spawned %v, want the hero then the two nearest %v", got, want)
	}
	second := next()
	next3 := []ID{units[2].NetID, units[3].NetID, units[4].NetID}
	if got := ids(second.Spawns); !slices.Equal(got, next3) {
		t.Fatalf("second snapshot spawned %v, want the next nearest %v", got, next3)
	}
	a.RemoveEntity(units[7])
	for _, u := range units[:5] {
		u.Vitals.HP = 1
	}
	third := next()
	if len(third.Spawns)+len(third.Updates) != 3 || len(third.Despawns) != 0 {
		t.Fatalf("third snapshot: %d spawns, %d updates, %d despawns",
			len(third.Spawns), len(third.Updates), len(third.Despawns))
	}
}

func TestKickDeliversWhatWasSentBefore(t *testing.T) {
	r := newRig(t, NewServer())
	var got []Pong
	HandleClient(r.cli(), func(p Pong) { got = append(got, p) })
	r.until("welcome", func() bool { return r.cli().ID() != 0 })
	r.srv().Send(r.cli().ID(), Pong{N: 1})
	r.srv().Send(r.cli().ID(), Pong{N: 2})
	r.srv().Kick(r.cli().ID())
	r.until("the kick", func() bool { return !r.cli().Connected() })
	if len(got) != 2 {
		t.Fatalf("the kicked client received %v before the connection closed", got)
	}
}

func TestEveryClientHandlerSeesItsMessages(t *testing.T) {
	r := newRig(t, NewServer())
	var a, b []int
	HandleClient(r.cli(), func(p Pong) { a = append(a, p.N) })
	HandleClient(r.cli(), func(p Pong) { b = append(b, p.N) })
	r.until("welcome", func() bool { return r.cli().ID() != 0 })
	r.srv().Send(r.cli().ID(), Pong{N: 4})
	r.until("both handlers", func() bool { return len(a) == 1 && len(b) == 1 })
}

func TestCloseAllDeliversWhatWasSentThenCloses(t *testing.T) {
	r := newRig(t, NewServer())
	var got Pong
	HandleClient(r.cli(), func(p Pong) { got = p })
	r.until("welcome", func() bool { return r.cli().ID() != 0 })
	r.srv().Send(r.cli().ID(), Pong{N: 7})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	r.srv().CloseAll(ctx)
	if ctx.Err() != nil {
		t.Fatal("CloseAll waited out its deadline")
	}
	for i := 0; i < 100 && r.cli().Connected(); i++ {
		r.client.RunFrames(1)
		time.Sleep(time.Millisecond)
	}
	if got.N != 7 || r.cli().Connected() {
		t.Fatalf("got %v, connected %v: the last message must arrive before the close", got, r.cli().Connected())
	}
}

func serverOf(w *world.World) *Server { s, _ := netside.Server(w).(*Server); return s }
func clientOf(w *world.World) *Client { c, _ := netside.Client(w).(*Client); return c }

// An entity embedding *world.BasicEntity and Network stays a world.Identifier:
// a Network field named like BasicEntity's ID method would make ID ambiguous.
var _ world.Identifier = &struct {
	*world.BasicEntity
	Network
}{}
