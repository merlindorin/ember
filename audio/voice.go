package audio

import (
	"math"
	"slices"
)

type voiceState uint8

const (
	playing voiceState = iota
	pausing
	paused
	stopping
	done
)

// A Voice is one sound playing in a Mixer. Its methods are safe from any
// goroutine and do nothing once it has finished.
type Voice struct {
	mixer *Mixer
	id    uint64
	src   source
	set   settings
	bus   *Bus
	state voiceState

	step   float64
	t      float64
	a, b   [2]float32
	primed bool

	gl, gr      float32
	fade        fade
	spatial     bool
	spatialGain float32
	spatialPan  float32

	stopped, stolen bool
}

type fade struct {
	from, to float32
	at, len  int
	power    bool
}

func (f *fade) value() float32 {
	if f.at >= f.len {
		return f.to
	}
	t := float64(f.at) / float64(f.len)
	if !f.power {
		return f.from + (f.to-f.from)*float32(t)
	}
	if f.to > f.from {
		return f.from + (f.to-f.from)*float32(math.Sin(t*math.Pi/2))
	}
	return f.from + (f.to-f.from)*float32(1-math.Cos(t*math.Pi/2))
}

func (f *fade) finished() bool { return f.at >= f.len }

// ID numbers the voices of a mixer in the order they started.
func (v *Voice) ID() uint64 { return v.id }

// Playing reports whether the voice has yet to finish: sounding, paused or
// fading out.
func (v *Voice) Playing() bool {
	v.mixer.mu.Lock()
	defer v.mixer.mu.Unlock()
	return v.state < done
}

func (v *Voice) Paused() bool {
	v.mixer.mu.Lock()
	defer v.mixer.mu.Unlock()
	return v.state == pausing || v.state == paused
}

// Stop ends the voice, over FadeOut when given.
func (v *Voice) Stop(opts ...Option) {
	set := newSettings(opts)
	v.mixer.mu.Lock()
	defer v.mixer.mu.Unlock()
	if v.state >= stopping {
		return
	}
	d := set.fadeOut
	if d < smoothing {
		d = smoothing
	}
	v.stop(v.mixer.frames(d), false)
}

func (v *Voice) stop(frames int, power bool) {
	if v.state == paused || v.state >= stopping {
		v.state = done
		v.stopped = true
		return
	}
	v.fade = fade{from: v.fade.value(), to: 0, len: frames, power: power}
	v.state = stopping
	v.stopped = true
}

func (v *Voice) Pause() {
	v.mixer.mu.Lock()
	defer v.mixer.mu.Unlock()
	if v.state != playing {
		return
	}
	v.fade = fade{from: v.fade.value(), to: 0, len: v.mixer.frames(smoothing)}
	v.state = pausing
}

func (v *Voice) Resume() {
	v.mixer.mu.Lock()
	defer v.mixer.mu.Unlock()
	if v.state != pausing && v.state != paused {
		return
	}
	v.fade = fade{from: v.fade.value(), to: 1, len: v.mixer.frames(smoothing)}
	v.state = playing
}

func (v *Voice) Volume() float32 { return v.read(func() float32 { return v.set.volume }) }
func (v *Voice) Pan() float32    { return v.read(func() float32 { return v.set.pan }) }
func (v *Voice) Pitch() float32  { return v.read(func() float32 { return v.set.pitch }) }

func (v *Voice) SetVolume(vol float32) { v.write(func() { v.set.volume = max(vol, 0) }) }
func (v *Voice) SetPan(p float32)      { v.write(func() { v.set.pan = clampPan(p) }) }
func (v *Voice) SetPitch(p float32)    { v.write(func() { v.set.pitch = clampPitch(p) }) }

// Loop reports whether the voice was started with Loop.
func (v *Voice) Loop() bool { return v.set.loop }

func (v *Voice) Priority() int { return v.set.priority }

// Effects are what the voice runs through before its bus.
func (v *Voice) Effects() []Effect {
	v.mixer.mu.Lock()
	defer v.mixer.mu.Unlock()
	return slices.Clone(v.set.effects)
}

func (v *Voice) SetEffects(e ...Effect) { v.write(func() { v.set.effects = slices.Clone(e) }) }

// Bus is the name of the bus the voice plays on, "master" when none.
func (v *Voice) Bus() string {
	if v.bus == nil {
		return masterBus
	}
	return v.bus.name
}

func (v *Voice) read(f func() float32) float32 {
	v.mixer.mu.Lock()
	defer v.mixer.mu.Unlock()
	return f()
}

func (v *Voice) write(f func()) {
	v.mixer.mu.Lock()
	defer v.mixer.mu.Unlock()
	f()
}

func (v *Voice) setSpatial(gain, pan float32) {
	v.spatial, v.spatialGain, v.spatialPan = true, gain, pan
}

// target is the per-channel gain the voice is heading to, before fades.
func (v *Voice) target() (float32, float32) {
	g := v.set.volume * v.bus.gain() * v.mixer.master.gain()
	pan := v.set.pan
	if v.spatial {
		g *= v.spatialGain
		pan = v.spatialPan
	}
	l, r := panGains(pan)
	return g * l, g * r
}

// panGains leaves a centred sound as it is and moves it to a side at equal
// power.
func panGains(pan float32) (float32, float32) {
	theta := float64(pan+1) * math.Pi / 4
	return min(float32(math.Sqrt2*math.Cos(theta)), 1), min(float32(math.Sqrt2*math.Sin(theta)), 1)
}

func approach(cur, target, step float32) float32 {
	switch {
	case cur < target-step:
		return cur + step
	case cur > target+step:
		return cur - step
	}
	return target
}

func (v *Voice) pull() bool {
	l, r, looped, ok := v.src.next(v.set.loop)
	if looped {
		v.mixer.events = append(v.mixer.events, VoiceLooped{Voice: v})
	}
	v.b = [2]float32{l, r}
	return ok
}

func (v *Voice) mix(p []float32, smooth float32) {
	if v.state == paused || v.state == done {
		return
	}
	if !v.primed {
		v.primed = true
		if !v.pull() {
			v.state = done
			return
		}
		v.a = v.b
		if !v.pull() {
			v.b = v.a
		}
	}
	tl, tr := v.target()
	step := v.step * float64(v.set.pitch)
	for i := 0; i+1 < len(p); i += 2 {
		v.gl = approach(v.gl, tl, smooth)
		v.gr = approach(v.gr, tr, smooth)
		f := v.fade.value()
		v.fade.at++
		k := float32(v.t)
		p[i] += (v.a[0] + (v.b[0]-v.a[0])*k) * v.gl * f
		p[i+1] += (v.a[1] + (v.b[1]-v.a[1])*k) * v.gr * f
		if v.fade.finished() {
			switch v.state {
			case pausing:
				v.state = paused
				return
			case stopping:
				v.state = done
				return
			case playing, paused, done: // only the fades of pausing and stopping end in a new state
			}
		}
		v.t += step
		for v.t >= 1 {
			v.t--
			v.a = v.b
			if !v.pull() {
				v.state = done
				return
			}
		}
	}
}
