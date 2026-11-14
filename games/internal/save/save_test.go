package save_test

import (
	"testing"
	"time"

	"github.com/lantervale/ember/games/internal/save"
)

func TestTheLeaderboardKeepsTheBestInOrder(t *testing.T) {
	d := save.Defaults()
	now := time.Now()
	for _, s := range []int{50, 300, 100} {
		d.Add(s, now)
	}
	if got := [3]int{d.Scores[0].Score, d.Scores[1].Score, d.Scores[2].Score}; got != [3]int{300, 100, 50} {
		t.Fatalf("best first: %v", got)
	}
	if place := d.Add(200, now); place != 2 {
		t.Fatalf("200 is second: %d", place)
	}
	if place := d.Add(100, now); place != 4 {
		t.Fatalf("an equal score goes below the one there: %d", place)
	}
	for range save.Places {
		d.Add(1000, now)
	}
	if len(d.Scores) != save.Places || d.Add(1, now) != 0 {
		t.Fatalf("ten places, and a score below them all is not kept: %d", len(d.Scores))
	}
}

func TestDataRoundTripsThroughItsFile(t *testing.T) {
	t.Setenv("EMBER_GAMES_DIR", t.TempDir())
	if d, err := save.Load("tetris"); err != nil || len(d.Scores) != 0 || d.Music != save.Defaults().Music {
		t.Fatalf("no file yet is the defaults: %+v, %v", d, err)
	}
	d := save.Defaults()
	d.Add(42, time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC))
	d.Music, d.MusicOff = 0.3, true
	if err := save.Store("tetris", d); err != nil {
		t.Fatal(err)
	}
	got, err := save.Load("tetris")
	if err != nil || len(got.Scores) != 1 || got.Scores[0].Score != 42 || got.Music != 0.3 || !got.MusicOff {
		t.Fatalf("read back %+v, %v", got, err)
	}
}
