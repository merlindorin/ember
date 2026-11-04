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

func play(t *testing.T) (*Play, *world.World, func(input.Button)) {
	t.Helper()
	t.Setenv("EMBER_GAMES_DIR", t.TempDir())
	win := look.Window("flap")
	data := save.Defaults()
	p := newPlay(win, func() uint64 { return 4 }, &data)
	win.Add(look.Camera(), p.sky, p.puffs, p.hud)
	w := ember.New().AddEntities(win, p.player.Mixer).AddSystems(p, p.fit)
	w.RunFrames(1)
	press := func(b input.Button) {
		in := win.Input()
		switch b := b.(type) {
		case input.Key:
			in.SetKey(b, true)
			w.RunFrames(1)
			in.SetKey(b, false)
		case input.MouseButton:
			in.SetMouseButton(b, true)
			w.RunFrames(1)
			in.SetMouseButton(b, false)
		}
		w.RunFrames(1)
	}
	return p, w, press
}

func TestItWaitsThenAFlapLiftsAndIsHeard(t *testing.T) {
	p, w, press := play(t)
	rec := audio.NewRecorder(p.player.Mixer)
	y := p.game.Y
	w.RunFrames(60)
	if p.game.Y != y || p.game.Travel != 0 {
		t.Fatal("the bird waits for the first flap")
	}
	press(input.KeySpace)
	if p.game.Y >= y || p.puffs.Live() == 0 || rec.Peak(0, rec.Frames()) == 0 {
		t.Fatalf("Space flaps it up, puffs and is heard: y %v from %v", p.game.Y, y)
	}
}

func TestAClickOnTheBoardFlapsButNotOnTheHUD(t *testing.T) {
	p, w, press := play(t)
	in := p.in
	r, _ := p.hud.BoxOf(p.hud.ByID("side"))
	in.SetMouse(r.Center())
	w.RunFrames(1)
	press(input.MouseLeft)
	if p.started {
		t.Fatal("a click on the HUD isn't a flap")
	}
	in.SetMouse(p.sky.Position())
	w.RunFrames(1)
	press(input.MouseLeft)
	if !p.started {
		t.Fatal("a click on the sky flaps")
	}
}

func TestACrashIsOnTheLeaderboard(t *testing.T) {
	p, w, press := play(t)
	press(input.KeySpace)
	w.RunFrames(240)
	if !p.game.Over {
		t.Fatal("without flapping it falls to the ground")
	}
	if _, open := p.over.Attr("open"); !open || len(p.data.Scores) != 1 {
		t.Fatalf("the question and the score: open %v, %+v", open, p.data.Scores)
	}
}

func TestEscapeOpensTheMenuAndTheWorldWaits(t *testing.T) {
	p, w, press := play(t)
	press(input.KeySpace)
	press(input.KeyEscape)
	travel := p.game.Travel
	w.RunFrames(60)
	if !p.menu.Open() || p.game.Travel != travel {
		t.Fatalf("the menu stops the world: open %v", p.menu.Open())
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
	y := p.game.Y
	pad(input.GamepadSouth)
	if p.game.Y >= y {
		t.Fatalf("A flaps: y %v from %v", p.game.Y, y)
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
