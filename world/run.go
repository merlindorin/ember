package world

import (
	"reflect"
	"sort"
	"time"
)

// Frame is what a system or a function system receives on each update: the
// step it advances by, the time so far, and the World it runs in.
type Frame struct {
	// DT is the step: the frame's delta, or the fixed delta in FixedUpdate.
	DT      float32
	Elapsed float32
	// FixedFrame counts the fixed steps run so far.
	FixedFrame uint64
	// Time is the World's clock.
	Time  *Time
	World *World
}

// Quit asks the runner to stop after the current frame.
func (f Frame) Quit() { f.World.Quit() }

// Time is the clock resource. Delta is the current phase's step: the frame
// delta in Update, FixedDelta inside FixedUpdate.
type Time struct {
	Delta   float32
	Elapsed float32
	// RealElapsed is the time the frames have taken, unscaled and never
	// paused: for what runs in a pause menu, such as a double click.
	RealElapsed float32
	Frame       uint64
	FixedFrame  uint64
	FixedDelta  float32
	// Alpha is how far the frame sits between two fixed steps (0..1).
	Alpha float32
	// Scale slows down or speeds up simulated time (0 pauses).
	Scale float32
	// MaxDelta clamps long frames (debugger pauses, window drags).
	MaxDelta float32

	accumulator float32
}

// Stats holds smoothed frame metrics for overlays and profilers; presenters
// fill in what they draw.
type Stats struct {
	FPS        float32
	FrameTime  time.Duration
	TickTime   time.Duration
	RenderTime time.Duration
	DrawCalls  int
	Triangles  int
	// Draws2D are the 2D renderer's draw calls in the last frame, Binds2D
	// the texture changes between them, Vertices2D what they drew and
	// Passes2D the offscreen pictures it drew first: cameras with a
	// Resolution or Into a texture, masks and blended layers.
	Draws2D, Binds2D, Vertices2D, Passes2D int

	world      *World
	history    [120]float32
	historyPos int
	recorded   []*SystemStat
	counted    []*Counter
}

// Counter is how many times something happened since the World started,
// as Count was told: a UI's replayed and repainted frames, by reason.
type Counter struct {
	Name string
	N    int
}

// Record adds a sample of d to the smoothed cost called name, for parts of
// a frame no system covers: a renderer's passes, a GPU's time. Overlays
// read them back with Recorded.
func (s *Stats) Record(name string, d time.Duration) {
	for _, r := range s.recorded {
		if r.Name == name {
			r.Time, r.Spent = smooth(r.Time, d), d
			return
		}
	}
	s.recorded = append(s.recorded, &SystemStat{Name: name, Time: d, Spent: d})
}

// Recorded lists the costs Record was given, smoothed, in the order they
// were first recorded.
func (s *Stats) Recorded() []*SystemStat { return s.recorded }

// Count adds one to the counter called name, for overlays to show how
// often something happens.
func (s *Stats) Count(name string) {
	for _, c := range s.counted {
		if c.Name == name {
			c.N++
			return
		}
	}
	s.counted = append(s.counted, &Counter{Name: name, N: 1})
}

// Counted lists the counters, in the order they were first counted.
func (s *Stats) Counted() []*Counter { return s.counted }

func (s *Stats) record(dt float32, tick time.Duration) {
	if dt > 0 {
		if s.FPS == 0 {
			s.FPS = 1 / dt
		}
		s.FPS = s.FPS*0.95 + (1/dt)*0.05
		s.FrameTime = smooth(s.FrameTime, time.Duration(float64(dt)*float64(time.Second)))
	}
	s.TickTime = smooth(s.TickTime, tick)
	s.history[s.historyPos] = dt * 1000
	s.historyPos = (s.historyPos + 1) % len(s.history)
}

// Work is what the CPU spends on a frame: its tick and its render,
// without waiting for the display. Under vsync the frame rate reads the
// display's, and 1/Work is the rate the frame's work would allow.
func (s *Stats) Work() time.Duration { return s.TickTime + s.RenderTime }

// Entities returns the number of live entities.
func (s *Stats) Entities() int { return s.world.Len() }

// FrameHistory returns recent frame times in milliseconds, oldest first.
func (s *Stats) FrameHistory() []float32 {
	out := make([]float32, 0, len(s.history))
	out = append(out, s.history[s.historyPos:]...)
	return append(out, s.history[:s.historyPos]...)
}

