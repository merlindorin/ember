package main

import (
	"testing"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
)

func TestTheCrateFollowsTheGestures(t *testing.T) {
	tex, err := twod.LoadTexture(files, "crate.png")
	if err != nil {
		t.Fatal(err)
	}
	crate := twod.NewSprite(tex, twod.At(320, 200), twod.Scale(8, 8))
	win := platform.NewWindow(platform.Size(640, 420))
	win.Add(crate)
	in := win.Input()
	w := ember.New().AddEntities(win).AddSystems(&Handle{crate: crate, in: in})

	in.AddPinch(0.5)
	in.AddRotation(0.3)
	w.RunFrames(1)
	if crate.Scale() != geom.V2(12, 12) || crate.Rotation() != 0.3 {
		t.Fatalf("a pinch of 0.5 zooms by half, the rotation turns it: %v %v", crate.Scale(), crate.Rotation())
	}
	in.AddPinch(100)
	w.RunFrames(1)
	if crate.Scale().X != 32 {
		t.Fatalf("the zoom stops at four times: %v", crate.Scale())
	}

	in.AddTouch(1, input.Trackpad, geom.V2(0.4, 0.5))
	in.AddTouch(2, input.Trackpad, geom.V2(0.6, 0.5))
	w.RunFrames(1)
	in.MoveTouch(1, geom.V2(0.5, 0.5))
	in.MoveTouch(2, geom.V2(0.7, 0.5))
	w.RunFrames(1)
	if p := crate.Position(); geom.Abs(p.X-384) > 1e-3 || p.Y != 200 {
		t.Fatalf("two fingers a tenth of the pad right move it 64 points: %v", p)
	}

	in.AddSwipe(geom.V2(1, 0))
	w.RunFrames(1)
	if crate.Position() != home() || crate.Rotation() != 0 || crate.Scale() != geom.V2(8, 8) {
		t.Fatal("a swipe puts it back")
	}
}
