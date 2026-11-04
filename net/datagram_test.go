package net //nolint:testpackage // white-box: a mirror's samples and Client.dispatch

import (
	"math"
	"testing"
	"time"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/scene"
	"github.com/merlindorin/ember/world"
)

func newLossy(t *testing.T, o LagOptions) *lockstep {
	l := &lockstep{t: t, clk: &testClock{now: time.Unix(1_800_000_000, 0)}, frame: time.Second / 144}
	se, ce := LagWith(l.clk.Now, o)
	l.srv = world.New()
	l.srv.Time().FixedDelta = 1.0 / 30
	l.srv.AddPlugins(NewServer(TickRate(20)))
	l.srv.AddSystems(&walkers{})
	serverOf(l.srv).ConnectEndpoint(se)
	l.cli = world.New().AddPlugins(NewClient(nil, WithEndpoint(ce), Clock(l.clk.Now), TickRate(30)))
	l.run(200 * time.Millisecond)
	return l
}

func (l *lockstep) remote(w *walker) *walker {
	l.srv.AddEntities(w)
	for n := 0; n < 1000; n++ {
		l.step()
		if m, ok := l.cliRes().mirrors[w.NetID]; ok {
			mirrored, _ := m.e.(*walker)
			return mirrored
		}
	}
	l.t.Fatal("never replicated")
	return nil
}

func lossy(loss float64, datagrams bool) LagOptions {
	return LagOptions{
		Delay:     40 * time.Millisecond,
		Jitter:    30 * time.Millisecond,
		Loss:      loss,
		Seed:      3,
		Datagrams: datagrams,
	}
}

func TestPosesTravelAsDatagramsAndArriveDespiteLoss(t *testing.T) {
	l := newLossy(t, lossy(0.05, true))
	w := &walker{BasicEntity: world.NewBasic(), Transform: scene.At(0, 0, 0), Network: Network{Kind: "walker"}, v: 4}
	shown := l.remote(w)
	if !serverOf(l.srv).clients[l.cliRes().ID()].peer.datagrams() {
		t.Fatal("the lossy link offers no datagrams")
	}
	for i := range 20 {
		l.run(time.Duration(300+i*37) * time.Millisecond)
		w.v = 0
		l.run(time.Second)
		if shown.Position.X != w.Position.X {
			t.Fatalf("stop %d: shown at %.3f, stopped at %.3f: the last pose must arrive despite loss",
				i, shown.Position.X, w.Position.X)
		}
		w.v = 4
	}
}

func TestReorderedDatagramsNeverMoveAMirrorBack(t *testing.T) {
	o := lossy(0.05, true)
	o.Jitter = 120 * time.Millisecond
	l := newLossy(t, o)
	w := &walker{BasicEntity: world.NewBasic(), Transform: scene.At(0, 0, 0), Network: Network{Kind: "walker"}, v: 4}
	shown := l.remote(w)
	prev := shown.Position.X
	for _, x := range l.track(shown, 5*time.Second) {
		if x < prev-1e-4 {
			t.Fatalf("shown at %.3f after %.3f: an older datagram moved the mirror back", x, prev)
		}
		prev = x
	}
}

func TestStateStaysReliableWhenDatagramsAreLost(t *testing.T) {
	l := newLossy(t, lossy(0.2, true))
	u := newUnit(geom.Vec3{}, 0)
	l.srv.AddEntities(u)
	for range 300 {
		l.step()
	}
	for hp := 1; hp <= 30; hp++ {
		u.Vitals.HP = hp
		l.step()
	}
	l.run(2 * time.Second)
	m, ok := l.cliRes().mirrors[u.NetID]
	var mirrored *unit
	if ok {
		mirrored, ok = m.e.(*unit)
	}
	if !ok || mirrored.Vitals.HP != 30 {
		t.Fatalf("mirror %v: the last state must arrive", m)
	}
}

func TestTheOwnedEntityAgreesWithTheServerOverDatagrams(t *testing.T) {
	l := newLossy(t, lossy(0.05, true))
	w := &walker{
		BasicEntity: world.NewBasic(),
		Transform:   scene.At(0, 0, 0),
		Network:     Network{Kind: "walker", Owner: l.cliRes().ID()},
	}
	l.srv.AddEntities(w)
	Handle(serverOf(l.srv), func(_ ClientID, m Walk) { w.v = m.V })
	var mine *walker
	for mine == nil {
		l.step()
		if m, ok := l.cliRes().Owned(); ok {
			mine, _ = m.(*walker)
		}
	}
	l.cliRes().Send(Walk{V: 3})
	l.run(2 * time.Second)
	l.cliRes().Send(Walk{V: 0})
	l.run(time.Second)
	if d := math.Abs(float64(mine.Position.X - w.Position.X)); d > 1e-2 {
		t.Fatalf("the owned entity rests %.3f m from the server", d)
	}
}

