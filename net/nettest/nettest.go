// Package nettest checks that a transport carries the whole net protocol:
// replication, owner-only state, intents and acks, typed messages and
// disconnects. Every transport runs the same suite:
//
//	nettest.Run(t, func(t *testing.T) (server, client stdnet.Conn) { ... })
package nettest

import (
	"fmt"
	stdnet "net"
	"testing"
	"time"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/net"
	"github.com/merlindorin/ember/scene"
	"github.com/merlindorin/ember/world"
)

// Connect returns the two ends of one connection.
type Connect func(t *testing.T) (server, client stdnet.Conn)

type Vitals struct{ HP int }

type Purse struct{ Coins int }

// Unit is a replicated entity with public and owner-only state.
type Unit struct {
	*world.BasicEntity
	scene.Transform
	net.Network
	Vitals Vitals
	Purse  Purse
}

func (u *Unit) NetState() any { return u.Vitals }

func (u *Unit) ApplyNetState(s any) {
	v, ok := s.(Vitals)
	if !ok {
		panic(fmt.Sprintf("nettest: Unit's net state is a Vitals, not a %T", s))
	}
	u.Vitals = v
}

func (u *Unit) OwnerState() any { return u.Purse }

func (u *Unit) ApplyOwnerState(s any) {
	p, ok := s.(Purse)
	if !ok {
		panic(fmt.Sprintf("nettest: Unit's owner state is a Purse, not a %T", s))
	}
	u.Purse = p
}

func NewUnit(pos geom.Vec3, owner net.ClientID) *Unit {
	return &Unit{
		BasicEntity: world.NewBasic(),
		Transform:   scene.AtVec(pos),
		Network:     net.Network{Kind: kind, Owner: owner},
		Vitals:      Vitals{HP: 10},
	}
}

type Ping struct{ N int }

type Pong struct{ N int }

const kind = "nettest.unit"

func init() { //nolint:gochecknoinits // registers the suite's types in a shared init, as net.Register asks
	net.Register(Vitals{})
	net.Register(Purse{})
	net.RegisterKind(kind, func() net.Replicated {
		return &Unit{BasicEntity: world.NewBasic(), Transform: scene.At(0, 0, 0)}
	})
}

type rig struct {
	t       *testing.T
	server  *world.World
	srv     *net.Server
	clients []*world.World
}

func newRig(t *testing.T) *rig {
	r := &rig{t: t, server: world.New(), srv: net.NewServer(net.SnapshotBudget(4096))}
	r.server.AddPlugins(r.srv)
	return r
}

func (r *rig) connect(c Connect) *net.Client {
	sc, cc := c(r.t)
	r.srv.Connect(sc)
	client := net.NewClient(cc)
	r.t.Cleanup(client.Close)
	r.clients = append(r.clients, world.New().AddPlugins(client))
	return client
}

func (r *rig) until(what string, cond func() bool) {
	r.t.Helper()
	r.untilProgress(what, cond, func() int { return 0 })
}

// untilProgress waits like until, but gives cond 5 more seconds each time
// progress changes, so a long transfer that keeps moving is not cut short.
func (r *rig) untilProgress(what string, cond func() bool, progress func() int) {
	r.t.Helper()
	last := progress()
	for deadline := time.Now().Add(5 * time.Second); !cond(); {
		if p := progress(); p != last {
			last, deadline = p, time.Now().Add(5*time.Second)
		}
		if time.Now().After(deadline) {
			r.t.Fatalf("timed out waiting for %s", what)
		}
		r.server.RunFrames(1)
		time.Sleep(time.Millisecond)
		for _, c := range r.clients {
			c.RunFrames(1)
		}
	}
}

// Run runs the suite over connections made by connect.
func Run(t *testing.T, connect Connect) {
	t.Run("ReplicatesSpawnsStateAndDespawns", func(t *testing.T) { replicates(t, connect) })
	t.Run("IntentsAndMessagesBothWays", func(t *testing.T) { messages(t, connect) })
	t.Run("OwnerStateReachesOnlyItsOwner", func(t *testing.T) { ownerState(t, connect) })
	t.Run("LargeSnapshots", func(t *testing.T) { largeSnapshots(t, connect) })
	t.Run("DisconnectsAreSeenOnBothSides", func(t *testing.T) { disconnects(t, connect) })
}

