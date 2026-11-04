package main

import (
	"testing"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/internal/draw"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
)

func TestTheWorldDrawsAt320By180AndTheHeroWalks(t *testing.T) {
	walker, err := twod.LoadTexture(files, "walker.png", twod.Pixelated())
	if err != nil {
		t.Fatal(err)
	}
	frames := twod.Strip(walker, twod.Cell(16, 16))
	clips := twod.NewAnimations().Add("side", 8, frames[8:12], twod.Loop())
	win := platform.NewWindow(platform.Size(1000, 600))
	hero := twod.NewSprite(walker, twod.Animate(clips), twod.At(150, 120))
	eye := twod.NewCamera(twod.Resolution(320, 180), twod.Follow(hero))
	win.AppendChild(hero)
	win.AppendChild(eye)
	keys := win.Input()
	w := ember.New().AddEntities(win).AddSystems(&Walk{hero: hero, keys: keys})

	keys.SetKey(input.KeyRight, true)
	for range 60 {
		w.Tick(1.0 / 60)
	}
	if x := hero.Position().X; x < 208 || x > 212 {
		t.Fatalf("a second right at 60 pixels a second: x %v", x)
	}
	passes := draw.PassesFor(w, win)
	if len(passes) != 1 || passes[0].Width != 320 || passes[0].Height != 180 {
		t.Fatalf("one 320×180 pass: %+v", passes)
	}
	if at := eye.WorldToScreen(hero.GlobalPosition()); at.Dist(geom.V2(500, 300)) > 3 {
		t.Fatalf("the followed hero shows in the middle of the window: %v", at)
	}
}
