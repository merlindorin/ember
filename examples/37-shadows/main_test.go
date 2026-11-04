package main

import (
	"testing"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/twod/light"
)

func TestTheLampFollowsTheMouseAndRises(t *testing.T) {
	win := platform.NewWindow(platform.Size(640, 384))
	lamp, camera, label := newLamp(), twod.NewCamera(twod.At(320, 192)), twod.NewText("")
	win.Add(camera, lamp, label)
	w := ember.New().AddEntities(win).AddSystems(&Lamp{lamp: lamp, camera: camera, in: win.Input(), label: label})

	win.Input().SetMouse(geom.V2(100, 80))
	win.Input().SetKey(input.KeyUp, true)
	w.RunFrames(30)
	if lamp.Position() != geom.V2(100, 80) {
		t.Fatalf("the lamp is under the mouse: %v", lamp.Position())
	}
	if lamp.Elevation() <= 100 {
		t.Fatalf("up raises it: %v", lamp.Elevation())
	}
	win.Input().SetKey(input.KeyUp, false)
	win.Input().SetKey(input.KeyDown, true)
	w.RunFrames(300)
	if lamp.Elevation() != 0 {
		t.Fatalf("down lowers it to the ground, no further: %v", lamp.Elevation())
	}
}

func TestSAndFSoftenAndFadeTheShadows(t *testing.T) {
	win := platform.NewWindow(platform.Size(640, 384))
	lamp, camera, label := newLamp(), twod.NewCamera(), twod.NewText("")
	win.Add(camera, lamp, label)
	w := ember.New().AddEntities(win).AddSystems(&Lamp{lamp: lamp, camera: camera, in: win.Input(), label: label})

	win.Input().SetKey(input.KeyS, true)
	win.Input().SetKey(input.KeyF, true)
	w.RunFrames(2)
	if lamp.Size() != 12 || lamp.ShadowFade() != light.FadeLinear {
		t.Fatalf("S gives the lamp a size, F fades its shadows: %v, %v", lamp.Size(), lamp.ShadowFade())
	}
}

func TestFStepsThroughTheFadeCurves(t *testing.T) {
	f := light.FadeNone
	for _, want := range []light.Fade{light.FadeLinear, light.FadeSmooth, light.FadeSoft, light.FadeNone} {
		if got := next(f); got != want {
			t.Fatalf("after %v comes %v, not %v", f, want, got)
		}
		f = want
	}
}
