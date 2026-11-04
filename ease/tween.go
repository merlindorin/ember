package ease

import "github.com/merlindorin/ember/world"

// Tween is an entity that drives a value over a duration: each frame the
// ease system advances it and calls its apply function with the eased
// progress. Added as a child of what it moves, it leaves with it.
type Tween struct {
	*world.BasicEntity
	seconds  float32
	apply    func(t float32)
	easing   Func
	delay    float32
	loop     bool
	pingPong bool
	elapsed  float32
	playing  bool
	finished bool
}

// TweenFinished is emitted when a tween that does not repeat reaches its
// end; the tween then leaves the World and its parent. What it moved stays:
// a game removes that on TweenFinished if it is done with it.
type TweenFinished struct{ Tween *Tween }

// TweenOption sets up a Tween.
type TweenOption func(*Tween)

// NewTween returns a playing tween of seconds that calls apply with the
// eased progress, from 0 to 1, every frame.
func NewTween(seconds float32, apply func(t float32), opts ...TweenOption) *Tween {
	t := &Tween{BasicEntity: world.NewBasic(), seconds: max(seconds, 1e-6), apply: apply, easing: Linear, playing: true}
	for _, o := range opts {
		o(t)
	}
	return t
}

// Easing bends the progress with f (default Linear).
func Easing(f Func) TweenOption { return func(t *Tween) { t.SetEasing(f) } }

// Delay waits seconds before the tween starts.
func Delay(seconds float32) TweenOption { return func(t *Tween) { t.delay = max(seconds, 0) } }

// Once runs the tween to its end, where it stops and leaves the World,
// the default.
func Once() TweenOption { return func(t *Tween) { t.loop, t.pingPong = false, false } }

// Loop starts the tween over each time it ends.
func Loop() TweenOption { return func(t *Tween) { t.loop = true } }

// PingPong runs the tween forth and back, again and again.
func PingPong() TweenOption { return func(t *Tween) { t.pingPong = true } }

func (t *Tween) GetTween() *Tween { return t }

func (t *Tween) Seconds() float32 { return t.seconds }
func (t *Tween) Delay() float32   { return t.delay }
func (t *Tween) Loop() bool       { return t.loop }
func (t *Tween) PingPong() bool   { return t.pingPong }

func (t *Tween) Easing() Func { return t.easing }

func (t *Tween) SetEasing(f Func) {
	if f != nil {
		t.easing = f
	}
}

func (t *Tween) Playing() bool { return t.playing }
func (t *Tween) Pause()        { t.playing = false }

// Resume carries on from where Pause stopped.
func (t *Tween) Resume() { t.playing = !t.finished }

// Restart plays the tween again from its start, after its Delay. A Once
// tween that finished has left the World: add it again after Restart.
func (t *Tween) Restart() { t.elapsed, t.playing, t.finished = 0, true, false }

// Progress is how far the tween is, from 0 to 1, before easing: on the
// way back for a PingPong.
func (t *Tween) Progress() float32 {
	run := max(t.elapsed-t.delay, 0) / t.seconds
	switch {
	case t.pingPong:
		cycle := run - 2*float32(int(run/2))
		if cycle > 1 {
			return 2 - cycle
		}
		return cycle
	case t.loop:
		return run - float32(int(run))
	}
	return min(run, 1)
}

// step advances the tween by dt seconds, applies it and reports whether
// it just finished.
func (t *Tween) step(dt float32) bool {
	if !t.playing {
		return false
	}
	t.elapsed += dt
	if t.elapsed < t.delay {
		return false
	}
	done := !t.loop && !t.pingPong && t.elapsed-t.delay >= t.seconds
	if t.apply != nil {
		t.apply(t.easing(t.Progress()))
	}
	if done {
		t.playing, t.finished = false, true
	}
	return done
}

// Plugin installs the system that plays tweens; ember.New adds it.
type Plugin struct{}

func (Plugin) Build(w *world.World) { w.AddSystems(&System{}) }

// System plays every Tween in the World, early in Update, so what they
// move is drawn where they put it.
type System struct {
	world.Group[*Tween]
}

func (s *System) Priority() int { return 100 }

func (s *System) Update(f world.Frame) {
	for t := range s.All() {
		if t.step(f.DT) {
			world.Emit(f.World, TweenFinished{Tween: t})
			if p := t.Parent(); p != nil {
				p.RemoveChild(t.BasicEntity)
			}
			f.World.RemoveEntity(t)
		}
	}
}
