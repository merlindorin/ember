// Package sound makes the games' music and effects in code, so they ship no
// recording: soft-edged notes on a few waves, mixed into clips.
package sound

import (
	"math"
	"math/rand/v2"

	"github.com/merlindorin/ember/audio"
)

// Rate is the clips' sample rate.
const Rate = 22050

// A Wave is one cycle of a sound, for a phase from 0 to 1.
type Wave func(phase float64) float64

func Sine(p float64) float64     { return math.Sin(2 * math.Pi * p) }
func Triangle(p float64) float64 { return 4*math.Abs(p-math.Floor(p+0.5)) - 1 }

// Square is softened, its edges rounded, so it buzzes less than a true one.
func Square(p float64) float64 { return math.Tanh(4 * Sine(p)) }

// Note is one note of a sound: when it starts and how long it lasts in
// seconds, its pitch in Hz (Slide to end at another), its wave and loudness.
type Note struct {
	At, Length    float64
	Hz, Slide     float64
	Wave          Wave
	Volume        float64
	Attack, Decay float64 // the fade in and out, in seconds
}

// Mix renders notes into a clip seconds long. With loop, a note running past
// the end comes back in at the start, so the clip loops without a seam.
func Mix(seconds float64, loop bool, notes ...Note) []float32 {
	out := make([]float32, int(seconds*Rate))
	for _, n := range notes {
		start, length := int(n.At*Rate), int(n.Length*Rate)
		phase := 0.0
		for i := range length {
			t := float64(i) / Rate
			hz := n.Hz
			if n.Slide != 0 {
				hz += (n.Slide - n.Hz) * t / n.Length
			}
			phase += hz / Rate
			env := math.Min(1, math.Min(t/max(n.Attack, 1e-4), (n.Length-t)/max(n.Decay, 1e-4)))
			j := start + i
			if loop {
				j %= len(out)
			} else if j >= len(out) {
				break
			}
			out[j] += float32(n.Volume * env * n.Wave(phase-math.Floor(phase)))
		}
	}
	return limit(out)
}

// Noise is a burst of white noise, for hits and hats.
func Noise(at, length, volume float64, seed uint64) []Note {
	r := rand.New(rand.NewPCG(seed, seed)) //nolint:gosec // a sound's randomness, not a secret
	values := make([]float64, int(length*Rate))
	for i := range values {
		values[i] = r.Float64()*2 - 1
	}
	wave := func(p float64) float64 { return values[int(p*float64(len(values)-1))] }
	return []Note{{At: at, Length: length, Hz: 1 / length, Wave: wave, Volume: volume, Attack: 0.002, Decay: length}}
}

// Clip makes an audio clip of samples at Rate.
func Clip(samples []float32) *audio.Clip {
	c, err := audio.NewClip(Rate, 1, samples)
	if err != nil {
		panic(err) // only for an empty clip, which no sound here is
	}
	return c
}

// Hz is the pitch of a note named by its semitones above A4 (440 Hz).
func Hz(semitones float64) float64 { return 440 * math.Pow(2, semitones/12) }

// limit scales a mix whose peak passes 0.8 back to 0.8.
func limit(s []float32) []float32 {
	var peak float32
	for _, v := range s {
		peak = max(peak, v, -v)
	}
	if peak > 0.8 {
		for i := range s {
			s[i] *= 0.8 / peak
		}
	}
	return s
}
