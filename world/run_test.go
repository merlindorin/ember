package world //nolint:testpackage // white-box: shares the unexported mover helper of world_test.go

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type frames struct{ seen []Frame }

func (f *frames) Update(fr Frame) { f.seen = append(f.seen, fr) }
func (f *frames) Phase() Phase    { return FixedUpdate }

func TestSystemsReceiveTheFrame(t *testing.T) {
	w := New()
	fixed := &frames{}
	var update []Frame
	w.AddSystems(fixed).On(Update, "watch", func(f Frame) { update = append(update, f) })
	w.RunFrames(3)
	require.Len(t, update, 3)
	assert.InDelta(t, 1.0/60, update[0].DT, 1e-6)
	assert.InDelta(t, 3.0/60, update[2].Elapsed, 1e-5)
	assert.Same(t, w, update[0].World)
	require.Len(t, fixed.seen, 3)
	assert.Equal(t, uint64(2), fixed.seen[2].FixedFrame, "counted from zero")
	assert.Equal(t, uint64(3), update[2].FixedFrame)
	assert.Equal(t, "frames", w.Timings()[0].Name, "names drop the package")
	assert.Equal(t, "watch", w.Timings()[1].Name)
}

func TestTheFixedStepAccumulates(t *testing.T) {
	w := New()
	fixed := &frames{}
	w.AddSystem(fixed)
	w.Frame(1.0 / 30)
	assert.Len(t, fixed.seen, 2, "a long frame runs two fixed steps")
	w.Frame(1.0 / 120)
	assert.Len(t, fixed.seen, 2)
	w.Frame(1.0 / 120)
	assert.Len(t, fixed.seen, 3)
	assert.InDelta(t, 1.0/60, fixed.seen[0].DT, 1e-6, "fixed steps carry the fixed delta")
	assert.Equal(t, uint64(3), w.Time().Frame, "frames counted")
}

func TestQuitStopsTheRunner(t *testing.T) {
	var quitAt uint64
	w := New().On(PostUpdate, "stop", func(f Frame) {
		if f.Elapsed >= 0.05 && quitAt == 0 {
			quitAt = f.World.Time().Frame
			f.Quit()
		}
	})
	require.NoError(t, w.Run())
	assert.True(t, w.ShouldQuit())
	assert.Equal(t, quitAt, w.Time().Frame, "the runner stops on the frame that quits")
	at := w.Time().Elapsed
	w.RunFrames(10)
	assert.Equal(t, at, w.Time().Elapsed, "a quit world runs no more frames")
}

type presenter struct{ calls []float32 }

func (p *presenter) Present(_ *World, alpha float32) { p.calls = append(p.calls, alpha) }

type plugin struct{ built *int }

func (p plugin) Build(w *World) { *p.built++; w.AddPresenter(&presenter{}) }

func TestPluginsAndPresenters(t *testing.T) {
	built := 0
	p := &presenter{}
	var order []string
	w := New().
		AddPlugins(plugin{&built}, plugin{&built}).
		AddPlugins(PluginFunc(func(*World) { order = append(order, "func") })).
		AddPlugins(PluginFunc(func(*World) { order = append(order, "func") })).
		AddPresenter(p).
		On(Update, "tick", func(Frame) { order = append(order, "tick") })
	assert.Equal(t, 1, built, "a plugin type builds once")
	w.Frame(1.0 / 120)
	assert.Equal(t, []string{"func", "func", "tick"}, order, "function plugins always build")
	require.Len(t, p.calls, 1, "presenters run after the frame's systems")
	assert.InDelta(t, 0.5, p.calls[0], 1e-5, "with the fixed-step alpha")
}

type watcher struct{ Group[Movable] }

type childless struct{ updates int }

func (c *childless) Update(Frame) { c.updates++ }

type keeper struct{ removed []uint64 }

func (k *keeper) Update(Frame)        {}
func (k *keeper) Remove(e Identifier) { k.removed = append(k.removed, e.ID()) }

func TestChainsParentsAndOptionalRemove(t *testing.T) {
	view := &watcher{}
	parent := &mover{BasicEntity: NewBasic()}
	child := &mover{BasicEntity: NewBasic()}
	parent.AppendChild(child)
	plain, kept := &childless{}, &keeper{}
	w := New().AddView(view).AddSystems(plain, kept).AddEntities(parent)
	assert.True(t, w.Has(child), "adding a parent adds its children")
	assert.Equal(t, 2, view.Len())
	w.RemoveEntity(parent)
	assert.Equal(t, 0, view.Len(), "removal reaches views and skips systems without Remove")
	assert.ElementsMatch(t, []uint64{parent.ID(), child.ID()}, kept.removed, "a system with Remove hears of each removal")
	w.Update(0)
	assert.Equal(t, 1, plain.updates)
}

