package main

import (
	"testing"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/platform"
)

func TestOnlyTheCrystalCastsShadows(t *testing.T) {
	at := geom.V2(0, 0)
	if newNeon(at).Shadows() || newRune(at).Shadows() {
		t.Fatal("a shape light shines through everything")
	}
	if !newCrystal(at).Shadows() {
		t.Fatal("the crystal's light is blocked by what's solid")
	}
}

func TestTheLavaIsHotAtItsHeart(t *testing.T) {
	img := lavaPicture().Image
	b := img.Bounds()
	heart := img.At(b.Dx()/2, b.Dy()/2)
	_, g, _, a := heart.RGBA()
	_, _, _, corner := img.At(0, 0).RGBA()
	if a == 0 || g < 0xc000 || corner != 0 {
		t.Fatalf("the pool is an oval, yellow at its heart: heart %v, corner alpha %v", heart, corner)
	}
}

func TestXMakesTheCrystalShineThrough(t *testing.T) {
	win := platform.NewWindow()
	crystal := newCrystal(geom.V2(0, 0))
	win.Add(crystal)
	w := ember.New().AddEntities(win).AddSystems(&Keys{crystal: crystal, in: win.Input()})
	win.Input().SetKey(input.KeyX, true)
	w.RunFrames(2)
	if crystal.Shadows() {
		t.Fatal("X lets the crystal's light through the trees and pillars")
	}
}
