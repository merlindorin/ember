package main

import (
	"testing"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
)

func TestCratesDragThroughTheZoomAndMovesUndo(t *testing.T) {
	tex, err := twod.LoadTexture(files, "crate.png")
	if err != nil {
		t.Fatal(err)
	}
	crate := twod.NewSprite(tex, twod.At(0, 0), twod.Size(64, 64))
	eye := twod.NewCamera()
	win := platform.NewWindow(platform.Size(640, 360))
	win.Add(twod.NewLayer(0).Add(crate), eye)
	in := win.Input()
	carry := moveCursor()
	w := ember.New().
		AddEntities(win).
		AddSystems(&Drag{window: win, eye: eye, crates: []*twod.Sprite{crate}, carry: carry})
	click := func(b input.MouseButton) {
		in.SetMouseButton(b, true)
		w.RunFrames(1)
		in.SetMouseButton(b, false)
		w.RunFrames(1)
	}

	in.AddWheel(geom.V2(0, -10))
	w.RunFrames(2)
	if eye.Zoom() != 2 {
		t.Fatalf("ten lines up zoom in twice: %v", eye.Zoom())
	}

	in.SetMouse(geom.V2(330, 190))
	w.RunFrames(1)
	if win.Cursor() != platform.CursorHand {
		t.Fatal("the hand shows over a crate")
	}
	in.SetMouseButton(input.MouseLeft, true)
	w.RunFrames(1)
	in.SetMouse(geom.V2(430, 190))
	w.RunFrames(1)
	if win.Cursor() != carry {
		t.Fatal("the move cursor shows while a crate is carried")
	}
	in.SetMouseButton(input.MouseLeft, false)
	w.RunFrames(1)
	if crate.Position() != geom.V2(50, 0) {
		t.Fatalf("a hundred points on screen are fifty in the world at zoom 2: %v", crate.Position())
	}

	click(input.MouseBack)
	if crate.Position() != geom.V2(0, 0) {
		t.Fatalf("Back undoes the move: %v", crate.Position())
	}
	click(input.MouseForward)
	if crate.Position() != geom.V2(50, 0) {
		t.Fatalf("Forward redoes it: %v", crate.Position())
	}
	in.SetMouse(geom.V2(20, 20))
	w.RunFrames(1)
	if win.Cursor() != platform.CursorArrow {
		t.Fatal("away from the crates, the arrow")
	}
}
