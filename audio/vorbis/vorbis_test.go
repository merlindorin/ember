package vorbis_test

import (
	"bytes"
	"math"
	"os"
	"testing"
	"testing/fstest"

	"github.com/merlindorin/ember/audio"
	"github.com/merlindorin/ember/audio/vorbis"
)

// testdata/tone.ogg is half a second of 440 Hz left and 660 Hz right at
// 22050 Hz and amplitude 0.5, synthesised for these tests and encoded with
// ffmpeg's libvorbis.
func tone(t testing.TB) []byte {
	b, err := os.ReadFile("testdata/tone.ogg")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// pitch estimates a channel's frequency from its zero crossings.
func pitch(s []float32, ch, rate int) float64 {
	crossings := 0
	frames := len(s) / 2
	for i := 1; i < frames; i++ {
		if (s[2*(i-1)+ch] < 0) != (s[2*i+ch] < 0) {
			crossings++
		}
	}
	return float64(crossings) / 2 / (float64(frames) / float64(rate))
}

func TestDecodesTheTone(t *testing.T) {
	c, err := audio.Load(fstest.MapFS{"tone.ogg": {Data: tone(t)}}, "tone.ogg")
	if err != nil {
		t.Fatal(err)
	}
	if c.SampleRate() != 22050 || c.Channels() != 2 || c.Frames() != 11025 {
		t.Fatalf("%d Hz, %d channels, %d frames", c.SampleRate(), c.Channels(), c.Frames())
	}
	var peak float64
	for _, s := range c.Samples() {
		peak = math.Max(peak, math.Abs(float64(s)))
	}
	if peak < 0.4 || peak > 0.6 {
		t.Fatalf("peak %v, want about 0.5", peak)
	}
	if l, r := pitch(c.Samples(), 0, 22050), pitch(c.Samples(), 1, 22050); math.Abs(l-440) > 10 || math.Abs(r-660) > 10 {
		t.Fatalf("left %v Hz, right %v Hz", l, r)
	}
}

func TestStreamsAndRewinds(t *testing.T) {
	d, err := vorbis.NewDecoder(bytes.NewReader(tone(t)))
	if err != nil {
		t.Fatal(err)
	}
	first, err := audio.ReadAll(d)
	if err != nil {
		t.Fatal(err)
	}
	if err = d.Rewind(); err != nil {
		t.Fatal(err)
	}
	again, err := audio.ReadAll(d)
	if err != nil || again.Frames() != first.Frames() {
		t.Fatal(again.Frames(), err)
	}
	for i, s := range first.Samples() {
		if again.Samples()[i] != s {
			t.Fatalf("sample %d differs after a rewind", i)
		}
	}
	s, err := audio.Open(fstest.MapFS{"tone.ogg": {Data: tone(t)}}, "tone.ogg")
	if err != nil || s.SampleRate() != 22050 || s.Channels() != 2 {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestTruncatedStreamsFailCleanly(t *testing.T) {
	b := tone(t)
	for n := 0; n < len(b); n += 7 {
		d, err := vorbis.NewDecoder(bytes.NewReader(b[:n]))
		if err != nil {
			continue
		}
		if c, rerr := audio.ReadAll(d); rerr == nil && c.Frames() > 11025 {
			t.Fatalf("truncated at %d: %d frames", n, c.Frames())
		}
	}
}

func FuzzDecode(f *testing.F) {
	f.Add(tone(f))
	f.Fuzz(func(_ *testing.T, b []byte) {
		d, err := vorbis.NewDecoder(bytes.NewReader(b))
		if err != nil {
			return
		}
		p := make([]float32, 4096)
		for range 1 << 10 {
			if _, err = d.Read(p); err != nil {
				return
			}
		}
	})
}

func TestAStreamLoopsThroughTheMixer(t *testing.T) {
	s, err := audio.Open(fstest.MapFS{"tone.ogg": {Data: tone(t)}}, "tone.ogg")
	if err != nil {
		t.Fatal(err)
	}
	m := audio.NewMixer()
	rec := audio.NewRecorder(m)
	v := m.Play(s, audio.Loop())
	rec.Mix(3 * 48000 / 2)
	if !v.Playing() {
		t.Fatal("a looping stream plays on past its end")
	}
	for _, at := range []int{12000, 36000, 60000} {
		if rec.Peak(at, at+1000) < 0.3 {
			t.Fatalf("silence around frame %d", at)
		}
	}
}
