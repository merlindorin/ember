//go:build ignore

// Synthesises the example's footstep, clap and the drum loop behind the
// door, so they are ours and free to use (CC0), and encodes them to Ogg
// Vorbis with ffmpeg:
//
//	go generate ./examples/55-audio-effects
package main

import (
	"log"
	"math"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/lantervale/ember/audio"
	"github.com/lantervale/ember/audio/wav"
)

const rate = 22050

var rng = rand.New(rand.NewPCG(5, 8))

func main() {
	write("step.ogg", step())
	write("clap.ogg", clap())
	write("drums.ogg", drums())
}

func add(out []float64, at float64, n int, f func(t float64) float64) {
	start := int(at * rate)
	for i := 0; i < n && start+i < len(out); i++ {
		out[start+i] += f(float64(i) / rate)
	}
}

func step() []float32 {
	out := make([]float64, rate/8)
	add(out, 0, len(out), func(t float64) float64 {
		return 0.8*math.Sin(2*math.Pi*(90-200*t)*t)*math.Exp(-40*t) + 0.25*(rng.Float64()*2-1)*math.Exp(-120*t)
	})
	return normal(out, 0.7)
}

func clap() []float32 {
	out := make([]float64, rate/4)
	for _, at := range []float64{0, 0.011, 0.023} {
		add(out, at, rate/5, func(t float64) float64 { return (rng.Float64()*2 - 1) * math.Exp(-60*t) })
	}
	return normal(out, 0.8)
}

// drums is two bars at 120 bpm: kick, snare, eighth-note hats and a bass.
func drums() []float32 {
	const beat = 0.5
	out := make([]float64, 8*beat*rate)
	kick := func(t float64) float64 { return math.Sin(2*math.Pi*(55+120*math.Exp(-30*t))*t) * math.Exp(-8*t) }
	snare := func(t float64) float64 {
		return (0.7*(rng.Float64()*2-1) + 0.3*math.Sin(2*math.Pi*190*t)) * math.Exp(-18*t)
	}
	hat := func(t float64) float64 {
		n := rng.Float64()*2 - 1
		return 0.35 * n * math.Exp(-70*t)
	}
	bass := []float64{55, 55, 65.4, 49}
	for b := range 8 {
		at := float64(b) * beat
		if b%2 == 0 {
			add(out, at, rate/2, kick)
		} else {
			add(out, at, rate/3, snare)
		}
		add(out, at, rate/10, hat)
		add(out, at+beat/2, rate/10, hat)
		hz := bass[b/2]
		add(out, at, rate*9/20, func(t float64) float64 {
			return 0.3 * math.Copysign(1, math.Sin(2*math.Pi*hz*t)) * math.Exp(-3*t)
		})
	}
	return normal(out, 0.7)
}

func normal(out []float64, to float64) []float32 {
	peak := 0.0
	for _, s := range out {
		peak = math.Max(peak, math.Abs(s))
	}
	f := make([]float32, len(out))
	for i, s := range out {
		f[i] = float32(s / peak * to)
	}
	return f
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
	cmd := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-y", "-i", tmp,
		"-c:a", "libvorbis", "-q:a", "3", "-map_metadata", "-1", "-fflags", "+bitexact", "-flags:a", "+bitexact", name)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
}
