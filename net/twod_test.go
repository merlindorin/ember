package net //nolint:testpackage // white-box: shares the package's test rigs

import (
	"math"
	"testing"
	"time"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/world"
)

type Walk2D struct{ V geom.Vec2 }

// flat is a 2D placement with the methods a twod.Item has, which this
// package's own tests can't import (twod reaches net through automation);
// twod_ext_test.go replicates a real twod.Item.
type flat struct {
	at, scale geom.Vec2
	turn      float32
}

func (f *flat) Position() geom.Vec2       { return f.at }
func (f *flat) SetPosition(at geom.Vec2)  { f.at = at }
func (f *flat) Translate(d geom.Vec2)     { f.at = f.at.Add(d) }
func (f *flat) Rotation() float32         { return f.turn }
func (f *flat) SetRotation(r float32)     { f.turn = r }
func (f *flat) Scale() geom.Vec2          { return f.scale }
func (f *flat) SetScale(s geom.Vec2)      { f.scale = s }
func (f *flat) GlobalPosition() geom.Vec2 { return f.at }

// mover is a 2D entity that walks at a velocity only its owner and the
// server know.
type mover struct {
	*world.BasicEntity
	flat
	Network
	v geom.Vec2
}

func (m *mover) PredictIntent(msg any) {
	if w, ok := msg.(Walk2D); ok {
		m.v = w.V
	}
}

func (m *mover) PredictStep(dt float32) { m.Translate(m.v.Scale(dt)) }

func newMover(at geom.Vec2, owner ClientID) *mover {
	return &mover{
		BasicEntity: world.NewBasic(),
		flat:        flat{at: at, scale: geom.V2(1, 1)},
		Network:     Network{Kind: "mover", Owner: owner},
	}
}

func registerMover() {
	Register(Walk2D{})
	RegisterKind("mover", func() Replicated { return newMover(geom.Vec2{}, 0) })
}

type movers struct{ world.Group[*mover] }

func (*movers) Phase() world.Phase { return world.FixedUpdate }
func (s *movers) Update(f world.Frame) {
	for m := range s.All() {
		m.Translate(m.v.Scale(f.DT))
	}
}

func TestA2DEntityReplicatesItsPlacement(t *testing.T) {
	r := newRig(t, NewServer())
	m := newMover(geom.V2(100, 40), 0)
	m.SetRotation(0.5)
	m.SetScale(geom.V2(2, 3))
	r.server.AddEntities(m)

	var mirror *mover
	r.until("spawn", func() bool {
		e, ok := r.cli().Lookup(m.NetID)
		mirror, _ = e.(*mover)
		return ok
	})
	if mirror == nil {
		t.Fatal("the mirror is not a mover")
	}
	if mirror.Position().Dist(geom.V2(100, 40)) > 1e-3 || !near(mirror.Rotation(), 0.5) ||
		mirror.Scale().Dist(geom.V2(2, 3)) > 1e-3 {
		t.Fatalf("mirror at %v turned %v scaled %v", mirror.Position(), mirror.Rotation(), mirror.Scale())
	}

	m.SetPosition(geom.V2(140, -20))
	m.SetRotation(-1)
	r.until("the new placement", func() bool {
		return mirror.Position().Dist(geom.V2(140, -20)) < 1e-2 && near(mirror.Rotation(), -1)
	})
}

func near(a, b float32) bool { return math.Abs(float64(a-b)) < 1e-3 }

func TestInterestIsMeasuredOnThe2DPlane(t *testing.T) {
	r := newRig(t, NewServer(Interest(50, 0)))
	r.until("welcome", func() bool { return r.cli().ID() != 0 })
	hero := newMover(geom.V2(0, 0), r.cli().ID())
	far := newMover(geom.V2(0, 200), 0)
	r.server.AddEntities(hero, far)

	r.until("hero spawn", func() bool { _, ok := r.cli().Lookup(hero.NetID); return ok })
	r.server.RunFrames(6)
	r.client.RunFrames(2)
	if _, ok := r.cli().Lookup(far.NetID); ok {
		t.Fatal("an entity 200 below the hero must not be replicated within 50")
	}

	hero.SetPosition(geom.V2(0, 180))
	r.until("far spawn", func() bool { _, ok := r.cli().Lookup(far.NetID); return ok })

	hero.SetPosition(geom.V2(0, 0))
	r.until("far despawn", func() bool { _, ok := r.cli().Lookup(far.NetID); return !ok })
}

// walk2D runs a client predicting a mover sent at v px/s and returns the
// mover's x on every rendered frame of the second second.
func walk2D(t *testing.T, v float32, opts ...ClientOption) []float32 {
	t.Helper()
	l := newLockstep(t, 30, 40*time.Millisecond, 0, opts...)
	l.srv.AddSystems(&movers{})
	m := newMover(geom.V2(0, 0), l.cliRes().ID())
	l.srv.AddEntities(m)
	Handle(serverOf(l.srv), func(_ ClientID, w Walk2D) { m.v = w.V })
	var mine *mover
	for mine == nil {
		l.step()
		if e, ok := l.cliRes().Owned(); ok {
			mine, _ = e.(*mover)
		}
	}
	l.cliRes().Send(Walk2D{V: geom.V2(v, 0)})
	l.run(time.Second)
	var xs []float32
	for end := l.clk.Now().Add(time.Second); l.clk.Now().Before(end); {
		l.step()
		xs = append(xs, mine.Position().X)
	}
	return xs
}

func TestAPredicted2DWalkerMovesEveryRenderedFrame(t *testing.T) {
	const speed = 120
	want := speed / float64(lockstepFPS)
	for i, d := range steps(walk2D(t, speed, SnapDistance(32))) {
		if d <= 0 || math.Abs(d-want) > want/2 {
			t.Fatalf("frame %d moved %.3f px, want about %.3f: a walk is blended, never snapped", i, d, want)
		}
	}
	stood := 0
	for _, d := range steps(walk2D(t, speed)) {
		if d == 0 {
			stood++
		}
	}
	if stood == 0 {
		t.Fatal("at the default snap distance of 1, a 4 px step is a jump: the walk should stair-step")
	}
}

func TestAPredictedJumpBeyondTheSnapDistanceShowsWhole(t *testing.T) {
	const speed = 3000
	most := 0.0
	for _, d := range steps(walk2D(t, speed, SnapDistance(32))) {
		most = math.Max(most, d)
	}
	if most < 90 {
		t.Fatalf("the longest frame moved %.1f px: 100 px steps must show whole, not blended over the step", most)
	}
}

type noPose struct {
	*world.BasicEntity
	Network
}

func TestAnEntityWithoutAPoseIsRejected(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("replicating an entity with neither a scene.Transform nor a twod placement must panic")
		}
	}()
	w := world.New().AddPlugins(NewServer())
	w.AddEntities(&noPose{BasicEntity: world.NewBasic(), Network: Network{Kind: "posless"}})
	w.RunFrames(1)
}
