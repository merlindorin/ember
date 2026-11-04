package main

import (
	"testing"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
)

func TestAClickThrowsABurstThatRemovesItself(t *testing.T) {
	win := platform.NewWindow(platform.Size(640, 360))
	layer := twod.NewLayer(1)
	win.Add(layer)
	w := ember.New().AddEntities(win).AddSystems(&Bursts{window: win, layer: layer})

	win.Input().SetMouse(geom.V2(200, 150))
	win.Input().SetMouseButton(input.MouseLeft, true)
	w.Tick(1.0 / 60)
	win.Input().SetMouseButton(input.MouseLeft, false)
	w.Tick(1.0 / 60)
	kids := layer.GetSceneNode().ChildNodes()
	if len(kids) != 1 {
		t.Fatalf("one burst: %d", len(kids))
	}
	burst, ok := kids[0].Entity().(*twod.Particles)
	if !ok {
		t.Fatalf("the burst is a %T", kids[0].Entity())
	}
	if burst.Live() != 80 || burst.Position() != geom.V2(200, 150) {
		t.Fatalf("80 embers where clicked: %d at %v", burst.Live(), burst.Position())
	}
	for range 120 {
		w.Tick(1.0 / 60)
	}
	if !burst.Done() || w.Has(burst) || len(layer.GetSceneNode().ChildNodes()) != 0 {
		t.Fatal("gone once its embers die")
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
