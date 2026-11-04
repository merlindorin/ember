package audio

import (
	"math"
	"sync"
	"time"
)

// An Effect changes the sound that runs through it: a Reverb, a Filter or
// an Echo. Put it on a bus with WithBus or Bus.SetEffects, or on a voice
// with the Effects option. An effect keeps its own state, so each one
// belongs to one bus or one voice.
type Effect interface {
	process(p []float32, rate int)
}

// Effects runs the sound through effects, in order: for Mixer.Play,
// Mixer.PlayMusic and NewEmitter on that voice alone, and for WithBus on
// the bus's mix. A voice's effects end with it; a bus's ring on.
func Effects(e ...Effect) Option { return func(s *settings) { s.effects = e } }

// effectBlock is how many frames run between parameter steps.
const effectBlock = 32

// param moves towards its target by a share of the gap each block, so it
// gets there in about smoothing whatever the rate.
type param struct{ cur, target float32 }

func newParam(v float32) param { return param{v, v} }

func (p *param) step(k float32) { p.cur += (p.target - p.cur) * k }

func blockStep(rate int) float32 {
	return float32(1 - math.Exp(-float64(effectBlock)/(smoothing.Seconds()*float64(rate))))
}

func runEffects(effects []Effect, p []float32, rate int) {
	for _, e := range effects {
		e.process(p, rate)
	}
}

// Reverb is a room's echoes: Freeverb's eight comb and four all-pass
// filters per ear.
type Reverb struct {
	mu                 sync.Mutex
	room, damping, wet param
	rate               int
	combs              [2][8]comb
	allpasses          [2][4]allpass
}

// A ReverbOption sets up a Reverb.
type ReverbOption interface{ reverb(*Reverb) }

type roomSize float32
type damping float32

func (o roomSize) reverb(r *Reverb) { r.room = newParam(clamp01(float32(o))) }
func (o damping) reverb(r *Reverb)  { r.damping = newParam(clamp01(float32(o))) }

// RoomSize is how large the room sounds, from 0 (a closet) to 1 (a hall);
// 0.5 by default.
func RoomSize(v float32) ReverbOption { return roomSize(v) }

// Damping is how fast high notes fade in the room, from 0 to 1; 0.5 by
// default.
func Damping(v float32) ReverbOption { return damping(v) }

// WetOption is Wet, which works on a Reverb and an Echo.
type WetOption float32

func (o WetOption) reverb(r *Reverb) { r.wet = newParam(clamp01(float32(o))) }
func (o WetOption) echo(e *Echo)     { e.wet = newParam(clamp01(float32(o))) }

// Wet is the effect's share of the output, from 0 (the dry sound) to 1
// (the effect alone); 0.3 by default.
func Wet(v float32) WetOption { return WetOption(v) }

// NewReverb returns a reverb.
func NewReverb(opts ...ReverbOption) *Reverb {
	r := &Reverb{room: newParam(0.5), damping: newParam(0.5), wet: newParam(0.3)}
	for _, o := range opts {
		o.reverb(r)
	}
	return r
}

func (r *Reverb) RoomSize() float32 { return r.get(&r.room) }
func (r *Reverb) Damping() float32  { return r.get(&r.damping) }
func (r *Reverb) Wet() float32      { return r.get(&r.wet) }

func (r *Reverb) SetRoomSize(v float32) { r.set(&r.room, clamp01(v)) }
func (r *Reverb) SetDamping(v float32)  { r.set(&r.damping, clamp01(v)) }
func (r *Reverb) SetWet(v float32)      { r.set(&r.wet, clamp01(v)) }

func (r *Reverb) get(p *param) float32 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return p.target
}

func (r *Reverb) set(p *param, v float32) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p.target = v
}

var (
	combTuning    = [8]int{1116, 1188, 1277, 1356, 1422, 1491, 1557, 1617} //nolint:gochecknoglobals // Freeverb's tuning
	allpassTuning = [4]int{556, 441, 341, 225}                             //nolint:gochecknoglobals // Freeverb's tuning
)

const (
	stereoSpread = 23
	reverbGain   = 0.015
	reverbWet    = 3
)

func (r *Reverb) tune(rate int) {
	r.rate = rate
	scale := float64(rate) / 44100
	for ear := range 2 {
		for i, n := range combTuning {
			r.combs[ear][i] = comb{buf: make([]float32, max(1, int(float64(n+ear*stereoSpread)*scale)))}
		}
		for i, n := range allpassTuning {
			r.allpasses[ear][i] = allpass{buf: make([]float32, max(1, int(float64(n+ear*stereoSpread)*scale)))}
		}
	}
}