func TestMaxDeltaClampsSpikes(t *testing.T) {
	w := New()
	w.Tick(10)
	assert.InDelta(t, 0.25, w.Time().Delta, 1e-6)
}

func TestStatsReportSlowestSystems(t *testing.T) {
	w := New().On(Update, "only", func(Frame) {})
	w.RunFrames(1)
	s := w.Stats().Slowest(5)
	require.Len(t, s, 1)
	assert.Equal(t, "only", s[0].Name)
	assert.Equal(t, 0, w.Stats().Entities())
}

func TestTimeAndStatsAreWorldAPI(t *testing.T) {
	w := New()
	var seen *Time
	w.On(Update, "clock", func(f Frame) { seen = f.Time })
	w.Frame(1.0 / 60)
	assert.Same(t, w.Time(), seen, "the frame carries the World's clock")
	assert.Equal(t, uint64(1), w.Time().Frame)
	assert.Equal(t, w, w.Stats().world)
	assert.Len(t, w.Timings(), 1)
	_, isResource := TryResource[Time](w)
	assert.False(t, isResource, "the clock is no resource")
	_, isResource = TryResource[Stats](w)
	assert.False(t, isResource)
}

func TestRunFramesRunsOneFixedStepAFrameAfterUpdate(t *testing.T) {
	w := New()
	var order []string
	w.On(FixedUpdate, "step", func(Frame) { order = append(order, "step") })
	w.On(Update, "update", func(Frame) { order = append(order, "update") })
	w.RunFrames(60)
	if n := w.Time().FixedFrame; n != 60 {
		t.Fatalf("60 frames at the fixed delta run 60 steps: %d", n)
	}
	if order[0] != "step" || order[1] != "update" {
		t.Fatalf("each frame steps before it updates: %v", order[:2])
	}
}

func TestStatsRecordSmoothsNamedCosts(t *testing.T) {
	w := New()
	st := w.Stats()
	st.Record("gpu", 10*time.Millisecond)
	st.Record("collect", 2*time.Millisecond)
	st.Record("gpu", 0)
	got := st.Recorded()
	assert.Len(t, got, 2)
	assert.Equal(t, "gpu", got[0].Name, "in the order first recorded")
	assert.Less(t, got[0].Time, 10*time.Millisecond, "smoothed toward the new sample")
	assert.Positive(t, got[0].Time)
	assert.Equal(t, 2*time.Millisecond, got[1].Time)
}

func TestFrameStatsFollowTheWallClockWhilePausedOrSlow(t *testing.T) {
	w := New()
	w.Time().Scale = 0
	for range 200 {
		w.Tick(0.4)
	}
	if s := w.Stats(); s.FPS < 2.4 || s.FPS > 2.6 || s.FrameTime < 390*time.Millisecond {
		t.Fatalf("paused, 400 ms frames read as such: %.1f fps, %v", s.FPS, s.FrameTime)
	}
}

func TestASystemsSpentIsWhatItCostInTheLastFrame(t *testing.T) {
	w := New()
	runs := 0
	w.On(FixedUpdate, "step", func(Frame) { runs++; time.Sleep(time.Millisecond) })
	w.Tick(3.5 / 60)
	var step *SystemStat
	for _, st := range w.Timings() {
		if st.Name == "step" {
			step = st
		}
	}
	if runs != 3 || step.Spent < 3*time.Millisecond {
		t.Fatalf("three fixed steps, all counted: %d runs, %v", runs, step.Spent)
	}
	w.Tick(0)
	if step.Spent != 0 {
		t.Fatalf("a frame without a step spent nothing: %v", step.Spent)
	}
}

func TestAFramesWorkIsItsTickAndRenderWithoutTheWait(t *testing.T) {
	s := Stats{FrameTime: 16700 * time.Microsecond, TickTime: time.Millisecond, RenderTime: 600 * time.Microsecond}
	if s.Work() != 1600*time.Microsecond {
		t.Fatalf("tick and render, not the vsync wait in the frame: %v", s.Work())
	}
}