func replicates(t *testing.T, connect Connect) {
	r := newRig(t)
	c := r.connect(connect)
	u := NewUnit(geom.V3(1, 0, 2), 0)
	r.server.AddEntities(u)
	var mirror *Unit
	r.until("spawn", func() bool {
		if m, ok := c.Lookup(u.NetID); ok {
			mirror, _ = m.(*Unit)
		}
		return mirror != nil
	})
	u.Position, u.Vitals.HP = geom.V3(5, 0, 2), 3
	r.until("pose and state", func() bool {
		return mirror.Position.ApproxEq(geom.V3(5, 0, 2), 1e-2) && mirror.Vitals.HP == 3
	})
	r.server.RemoveEntity(u)
	r.until("despawn", func() bool { _, ok := c.Lookup(u.NetID); return !ok })
}

func messages(t *testing.T, connect Connect) {
	r := newRig(t)
	c := r.connect(connect)
	var pings []int
	net.Handle(r.srv, func(from net.ClientID, p Ping) {
		pings = append(pings, p.N)
		r.srv.Send(from, Pong{N: p.N * 2})
	})
	var pong Pong
	net.HandleClient(c, func(p Pong) { pong = p })
	r.until("welcome", func() bool { return c.ID() != 0 })
	var last uint32
	for n := 1; n <= 20; n++ {
		last = c.Send(Ping{N: n})
	}
	r.until("acks and replies", func() bool { return c.Acked() == last && pong.N == 40 })
	for i, n := range pings {
		if n != i+1 {
			t.Fatalf("intents applied out of order: %v", pings)
		}
	}
}

func ownerState(t *testing.T, connect Connect) {
	r := newRig(t)
	a, b := r.connect(connect), r.connect(connect)
	r.until("welcomes", func() bool { return a.ID() != 0 && b.ID() != 0 })
	u := NewUnit(geom.V3(0, 0, 0), a.ID())
	u.Purse.Coins = 5
	r.server.AddEntities(u)
	r.until("both mirrors", func() bool {
		_, okA := a.Lookup(u.NetID)
		_, okB := b.Lookup(u.NetID)
		return okA && okB
	})
	ma, _ := a.Lookup(u.NetID)
	mb, _ := b.Lookup(u.NetID)
	r.until("the owner's purse", func() bool { m, ok := ma.(*Unit); return ok && m.Purse.Coins == 5 })
	u.Owner = b.ID()
	r.until("the handover", func() bool {
		o, ok := b.Owned()
		_, still := a.Owned()
		m, isUnit := mb.(*Unit)
		return ok && o == mb && isUnit && m.Purse.Coins == 5 && !still
	})
}

func largeSnapshots(t *testing.T, connect Connect) {
	r := newRig(t)
	c := r.connect(connect)
	r.until("welcome", func() bool { return c.ID() != 0 })
	units := make([]world.Identifier, 3000)
	var last *Unit
	for i := range units {
		last = NewUnit(geom.V3(float32(i%50), 0, float32(i/50)), 0)
		units[i] = last
	}
	r.server.AddEntities(units...)
	r.untilProgress("every unit", func() bool {
		_, ok := c.Lookup(last.NetID)
		return ok && r.clients[0].Len() == len(units)
	}, r.clients[0].Len)
}

func disconnects(t *testing.T, connect Connect) {
	r := newRig(t)
	var left world.Reader[net.ClientDisconnected]
	var gone []net.ClientDisconnected
	r.server.On(world.Update, "left", func(world.Frame) {
		for ev := range left.Read(r.server) {
			gone = append(gone, ev)
		}
	})
	quits, kicked := r.connect(connect), r.connect(connect)
	r.until("welcomes", func() bool { return quits.ID() != 0 && kicked.ID() != 0 })
	seq := quits.Send(Ping{N: 1})
	r.until("ack", func() bool { return quits.Acked() == seq })
	quits.Close()
	r.until("the server to see the client leave", func() bool { return len(gone) == 1 })
	if gone[0].ID != quits.ID() || gone[0].Ack != seq {
		t.Fatalf("disconnect %+v, want client %d with ack %d", gone[0], quits.ID(), seq)
	}
	r.srv.Kick(kicked.ID())
	r.until("the client to see the kick", func() bool { return !kicked.Connected() && len(gone) == 2 })
}
