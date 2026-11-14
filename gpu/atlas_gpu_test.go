//go:build gpu

package gpu //nolint:testpackage // white-box: drives the renderer's unexported passes

import (
	"image"
	"testing"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/internal/draw"
	"github.com/lantervale/ember/internal/wgpu"
	"github.com/lantervale/ember/internal/wm"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/world"
)

// glyphFrame draws "A" in a size of its own, then, when more is set, a
// run of new glyphs added only in the frame drawn, after the A: enough to
// grow the size's atlas while that frame is being drawn.
func glyphFrame(t *testing.T, more bool) *image.RGBA {
	t.Helper()
	d := device(t)
	r := &webgpu2D{}
	if err := r.init(d); err != nil {
		t.Fatal(err)
	}
	w := world.New().AddPlugins(wm.Plugin{}, twod.Plugin{})
	win := platform.NewWindow(platform.Size(200, 100), platform.Background(color.Black))
	win.Add(twod.NewCamera(twod.At(100, 50)))
	layer := twod.NewLayer(0).Add(twod.NewText("A", twod.FontSize(23), twod.At(20, 30), twod.Z(0)))
	win.Add(layer)
	w.AddEntities(win)
	w.RunFrames(1)
	if more {
		var run []rune
		for c := rune('!'); c <= 0x17f; c++ {
			if c != 'A' && (c < 0x7f || c > 0xa0) {
				run = append(run, c)
			}
		}
		layer.Add(twod.NewText(string(run), twod.FontSize(23), twod.At(1.3, 80), twod.Z(1)))
	}
	w.RunFrames(1)
	target := d.CreateTexture(200, 100, wgpu.RGBA8Unorm, wgpu.UsageRenderAttachment|wgpu.UsageCopySrc)
	r.window(w, win, target, d.NewTimers(), draw.PassesFor(w, win), draw.BackFor(w, win), draw.FrontFor(w, win))
	if err := wgpu.TakeError(); err != nil {
		t.Fatal(err)
	}
	img, err := d.Read(target)
	if err != nil {
		t.Fatal(err)
	}
	return img
}

func TestGlyphsDrawnBeforeTheAtlasGrowsInTheSameFrameStayRight(t *testing.T) {
	alone := glyphFrame(t, false)
	grown := glyphFrame(t, true)
	differ := 0
	for y := 0; y < 60; y++ {
		for x := 0; x < 60; x++ {
			if alone.RGBAAt(x, y) != grown.RGBAAt(x, y) {
				differ++
			}
		}
	}
	if differ > 0 {
		t.Fatalf("the A drawn before its atlas grew differs in %d pixels from the A alone", differ)
	}
}
