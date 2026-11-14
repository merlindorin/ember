package twod //nolint:testpackage // white-box: shares twod's test helpers, reaches unexported state

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/internal/draw"
	"github.com/lantervale/ember/render"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLetterboxScalesByWholeFactorsCentredOnPixels(t *testing.T) {
	m := letterbox(geom.V2(320, 180), geom.V2(800, 600), geom.Vec2{}, false)
	assert.Equal(t, geom.V2(2, 0), m.X, "800/320 is 2.5: the largest whole factor is 2")
	assert.Equal(t, geom.V2(80, 120), m.Origin, "640×360 centred in 800×600")

	m = letterbox(geom.V2(320, 180), geom.V2(800, 600), geom.V2(1600, 1200), false)
	assert.Equal(t, geom.V2(2.5, 0), m.X, "a factor of 5 framebuffer pixels, at 2 pixels a point")
	assert.Equal(t, geom.V2(0, 75), m.Origin)

	m = letterbox(geom.V2(320, 180), geom.V2(800, 600), geom.Vec2{}, true)
	assert.Equal(t, geom.V2(2.5, 0), m.X, "FitWindow fills the width")
	assert.Equal(t, geom.V2(0, 75), m.Origin)

	m = letterbox(geom.V2(320, 180), geom.V2(160, 90), geom.Vec2{}, false)
	assert.Equal(t, geom.V2(0.5, 0), m.X, "a window smaller than the resolution shrinks it")
}

func TestAResolutionCameraDrawsItsWorldIntoAPassShownLetterboxed(t *testing.T) {
	w, win, c := app()
	cam := NewCamera(Resolution(320, 180), At(160, 90))
	hero := NewSprite(atlas(16, 16), At(160, 90))
	hud := NewLayer(1, Fixed())
	hud.Add(NewLine([]geom.Vec2{geom.V2(0, 10), geom.V2(100, 10)}))
	win.AppendChild(cam)
	win.AppendChild(hero)
	win.AppendChild(hud)
	w.AddEntities(cam, hero, hud)
	w.RunFrames(1)

	passes := draw.PassesFor(w, win)
	require.Len(t, passes, 1)
	p := passes[0]
	assert.Equal(t, 320, p.Width)
	assert.Equal(t, 180, p.Height)
	assert.Equal(t, render.FilterNearest, p.Target.Filter, "sharp by default")
	lo, hi := corners(p.List.Unindexed())
	assert.Equal(t, geom.V2(152, 82), lo, "the hero in the middle of 320×180 pixels")
	assert.Equal(t, geom.V2(168, 98), hi)

	require.Len(t, c.Unindexed(), 12, "the pass's picture, then the fixed HUD at the window's resolution")
	assert.Equal(t, p.Target, c.Batches[0].Image())
	lo, hi = corners(c.Unindexed()[:6])
	assert.Equal(t, geom.V2(80, 120), lo, "scaled by 2 and centred in 800×600")
	assert.Equal(t, geom.V2(720, 480), hi)
	assert.Equal(t, geom.V2(0, 0), c.Unindexed()[0].UV, "the whole picture, like any image")
	assert.Equal(t, float32(draw.KindImage), c.Unindexed()[0].Shape.X)
	lo, _ = corners(c.Unindexed()[6:])
	assert.InDelta(t, 0, lo.X, 1e-4, "the HUD is not scaled")

	at, ok := toScreen(w, hero)
	require.True(t, ok)
	assert.Equal(t, geom.V2(400, 300), at, "through the camera and the scaling")
	assert.Equal(t, geom.V2(160, 90), cam.ScreenToWorld(geom.V2(400, 300)))
	assert.Equal(t, geom.V2(0, 0), cam.ScreenToWorld(geom.V2(80, 120)), "the picture's top-left corner")
}

func TestAResolutionOfZeroDrawsAtTheWindowsResolutionAgain(t *testing.T) {
	w, win, _ := app()
	cam := NewCamera(Resolution(320, 180))
	win.AppendChild(cam)
	win.AppendChild(NewSprite(atlas(16, 16)))
	w.AddEntities(cam)
	w.RunFrames(1)
	require.Len(t, draw.PassesFor(w, win), 1)
	cam.SetResolution(0, 0)
	w.RunFrames(1)
	assert.Empty(t, draw.PassesFor(w, win), "no pass: the world draws in the window")
	rw, rh := cam.Resolution()
	assert.Equal(t, [2]int{0, 0}, [2]int{rw, rh})
	cam.SetResolution(400, 300)
	w.RunFrames(1)
	require.Len(t, draw.PassesFor(w, win), 1, "and a resolution again draws into one")
}

func TestUpscaleAndFitWindow(t *testing.T) {
	w, win, c := app()
	cam := NewCamera(Resolution(320, 180), Upscale(Linear), FitWindow())
	win.AppendChild(cam)
	win.AppendChild(NewSprite(atlas(16, 16)))
	w.AddEntities(cam)
	w.RunFrames(1)
	require.Len(t, draw.PassesFor(w, win), 1)
	assert.Equal(t, render.FilterLinear, draw.PassesFor(w, win)[0].Target.Filter)
	lo, hi := corners(c.Unindexed()[:6])
	assert.Equal(t, geom.V2(0, 75), lo, "fills the width")
	assert.Equal(t, geom.V2(800, 525), hi)
}

func TestTheMouseMapsIntoAResolutionCamerasWorld(t *testing.T) {
	w, win, _ := app()
	cam := NewCamera(Resolution(320, 180), At(160, 90))
	probe := &mouseProbe{Item: Base()}
	win.AppendChild(cam)
	win.AppendChild(Build(probe, "probe"))
	w.AddEntities(cam)
	win.Input().SetMouse(geom.V2(400, 300))
	w.RunFrames(1)
	assert.Equal(t, geom.V2(160, 90), probe.mouse, "the middle of the window is the middle of the world")
	assert.Equal(t, geom.V2(320, 180), probe.screen, "the screen is the pass's")
}

type mouseProbe struct {
	Item
	mouse, screen geom.Vec2
}

func (p *mouseProbe) Draw(c *Canvas) {
	p.mouse, p.screen = c.Mouse(), sizeOf(c.Screen())
}
