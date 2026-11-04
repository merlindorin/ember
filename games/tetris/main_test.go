package main

import (
	"strconv"
	"testing"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/games/internal/look"
	"github.com/merlindorin/ember/games/internal/save"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/ui/align"
	"github.com/merlindorin/ember/world"
)

func play(t *testing.T) (*Play, *world.World, func(input.Key)) {
	t.Helper()
	t.Setenv("EMBER_GAMES_DIR", t.TempDir())
	win := look.Window("tetris")
	data := save.Defaults()
	p := newPlay(win.Input(), func() uint64 { return 1 }, &data)
	win.Add(p.well, p.sparks, p.hud)
	w := ember.New().AddEntities(win, p.player.Mixer).AddSystems(p)
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
	p.hud.Close(p.over, "again")
	w.RunFrames(1)
	if p.game.Over || p.score.Text() != "0" {
		t.Fatalf("New game starts again: over %v, score %q", p.game.Over, p.score.Text())
	}
}

func TestTheBoardSitsLeftOfACleanHUD(t *testing.T) {
	p, _, _ := play(t)
	panel, ok := p.hud.BoxOf(p.hud.ByID("side"))
	if !ok || panel.X < look.HUDLeft {
		t.Fatalf("the HUD keeps right of the board's area: %v", panel)
	}
	if next, shown := p.hud.BoxOf(p.next); !shown || next.W != 64 || next.H != 32 {
		t.Fatalf("the preview is 64 by 32: %v", next)
	}
	if faults := align.Check(p.hud); len(faults) > 0 {
		t.Fatalf("layout faults: %v", faults)
	}
	if warns := p.hud.Warnings(); len(warns) > 0 {
		t.Fatalf("CSS warnings: %v", warns)
	}
}

func TestEnterAnswersTheGameOverQuestion(t *testing.T) {
	p, w, press := play(t)
	for y := 2; y < H; y++ {
		for x := range W - 1 {
			p.game.Board[y][x] = Z
		}
	}
	press(input.KeySpace)
	w.RunFrames(40)
	if warns := p.hud.Warnings(); len(warns) > 0 {
		t.Fatalf("the question's CSS: %v", warns)
	}
	press(input.KeyEnter)
	w.RunFrames(1)
	if _, open := p.over.Attr("open"); open || p.game.Over {
		t.Fatalf("Enter presses New game: open %v, over %v", open, p.game.Over)
	}
}
