package sound

// A Tune is a music loop: chords in turn, each held for some beats, under a
// soft pad and a bass, with an arpeggio and hats when asked.
type Tune struct {
	BPM           float64
	Chords        [][]float64 // each chord's notes, in semitones from A4
	BeatsPerChord int
	Arpeggio      bool // eighth notes up the chord
	Hats          bool // a tick on each off-beat
}

// Music renders a tune's loop, once through its chords; play it with
// audio.Loop.
func Music(t Tune) []float32 {
	beat := 60 / t.BPM
	hold := beat * float64(t.BeatsPerChord)
	var notes []Note
	for i, chord := range t.Chords {
		at := float64(i) * hold
		for _, s := range chord {
			notes = append(notes, Note{At: at, Length: hold + 0.4, Hz: Hz(s - 12), Wave: Sine, Volume: 0.07,
				Attack: 0.4, Decay: 0.6})
		}
		for b := range t.BeatsPerChord {
			notes = append(notes, Note{At: at + float64(b)*beat, Length: beat * 0.9, Hz: Hz(chord[0] - 24),
				Wave: Triangle, Volume: 0.16, Attack: 0.01, Decay: 0.25})
			if t.Hats {
				notes = append(notes, Noise(at+(float64(b)+0.5)*beat, 0.04, 0.05, uint64(i*16+b))...)
			}
		}
		if t.Arpeggio {
			for e := range 2 * t.BeatsPerChord {
				s := chord[e%len(chord)] + 12*float64(e/len(chord)%2)
				notes = append(notes, Note{At: at + float64(e)*beat/2, Length: beat * 0.8, Hz: Hz(s),
					Wave: Triangle, Volume: 0.06, Attack: 0.005, Decay: 0.3})
			}
		}
	}
	return Mix(hold*float64(len(t.Chords)), true, notes...)
}