func (r *Reverb) process(p []float32, rate int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.rate != rate {
		r.tune(rate)
	}
	k := blockStep(rate)
	for start := 0; start < len(p)/2; start += effectBlock {
		r.room.step(k)
		r.damping.step(k)
		r.wet.step(k)
		feedback := 0.28*r.room.cur + 0.7
		damp := 0.4 * r.damping.cur
		wet, dry := r.wet.cur*reverbWet, 1-r.wet.cur
		for i := start; i < min(start+effectBlock, len(p)/2); i++ {
			l, rr := p[2*i], p[2*i+1]
			in := (l + rr) * reverbGain
			var out [2]float32
			for ear := range 2 {
				for c := range r.combs[ear] {
					out[ear] += r.combs[ear][c].process(in, feedback, damp)
				}
				for a := range r.allpasses[ear] {
					out[ear] = r.allpasses[ear][a].process(out[ear])
				}
			}
			p[2*i] = l*dry + out[0]*wet
			p[2*i+1] = rr*dry + out[1]*wet
		}
	}
}

type comb struct {
	buf    []float32
	at     int
	filter float32
}

func (c *comb) process(in, feedback, damp float32) float32 {
	out := c.buf[c.at]
	c.filter = out*(1-damp) + c.filter*damp
	c.buf[c.at] = in + c.filter*feedback
	if c.at++; c.at == len(c.buf) {
		c.at = 0
	}
	return out
}

type allpass struct {
	buf []float32
	at  int
}

func (a *allpass) process(in float32) float32 {
	held := a.buf[a.at]
	a.buf[a.at] = in + held*0.5
	if a.at++; a.at == len(a.buf) {
		a.at = 0
	}
	return held - in
}

// Filter is a low-pass or a high-pass biquad, 12 dB an octave past its
// cutoff.
type Filter struct {
	mu                sync.Mutex
	high              bool
	cutoff, resonance param
	b0, b1, b2        float32
	a1, a2            float32
	z                 [2][2]float32
	tuned             [2]float32
}

// A FilterOption sets up a Filter.
type FilterOption interface{ filter(*Filter) }

type cutoff float32
type resonance float32

func (o cutoff) filter(f *Filter)    { f.cutoff = newParam(clampCutoff(float32(o))) }
func (o resonance) filter(f *Filter) { f.resonance = newParam(clampResonance(float32(o))) }

// Cutoff is the frequency in Hz where the filter starts cutting: highs
// above it for a low-pass, lows below it for a high-pass.
func Cutoff(hz float32) FilterOption { return cutoff(hz) }

// Resonance is the filter's Q: 0.707 by default, flat; higher rings at the
// cutoff.
func Resonance(q float32) FilterOption { return resonance(q) }

// NewLowPass returns a filter that keeps what is below its cutoff, 1000 Hz
// by default: a sound behind a wall, or under water.
func NewLowPass(opts ...FilterOption) *Filter { return newFilter(false, opts) }

// NewHighPass returns a filter that keeps what is above its cutoff, 1000 Hz
// by default: a radio, or a small speaker.
func NewHighPass(opts ...FilterOption) *Filter { return newFilter(true, opts) }

func newFilter(high bool, opts []FilterOption) *Filter {
	f := &Filter{high: high, cutoff: newParam(1000), resonance: newParam(math.Sqrt2 / 2)}
	for _, o := range opts {
		o.filter(f)
	}
	return f
}

func (f *Filter) Cutoff() float32    { return f.get(&f.cutoff) }
func (f *Filter) Resonance() float32 { return f.get(&f.resonance) }

func (f *Filter) SetCutoff(hz float32)   { f.set(&f.cutoff, clampCutoff(hz)) }
func (f *Filter) SetResonance(q float32) { f.set(&f.resonance, clampResonance(q)) }

func (f *Filter) get(p *param) float32 {
	f.mu.Lock()
	defer f.mu.Unlock()
	return p.target
}

func (f *Filter) set(p *param, v float32) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p.target = v
}

func clampCutoff(hz float32) float32   { return min(max(hz, 10), 24000) }
func clampResonance(q float32) float32 { return min(max(q, 0.1), 20) }
func clamp01(v float32) float32        { return min(max(v, 0), 1) }

// stepCutoff moves the cutoff geometrically, as the ear hears pitch.
func (f *Filter) stepCutoff(k float32) {
	c := float64(f.cutoff.cur)
	f.cutoff.cur = float32(c * math.Pow(float64(f.cutoff.target)/c, float64(k)))
}

