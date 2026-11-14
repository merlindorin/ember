package main

import (
	"testing"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/internal/draw"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/twod"
)

func TestTheMinimapCameraFeedsItsTexture(t *testing.T) {
	win := platform.NewWindow(platform.Size(960, 540))
	minimap := twod.NewRenderTexture(240, 128)
	hero := twod.NewSprite(twod.NewRenderTexture(16, 16), twod.At(240, 128))
	win.AppendChild(hero)
	win.AppendChild(twod.NewCamera(twod.Into(minimap), twod.Zoom(0.5), twod.At(240, 128)))
	hud := twod.NewLayer(1, twod.Fixed())
	hud.Add(twod.NewSprite(minimap, twod.At(824, 80)))
	win.AppendChild(hud)
	w := ember.New().AddEntities(win)
	w.Tick(1.0 / 60)

	passes := draw.PassesFor(w, win)
	if len(passes) != 1 || passes[0].Target != &minimap.Texture || passes[0].Width != 240 {
		t.Fatalf("one 240×128 pass into the minimap: %+v", passes)
	}
	if n := len(passes[0].List.Unindexed()); n != 6 {
		t.Fatalf("the hero alone in the minimap, not the HUD: %d vertices", n)
	}
	if at, _ := draw.ToScreen(w, hero); at.X != 240 || at.Y != 128 {
		t.Fatalf("without a camera of its own, the window shows the world as it is: %v", at)
	}
}
