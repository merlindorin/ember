package main

import (
	"math"
	"testing"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/twod/light"
)

func TestTheFlashlightPointsAtTheMouse(t *testing.T) {
	win := platform.NewWindow(platform.Size(640, 384))
	hero, err := newHero(geom.V2(320, 192), light.Unlit())
	if err != nil {
		t.Fatal(err)
	}
	beam := newFlashlight()
	hero.Add(beam)
	camera := twod.NewCamera(twod.At(320, 192))
	win.Add(hero, camera)
	w := ember.New().AddEntities(win).AddSystems(&Flashlight{beam: beam, camera: camera, in: win.Input()})

	win.Input().SetMouse(geom.V2(320, 50))
	w.RunFrames(2)
	if a := beam.Angle(); math.Abs(float64(a)+math.Pi/2) > 0.01 {
		t.Fatalf("the mouse is straight up from the hero, the beam points at %v", a)
	}
	win.Input().SetKey(input.KeyF, true)
	w.RunFrames(2)
	if !beam.Hidden() {
		t.Fatal("F switches the flashlight off")
	}
}

func TestSAndKTryTheBeamsSpreadAndSize(t *testing.T) {
	if nextSpread(0.3) != 0.6 || nextSpread(0.6) != 1 || nextSpread(1) != 0.3 {
		t.Fatal("S steps the spread through 0.3, 0.6 and 1")
	}
	win := platform.NewWindow(platform.Size(640, 384))
	beam := newFlashlight()
	camera := twod.NewCamera()
	win.Add(beam, camera)
	w := ember.New().AddEntities(win).AddSystems(&Flashlight{beam: beam, camera: camera, in: win.Input()})
	win.Input().SetKey(input.KeyK, true)
	w.RunFrames(2)
	if beam.Size() != 0 {
		t.Fatalf("K makes the lamp a point, its shadows sharp: %v", beam.Size())
	}
}
