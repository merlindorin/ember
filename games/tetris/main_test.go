package main

import (
	"strconv"
	"testing"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/games/internal/look"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/ui/align"
	"github.com/merlindorin/ember/world"
)

func play(t *testing.T) (*Play, *world.World, func(input.Key)) {
	t.Helper()
	win := look.Window("tetris")
	p := newPlay(win.Input(), func() uint64 { return 1 })
	win.Add(p.doc)
	w := ember.New().AddEntities(win).AddSystems(p)
	w.RunFrames(1)
	press := func(k input.Key) {
		win.Input().SetKey(k, true)
		w.RunFrames(1)
		win.Input().SetKey(k, false)
		w.RunFrames(1)
	}
	return p, w, press
}

func TestTheKeysSteerTurnAndDrop(t *testing.T) {
	p, _, press := play(t)
	x := p.game.Piece.X
	press(input.KeyLeft)
	if p.game.Piece.X != x-1 {
		t.Fatalf("a tap on Left moves one column: %d then %d", x, p.game.Piece.X)
	}
	kind := p.game.Next
	press(input.KeySpace)
	if p.game.Piece.Kind != kind || p.game.Score == 0 || p.score.Text() != strconv.Itoa(p.game.Score) {
		t.Fatalf("Space drops and the next piece comes: kind %d, score %d, panel %q",
			p.game.Piece.Kind, p.game.Score, p.score.Text())
	}
}

func TestHoldingAnArrowRepeats(t *testing.T) {
	p, w, _ := play(t)
	in := p.in
	x := p.game.Piece.X
	in.SetKey(input.KeyRight, true)
	w.RunFrames(20)
	in.SetKey(input.KeyRight, false)
	w.RunFrames(1)
	if moved := p.game.Piece.X - x; moved < 2 {
		t.Fatalf("a third of a second held moves on: %d columns", moved)
	}
}

func TestAToppedOutGameAsksForANewOne(t *testing.T) {
	p, w, press := play(t)
	for y := 2; y < H; y++ {
		for x := range W - 1 {
			p.game.Board[y][x] = Z
		}
	}
	press(input.KeySpace)
	if !p.game.Over {
		t.Fatal("the stack reached the top")
	}
	if _, open := p.over.Attr("open"); open {
		t.Fatal("the question waits, so the Space that dropped doesn't answer it")
	}
	w.RunFrames(40)
	if p.game.Over != true {
		t.Fatal("nothing answered the question meanwhile")
	}
	if _, open := p.over.Attr("open"); !open {
		t.Fatal("game over shows the question")
	}
	p.doc.Close(p.over, "again")
	w.RunFrames(1)
	if p.game.Over || p.score.Text() != "0" {
		t.Fatalf("New game starts again: over %v, score %q", p.game.Over, p.score.Text())
	}
}

func TestTheScreenIsClean(t *testing.T) {
	p, _, _ := play(t)
	if r, ok := p.doc.BoxOf(p.canvas); !ok || r.W != 196 || r.H != 386 {
		t.Fatalf("the board is ten by twenty cells: %v", r)
	}
	if faults := align.Check(p.doc); len(faults) > 0 {
		t.Fatalf("layout faults: %v", faults)
	}
	if warns := p.doc.Warnings(); len(warns) > 0 {
		t.Fatalf("CSS warnings: %v", warns)
	}
}
