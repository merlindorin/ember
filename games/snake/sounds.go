package main

import (
	"github.com/merlindorin/ember/audio"
	"github.com/merlindorin/ember/games/internal/sound"
)

// Sounds are Snake's effects, made in code.
type Sounds struct {
	turn, over *audio.Clip
	eat        [12]*audio.Clip // higher as the snake grows
}

// music is a bouncy loop: Em, C, G, D, an arpeggio over ticking hats.
func music() []float32 {
	return sound.Music(sound.Tune{BPM: 104, BeatsPerChord: 4, Arpeggio: true, Hats: true, Chords: [][]float64{
		{-5, -2, 2}, {-9, -5, -2}, {-14, -10, -7}, {-7, -3, 0},
	}})
}

func newSounds() *Sounds {
	s := &Sounds{
		turn: sound.Clip(sound.Mix(0.025, false,
			sound.Note{Length: 0.025, Hz: 900, Wave: sound.Square, Volume: 0.06, Attack: 0.002, Decay: 0.02})),
		over: sound.Clip(sound.Mix(0.9, false,
			sound.Note{Length: 0.9, Hz: 330, Slide: 80, Wave: sound.Square, Volume: 0.25, Attack: 0.01, Decay: 0.6})),
	}
	for i := range s.eat {
		hz := sound.Hz(float64(i))
		s.eat[i] = sound.Clip(sound.Mix(0.16, false,
			sound.Note{Length: 0.08, Hz: hz, Wave: sound.Triangle, Volume: 0.3, Attack: 0.003, Decay: 0.07},
			sound.Note{At: 0.06, Length: 0.1, Hz: hz * 1.5, Wave: sound.Triangle, Volume: 0.3, Attack: 0.003, Decay: 0.09}))
	}
	return s
}

func (s *Sounds) eatOf(eaten int) *audio.Clip { return s.eat[min(eaten, len(s.eat)-1)] }
