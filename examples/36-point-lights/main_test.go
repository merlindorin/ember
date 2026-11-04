package main

import (
	"testing"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod/light"
)

func TestTheHeroCarriesTheTorch(t *testing.T) {
	hero, err := newHero(geom.V2(100, 100), light.Unlit())
	if err != nil {
		t.Fatal(err)
	}
	torch := newTorch()
	hero.Add(torch)
	if torch.GlobalPosition() != geom.V2(100, 92) {
		t.Fatalf("the torch is held at the hero, 12 of his pixels up: %v", torch.GlobalPosition())
	}
	hero.SetPosition(geom.V2(200, 150))
	if torch.GlobalPosition() != geom.V2(200, 142) {
		t.Fatalf("and goes where he goes: %v", torch.GlobalPosition())
	}
}

func TestTheTorchFlickersWithinAFewPercent(t *testing.T) {
	torch := newTorch()
	w := ember.New().AddEntities(torch).AddSystems(&Flicker{torch: torch})
	lo, hi := float32(2), float32(0)
	for range 240 {
		w.RunFrames(1)
		lo, hi = min(lo, torch.Intensity()), max(hi, torch.Intensity())
	}
	if lo < 0.8 || hi > 1 || hi-lo < 0.05 {
		t.Fatalf("a flame wavers a little: %v to %v", lo, hi)
	}
}

func TestRAndXTryTheTorchsOptions(t *testing.T) {
	win := platform.NewWindow()
	torch := newTorch()
	win.Add(torch)
	w := ember.New().AddEntities(win).AddSystems(&Keys{torch: torch, in: win.Input()})

	win.Input().SetKey(input.KeyR, true)
	win.Input().SetKey(input.KeyX, true)
	w.RunFrames(2)
	if torch.Radius() != 220 || torch.Shadows() {
		t.Fatalf("R steps the radius from 150 to 220, X makes it x-ray: %v, %v", torch.Radius(), !torch.Shadows())
	}
	for _, r := range []float32{100, 150, 220} {
		if got := nextRadius(r); got == r {
			t.Fatalf("R leaves the radius at %v", r)
		}
	}
	if nextRadius(220) != 100 {
		t.Fatal("after the widest, the radius comes back to 100")
	}
}
