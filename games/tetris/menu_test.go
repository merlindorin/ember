package main

import (
	"testing"

	"github.com/lantervale/ember/audio"
	"github.com/lantervale/ember/input"
)

func clearRow(p *Play) {
	for x := range W - 4 {
		p.game.Board[H-1][x+4] = T
	}
	p.game.Piece = spawn(I)
	for range W {
		p.game.Move(-1)
	}
}

func TestAClearFlashesSparksAndPausesTheGame(t *testing.T) {
	p, w, press := play(t)
	rec := audio.NewRecorder(p.player.Mixer)
	clearRow(p)
	press(input.KeySpace)
	if p.game.Lines != 1 || p.clearing >= clearTime {
		t.Fatalf("the clear is being drawn: lines %d, clearing %v", p.game.Lines, p.clearing)
	}
	if p.sparks.Live() == 0 {
		t.Fatal("sparks fly from the cleared row")
	}
	y := p.game.Piece.Y
	w.RunFrames(10)
	if p.game.Piece.Y != y {
		t.Fatal("the next piece waits while the row clears")
	}
	w.RunFrames(60)
	if p.clearing < clearTime || rec.Peak(0, rec.Frames()) == 0 {
		t.Fatalf("the clear ends and was heard: clearing %v", p.clearing)
	}
}

func TestEscapeOpensTheMenuAndTheGameWaits(t *testing.T) {
	p, w, press := play(t)
	press(input.KeyEscape)
	if !p.menu.Open() {
		t.Fatal("Escape opens the menu")
	}
	y := p.game.Piece.Y
	w.RunFrames(120)
	if p.game.Piece.Y != y {
		t.Fatal("the piece doesn't fall while the menu is open")
	}
}

func TestAGameOverIsOnTheLeaderboard(t *testing.T) {
	p, w, press := play(t)
	for y := 2; y < H; y++ {
		for x := range W - 1 {
			p.game.Board[y][x] = Z
		}
	}
	p.game.Score = 777
	press(input.KeySpace)
	w.RunFrames(40)
	if len(p.data.Scores) != 1 || p.data.Scores[0].Score != 777 {
		t.Fatalf("the score is kept: %+v", p.data.Scores)
	}
}
