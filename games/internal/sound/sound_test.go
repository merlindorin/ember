package sound_test

import (
	"testing"

	"github.com/merlindorin/ember/games/internal/sound"
)

func TestAMixStaysUnderItsLimit(t *testing.T) {
	loud := make([]sound.Note, 8)
	for i := range loud {
		loud[i] = sound.Note{Length: 0.2, Hz: 220, Wave: sound.Square, Volume: 1, Attack: 0.01, Decay: 0.01}
	}
	for _, v := range sound.Mix(0.2, false, loud...) {
		if v > 0.8 || v < -0.8 {
			t.Fatalf("a sample past 0.8: %v", v)
		}
	}
}

func TestALoopedNoteComesBackAtTheStart(t *testing.T) {
	s := sound.Mix(1, true,
		sound.Note{At: 0.9, Length: 0.3, Hz: 440, Wave: sound.Sine, Volume: 0.5, Attack: 0.01, Decay: 0.01})
	quiet := true
	for _, v := range s[:sound.Rate/10] {
		quiet = quiet && v == 0
	}
	if quiet {
		t.Fatal("the note's last 0.2 s wraps round to the start")
	}
}

func TestMusicIsItsChordsLong(t *testing.T) {
	m := sound.Music(sound.Tune{BPM: 120, Chords: [][]float64{{0, 3, 7}, {-4, 0, 3}}, BeatsPerChord: 4,
		Arpeggio: true, Hats: true})
	if want := 2 * 4 * sound.Rate / 2; len(m) != want {
		t.Fatalf("two chords of four beats at 120 bpm are 4 s: %d samples, want %d", len(m), want)
	}
	if sound.Clip(m).Duration() <= 0 {
		t.Fatal("it makes a clip")
	}
}
