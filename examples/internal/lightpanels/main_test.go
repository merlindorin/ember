package main

import (
	"testing"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/twod/light"
)

func TestEveryPanelBuilds(t *testing.T) {
	for name, p := range panels() {
		if p.call == "" {
			t.Errorf("%s shows no call", name)
		}
		win := scene(p)
		w := ember.New().AddEntities(win)
		light.SetDebug(win, p.debug)
		w.RunFrames(2)
		if light.DebugOf(win) != p.debug {
			t.Errorf("%s shows the debug view %v, want %v", name, light.DebugOf(win), p.debug)
		}
	}
}

func TestTheStrengthIsFullAtTheLampAndNothingAtTheRadius(t *testing.T) {
	for _, f := range []float32{0, 0.5, 1} {
		pts := strength(f, 96, 48)
		if first, last := pts[0], pts[len(pts)-1]; first.Y != 0 || last.Y != 48 {
			t.Errorf("falloff %v: from %v to %v", f, first, last)
		}
	}
	if half := strength(0.5, 96, 48)[20]; half.Y != 0 {
		t.Errorf("with falloff 0.5, the light is full before half the radius: %v", half)
	}
}
