package main

import (
	"github.com/lantervale/ember/audio"
	"github.com/lantervale/ember/games/internal/sound"
)

// Sounds are Flap's effects, made in code.
type Sounds struct {
	flap, point, hit *audio.Clip
}

// music is an airy loop: Dmaj7, Bm7, Gmaj7, A, rippling over a soft beat.
func music() []float32 {
	return sound.Music(sound.Tune{BPM: 96, BeatsPerChord: 4, Arpeggio: true, Hats: true, Chords: [][]float64{
		{-7, -3, 0, 4}, {-10, -7, -3, 0}, {-14, -10, -7, -3}, {-12, -8, -5, -1},
	}})
}

func newSounds() *Sounds {
	return &Sounds{
		flap: sound.Clip(sound.Mix(0.1, false, append(sound.Noise(0, 0.08, 0.12, 3),
			sound.Note{Length: 0.1, Hz: 300, Slide: 520, Wave: sound.Sine, Volume: 0.2, Attack: 0.005, Decay: 0.08})...)),
		point: sound.Clip(sound.Mix(0.3, false,
			sound.Note{Length: 0.12, Hz: sound.Hz(7), Wave: sound.Triangle, Volume: 0.3, Attack: 0.003, Decay: 0.1},
			sound.Note{At: 0.08, Length: 0.22, Hz: sound.Hz(14), Wave: sound.Triangle, Volume: 0.3, Attack: 0.003, Decay: 0.2})),
		hit: sound.Clip(sound.Mix(0.7, false, append(sound.Noise(0, 0.15, 0.3, 4),
			sound.Note{Length: 0.7, Hz: 220, Slide: 60, Wave: sound.Square, Volume: 0.25, Attack: 0.005, Decay: 0.5})...)),
	}
}
