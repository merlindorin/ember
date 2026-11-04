package main

import (
	"testing"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/audio"
	"github.com/merlindorin/ember/games/internal/look"
	"github.com/merlindorin/ember/games/internal/save"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/ui/align"
	"github.com/merlindorin/ember/world"
)

func play(t *testing.T) (*Play, *world.World, func(input.Key)) {
	t.Helper()
	t.Setenv("EMBER_GAMES_DIR", t.TempDir())
	win := look.Window("snake")
	data := save.Defaults()
	p := newPlay(win.Input(), func() uint64 { return 1 }, &data)
	win.Add(p.pit, p.sparks, p.hud)
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

func TestItWaitsForAnArrowThenMoves(t *testing.T) {
	p, w, press := play(t)
	head := p.game.Head()
	w.RunFrames(60)
	if p.game.Head() != head {
		t.Fatal("the snake waits for the first arrow")
	}
	press(input.KeyDown)
	w.RunFrames(30)
	if p.game.Head().Y <= head.Y {
		t.Fatalf("Down starts it, heading down: %v from %v", p.game.Head(), head)
	}
	if _, hidden := p.start.Attr("hidden"); !hidden {
		t.Fatal("the start line goes")
	}
}

func TestAnAppleSparksIsHeardAndCounts(t *testing.T) {
	p, w, press := play(t)
	rec := audio.NewRecorder(p.player.Mixer)
	head := p.game.Head()
	p.game.Apple = Cell{head.X + 1, head.Y}
	press(input.KeyUp)
	p.game.turns = nil
	w.RunFrames(12)
	if p.game.Eaten != 1 || p.score.Text() != "10" || p.length.Text() != "4" {
		t.Fatalf("eaten %d, HUD %q %q", p.game.Eaten, p.score.Text(), p.length.Text())
	}
	if p.sparks.Live() == 0 || rec.Peak(0, rec.Frames()) == 0 {
		t.Fatal("sparks fly and the bite is heard")
	}
}

func TestACrashIsOnTheLeaderboard(t *testing.T) {
	p, w, press := play(t)
	press(input.KeyRight)
	w.RunFrames(240)
	if !p.game.Over {
		t.Fatal("heading right into the wall")
	}
	if _, open := p.over.Attr("open"); !open || len(p.data.Scores) != 1 {
		t.Fatalf("the question and the score: open %v, %+v", open, p.data.Scores)
	}
}

func TestEscapeOpensTheMenuAndTheSnakeWaits(t *testing.T) {
	p, w, press := play(t)
	press(input.KeyRight)
	press(input.KeyEscape)
	head := p.game.Head()
	w.RunFrames(60)
	if !p.menu.Open() || p.game.Head() != head {
		t.Fatalf("the menu stops the snake: open %v", p.menu.Open())
	}
	if faults := align.Check(p.hud); len(faults) > 0 {
		t.Fatalf("layout faults: %v", faults)
	}
	if warns := p.hud.Warnings(); len(warns) > 0 {
		t.Fatalf("CSS warnings: %v", warns)
	}
}
