package main

import (
	"testing"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/games/internal/look"
	"github.com/merlindorin/ember/games/internal/save"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/ui/align"
	"github.com/merlindorin/ember/world"
)

func play(t *testing.T) (*Game, *world.World, func(input.Key)) {
	t.Helper()
	t.Setenv("EMBER_GAMES_DIR", t.TempDir())
	win := look.Window("2048")
	data := save.Defaults()
	g, tiles := newGame(win.Input(), func() uint64 { return 1 }, &data)
	win.Add(tiles, g.hud)
	w := ember.New().AddEntities(win, g.player.Mixer).AddSystems(g)
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
	g.hud.Close(g.over, "again")
	w.RunFrames(1)
	if g.board.Over() || g.score.Text() != "0" {
		t.Fatalf("New game starts again: %v, score %q", g.board.Cells, g.score.Text())
	}
}

func TestTheBoardSitsLeftOfACleanHUD(t *testing.T) {
	g, _, _ := play(t)
	panel, ok := g.hud.BoxOf(g.hud.ByID("side"))
	if !ok || panel.X < look.HUDLeft {
		t.Fatalf("the HUD keeps right of the board's area: %v", panel)
	}
	if faults := align.Check(g.hud); len(faults) > 0 {
		t.Fatalf("layout faults: %v", faults)
	}
	if warns := g.hud.Warnings(); len(warns) > 0 {
		t.Fatalf("CSS warnings: %v", warns)
	}
}

func TestAMoveAnimatesThenSettles(t *testing.T) {
	g, w, press := play(t)
	g.board.Cells = [4][4]int{{2, 2, 0, 0}}
	press(input.KeyLeft)
	if g.since >= slideTime+popTime || len(g.moved.Slides) != 2 || len(g.moved.Merged) != 1 {
		t.Fatalf("the move is being drawn: since %v, %+v", g.since, g.moved)
	}
	w.RunFrames(30)
	if g.since < slideTime+popTime {
		t.Fatalf("half a second later it has settled: since %v", g.since)
	}
}

func TestAfterNewGameTheArrowsStillPlay(t *testing.T) {
	g, w, press := play(t)
	in := g.in
	r, _ := g.hud.BoxOf(g.hud.ByID("new-game"))
	in.SetMouse(r.Center())
	w.RunFrames(1)
	in.SetMouseButton(input.MouseLeft, true)
	w.RunFrames(1)
	in.SetMouseButton(input.MouseLeft, false)
	w.RunFrames(2)
	g.board.Cells = [4][4]int{{0, 2}}
	press(input.KeyLeft)
	if g.board.Cells[0][0] != 2 {
		t.Fatalf("Left reaches the game after the click: %v", g.board.Cells)
	}
}
