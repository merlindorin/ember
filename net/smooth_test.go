package net //nolint:testpackage // white-box: Client.mirrors and the package's test rigs

import (
	"math"
	"testing"
	"time"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/scene"
	"github.com/merlindorin/ember/world"
)

type testClock struct{ now time.Time }

func (c *testClock) Now() time.Time          { return c.now }
func (c *testClock) Advance(d time.Duration) { c.now = c.now.Add(d) }

// lockstep runs a 30 Hz server and a client rendering at lockstepFPS, on
// one virtual clock, through a link with the given delay and jitter.
type lockstep struct {
	t        *testing.T
	clk      *testClock
	srv, cli *world.World
	frame    time.Duration
}

const lockstepFPS = 144

func newLockstep(t *testing.T, snapshots int, delay, jitter time.Duration, opts ...ClientOption) *lockstep {
	l := &lockstep{t: t, clk: &testClock{now: time.Unix(1_800_000_000, 0)}, frame: time.Second / lockstepFPS}
	se, ce := LagJitter(l.clk.Now, delay, jitter, 7)
	l.srv = world.New()
	l.srv.Time().FixedDelta = 1.0 / 30
	l.srv.AddPlugins(NewServer(TickRate(snapshots)))
	l.srv.AddSystems(&walkers{})
	serverOf(l.srv).ConnectEndpoint(se)
	opts = append([]ClientOption{WithEndpoint(ce), Clock(l.clk.Now), TickRate(30)}, opts...)
	l.cli = world.New().AddPlugins(NewClient(nil, opts...))
	l.run(200 * time.Millisecond)
	return l
}

func (l *lockstep) cliRes() *Client { return clientOf(l.cli) }

func (l *lockstep) step() {
	dt := float32(l.frame.Seconds())
	l.srv.Frame(dt)
	l.cli.Frame(dt)
	l.clk.Advance(l.frame)
}

func (l *lockstep) run(d time.Duration) {
	for end := l.clk.Now().Add(d); l.clk.Now().Before(end); {
		l.step()
	}
}

// track records x of a mirror each rendered frame.
func (l *lockstep) track(e *walker, d time.Duration) []float32 {
	var xs []float32
	for end := l.clk.Now().Add(d); l.clk.Now().Before(end); {
		l.step()
		xs = append(xs, e.Position.X)
	}
	return xs
}

func steps(xs []float32) []float64 {
	out := make([]float64, 0, len(xs)-1)
	for i := 1; i < len(xs); i++ {
		out = append(out, float64(xs[i]-xs[i-1]))
	}
	return out
}

func meanStd(v []float64) (float64, float64) {
	var sum, sq float64
	for _, x := range v {
		sum += x
	}
	m := sum / float64(len(v))
	for _, x := range v {
		sq += (x - m) * (x - m)
	}
	return m, math.Sqrt(sq / float64(len(v)))
}

func TestTheOwnedEntityMovesEveryRenderedFrame(t *testing.T) {
	l := newLockstep(t, 30, 40*time.Millisecond, 0)
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
	l.run(time.Second)
	want := 3 * l.frame.Seconds()
	for i, d := range steps(l.track(mine, time.Second)) {
		if d <= 0 || math.Abs(d-want) > want/2 {
			t.Fatalf("frame %d moved %.4f m, want about %.4f: the display must not stair-step at the 30 Hz prediction rate",
				i, d, want)
		}
	}
}

func TestARemoteWalkerIsShownAtAConstantSpeed(t *testing.T) {
	l := newLockstep(t, 20, 20*time.Millisecond, 60*time.Millisecond)
	w := &walker{BasicEntity: world.NewBasic(), Transform: scene.At(0, 0, 0), Network: Network{Kind: "walker"}, v: 4}
	l.srv.AddEntities(w)
	var shown *walker
	for shown == nil {
		l.step()
		for _, m := range l.cliRes().mirrors {
			shown, _ = m.e.(*walker)
		}
	}
	l.run(time.Second)
	m, sd := meanStd(steps(l.track(shown, 5*time.Second)))
	want := 4 * l.frame.Seconds()
	if math.Abs(m-want) > want*0.1 || sd > want*0.25 {
		t.Fatalf("per-frame steps average %.4f m (want %.4f) with spread %.4f: remote motion must be even", m, want, sd)
	}
	if lag := w.Position.X - shown.Position.X; lag < 0 || lag > 4*0.35 {
		t.Fatalf("the remote walker is shown %.2f m behind the server", lag)
	}
}

func TestARemoteTeleportSnaps(t *testing.T) {
	l := newLockstep(t, 20, 50*time.Millisecond, 30*time.Millisecond)
	w := &walker{BasicEntity: world.NewBasic(), Transform: scene.At(0, 0, 0), Network: Network{Kind: "walker"}, v: 2}
	l.srv.AddEntities(w)
	var shown *walker
	for shown == nil {
		l.step()
		for _, m := range l.cliRes().mirrors {
			shown, _ = m.e.(*walker)
		}
	}
	l.run(time.Second)
	w.Position.X += 50
	from := w.Position.X - 50
	for _, x := range l.track(shown, time.Second) {
		if x > from+3 && x < from+47 {
			t.Fatalf("the teleport was shown in flight at %.2f", x)
		}
	}
	if geom.Abs(shown.Position.X-w.Position.X) > 1.5 {
		t.Fatalf("after the teleport the mirror sits at %.2f, the server at %.2f", shown.Position.X, w.Position.X)
	}
}

func TestARemoteWalkerStopsWithoutOvershooting(t *testing.T) {
	l := newLockstep(t, 20, 20*time.Millisecond, 60*time.Millisecond)
	w := &walker{BasicEntity: world.NewBasic(), Transform: scene.At(0, 0, 0), Network: Network{Kind: "walker"}, v: 4}
	l.srv.AddEntities(w)
	var shown *walker
	for shown == nil {
		l.step()
		for _, m := range l.cliRes().mirrors {
			shown, _ = m.e.(*walker)
		}
	}
	for range 5 {
		l.run(700 * time.Millisecond)
		w.v = 0
		stop := w.Position.X
		prev := shown.Position.X
		for _, x := range l.track(shown, 500*time.Millisecond) {
			if x > stop+1e-3 || x < prev-1e-3 {
				t.Fatalf("shown at %.3f after %.3f, stopped at %.3f: a stop must not overshoot nor step back",
					x, prev, stop)
			}
			prev = x
		}
		w.v = 4
	}
}

func TestTheShownClockSlewsInsteadOfJumping(t *testing.T) {
	var tl timeline
	start := time.Unix(1_800_000_000, 0)
	const step = float32(1.0 / 30)
	tl.observe(30, start, step)
	tl.advance(start, 1.0/144, step)
	tl.offset -= 0.1
	prev := tl.shown
	for i := 1; i <= 144; i++ {
		now := start.Add(time.Duration(i) * time.Second / 144)
		got := tl.advance(now, 1.0/144, step)
		ticks := got - prev
		if want := (1.0 / 144) / float64(step); ticks > want*1.11 || ticks < want*0.89 {
			t.Fatalf("frame %d moved the clock %.4f ticks, real time is %.4f", i, ticks, want)
		}
		prev = got
	}
	if gap := tl.renderTick(start.Add(time.Second), step) - tl.shown; gap > 3*0.9 || gap < 0 {
		t.Fatalf("after a second the shown clock is %.2f ticks behind its target, want it catching up from 3", gap)
	}
}