// Under loss, datagram poses keep remote motion even where a reliable
// stream stalls behind each lost packet.
func TestDatagramsBeatAReliableStreamUnderLoss(t *testing.T) {
	spread := func(o LagOptions) float64 {
		l := newLossy(t, o)
		w := &walker{
			BasicEntity: world.NewBasic(),
			Transform:   scene.At(0, 0, 0),
			Network:     Network{Kind: "walker"},
			v:           4,
		}
		shown := l.remote(w)
		l.run(time.Second)
		_, sd := meanStd(steps(l.track(shown, 10*time.Second)))
		return sd / (4 * l.frame.Seconds())
	}
	for _, loss := range []float64{0.01, 0.05} {
		stream, dgram := spread(lossy(loss, false)), spread(lossy(loss, true))
		t.Logf("%.0f%% loss: per-frame spread %.0f%% over the stream, %.0f%% over datagrams",
			loss*100, stream*100, dgram*100)
		if dgram > 0.35 || dgram >= stream {
			t.Errorf("%.0f%% loss: datagrams %.0f%%, stream %.0f%%", loss*100, dgram*100, stream*100)
		}
	}
}

func TestAPoseArrivingBeforeItsSpawnIsKept(t *testing.T) {
	clk := &testClock{now: time.Unix(1_800_000_000, 0)}
	_, ce := LagWith(clk.Now, LagOptions{Datagrams: true})
	cli := world.New().AddPlugins(NewClient(nil, WithEndpoint(ce), Clock(clk.Now), TickRate(30)))
	c := clientOf(cli)
	at := func(x float32) Pose {
		return Pose{Position: geom.V3(x, 0, 0), Rotation: geom.QuatIdent(), Scale: geom.V3(1, 1, 1)}
	}
	c.dispatch(Poses{Seq: 1, Tick: 12, Updates: []Update{{ID: 9, Pose: at(5)}}})
	c.dispatch(Snapshot{Tick: 10, Spawns: []Spawn{{ID: 9, Kind: "walker", Pose: at(1)}}})
	m := c.mirrors[9]
	if m == nil || m.target.Position.X != 5 || m.samples[len(m.samples)-1].tick != 12 {
		t.Fatalf("mirror %+v: the pose that came before the spawn was lost", m)
	}
	if c.dgLast != 1 || !c.ackDue {
		t.Fatal("the datagram was not acknowledged")
	}
}

func TestAcknowledgementsMergeOutOfOrder(t *testing.T) {
	var r remote
	r.acknowledge(PosesAck{Last: 10, Mask: 0b101})
	r.acknowledge(PosesAck{Last: 7, Mask: 0b1})
	for seq, want := range map[uint32]bool{10: true, 9: true, 8: false, 7: true, 6: true, 11: false} {
		if r.delivered(seq) != want {
			t.Errorf("seq %d delivered %v, want %v", seq, !want, want)
		}
	}
	r.acknowledge(PosesAck{Last: 12})
	if !r.delivered(10) || !r.delivered(12) || r.delivered(11) {
		t.Fatal("a newer acknowledgement dropped the older ones")
	}
}

func TestAnAcknowledgementOfTheSameLastAddsItsMask(t *testing.T) {
	var r remote
	r.acknowledge(PosesAck{Last: 5})
	r.acknowledge(PosesAck{Last: 5, Mask: 0b1})
	if !r.delivered(4) {
		t.Fatal("a later acknowledgement of the same datagram lost its mask")
	}
}

func TestAnOlderPoseOfTheOwnedEntityIsIgnored(t *testing.T) {
	clk := &testClock{now: time.Unix(1_800_000_000, 0)}
	_, ce := LagWith(clk.Now, LagOptions{Datagrams: true})
	cli := world.New().AddPlugins(NewClient(nil, WithEndpoint(ce), Clock(clk.Now), TickRate(30)))
	c := clientOf(cli)
	at := func(x float32) Pose {
		return Pose{Position: geom.V3(x, 0, 0), Rotation: geom.QuatIdent(), Scale: geom.V3(1, 1, 1)}
	}
	c.dispatch(Welcome{ID: 4})
	c.dispatch(Snapshot{Tick: 10, Spawns: []Spawn{{ID: 9, Kind: "walker", Owner: 4, Pose: at(1)}}})
	c.dispatch(Poses{Seq: 2, Tick: 20, Updates: []Update{{ID: 9, Pose: at(5)}}})
	c.dispatch(Poses{Seq: 1, Tick: 18, Updates: []Update{{ID: 9, Pose: at(3)}}})
	if m := c.mirrors[9]; c.self != m || m.target.Position.X != 5 || c.selfTick != 20 {
		t.Fatalf("self target %v at tick %d: an older pose was taken over a newer one", m.target.Position, c.selfTick)
	}
}
