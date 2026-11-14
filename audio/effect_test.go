package audio_test

import (
	"math"
	"testing"
	"time"

	"github.com/lantervale/ember/audio"
)

func sine(t *testing.T, hz float64, frames int) *audio.Clip {
	t.Helper()
	s := make([]float32, frames)
	for i := range s {
		s[i] = float32(0.5 * math.Sin(2*math.Pi*hz*float64(i)/rate))
	}
	c, err := audio.NewClip(rate, 1, s)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func impulse(t testing.TB) *audio.Clip {
	c, err := audio.NewClip(rate, 1, []float32{1})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// filtered is the gain f gives a sine at hz, played quietly so even a
// resonant peak stays under the mixer's clamp.
func filtered(t *testing.T, f *audio.Filter, hz float64) float32 {
	t.Helper()
	m := audio.NewMixer()
	rec := audio.NewRecorder(m)
	m.Play(sine(t, hz, rate/2), audio.Effects(f), audio.Volume(0.1))
	rec.Mix(rate / 2)
	return rec.Peak(rate/4, rate/2) / 0.05
}

func TestFiltersKeepTheirSideOfTheCutoff(t *testing.T) {
	for _, c := range []struct {
		name   string
		filter func() *audio.Filter
		pass   float64
		cut    float64
	}{
		{"low-pass", func() *audio.Filter { return audio.NewLowPass(audio.Cutoff(500)) }, 100, 8000},
		{"high-pass", func() *audio.Filter { return audio.NewHighPass(audio.Cutoff(2000)) }, 10000, 100},
	} {
		if g := filtered(t, c.filter(), c.pass); g < 0.9 || g > 1.05 {
			t.Fatalf("%s passes %v Hz at %v", c.name, c.pass, g)
		}
		if g := filtered(t, c.filter(), c.cut); g > 0.02 {
			t.Fatalf("%s cuts %v Hz to %v", c.name, c.cut, g)
		}
	}
	if g := filtered(t, audio.NewLowPass(audio.Cutoff(1000), audio.Resonance(4)), 1000); g < 3 {
		t.Fatalf("resonance rings at the cutoff: %v", g)
	}
	f := audio.NewLowPass(audio.Cutoff(5), audio.Resonance(100))
	if f.Cutoff() != 10 || f.Resonance() != 20 {
		t.Fatal("cutoff and resonance are clamped and read back")
	}
	f.SetCutoff(3000)
	f.SetResonance(1)
	if f.Cutoff() != 3000 || f.Resonance() != 1 {
		t.Fatal("setters read back")
	}
}

func TestReverbRingsLongerInALargerRoom(t *testing.T) {
	tail := func(room float32) float32 {
		m := audio.NewMixer(audio.WithBus("sfx", 1, audio.Effects(audio.NewReverb(audio.RoomSize(room), audio.Wet(1)))))
		rec := audio.NewRecorder(m)
		m.Play(impulse(t), audio.OnBus("sfx"))
		rec.Mix(2 * rate)
		return rec.Peak(rate, 2*rate)
	}
	small, large := tail(0.2), tail(0.95)
	if small <= 0 || large < 5*small {
		t.Fatalf("a second after a clap, the hall rings on (%v) far more than the closet (%v)", large, small)
	}

	m := audio.NewMixer(audio.WithBus("sfx", 1, audio.Effects(audio.NewReverb(audio.Wet(0)))))
	rec := audio.NewRecorder(m)
	m.Play(mono(t, rate, 100, 0.5), audio.OnBus("sfx"))
	rec.Mix(1000)
	if rec.Peak(0, 100) != 0.5 || rec.Peak(100, 1000) != 0 {
		t.Fatal("a dry reverb leaves the sound as it is")
	}

	r := audio.NewReverb(audio.RoomSize(2), audio.Damping(0.1), audio.Wet(-1))
	if r.RoomSize() != 1 || r.Damping() != 0.1 || r.Wet() != 0 {
		t.Fatal("reverb options are clamped and read back")
	}
}

func TestEchoRepeatsAfterItsDelay(t *testing.T) {
	e := audio.NewEcho(audio.Delay(100*time.Millisecond), audio.Feedback(0.5), audio.Wet(0.5))
	m := audio.NewMixer()
	rec := audio.NewRecorder(m)
	m.Play(impulse(t), audio.Effects(e))
	rec.Mix(rate / 2)
	s := rec.Samples()
	at := func(frame int) float32 { return s[2*frame] }
	if at(0) != 0.5 || !near(at(rate/10), 0.5, 1e-6) || !near(at(rate/5), 0.25, 1e-6) {
		t.Fatalf("dry half now, then 0.5 after 100 ms, then 0.25: %v %v %v", at(0), at(rate/10), at(rate/5))
	}
	if rec.Peak(1, rate/10) != 0 {
		t.Fatal("silence between the repeats")
	}
	e.SetDelay(3 * time.Second)
	e.SetFeedback(1)
	if e.Delay() != 2*time.Second || e.Feedback() != 0.95 || e.Wet() != 0.5 {
		t.Fatal("echo settings are clamped and read back")
	}
}

func TestChangingAnEffectDoesNotClick(t *testing.T) {
	f := audio.NewLowPass(audio.Cutoff(200))
	m := audio.NewMixer()
	rec := audio.NewRecorder(m)
	m.Play(sine(t, 300, rate), audio.Effects(f))
	rec.Mix(rate / 4)
	f.SetCutoff(15000)
	rec.Mix(rate / 4)
	s := rec.Samples()
	var jump float32
	for i := 2; i < len(s); i += 2 {
		jump = max(jump, abs32(s[i]-s[i-2]))
	}
	smooth := float32(2 * math.Pi * 300 / rate * 0.5 * 1.5)
	if jump > smooth {
		t.Fatalf("the cutoff glides: the biggest step between samples is %v, a 300 Hz sine moves %v", jump, smooth)
	}
	if g := rec.Peak(rate/4+rate/50, rate/2) / 0.5; g < 0.95 {
		t.Fatalf("after 20 ms the filter is open: %v", g)
	}
}

func TestEffectsRunOnTheirBusAndRingOn(t *testing.T) {
	m := audio.NewMixer(
		audio.WithBus("sfx", 1, audio.Effects(audio.NewEcho(audio.Delay(50*time.Millisecond), audio.Wet(0.5)))),
		audio.WithBus("music", 1),
	)
	rec := audio.NewRecorder(m)
	m.Play(impulse(t), audio.OnBus("music"))
	rec.Mix(rate / 10)
	if rec.Peak(1, rate/10) != 0 {
		t.Fatal("another bus's echo leaves this one alone")
	}
	rec.Reset()
	v := m.Play(impulse(t), audio.OnBus("sfx"))
	rec.Mix(rate / 10)
	if v.Playing() || rec.Peak(rate/20, rate/20+1) == 0 {
		t.Fatal("the bus echoes after its voice has ended")
	}
	if len(m.Bus("sfx").Effects()) != 1 {
		t.Fatal("bus effects read back")
	}
	m.Bus("sfx").SetEffects()
	m.Master().SetEffects(audio.NewHighPass(audio.Cutoff(20000)))
	rec.Reset()
	m.Play(sine(t, 100, rate/10), audio.OnBus("sfx"))
	rec.Mix(rate / 10)
	if rec.Peak(rate/20, rate/10) > 0.01 {
		t.Fatal("master's effects run on everything")
	}
	w := m.Play(impulse(t), audio.Effects(audio.NewLowPass()))
	if len(w.Effects()) != 1 {
		t.Fatal("voice effects read back")
	}
}

func TestEffectsAreDeterministic(t *testing.T) {
	run := func() []float32 {
		m := audio.NewMixer(audio.WithBus("sfx", 1, audio.Effects(audio.NewReverb(), audio.NewEcho())))
		rec := audio.NewRecorder(m)
		m.Play(sine(t, 440, rate/4), audio.OnBus("sfx"), audio.Effects(audio.NewLowPass(audio.Cutoff(800))))
		m.Play(sine(t, 660, rate/4), audio.OnBus("sfx"))
		rec.Mix(rate / 2)
		return rec.Samples()
	}
	a, b := run(), run()
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("sample %d differs between two runs", i)
		}
	}
}

func abs32(x float32) float32 { return float32(math.Abs(float64(x))) }

// BenchmarkMixWithEffects mixes a 60 Hz frame (800 frames) of 32 voices,
// each through a low-pass, into an sfx bus with a reverb and an echo.
func BenchmarkMixWithEffects(b *testing.B) {
	m := audio.NewMixer(audio.WithBus("sfx", 1, audio.Effects(audio.NewReverb(), audio.NewEcho())))
	rec := audio.NewRecorder(m)
	s := make([]float32, 10*rate)
	for i := range s {
		s[i] = float32(math.Sin(float64(i) * 0.05))
	}
	c, _ := audio.NewClip(rate, 1, s)
	for range 32 {
		m.Play(c, audio.Loop(), audio.OnBus("sfx"), audio.Effects(audio.NewLowPass(audio.Cutoff(2000))))
	}
	b.ReportAllocs()
	for b.Loop() {
		rec.Mix(rate / 60)
		rec.Reset()
	}
}
