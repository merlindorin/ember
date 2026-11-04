package main

import (
	"math/bits"

	"github.com/merlindorin/ember/audio"
	"github.com/merlindorin/ember/games/internal/sound"
)

// Sounds are 2048's effects, made in code.
type Sounds struct {
	slide, over *audio.Clip
	merge       [12]*audio.Clip // by the merged tile, 4 up to 8192
}

// music is a calm loop: Cmaj7, Am7, Fmaj7, G, rippling.
func music() []float32 {
	return sound.Music(sound.Tune{BPM: 84, BeatsPerChord: 4, Arpeggio: true, Chords: [][]float64{
		{-9, -5, -2, 2}, {-12, -9, -5, -2}, {-16, -12, -9, -5}, {-14, -10, -7, -2},
	}})
}

func newSounds() *Sounds {
	s := &Sounds{
		slide: sound.Clip(sound.Mix(0.09, false, append(sound.Noise(0, 0.07, 0.12, 1),
			sound.Note{Length: 0.09, Hz: 210, Slide: 140, Wave: sound.Sine, Volume: 0.25, Attack: 0.005, Decay: 0.07})...)),
		over: sound.Clip(sound.Mix(1.1, false,
			sound.Note{At: 0, Length: 0.3, Hz: sound.Hz(3), Wave: sound.Triangle, Volume: 0.35, Attack: 0.01, Decay: 0.2},
			sound.Note{At: 0.25, Length: 0.3, Hz: sound.Hz(-2), Wave: sound.Triangle, Volume: 0.35, Attack: 0.01, Decay: 0.2},
			sound.Note{At: 0.5, Length: 0.6, Hz: sound.Hz(-9), Wave: sound.Triangle, Volume: 0.35, Attack: 0.01, Decay: 0.5})),
	}
	for i := range s.merge {
		hz := sound.Hz(float64(i * 2))
		s.merge[i] = sound.Clip(sound.Mix(0.25, false,
			sound.Note{Length: 0.25, Hz: hz, Wave: sound.Triangle, Volume: 0.35, Attack: 0.004, Decay: 0.22},
			sound.Note{Length: 0.18, Hz: hz * 2, Wave: sound.Sine, Volume: 0.12, Attack: 0.004, Decay: 0.16}))
	}
	return s
}

// mergeOf is the sound of a merge making v: higher for bigger tiles.
func (s *Sounds) mergeOf(v int) *audio.Clip {
	return s.merge[min(max(bits.Len(uint(v))-3, 0), len(s.merge)-1)]
}
