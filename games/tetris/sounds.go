package main

import (
	"github.com/merlindorin/ember/audio"
	"github.com/merlindorin/ember/games/internal/sound"
)

// Sounds are Tetris's effects, made in code.
type Sounds struct {
	move, turn, drop, lock, over *audio.Clip
	clear                        [5]*audio.Clip // by the rows cleared at once
}

// music is a brisk loop: Am, F, C, G, an arpeggio over a ticking beat.
func music() []float32 {
	return sound.Music(sound.Tune{BPM: 120, BeatsPerChord: 4, Arpeggio: true, Hats: true, Chords: [][]float64{
		{-12, -9, -5}, {-16, -12, -9}, {-9, -5, -2}, {-14, -10, -7},
	}})
}

func newSounds() *Sounds {
	blip := func(hz, to, length, volume float64) *audio.Clip {
		return sound.Clip(sound.Mix(length, false,
			sound.Note{Length: length, Hz: hz, Slide: to, Wave: sound.Square, Volume: volume, Attack: 0.002, Decay: length}))
	}
	s := &Sounds{
		move: blip(660, 620, 0.03, 0.12),
		turn: blip(520, 780, 0.06, 0.15),
		lock: blip(180, 120, 0.06, 0.2),
		drop: sound.Clip(sound.Mix(0.18, false, append(sound.Noise(0, 0.08, 0.25, 2),
			sound.Note{Length: 0.18, Hz: 140, Slide: 55, Wave: sound.Sine, Volume: 0.6, Attack: 0.002, Decay: 0.15})...)),
		over: sound.Clip(sound.Mix(1.2, false,
			sound.Note{At: 0, Length: 0.25, Hz: sound.Hz(0), Wave: sound.Square, Volume: 0.2, Attack: 0.01, Decay: 0.2},
			sound.Note{At: 0.2, Length: 0.25, Hz: sound.Hz(-4), Wave: sound.Square, Volume: 0.2, Attack: 0.01, Decay: 0.2},
			sound.Note{At: 0.4, Length: 0.25, Hz: sound.Hz(-7), Wave: sound.Square, Volume: 0.2, Attack: 0.01, Decay: 0.2},
			sound.Note{At: 0.6, Length: 0.6, Hz: sound.Hz(-12), Wave: sound.Square, Volume: 0.2, Attack: 0.01, Decay: 0.5})),
	}
	for n := 1; n <= 4; n++ {
		var notes []sound.Note
		for i, step := range []float64{0, 4, 7, 12, 16, 19, 24}[:n+2] {
			notes = append(notes, sound.Note{At: float64(i) * 0.05, Length: 0.25, Hz: sound.Hz(step), Wave: sound.Triangle,
				Volume: 0.3, Attack: 0.003, Decay: 0.22})
		}
		s.clear[n] = sound.Clip(sound.Mix(float64(n+2)*0.05+0.25, false, notes...))
	}
	return s
}