func (f *Filter) coefficients(rate int) {
	hz, q := min(f.cutoff.cur, 0.49*float32(rate)), f.resonance.cur
	if f.tuned == [2]float32{hz, q} {
		return
	}
	f.tuned = [2]float32{hz, q}
	w := 2 * math.Pi * float64(hz) / float64(rate)
	cos, alpha := math.Cos(w), math.Sin(w)/(2*float64(q))
	a0 := 1 + alpha
	b1 := 1 - cos
	if f.high {
		b1 = -(1 + cos)
	}
	f.b0 = float32(math.Abs(b1) / 2 / a0)
	f.b1 = float32(b1 / a0)
	f.b2 = f.b0
	f.a1 = float32(-2 * cos / a0)
	f.a2 = float32((1 - alpha) / a0)
}

func (f *Filter) process(p []float32, rate int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	k := blockStep(rate)
	for start := 0; start < len(p)/2; start += effectBlock {
		f.stepCutoff(k)
		f.resonance.step(k)
		f.coefficients(rate)
		for i := start; i < min(start+effectBlock, len(p)/2); i++ {
			for ear := range 2 {
				x := p[2*i+ear]
				z := &f.z[ear]
				y := f.b0*x + z[0]
				z[0] = f.b1*x - f.a1*y + z[1]
				z[1] = f.b2*x - f.a2*y
				p[2*i+ear] = y
			}
		}
	}
}

// maxEcho is the longest delay an Echo holds.
const maxEcho = 2 * time.Second

// Echo repeats the sound after a delay, each repeat quieter by its
// feedback.
type Echo struct {
	mu                   sync.Mutex
	delay, feedback, wet param
	rate                 int
	buf                  []float32
	at                   int
}

// An EchoOption sets up an Echo.
type EchoOption interface{ echo(*Echo) }

type delay time.Duration
type feedback float32

func (o delay) echo(e *Echo)    { e.delay = newParam(clampDelay(time.Duration(o))) }
func (o feedback) echo(e *Echo) { e.feedback = newParam(clampFeedback(float32(o))) }

// Delay is the time between repeats, up to 2 s; 300 ms by default.
func Delay(d time.Duration) EchoOption { return delay(d) }

// Feedback is how loud each repeat is against the one before, from 0 (one
// repeat) to 0.95; 0.4 by default.
func Feedback(v float32) EchoOption { return feedback(v) }

// NewEcho returns an echo.
func NewEcho(opts ...EchoOption) *Echo {
	e := &Echo{delay: newParam(0.3), feedback: newParam(0.4), wet: newParam(0.3)}
	for _, o := range opts {
		o.echo(e)
	}
	return e
}

func (e *Echo) Delay() time.Duration {
	e.mu.Lock()
	defer e.mu.Unlock()
	return time.Duration(float64(e.delay.target) * float64(time.Second))
}

func (e *Echo) Feedback() float32 { return e.get(&e.feedback) }
func (e *Echo) Wet() float32      { return e.get(&e.wet) }

func (e *Echo) SetDelay(d time.Duration) { e.set(&e.delay, clampDelay(d)) }
func (e *Echo) SetFeedback(v float32)    { e.set(&e.feedback, clampFeedback(v)) }
func (e *Echo) SetWet(v float32)         { e.set(&e.wet, clamp01(v)) }

func (e *Echo) get(p *param) float32 {
	e.mu.Lock()
	defer e.mu.Unlock()
	return p.target
}

func (e *Echo) set(p *param, v float32) {
	e.mu.Lock()
	defer e.mu.Unlock()
	p.target = v
}

func clampDelay(d time.Duration) float32 {
	return float32(min(max(d, time.Millisecond), maxEcho).Seconds())
}

func clampFeedback(v float32) float32 { return min(max(v, 0), 0.95) }

func (e *Echo) process(p []float32, rate int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.rate != rate {
		e.rate = rate
		e.buf = make([]float32, 2*(int(maxEcho.Seconds()*float64(rate))+2))
		e.at = 0
	}
	frames := len(e.buf) / 2
	k := blockStep(rate)
	for start := 0; start < len(p)/2; start += effectBlock {
		e.delay.step(k)
		e.feedback.step(k)
		e.wet.step(k)
		lag := math.Round(float64(e.delay.cur)*float64(rate)*1000) / 1000
		whole := int(lag)
		frac := float32(lag - float64(whole))
		wet, dry, fb := e.wet.cur, 1-e.wet.cur, e.feedback.cur
		for i := start; i < min(start+effectBlock, len(p)/2); i++ {
			a := (e.at - whole + frames) % frames
			b := (a - 1 + frames) % frames
			for ear := range 2 {
				y := e.buf[2*a+ear] + (e.buf[2*b+ear]-e.buf[2*a+ear])*frac
				x := p[2*i+ear]
				e.buf[2*e.at+ear] = x + y*fb
				p[2*i+ear] = x*dry + y*wet
			}
			if e.at++; e.at == frames {
				e.at = 0
			}
		}
	}
}
