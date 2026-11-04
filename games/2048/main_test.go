package main

import (
	"testing"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/games/internal/look"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/ui/align"
	"github.com/merlindorin/ember/world"
)

func play(t *testing.T) (*Game, *world.World, func(input.Key)) {
	t.Helper()
	win := look.Window("2048")
	g := newGame(win.Input(), func() uint64 { return 1 })
	win.Add(g.doc)
	w := ember.New().AddEntities(win).AddSystems(g)
	w.RunFrames(1)
	press := func(k input.Key) {
		win.Input().SetKey(k, true)
		w.RunFrames(1)
		win.Input().SetKey(k, false)
		w.RunFrames(1)
	}
	return g, w, press
}

func TestTheArrowsSlideAndThePanelFollows(t *testing.T) {
	g, _, press := play(t)
	g.board.Cells = [4][4]int{{2, 2, 0, 0}}
	press(input.KeyLeft)
	if g.board.Cells[0][0] != 4 || g.score.Text() != "4" || g.best.Text() != "4" {
		t.Fatalf("cells %v, score %q, best %q", g.board.Cells, g.score.Text(), g.best.Text())
	}
	press(input.KeyN)
	if g.board.Score != 0 || g.score.Text() != "0" {
		t.Fatalf("N starts a new game: score %d, panel %q", g.board.Score, g.score.Text())
	}
}

func TestAFullBoardAsksForANewGame(t *testing.T) {
	g, w, press := play(t)
	// After Right, the new tile lands at the bottom left beside a 32 and an 8:
	// a 2 or a 4 there leaves no pair.
	g.board.Cells = [4][4]int{{2, 4, 2, 4}, {4, 2, 4, 2}, {32, 4, 2, 4}, {8, 16, 8, 0}}
	press(input.KeyRight)
	if !g.board.Over() {
		t.Fatalf("the board is over: %v", g.board.Cells)
	}
	if _, open := g.over.Attr("open"); !open {
		t.Fatal("game over shows the question")
	}
	g.doc.Close(g.over, "again")
	w.RunFrames(1)
	if g.board.Over() || g.score.Text() != "0" {
		t.Fatalf("New game starts again: %v, score %q", g.board.Cells, g.score.Text())
	}
}

func TestTheScreenIsClean(t *testing.T) {
	g, _, _ := play(t)
	if r, ok := g.doc.BoxOf(g.canvas); !ok || r.W != 354 || r.H != 354 {
		t.Fatalf("the board is 354 square: %v", r)
	}
	if faults := align.Check(g.doc); len(faults) > 0 {
		t.Fatalf("layout faults: %v", faults)
	}
	if warns := g.doc.Warnings(); len(warns) > 0 {
		t.Fatalf("CSS warnings: %v", warns)
	}
}
