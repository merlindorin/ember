package audio_test

import (
	"io"
	"math"
	"testing"
	"time"

	"github.com/lantervale/ember/audio"
	"github.com/lantervale/ember/world"
)

const rate = 48000

func constant(t *testing.T, frames int, l, r float32) *audio.Clip {
	t.Helper()
	s := make([]float32, 2*frames)
	for i := range frames {
		s[2*i], s[2*i+1] = l, r
	}
	c, err := audio.NewClip(rate, 2, s)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func mono(t *testing.T, hz, frames int, v float32) *audio.Clip {
	t.Helper()
	s := make([]float32, frames)
	for i := range s {
		s[i] = v
	}
	c, err := audio.NewClip(hz, 1, s)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func near(a, b, tol float32) bool { return float32(math.Abs(float64(a-b))) <= tol }

func newWorld(opts ...audio.MixerOption) (*world.World, *audio.Mixer, *audio.Recorder) {
	m := audio.NewMixer(opts...)
	w := world.New().AddPlugins(audio.Plugin{}).AddEntities(m)
	return w, m, audio.NewRecorder(m)
}

// lastSound is the last frame with sound in it, or -1.
func lastSound(r *audio.Recorder) int {
	s := r.Samples()
	for i := len(s)/2 - 1; i >= 0; i-- {
		if s[2*i] != 0 || s[2*i+1] != 0 {
			return i
		}
	}
	return -1
}

func TestAOneShotPlaysAtTheWorldsPaceAndFinishes(t *testing.T) {
	w, m, rec := newWorld()
	var finished world.Reader[audio.VoiceFinished]
	v := m.Play(mono(t, rate, rate/10, 0.5))
	w.RunFrames(3)
	if !v.Playing() || rec.Frames() != 3*rate/60 {
		t.Fatalf("after 3 frames: playing %v, %d frames mixed", v.Playing(), rec.Frames())
	}
	if p := rec.Peak(0, rec.Frames()); !near(p, 0.5, 1e-6) {
		t.Fatalf("a centred mono sound plays as it is in both channels: peak %v", p)
	}
	w.RunFrames(3)
	if v.Playing() {
		t.Fatal("a tenth of a second plays out in 6 frames")
	}
	var got []audio.VoiceFinished
	for ev := range finished.Read(w) {
		got = append(got, ev)
	}
	if len(got) != 1 || got[0].Voice != v || got[0].Stopped {
		t.Fatalf("one Finished for the voice, not stopped: %+v", got)
	}
	if last := lastSound(rec); last < rate/10-2 || last > rate/10 {
		t.Fatalf("the sound lasts its length: last sound at frame %d", last)
	}
}

func TestSoundsSumAndBusesScale(t *testing.T) {
	_, m, rec := newWorld(audio.WithBus("sfx", 0.5), audio.WithBus("music", 0))
	c := mono(t, rate, rate, 0.25)
	m.Play(c)
	m.Play(c)
	rec.Mix(100)
	if p := rec.Peak(0, 100); !near(p, 0.5, 1e-6) {
		t.Fatalf("two voices sum: %v", p)
	}
	rec.Reset()
	m2 := audio.NewMixer(audio.WithBus("sfx", 0.5), audio.WithBus("music", 0))
	rec2 := audio.NewRecorder(m2)
	m2.Play(c, audio.OnBus("sfx"))
	m2.Play(c, audio.OnBus("music"))
	m2.Play(c, audio.OnBus("nowhere"), audio.Volume(2))
	rec2.Mix(100)
	if p := rec2.Peak(0, 100); !near(p, 0.125+0.5, 1e-6) {
		t.Fatalf("sfx at half, music silent, unknown buses on master: %v", p)
	}
	m2.Master().SetVolume(0.5)
	rec2.Reset()
	rec2.Mix(2000)
	if p := rec2.Peak(1000, 2000); !near(p, (0.125+0.5)/2, 1e-6) {
		t.Fatalf("master scales every bus: %v", p)
	}
	if m2.Bus("sfx").Volume() != 0.5 || m2.Bus("nope") != nil || m2.Bus("master") != m2.Master() {
		t.Fatal("buses read back")
	}
	m2.Bus("sfx").SetMuted(true)
	m2.Master().SetVolume(1)
	rec2.Reset()
	rec2.Mix(2000)
	if p := rec2.Peak(1000, 2000); !near(p, 0.5, 1e-6) || !m2.Bus("sfx").Muted() {
		t.Fatalf("a muted bus is silent: %v", p)
	}
}

func TestPanMovesASoundAtEqualPower(t *testing.T) {
	_, m, rec := newWorld()
	c := mono(t, rate, rate, 1)
	v := m.Play(c, audio.Pan(-1))
	rec.Mix(100)
	if l, r := rec.PeakLR(0, 100); !near(l, 1, 1e-6) || !near(r, 0, 1e-6) {
		t.Fatalf("hard left: %v %v", l, r)
	}
	v.SetPan(0.5)
	rec.Reset()
	rec.Mix(2000)
	l, r := rec.PeakLR(1000, 2000)
	theta := (0.5 + 1) * math.Pi / 4
	if !near(l, float32(math.Sqrt2*math.Cos(theta)), 1e-4) || !near(r, 1, 1e-6) || v.Pan() != 0.5 {
		t.Fatalf("half right, after smoothing: %v %v", l, r)
	}
}

func TestPitchAndSampleRateResample(t *testing.T) {
	_, m, rec := newWorld()
	m.Play(mono(t, rate, 4800, 1), audio.Pitch(2))
	rec.Mix(6000)
	if last := lastSound(rec); last < 2390 || last > 2400 {
		t.Fatalf("pitch 2 halves the length: %d", last)
	}
	rec.Reset()
	m.Play(mono(t, rate/2, 2400, 1))
	rec.Mix(6000)
	if last := lastSound(rec); last < 4790 || last > 4800 {
		t.Fatalf("a 24 kHz clip lasts as long at 48 kHz: %d", last)
	}
}

func TestMusicCrossfadesAtEqualPower(t *testing.T) {
	w, m, rec := newWorld(audio.WithBus("music", 1))
	var finished world.Reader[audio.VoiceFinished]
	day := m.PlayMusic(constant(t, 4*rate, 0.5, 0), audio.Loop())
	rec.Mix(rate / 10)
	night := m.PlayMusic(constant(t, 4*rate, 0, 0.5), audio.Crossfade(time.Second))
	if m.Music() != night || day.Bus() != "music" {
		t.Fatal("the new track is the music")
	}
	rec.Reset()
	rec.Mix(rate)
	half := rate / 2
	l, r := rec.PeakLR(half, half+1)
	if !near(l, 0.5*float32(math.Cos(math.Pi/4)), 1e-3) || !near(r, 0.5*float32(math.Sin(math.Pi/4)), 1e-3) {
		t.Fatalf("halfway, both tracks at equal power: %v %v", l, r)
	}
	rec.Mix(10)
	l, r = rec.PeakLR(rate, rate+10)
	if l != 0 || !near(r, 0.5, 1e-6) {
		t.Fatalf("after the crossfade only the new track plays: %v %v", l, r)
	}
	w.RunFrames(1)
	var ended []*audio.Voice
	for ev := range finished.Read(w) {
		if ev.Stopped {
			ended = append(ended, ev.Voice)
		}
	}
	if len(ended) != 1 || ended[0] != day || day.Playing() {
		t.Fatalf("the old track stops: %v", ended)
	}
	m.StopMusic(audio.FadeOut(100 * time.Millisecond))
	rec.Reset()
	rec.Mix(rate / 5)
	if lastSound(rec) > rate/10 || night.Playing() || m.Music() != nil {
		t.Fatalf("StopMusic fades out: last sound %d", lastSound(rec))
	}
}

func TestLoopsStopsPausesAndSmoothsVolume(t *testing.T) {
	w, m, rec := newWorld()
	var looped world.Reader[audio.VoiceLooped]
	v := m.Play(mono(t, rate, 100, 1), audio.Loop())
	w.RunFrames(1)
	n := 0
	for range looped.Read(w) {
		n++
	}
	if n != 8 || !v.Playing() || !v.Loop() {
		t.Fatalf("a 100-frame loop starts over every 100 frames: %d loops", n)
	}
	v.SetVolume(0)
	rec.Reset()
	rec.Mix(rate / 50)
	if rec.Peak(0, 1) < 0.9 || rec.Peak(rate/100+1, rate/50) != 0 {
		t.Fatalf("volume changes over 10 ms: %v then %v", rec.Peak(0, 1), rec.Peak(rate/100+1, rate/50))
	}
	v.SetVolume(1)
	v.Pause()
	rec.Reset()
	rec.Mix(rate / 10)
	if !v.Paused() || rec.Peak(rate/50, rate/10) != 0 {
		t.Fatal("a paused voice is silent")
	}
	v.Resume()
	rec.Reset()
	rec.Mix(rate / 10)
	if v.Paused() || rec.Peak(rate/50, rate/10) < 0.9 {
		t.Fatal("a resumed voice plays again")
	}
	v.Stop(audio.FadeOut(50 * time.Millisecond))
	rec.Reset()
	rec.Mix(rate / 10)
	if v.Playing() || lastSound(rec) > rate/20 || rec.Peak(rate/40-10, rate/40+10) > 0.6 {
		t.Fatalf("stop fades out over 50 ms: last sound %d", lastSound(rec))
	}
	if m.Voices() != 0 {
		t.Fatalf("%d voices left", m.Voices())
	}
}

func TestVoicesAreStolenByPriority(t *testing.T) {
	_, m, rec := newWorld(audio.MaxVoices(2))
	c := mono(t, rate, rate, 0.1)
	a := m.Play(c)
	b := m.Play(c, audio.Priority(1))
	d := m.Play(c)
	rec.Mix(rate / 50)
	if a.Playing() || !b.Playing() || !d.Playing() || m.Voices() != 2 {
		t.Fatal("the oldest of the lowest priority makes room")
	}
	e := m.Play(c, audio.Priority(5))
	rec.Mix(rate / 50)
	if d.Playing() || !b.Playing() || !e.Playing() || e.Priority() != 5 {
		t.Fatal("a higher priority takes the lowest voice")
	}
	f := m.Play(c)
	if f.Playing() || !b.Playing() || !e.Playing() || m.MaxVoices() != 2 {
		t.Fatal("a sound outranked by every voice is dropped")
	}
}

type samples struct {
	rate, channels int
	data           []float32
	at             int
}

func (s *samples) SampleRate() int { return s.rate }
func (s *samples) Channels() int   { return s.channels }
func (s *samples) Rewind() error   { s.at = 0; return nil }

func (s *samples) Read(p []float32) (int, error) {
	if s.at >= len(s.data) {
		return 0, io.EOF
	}
	n := copy(p, s.data[s.at:min(len(s.data), s.at+333)])
	s.at += n
	return n, nil
}

func ramp(n int) []float32 {
	d := make([]float32, n)
	for i := range d {
		d[i] = float32(i+1) / float32(n)
	}
	return d
}

func TestStreamsLoopAndRestart(t *testing.T) {
	w, m, rec := newWorld()
	var looped world.Reader[audio.VoiceLooped]
	s := audio.NewStream(&samples{rate: rate, channels: 1, data: ramp(10000)})
	v := m.Play(s, audio.Loop())
	rec.Mix(25000)
	got := rec.Samples()
	for _, i := range []int{0, 9999, 10000, 19999, 20000} {
		if want := float32(i%10000+1) / 10000; !near(got[2*i], want, 1e-6) {
			t.Fatalf("frame %d is %v, want %v", i, got[2*i], want)
		}
	}
	w.RunFrames(1)
	n := 0
	for range looped.Read(w) {
		n++
	}
	if n != 2 {
		t.Fatalf("two loops in 25000 frames: %d", n)
	}
	again := m.Play(s)
	rec.Reset()
	rec.Mix(10)
	if v.Playing() || !again.Playing() || !near(rec.Samples()[0], 1.0/10000, 1e-6) {
		t.Fatal("playing a stream again restarts it in a new voice")
	}
	if err := s.Close(); err != nil || again.Playing() {
		t.Fatal("closing a stream stops it")
	}
}

func TestADeviceReadingTheMixerDecodesStreamsAhead(t *testing.T) {
	w, m, _ := newWorld()
	m.Play(audio.NewStream(&samples{rate: rate, channels: 2, data: ramp(200000)}))
	buf := make([]float32, 2048)
	var heard float32
	deadline := time.Now().Add(5 * time.Second)
	for heard == 0 && time.Now().Before(deadline) {
		if n, err := m.Read(buf); err != nil || n != len(buf) {
			t.Fatal(n, err)
		}
		for _, s := range buf {
			heard = max(heard, s)
		}
	}
	if heard == 0 {
		t.Fatal("the decoding goroutine never filled the stream")
	}
	w.RunFrames(5)
	if n, _ := m.Read(buf); n != len(buf) {
		t.Fatal("the device keeps reading")
	}
}

func TestClipsCheckTheirShape(t *testing.T) {
	if _, err := audio.NewClip(rate, 3, nil); err == nil {
		t.Fatal("3 channels")
	}
	if _, err := audio.NewClip(0, 1, nil); err == nil {
		t.Fatal("0 Hz")
	}
	c, err := audio.ReadAll(&samples{rate: 22050, channels: 2, data: ramp(44100)})
	if err != nil || c.Frames() != 22050 || c.Duration() != time.Second || c.Channels() != 2 || c.SampleRate() != 22050 {
		t.Fatalf("%v %v", c, err)
	}
}