// Slowest returns the n most expensive systems.
func (s *Stats) Slowest(n int) []*SystemStat {
	st := s.world.Timings()
	sort.SliceStable(st, func(i, j int) bool { return st[i].Time > st[j].Time })
	return st[:min(n, len(st))]
}

// Plugin sets up a World: systems, resources, presenters, a runner.
type Plugin interface{ Build(w *World) }

// PluginFunc is a function used as a Plugin.
type PluginFunc func(w *World)

func (f PluginFunc) Build(w *World) { f(w) }

// Presenter draws the World after each frame's systems. It only reads the
// World, so a server needs none.
type Presenter interface {
	Present(w *World, alpha float32)
}

// Runner drives a World's frames until it quits.
type Runner func(w *World) error

// AddPlugins builds each plugin once: adding a plugin type twice is a
// no-op, so plugins can add the plugins they need. Function plugins always
// build.
func (w *World) AddPlugins(plugins ...Plugin) *World {
	for _, p := range plugins {
		t := reflect.TypeOf(p)
		if _, isFunc := p.(PluginFunc); !isFunc {
			if w.plugins[t] {
				continue
			}
			w.plugins[t] = true
		}
		p.Build(w)
	}
	return w
}

// Insert adds resources, each a pointer.
func (w *World) Insert(resources ...any) *World {
	for _, r := range resources {
		w.InsertResource(r)
	}
	return w
}

// On adds a function system to a phase.
func (w *World) On(phase Phase, name string, fn func(f Frame)) *World {
	return w.AddSystem(Func(phase, name, fn))
}

func (w *World) AddPresenter(p Presenter) *World {
	w.presenters = append(w.presenters, p)
	return w
}

// SetRunner replaces the runner: a desktop window loop, or the headless
// ticker the World starts with.
func (w *World) SetRunner(r Runner) *World {
	w.runner = r
	return w
}

// Run drives frames with the runner until Quit.
func (w *World) Run() error { return w.runner(w) }

func (w *World) MustRun() {
	if err := w.Run(); err != nil {
		panic(err)
	}
}

// Tick advances the simulation by dt seconds of wall time: Startup once,
// PreUpdate, FixedUpdate as many fixed steps as the time allows, then
// Update and PostUpdate.
func (w *World) Tick(dt float32) {
	start := time.Now()
	wall := dt
	for _, st := range w.stats {
		st.Spent = 0
	}
	t := w.time
	w.UpdatePhase(Startup, 0)

	t.RealElapsed += min(dt, t.MaxDelta)
	dt = min(dt, t.MaxDelta) * t.Scale
	t.Delta = dt
	t.Elapsed += dt
	t.Frame++

	w.UpdatePhase(PreUpdate, dt)
	t.accumulator += dt
	for t.accumulator >= t.FixedDelta {
		t.Delta = t.FixedDelta
		for _, q := range w.events {
			q.stepStarted()
		}
		w.UpdatePhase(FixedUpdate, t.FixedDelta)
		t.accumulator -= t.FixedDelta
		t.FixedFrame++
	}
	t.Delta = dt
	t.Alpha = t.accumulator / t.FixedDelta
	w.UpdatePhase(Update, dt)
	w.UpdatePhase(PostUpdate, dt)

	w.frameStats.record(wall, time.Since(start))
}

// Present runs every presenter on the World.
func (w *World) Present() {
	start := time.Now()
	for _, p := range w.presenters {
		p.Present(w, w.time.Alpha)
	}
	w.frameStats.RenderTime = smooth(w.frameStats.RenderTime, time.Since(start))
}

// Frame is Tick, then Present, then event aging: the body of every runner.
func (w *World) Frame(dt float32) {
	w.Tick(dt)
	w.Present()
	w.EndFrame()
}

// RunFrames runs n frames at the fixed step, the backbone of tests and
// dedicated servers; it stops early once the World quits.
func (w *World) RunFrames(n int) {
	for i := 0; i < n && !w.quit; i++ {
		w.Frame(w.time.FixedDelta)
	}
}

// Quit asks the runner to stop after the current frame.
func (w *World) Quit() { w.quit = true }

func (w *World) ShouldQuit() bool { return w.quit }

// Headless is the runner a World starts with: it ticks frames in real time
// at the fixed step, presenting nothing that needs a window.
func Headless(w *World) error {
	tick := time.NewTicker(time.Duration(float64(w.time.FixedDelta) * float64(time.Second)))
	defer tick.Stop()
	last := time.Now()
	for !w.quit {
		now := <-tick.C
		w.Frame(float32(now.Sub(last).Seconds()))
		last = now
	}
	return nil
}
