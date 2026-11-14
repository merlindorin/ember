package main

import (
	"testing"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/twod"
)

func TestEachFingerIsDrawnWhereItIsOnThePad(t *testing.T) {
	win := platform.NewWindow(platform.Size(640, 420))
	layer := twod.NewLayer(1)
	win.Add(newPad(), layer)
	in := win.Input()
	fingers := newFingers(in, layer)
	w := ember.New().AddEntities(win).AddSystems(fingers)

	in.AddTouch(1, input.Trackpad, geom.V2(0.5, 0.5))
	in.AddTouch(2, input.Trackpad, geom.V2(0, 1))
	w.RunFrames(1)
	if len(fingers.fingers) != 2 {
		t.Fatalf("two fingers: %d", len(fingers.fingers))
	}
	one := fingers.fingers[1]
	if one.spot.Position() != geom.V2(320, 200) || one.label.Text() != "1 · began" {
		t.Fatalf("the pad's middle: %v %q", one.spot.Position(), one.label.Text())
	}
	if p := fingers.fingers[2].spot.Position(); p != geom.V2(80, 350) {
		t.Fatalf("the pad's bottom-left: %v", p)
	}

	in.MoveTouch(1, geom.V2(0.75, 0.5))
	w.RunFrames(1)
	if one.spot.Position() != geom.V2(440, 200) || one.label.Text() != "1 · moving" {
		t.Fatalf("moved: %v %q", one.spot.Position(), one.label.Text())
	}
	w.RunFrames(1)
	if one.label.Text() != "1 · resting" {
		t.Fatalf("still: %q", one.label.Text())
	}

	in.EndTouch(1)
	w.RunFrames(1)
	if _, ok := fingers.fingers[1]; ok || len(layer.Children()) != 2 {
		t.Fatalf("a lifted finger goes: %d left on the layer", len(layer.Children()))
	}
}
