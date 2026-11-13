//go:build ignore

// Synthesises hit.wav, so the example's sound is ours and free to use (CC0):
//
//	go generate ./examples/53-play-a-sound
package main

import (
	"log"
	"math"
	"os"

	"github.com/merlindorin/ember/audio"
	"github.com/merlindorin/ember/audio/wav"
)

func main() {
	const rate = 22050
	s := make([]float32, rate*2/5)
	for i := range s {
		t := float64(i) / rate
		s[i] = float32(0.5*math.Sin(2*math.Pi*880*t)*math.Exp(-9*t) + 0.25*math.Sin(2*math.Pi*1320*t)*math.Exp(-14*t))
	}
	clip, err := audio.NewClip(rate, 1, s)
	if err != nil {
		log.Fatal(err)
	}
	f, err := os.Create("hit.wav")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if err := wav.Encode(f, clip); err != nil {
		log.Fatal(err)
	}
}
