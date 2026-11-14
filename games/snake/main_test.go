package main

import (
	"testing"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/audio"
	"github.com/lantervale/ember/games/internal/look"
	"github.com/lantervale/ember/games/internal/save"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/ui/align"
	"github.com/lantervale/ember/world"
)

func play(t *testing.T) (*Play, *world.World, func(input.Key)) {
	t.Helper()
	t.Setenv("EMBER_GAMES_DIR", t.TempDir())
	win := look.Window("snake")
	data := save.Defaults()
	p := newPlay(win, func() uint64 { return 1 }, &data)
	win.Add(look.Camera(), p.pit, p.sparks, p.hud)
	w := ember.New().AddEntities(win, p.player.Mixer).AddSystems(p, p.fit)
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

func TestAGamepadPlaysTheGameAndItsMenu(t *testing.T) {
	p, w, _ := play(t)
	in := p.in
	in.ConnectGamepad(0, "pad")
	pad := func(b input.GamepadButton) {
		in.SetGamepadButton(0, b, true)
		w.RunFrames(1)
		in.SetGamepadButton(0, b, false)
		w.RunFrames(1)
	}
	head := p.game.Head()
	pad(input.GamepadDown)
	w.RunFrames(30)
	if p.game.Head().Y <= head.Y {
		t.Fatalf("the cross steers: %v from %v", p.game.Head(), head)
	}
	if !p.hud.Body().HasClass("gamepad") {
		t.Fatal("the HUD shows the pad's prompts")
	}
	if faults := align.Check(p.hud); len(faults) > 0 {
		t.Fatalf("the pad's prompts' layout: %v", faults)
	}
	pad(input.GamepadStart)
	if !p.menu.Open() {
		t.Fatal("Menu opens the menu")
	}
	pad(input.GamepadEast)
	pad(input.GamepadBack)
	if p.menu.Open() || p.started {
		t.Fatalf("B closes the menu, View starts again: open %v, started %v", p.menu.Open(), p.started)
	}
}
