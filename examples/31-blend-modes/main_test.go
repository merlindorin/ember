package main

import (
	"testing"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/internal/draw"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/twod"
)

func TestTheLanternFollowsTheMouseThroughItsMask(t *testing.T) {
	win := platform.NewWindow(platform.Size(640, 384))
	flame := twod.NewSprite(glow(128), twod.At(320, 192))
	lantern := twod.NewLayer(3, twod.AsMask())
	lantern.Add(flame)
	day := twod.NewLayer(2, twod.MaskedBy(lantern))
	day.Add(twod.NewSprite(glow(16)))
	win.AppendChild(day)
	win.AppendChild(lantern)
	keys := win.Input()
	w := ember.New().AddEntities(win).AddSystems(&Lantern{light: flame, keys: keys})

	w.Tick(1.0 / 60)
	if at := flame.Position(); at != geom.V2(320, 192) {
		t.Fatalf("it stays put while the mouse does: %v", at)
	}
	keys.SetMouse(geom.V2(100, 50))
	w.Tick(1.0 / 60)
	if at := flame.Position(); at != geom.V2(100, 50) {
		t.Fatalf("then follows it: %v", at)
	}
	if n := len(draw.PassesFor(w, win)); n != 2 {
		t.Fatalf("the lantern and the day it shows draw as pictures: %d passes", n)
	}
}

func TestGlowFadesFromItsMiddle(t *testing.T) {
	img := glow(64).Image
	_, _, _, middle := img.At(32, 32).RGBA()
	_, _, _, edge := img.At(0, 32).RGBA()
	if middle < 0xf000 || edge > 0x100 {
		t.Fatalf("opaque in the middle, clear at the edge: %x %x", middle, edge)
	}
}
