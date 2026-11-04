//go:build ignore

// Synthesises the game's sound effects in the manner of sfxr (square and
// triangle waves, pitch slides, noise, fast envelopes), so they are ours
// and free to use (CC0), and encodes them to Ogg Vorbis with ffmpeg:
//
//	go generate ./examples/internal/crypt
package main

import (
	"log"
	"math"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/merlindorin/ember/audio"
	"github.com/merlindorin/ember/audio/wav"
)

const rate = 22050

// voice is one sfxr-style tone: a wave sliding from one pitch to another
// under an attack/decay envelope.
type voice struct {
	wave       func(phase float64) float64
	from, to   float64
	at, length float64
	attack     float64
	vibrato    float64
	gain       float64
}

func square(p float64) float64 {
	if math.Mod(p, 1) < 0.5 {
		return 1
	}
	return -1
}

func triangle(p float64) float64 { return 4*math.Abs(math.Mod(p, 1)-0.5) - 1 }

var rng = rand.New(rand.NewPCG(7, 11))

// noise is white noise held for one period of the pitch, like sfxr's.
func noise() func(float64) float64 {
	last, held := -1.0, 0.0
	return func(p float64) float64 {
		if int(p) != int(last) {
			held = rng.Float64()*2 - 1
		}
		last = p
		return held
	}
}

func render(peakTo, seconds float64, voices ...voice) []float32 {
	out := make([]float64, int(seconds*rate))
	for _, v := range voices {
		phase := 0.0
		start := int(v.at * rate)
		n := int(v.length * rate)
		for i := 0; i < n && start+i < len(out); i++ {
			t := float64(i) / float64(n)
			hz := v.from * math.Pow(v.to/v.from, t) * (1 + v.vibrato*math.Sin(2*math.Pi*12*float64(i)/rate))
			phase += hz / rate
			env := math.Min(float64(i)/math.Max(v.attack*rate, 1), 1) * math.Pow(1-t, 2)
			out[start+i] += v.gain * env * v.wave(phase)
		}
	}
	peak := 0.0
	for _, s := range out {
		peak = math.Max(peak, math.Abs(s))
	}
	f := make([]float32, len(out))
	for i, s := range out {
		f[i] = float32(s / peak * peakTo)
	}
	return f
}

func notes(wave func(float64) float64, step, gain float64, hz ...float64) []voice {
	var v []voice
	for i, h := range hz {
		length := step * 1.6
		if i == len(hz)-1 {
			length = step * 4
		}
		v = append(
			v,
			voice{
				wave:    wave,
				from:    h,
				to:      h,
				at:      float64(i) * step,
				length:  length,
				attack:  0.005,
				vibrato: 0.004,
				gain:    gain,
			},
		)
	}
	return v
}

func main() {
	write("shoot", render(0.35, 0.09, voice{wave: square, from: 1400, to: 600, length: 0.09, attack: 0.002, gain: 0.8}))
	write("hit", render(0.6, 0.12,
		voice{wave: noise(), from: 3000, to: 800, length: 0.1, gain: 0.7},
		voice{wave: square, from: 220, to: 90, length: 0.12, gain: 0.5}))
	write("kill", render(0.7, 0.4,
		voice{wave: noise(), from: 2500, to: 150, length: 0.4, gain: 0.8},
		voice{wave: square, from: 380, to: 60, length: 0.3, gain: 0.4}))
	write("powerup", render(0.7, 0.5, append(notes(square, 0.06, 0.5, 523, 659, 784, 1047),
		voice{wave: triangle, from: 400, to: 1600, length: 0.3, attack: 0.01, gain: 0.3})...))
	write("hurt", render(0.7, 0.28,
		voice{wave: square, from: 330, to: 110, length: 0.28, attack: 0.002, vibrato: 0.08, gain: 0.7},
		voice{wave: noise(), from: 1200, to: 400, length: 0.12, gain: 0.4}))
	lose := append(notes(triangle, 0.22, 0.6, 392, 330, 262, 196),
		notes(square, 0.22, 0.25, 196, 165, 131, 98)...)
	write("gameover", render(0.7, 1.6, lose...))
}

func write(name string, samples []float32) {
	clip, err := audio.NewClip(rate, 1, samples)
	if err != nil {
		log.Fatal(err)
	}
	tmp := filepath.Join(os.TempDir(), name+".wav")
	f, err := os.Create(tmp)
	if err != nil {
		log.Fatal(err)
	}
	if err := wav.Encode(f, clip); err != nil {
		log.Fatal(err)
	}
	f.Close()
	defer os.Remove(tmp)
	cmd := exec.Command(
		"ffmpeg",
		"-hide_banner",
		"-loglevel",
		"error",
		"-y",
		"-i",
		tmp,
		"-c:a",
		"libvorbis",
		"-q:a",
		"3",
		"-map_metadata",
		"-1",
		"-fflags",
		"+bitexact",
		"-flags:a",
		"+bitexact",
		filepath.Join("sound", name+".ogg"),
	)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
}
