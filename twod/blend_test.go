package twod //nolint:testpackage // white-box: shares twod's test helpers, reaches unexported state

import (
	"testing"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/internal/draw"
	"github.com/merlindorin/ember/render"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlendAppliesToTheEntityNotItsChildren(t *testing.T) {
	w, win, c := app()
	glow := NewSprite(atlas(8, 8), At(100, 100), Blend(Add))
	glow.Add(NewSprite(atlas(8, 8)))
	win.AppendChild(glow)
	w.AddEntities(glow)
	w.RunFrames(1)
	require.Len(t, c.Batches, 2)
	assert.Equal(t, draw.Add, c.Batches[0].Blend)
	assert.Equal(t, draw.Alpha, c.Batches[1].Blend, "the child keeps its own")
	assert.Equal(t, Add, glow.Blend())
}

func TestLightenKeepsItsModeToTheDrawListAndInScenes(t *testing.T) {
	w, win, c := app()
	ambient := NewSprite(atlas(8, 8), Blend(Lighten))
	win.AppendChild(ambient)
	w.AddEntities(ambient)
	w.RunFrames(1)
	require.Len(t, c.Batches, 1)
	assert.Equal(t, draw.Lighten, c.Batches[0].Blend)
	text, err := Lighten.MarshalText()
	require.NoError(t, err)
	var back BlendMode
	require.NoError(t, back.UnmarshalText(text))
	assert.Equal(t, "lighten", string(text))
	assert.Equal(t, Lighten, back)
}

func TestABlendedLayerShowsAsOnePicture(t *testing.T) {
	w, win, c := app()
	night := NewLayer(2, Blend(Multiply))
	night.Add(NewSprite(atlas(8, 8), At(10, 10)), NewSprite(atlas(8, 8), At(30, 10)))
	win.AppendChild(night)
	w.AddEntities(night)
	w.RunFrames(1)

	passes := draw.PassesFor(w, win)
	require.Len(t, passes, 1)
	assert.Len(t, passes[0].List.Unindexed(), 12, "both sprites, in the pass")
	require.Len(t, c.Batches, 1)
	assert.Equal(t, draw.Multiply, c.Batches[0].Blend, "the picture multiplies")
	assert.Equal(t, draw.Alpha, passes[0].List.Batches[0].Blend, "its sprites draw normally into it")
	shown := shownAt(c.Unindexed())
	assert.True(t, contains(shown, geom.Rect{X: 6, Y: 6, W: 28, H: 8}), "the picture covers both sprites: %v", shown)
	assert.Equal(t, geom.V2(shown.W, shown.H), passes[0].Screen, "its pass is that much of the window")
	assert.Equal(t, int(shown.W), passes[0].Width, "at the window's resolution")
	assert.Less(t, passes[0].Width, 800, "cropped to what was drawn")
}

// shownAt is the rectangle a picture's vertices cover.
func shownAt(vs []draw.Vertex) geom.Rect {
	lo, hi := corners(vs)
	return geom.Rect{X: lo.X, Y: lo.Y, W: hi.X - lo.X, H: hi.Y - lo.Y}
}

func contains(outer, inner geom.Rect) bool {
	return outer.X <= inner.X && outer.Y <= inner.Y && outer.X+outer.W >= inner.X+inner.W &&
		outer.Y+outer.H >= inner.Y+inner.H
}

func TestAScaledPassIsFilledSmallAndShownOverTheWholeWindow(t *testing.T) {
	w, win, c := app()
	light := NewLayer(2, Blend(Multiply), PassScale(0.25))
	plain := NewLayer(3, Blend(Add))
	light.Add(NewSprite(atlas(8, 8), At(10, 10)))
	plain.Add(NewSprite(atlas(8, 8), At(10, 10)))
	win.Add(light, plain)
	w.AddEntities(light, plain)
	w.RunFrames(1)

	sizes := func() (*draw.Pass, *draw.Pass) {
		passes := draw.PassesFor(w, win)
		require.Len(t, passes, 2)
		if passes[0].Width < passes[1].Width {
			return passes[0], passes[1]
		}
		return passes[1], passes[0]
	}
	scaled, full := sizes()
	assert.Equal(t, float32(full.Width), full.Screen.X, "an unscaled layer draws a pixel per point")
	assert.Equal(t, float32(scaled.Width)*4, scaled.Screen.X, "a quarter of that")
	assert.Equal(t, float32(scaled.Height)*4, scaled.Screen.Y)
	assert.Equal(t, Linear, scaled.Target.Filter, "stretched smooth")
	assert.True(t, contains(shownAt(c.Unindexed()[:6]), geom.Rect{X: 6, Y: 6, W: 8, H: 8}), "shown over its sprite")

	win.SetSize(geom.V2(1600, 900))
	w.RunFrames(1)
	scaled, full = sizes()
	assert.InDelta(
		t,
		float32(full.Width)/full.Screen.X/4,
		float32(scaled.Width)/scaled.Screen.X,
		1e-4,
		"still a quarter of the unscaled layer's pixels once the window grows",
	)
	light.SetPassScale(2)
	assert.Equal(t, float32(1), light.PassScale(), "never above the window's own resolution")
	assert.Equal(t, float32(1), NewLayer(0).PassScale())
}

func TestAMaskedLayerShowsThroughItsMask(t *testing.T) {
	w, win, c := app()
	lantern := NewLayer(3, AsMask())
	lantern.Add(NewSprite(atlas(8, 8), At(50, 50)))
	lit := NewLayer(2, MaskedBy(lantern))
	lit.Add(NewSprite(atlas(8, 8), At(10, 10)))
	win.AppendChild(lit)
	win.AppendChild(lantern)
	w.AddEntities(lit, lantern)
	w.RunFrames(1)

	passes := draw.PassesFor(w, win)
	require.Len(t, passes, 2)
	assert.Contains(t, passes, lantern.pass)
	assert.Contains(t, passes, lit.pass)
	assert.Len(t, lantern.pass.List.Unindexed(), 6, "the mask draws into its pass")
	require.Len(t, c.Batches, 1, "the mask itself never shows")
	assert.Equal(t, lit.pass.Target, c.Batches[0].Image())
	assert.Equal(t, lantern.pass.Target, c.Batches[0].Mask)
	assert.Equal(t, lantern.pass.Screen, lit.pass.Screen, "the two passes share one rectangle, so the mask lines up")
	assert.Equal(t, lantern.pass.Width, lit.pass.Width)
	assert.True(
		t,
		contains(shownAt(c.Unindexed()), geom.Rect{X: 46, Y: 46, W: 8, H: 8}),
		"cropped to the mask, which is where it can show",
	)
}

func TestAPassIsCroppedOnWholePixelsWithTransparentOnesAround(t *testing.T) {
	w, win, c := app()
	glow := NewLayer(1, Blend(Add))
	glow.Add(NewSprite(atlas(8, 8), At(400.5, 300)))
	win.Add(glow)
	w.AddEntities(glow)
	w.RunFrames(1)
	shown := shownAt(c.Unindexed())
	assert.Equal(t, shown.X, float32(int(shown.X)), "on a whole pixel")
	assert.Equal(t, 0, glow.pass.Width%cropStep, "sized in steps, so a moving picture keeps its target")
	e, ok := glow.pass.List.Extent()
	require.True(t, ok)
	assert.GreaterOrEqual(t, e.X, float32(cropMargin), "a margin of transparent pixels on the left")
	assert.GreaterOrEqual(t, shown.W-(e.X+e.W), float32(cropMargin), "and on the right")
	assert.Equal(t, geom.V2(400.5-4, 296), geom.V2(shown.X+e.X, shown.Y+e.Y), "drawn where it was")
}

func TestALayerWithEffectsKeepsItsWholePass(t *testing.T) {
	w, win, _ := app()
	water := NewLayer(1, WithEffect(render.NewEffect("fn effect(vUV: vec2f) -> vec4f { return color(vUV); }")))
	water.Add(NewSprite(atlas(8, 8)))
	win.Add(water)
	w.AddEntities(water)
	w.RunFrames(1)
	assert.Equal(t, 800, water.pass.Width, "effects may read the whole window")
}

func TestAMaskWithNothingInItStillHidesWhatItMasks(t *testing.T) {
	w, win, c := app()
	lantern := NewLayer(3, AsMask())
	lit := NewLayer(2, MaskedBy(lantern))
	lit.Add(NewSprite(atlas(8, 8)))
	win.AppendChild(lit)
	win.AppendChild(lantern)
	w.AddEntities(lit, lantern)
	w.RunFrames(1)
	assert.Len(t, draw.PassesFor(w, win), 2, "the empty mask is still drawn, transparent")
	require.Len(t, c.Batches, 1)
	assert.Equal(t, lantern.pass.Target, c.Batches[0].Mask)
}

func TestInnerPassesAreDrawnBeforeTheScaledWorld(t *testing.T) {
	w, win, _ := app()
	cam := NewCamera(Resolution(320, 180))
	glow := NewLayer(1, Blend(Add))
	glow.Add(NewSprite(atlas(8, 8)))
	win.AppendChild(cam)
	win.AppendChild(glow)
	w.AddEntities(cam, glow)
	w.RunFrames(1)
	passes := draw.PassesFor(w, win)
	require.Len(t, passes, 2)
	assert.Equal(t, glow.pass, passes[0], "the layer's picture first")
	assert.Equal(t, cam.pass, passes[1], "then the world it shows in")
	assert.Equal(t, float32(glow.pass.Width), glow.pass.Screen.X, "at the camera's resolution")
	require.Len(t, cam.pass.List.Batches, 1)
	assert.Equal(t, glow.pass.Target, cam.pass.List.Batches[0].Image())
	assert.Equal(t, draw.Add, cam.pass.List.Batches[0].Blend)
}

func TestALayerWithEffectsShowsAsOnePictureThroughThem(t *testing.T) {
	w, win, c := app()
	ripple := render.NewEffect("fn effect(vUV: vec2f) -> vec4f { return color(vUV); }")
	tint := render.NewEffect("fn effect(vUV: vec2f) -> vec4f { return color(vUV) * 0.5; }")
	water := NewLayer(1, WithEffect(ripple), WithEffect(tint), WithEffect(nil))
	water.Add(NewSprite(atlas(8, 8)))
	win.AppendChild(water)
	w.AddEntities(water)
	w.RunFrames(1)

	passes := draw.PassesFor(w, win)
	require.Len(t, passes, 1)
	assert.Equal(t, []*render.Effect{ripple, tint}, passes[0].Effects, "in order")
	require.Len(t, c.Batches, 1)
	assert.Equal(t, passes[0].Target, c.Batches[0].Image())
}

func TestAMultiplyLayerKeepsItsPlaceUnderAResolutionCameraAndAResize(t *testing.T) {
	w, win, _ := app()
	cam := NewCamera(Resolution(320, 180))
	ground := NewLayer(0).Add(NewSprite(atlas(8, 8)))
	dark := NewLayer(1, Blend(Multiply)).Add(NewSprite(atlas(8, 8)))
	top := NewLayer(2).Add(NewSprite(atlas(4, 4)))
	win.Add(cam, ground, dark, top)
	w.AddEntities(ground, dark, top)
	check := func(when string) {
		t.Helper()
		bs := cam.pass.List.Batches
		require.Len(t, bs, 3, when)
		assert.Equal(t, draw.Alpha, bs[0].Blend, "%s: the ground first", when)
		assert.Equal(t, dark.pass.Target, bs[1].Image(), "%s: then the darkness, as one picture", when)
		assert.Equal(t, draw.Multiply, bs[1].Blend, when)
		assert.Equal(t, draw.Alpha, bs[2].Blend, "%s: and what follows it on top", when)
		assert.Equal(t, float32(dark.pass.Width), dark.pass.Screen.X, "%s: at the camera's resolution", when)
		shown := shownAt(cam.pass.List.Unindexed()[bs[1].First : bs[1].First+bs[1].Count])
		assert.Equal(t, dark.pass.Screen, geom.V2(shown.W, shown.H), "%s: shown as large as its pass", when)
		assert.True(
			t,
			contains(shown, geom.Rect{X: 156, Y: 86, W: 8, H: 8}),
			"%s: over its sprite, at the scaled world's centre: %v",
			when,
			shown,
		)
	}
	w.RunFrames(1)
	check("at first")
	win.Sync(geom.V2(1280, 720), geom.V2(1280, 720), true)
	w.RunFrames(1)
	check("resized")
}

func TestAMultiplyLayerFollowsTheWindowWithoutAResolution(t *testing.T) {
	w, win, c := app()
	dark := NewLayer(1, Blend(Multiply)).Add(NewSprite(atlas(8, 8)))
	win.Add(NewCamera(), dark)
	w.AddEntities(dark)
	w.RunFrames(1)
	win.Sync(geom.V2(1000, 700), geom.V2(1000, 700), true)
	w.RunFrames(1)
	require.Len(t, c.Batches, 1)
	assert.Equal(t, float32(dark.pass.Width), dark.pass.Screen.X, "its picture at the window's resolution")
	assert.True(
		t,
		contains(shownAt(c.Unindexed()), geom.Rect{X: 496, Y: 346, W: 8, H: 8}),
		"over its sprite at the new window's centre",
	)
}
