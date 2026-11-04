package light //nolint:testpackage // white-box: shares light's test scene, reaches unexported state

import (
	"image"
	"testing"

	"github.com/merlindorin/ember/collide"
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/internal/draw"
	"github.com/merlindorin/ember/internal/wm"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// scene is a world with an 800×600 window whose camera, zoomed twice, looks
// at the world's origin, and the lighting.
func scene(t *testing.T, opts ...twod.Option) (*world.World, *platform.Window, *twod.Camera) {
	t.Helper()
	w := world.New().AddPlugins(wm.Plugin{}, twod.Plugin{}, Plugin{})
	win := platform.NewWindow(platform.Size(800, 600))
	cam := twod.NewCamera(twod.Zoom(2))
	for _, o := range opts {
		o(cam)
	}
	win.Add(cam)
	w.AddEntities(win)
	return w, win, cam
}

func picture(w, h int) *twod.Texture { return twod.NewTexture(image.NewRGBA(image.Rect(0, 0, w, h))) }

type walls []geom.Rect

func (ws walls) SolidIn(_ geom.Rect) []geom.Rect { return ws }

func TestALitLayersSpritesReadTheLightWhereTheyDraw(t *testing.T) {
	w, win, _ := scene(t)
	plain := twod.NewSprite(picture(10, 10), twod.At(-50, 0))
	off := twod.NewSprite(picture(10, 10), twod.At(0, 0), Unlit())
	wall := twod.NewSprite(picture(10, 40), twod.At(50, 0), LitAt(0, 20))
	layer := twod.NewLayer(0, Lit()).Add(plain, off, wall)
	win.Add(layer)
	w.AddEntities(layer)
	w.RunFrames(2)
	vs := draw.FrontFor(w, win).Vertices
	require.Len(t, vs, 12)
	toLight := draw.ToLight(win.Size())
	for _, v := range vs[:4] {
		assert.Equal(t, 1, v.Lit(), "per pixel")
		assert.InDelta(t, toLight.Xform(v.Pos).X, v.LightUV.X, 1e-5)
		assert.InDelta(t, toLight.Xform(v.Pos).Y, v.LightUV.Y, 1e-5)
	}
	for _, v := range vs[4:8] {
		assert.Equal(t, 0, v.Lit(), "Unlit draws as it is")
	}
	base := toLight.Xform(geom.V2(400+100, 300+40))
	for _, v := range vs[8:] {
		assert.Equal(t, 1, v.Lit())
		assert.InDelta(
			t,
			base.X,
			v.LightUV.X,
			1e-5,
			"the wall takes its light at its base, 20 below its centre, zoomed twice",
		)
		assert.InDelta(t, base.Y, v.LightUV.Y, 1e-5)
	}
}

func TestLightsAreListedWhereTheyShowWithTheirRadiusZoomed(t *testing.T) {
	w, win, _ := scene(t, Ambient(color.RGB(0.1, 0.1, 0.2)))
	hero := twod.NewSprite(picture(8, 8), twod.At(30, 10))
	torch := NewPoint(100, color.RGB(1, 0.8, 0.5), Intensity(0.5), At(0, -4))
	hero.Add(torch)
	far := NewPoint(50, color.White, At(2000, 0))
	hidden := NewPoint(50, color.White)
	hidden.SetHidden(true)
	layer := twod.NewLayer(0, Lit()).Add(hero, far, hidden)
	win.Add(layer)
	w.AddEntities(layer)
	w.RunFrames(2)
	l := draw.LightingFor(w, win)
	require.NotNil(t, l)
	assert.Equal(t, color.RGB(0.1, 0.1, 0.2), l.Ambient)
	require.Len(t, l.Lights, 1, "the far light and the hidden one are left out")
	got := l.Lights[0]
	assert.InDelta(t, 400+30*2, got.At.X, 1e-3)
	assert.InDelta(t, 300+6*2, got.At.Y, 1e-3)
	assert.InDelta(t, 200, got.Radius, 1e-3)
	assert.Equal(t, color.RGB(0.5, 0.4, 0.25), got.Color)
}

func TestSolidsAndOccluderSpritesBlockLightInTheLightPicturesSpace(t *testing.T) {
	space := collide.NewSpace()
	space.AddSolid(walls{{X: 0, Y: 0, W: 10, H: 10}})
	w, win, _ := scene(t, OccludersFrom(space))
	pillar := twod.NewSprite(picture(4, 4), twod.At(-20, 0), Occluder())
	layer := twod.NewLayer(0).Add(pillar, NewPoint(80, color.White))
	win.Add(layer)
	w.AddEntities(layer)
	w.RunFrames(2)
	l := draw.LightingFor(w, win)
	require.NotNil(t, l)
	vs := l.Occluders.Vertices
	require.Len(t, vs, 8, "the wall and the pillar, a quad each")
	m, g := float32(draw.LightMargin), float32(occluderGrow)
	assert.Equal(
		t,
		geom.V2(400+m-g, 300+m-g),
		vs[0].Pos,
		"the wall's corner, at the origin, in the light picture's points, half a texel out",
	)
	assert.Equal(t, geom.V2(400+20+m+g, 300+20+m+g), vs[2].Pos, "zoomed twice")
	assert.Equal(t, geom.V2(400-44+m-g, 300-4+m-g), vs[4].Pos, "the pillar's top-left corner")
	assert.Equal(t, draw.Occlude, l.Occluders.Batches[0].Blend, "groups and heights add up, never cover each other")
}

func TestAWindowWithoutLightOrAmbientIsNotLit(t *testing.T) {
	w, win, _ := scene(t)
	layer := twod.NewLayer(0, Lit()).Add(twod.NewSprite(picture(4, 4)))
	win.Add(layer)
	w.AddEntities(layer)
	w.RunFrames(2)
	assert.Nil(t, draw.LightingFor(w, win), "its lit layers show as they are")
}

// tallVertices are the vertices of a map's one tall tile, a 16×32 tile on
// a 32-point cell at the origin, drawn on a layer made with layerOpts, the
// map made with mapOpts.
func tallVertices(
	t *testing.T,
	layerOpts []twod.LayerOption,
	mapOpts ...twod.TileMapOption,
) ([]draw.Vertex, *platform.Window) {
	t.Helper()
	w, win, _ := scene(t)
	tiles := twod.NewTileset(picture(16, 32), twod.Cell(16, 32), twod.Tall(1))
	m := twod.NewTileMap(
		tiles,
		append([]twod.TileMapOption{twod.Cells([][]int{{1}}), twod.CellSize(32, 32)}, mapOpts...)...)
	layer := twod.NewLayer(0, layerOpts...).Add(m)
	win.Add(layer)
	w.AddEntities(layer)
	w.RunFrames(2)
	return draw.FrontFor(w, win).Vertices, win
}

func TestATallTileOnALitLayerTakesItsLightJustInFrontOfItsFoot(t *testing.T) {
	vs, win := tallVertices(t, []twod.LayerOption{Lit()})
	require.Len(t, vs, 4)
	foot := geom.V2(400+16*2, 300+32*2)
	want := draw.ToLight(win.Size()).Xform(foot.Add(geom.V2(0, 3*2)))
	for _, v := range vs {
		assert.Equal(t, 1, v.Lit())
		assert.InDelta(t, want.X, v.LightUV.X, 1e-5, "its cell's middle, zoomed twice")
		assert.InDelta(t, want.Y, v.LightUV.Y, 1e-5, "three points below its foot, zoomed twice")
	}
	vs, _ = tallVertices(t, nil)
	for _, v := range vs {
		assert.Equal(t, 0, v.Lit(), "on a layer that isn't lit, it isn't either")
	}
	vs, _ = tallVertices(t, []twod.LayerOption{Lit()}, PerPixel())
	require.Len(t, vs, 4)
	assert.NotEqual(t, vs[0].LightUV, vs[2].LightUV, "PerPixel on the map lights it at each pixel")
}

func TestHardBodiesBlockLightButTriggersAndSoftBodiesDont(t *testing.T) {
	space := collide.NewSpace()
	w, win, _ := scene(t, OccludersFrom(space))
	pillar := twod.NewSprite(picture(4, 4), twod.At(-20, 0))
	pillar.Add(collide.NewBody(collide.Box(10, 10)))
	plate := twod.NewSprite(picture(4, 4), twod.At(20, 0))
	plate.Add(collide.NewBody(collide.Box(10, 10), collide.Trigger()))
	walker := twod.NewSprite(picture(4, 4), twod.At(40, 0))
	walker.Add(collide.NewBody(collide.Box(10, 10), collide.Soft(1)))
	layer := twod.NewLayer(0).Add(pillar, plate, walker, NewPoint(80, color.White))
	win.Add(layer)
	w.AddSystems(collide.NewTracker(space))
	w.AddEntities(layer)
	w.RunFrames(3)
	l := draw.LightingFor(w, win)
	require.NotNil(t, l)
	vs := l.Occluders.Vertices
	require.Len(t, vs, 4, "only the pillar's hard body")
	m := float32(draw.LightMargin)
	g := float32(occluderGrow)
	assert.Equal(t, geom.V2(400+(-25)*2+m-g, 300-5*2+m-g), vs[0].Pos, "its box round its owner, zoomed twice")
}

func TestOccluderSpritesOfGroupsAreDrawnInTheirChannels(t *testing.T) {
	w, win, _ := scene(t)
	layer := twod.NewLayer(0).Add(
		twod.NewSprite(picture(4, 4), Occluder()),
		twod.NewSprite(picture(4, 4), Occluder(1, 2)),
		NewPoint(80, color.White, BlockedBy(2)),
	)
	win.Add(layer)
	w.AddEntities(layer)
	w.RunFrames(2)
	l := draw.LightingFor(w, win)
	require.NotNil(t, l)
	vs := l.Occluders.Vertices
	require.Len(t, vs, 8)
	assert.Equal(t, color.Color{R: 1, A: 1}, vs[0].Color, "group 0 by default, in red")
	assert.Equal(t, color.Color{G: 1, B: 1, A: 1}, vs[4].Color, "groups 1 and 2")
	assert.Equal(t, uint8(1<<2), l.Lights[0].Groups)
}

func TestOccludersOutOfReachOfTheViewAreLeftOut(t *testing.T) {
	w, win, _ := scene(t)
	layer := twod.NewLayer(0).
		Add(twod.NewSprite(picture(4, 4), twod.At(5000, 0), Occluder()), NewPoint(80, color.White))
	win.Add(layer)
	w.AddEntities(layer)
	w.RunFrames(2)
	l := draw.LightingFor(w, win)
	require.NotNil(t, l)
	assert.Empty(t, l.Occluders.Vertices)
}

func TestTheAmbientCanChangeWhileTheGameRuns(t *testing.T) {
	w, win, cam := scene(t)
	assert.Equal(t, color.Black, AmbientOf(cam), "black until set")
	SetAmbient(cam, color.RGB(0.5, 0.4, 0.3))
	assert.Equal(t, color.RGB(0.5, 0.4, 0.3), AmbientOf(cam))
	w.RunFrames(2)
	l := draw.LightingFor(w, win)
	require.NotNil(t, l, "an ambient set at run time lights the window")
	assert.Equal(t, color.RGB(0.5, 0.4, 0.3), l.Ambient)
}

func TestProbesInViewAreAskedAndThoseOutOfItAreStale(t *testing.T) {
	w, win, _ := scene(t, Ambient(color.Black))
	near, far := NewProbe(twod.At(10, 5)), NewProbe(twod.At(5000, 0))
	layer := twod.NewLayer(0).Add(near, far)
	win.Add(layer)
	w.AddEntities(layer)
	w.RunFrames(2)
	l := draw.LightingFor(w, win)
	require.NotNil(t, l)
	require.Len(t, l.Probes, 1, "only the probe in view is asked")
	assert.Equal(t, geom.V2(400+10*2, 300+5*2), l.Probes[0].At)
	assert.False(t, near.Stale())
	assert.True(t, far.Stale())
	assert.False(t, near.Ready(), "no answer yet: the GPU gives it")

	w2, win2, _ := scene(t)
	p := NewProbe()
	layer2 := twod.NewLayer(0).Add(p)
	win2.Add(layer2)
	w2.AddEntities(layer2)
	w2.RunFrames(2)
	assert.True(t, p.Ready())
	assert.Equal(t, color.White, p.Light(), "in a window nothing lights, a probe reads white")
}

func TestAnAmbientMapLightsAWindowWithItsTilesInView(t *testing.T) {
	painted := 0
	pic := twod.NewTiledPicture(
		func(_ twod.PictureTile) { painted++ },
		twod.TexelSize(32, 32),
		twod.TileTexels(16),
		twod.PaintAhead(0),
	)
	w, win, cam := scene(t)
	assert.Nil(t, AmbientMapOf(cam))
	SetAmbientMap(cam, pic)
	assert.Same(t, pic, AmbientMapOf(cam))
	w.RunFrames(2)
	l := draw.LightingFor(w, win)
	require.NotNil(t, l, "an ambient map alone lights the window")
	assert.NotEmpty(t, l.AmbientMap.Vertices, "its tiles in view, as quads in the light picture's points")
	assert.Positive(t, painted)
	SetAmbientMap(cam, nil)
	w.RunFrames(1)
	assert.Nil(t, draw.LightingFor(w, win))
}

func TestATileMapsShadowsAreItsOccludersInTheirGroups(t *testing.T) {
	tiles := twod.NewTileset(picture(16, 16), twod.Cell(16, 16), twod.ShadowShapes(1, geom.Rect{W: 16, H: 16}),
		twod.Solid(2), twod.CastShadow(false, 2), twod.Solid(3), twod.OccluderGroup(1, 3))
	m := twod.NewTileMap(tiles, twod.Cells([][]int{{1, 2, 3}}), twod.CellSize(16, 16))
	space := collide.NewSpace()
	space.AddSolid(m)
	w, win, _ := scene(t, OccludersFrom(space))
	layer := twod.NewLayer(0).Add(m, NewPoint(80, color.White))
	win.Add(layer)
	w.AddEntities(layer)
	w.RunFrames(2)
	l := draw.LightingFor(w, win)
	require.NotNil(t, l)
	vs := l.Occluders.Vertices
	require.Len(t, vs, 8, "tile 1's shadow shape and tile 3, not tile 2")
	assert.Equal(t, color.Color{R: 1, A: 1}, vs[0].Color, "group 0")
	assert.Equal(
		t,
		color.Color{G: 1, A: 32.0 / draw.MaxOccluderHeight},
		vs[4].Color,
		"tile 3 in group 1, a solid tile standing its cell high, 16, zoomed twice",
	)
}

func TestATallTileThatBlocksLightTakesItRoundWhatItBlocksWith(t *testing.T) {
	w, win, _ := scene(t)
	tiles := twod.NewTileset(picture(16, 32), twod.Cell(16, 32), twod.Tall(1), twod.Solid(1))
	m := twod.NewTileMap(tiles, twod.Cells([][]int{{1}}), twod.CellSize(32, 32))
	layer := twod.NewLayer(0, Lit()).Add(m)
	win.Add(layer)
	w.AddEntities(layer)
	w.RunFrames(2)
	vs := draw.FrontFor(w, win).Vertices
	require.Len(t, vs, 4)
	toLight := draw.ToLight(win.Size())
	mid := toLight.Xform(geom.V2(400+16*2, 300+16*2))
	half := toLight.Xform(geom.V2(400+(16+16+3)*2, 300+(16+16+3)*2)).Sub(mid)
	for _, v := range vs {
		assert.Equal(t, 2, v.Lit(), "round its solid cell")
		assert.InDelta(t, mid.X, v.LightUV.X, 1e-5, "the cell's middle, zoomed twice")
		assert.InDelta(t, mid.Y, v.LightUV.Y, 1e-5)
		assert.InDelta(t, half.X, v.LightSpread.X, 1e-5, "half the cell and three points, zoomed twice")
		assert.InDelta(t, half.Y, v.LightSpread.Y, 1e-5)
	}
}

func TestALightsElevationSizeAndFadeReachItsWindowZoomed(t *testing.T) {
	w, win, _ := scene(t)
	lamp := NewPoint(100, color.White, Elevation(30), Size(4), ShadowFade(FadeSmooth), FadeFrom(0.25))
	pillar := twod.NewSprite(picture(4, 4), twod.At(20, 0), Occluder(), Height(12))
	layer := twod.NewLayer(0).Add(lamp, pillar)
	win.Add(layer)
	w.AddEntities(layer)
	w.RunFrames(2)
	l := draw.LightingFor(w, win)
	require.NotNil(t, l)
	require.Len(t, l.Lights, 1)
	got := l.Lights[0]
	assert.Equal(t, float32(60), got.Elevation, "zoomed twice")
	assert.Equal(t, float32(8), got.ShadowSize)
	assert.Equal(t, int(FadeSmooth), got.ShadowFade)
	assert.Equal(t, float32(0.25), got.FadeFrom)
	assert.Equal(t, FadeLinear, NewPoint(1, color.White, ShadowFade()).ShadowFade(), "no curve given: linear")
	vs := l.Occluders.Vertices
	require.Len(t, vs, 4)
	assert.InDelta(
		t,
		24.0/draw.MaxOccluderHeight,
		vs[0].Color.A,
		1e-6,
		"the pillar's height, zoomed, in the occluder picture's alpha",
	)
	assert.Equal(t, float32(1), draw.OccluderColor(1, 0).A, "no height given: it blocks all the way")
}

func TestF6OutlinesEachOccluderInItsGroupsColour(t *testing.T) {
	w, win, _ := scene(t)
	layer := twod.NewLayer(0).Add(
		twod.NewSprite(picture(4, 4), Occluder()),
		twod.NewSprite(picture(4, 4), twod.At(20, 0), Occluder(1)),
		NewPoint(80, color.White),
	)
	win.Add(layer)
	w.AddEntities(layer)
	w.RunFrames(2)
	assert.Empty(t, draw.LightingFor(w, win).Debug.Vertices)
	SetOutlines(win, true)
	w.RunFrames(1)
	n := map[color.Color]int{}
	for _, v := range draw.LightingFor(w, win).Debug.Vertices {
		n[v.Color]++
	}
	red, green := color.Color{R: 1, A: 1}, color.Color{G: 1, A: 1}
	assert.Positive(t, n[red], "group 0's square outlined in red, over the debug view")
	assert.Equal(t, n[red], n[green], "group 1's in green, four edges each")
}

func TestF6OutlinesAreLightenedOverTheOccluderView(t *testing.T) {
	w, win, _ := scene(t)
	layer := twod.NewLayer(0).Add(twod.NewSprite(picture(4, 4), Occluder()), NewPoint(80, color.White))
	win.Add(layer)
	w.AddEntities(layer)
	SetDebug(win, DebugOccluders)
	SetOutlines(win, true)
	w.RunFrames(2)
	pink := 0
	for _, v := range draw.LightingFor(w, win).Debug.Vertices {
		if v.Color == color.RGB(1, 0.6, 0.6) {
			pink++
		}
		assert.NotEqual(t, color.Color{R: 1, A: 1}, v.Color, "no pure red over the red fills")
	}
	assert.Positive(t, pink, "group 0 outlined in a light red")
}

func TestAWindowsDebugViewIsSetAndReadByItsOwn(t *testing.T) {
	w, win, _ := scene(t)
	other := platform.NewWindow(platform.Size(100, 100))
	w.AddEntities(other)
	SetDebug(win, DebugShadows)
	SetOutlines(win, true)
	SetDebugArea(win, geom.Rect{X: 10, Y: 10, W: 100, H: 50})
	assert.Equal(t, DebugShadows, DebugOf(win))
	assert.True(t, OutlinesOf(win))
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 100, H: 50}, DebugAreaOf(win))
	assert.Equal(t, DebugOff, DebugOf(other), "each window its own")
	assert.False(t, OutlinesOf(other))
	assert.Equal(t, "shadow maps", DebugShadows.String())

	alone := platform.NewWindow()
	SetDebug(alone, DebugLight)
	assert.Equal(t, DebugOff, DebugOf(alone), "a window in no World keeps none")
}

func TestAnOccluderSpriteOnALitLayerTakesItsLightPastItsSidesUnlessPerPixel(t *testing.T) {
	w, win, _ := scene(t)
	crate := twod.NewSprite(picture(10, 10), twod.At(-50, 0), Occluder())
	perPixel := twod.NewSprite(picture(10, 10), twod.At(50, 0), Occluder(), PerPixel())
	plain := twod.NewSprite(picture(10, 10), twod.At(0, 50))
	layer := twod.NewLayer(0, Lit()).Add(crate, perPixel, plain, NewPoint(80, color.White))
	win.Add(layer)
	w.AddEntities(layer)
	w.RunFrames(2)
	vs := draw.FrontFor(w, win).Vertices
	require.Len(t, vs, 12)
	modes := []int{vs[0].Lit(), vs[4].Lit(), vs[8].Lit()}
	assert.Equal(
		t,
		[]int{3, 1, 1},
		modes,
		"the occluder round its left, right and bottom; PerPixel and the plain sprite per pixel",
	)
	box := draw.ToLight(win.Size())
	centre := box.Xform(geom.V2(400-100, 300))
	assert.InDelta(t, centre.X, vs[0].LightUV.X, 1e-4, "round the crate's centre, zoomed twice")
	assert.InDelta(t, centre.Y, vs[0].LightUV.Y, 1e-4)
}

func TestUnlitOnATileMapLeavesItsTallTilesUnlitWhereverTheyAreSorted(t *testing.T) {
	vs, _ := tallVertices(t, []twod.LayerOption{Lit()}, Unlit())
	require.Len(t, vs, 4)
	assert.Equal(t, 0, vs[0].Lit(), "in the map's own layer")

	w, win, _ := scene(t)
	tiles := twod.NewTileset(picture(16, 32), twod.Cell(16, 32), twod.Tall(1), twod.Solid(1))
	sorted := twod.NewLayer(1, Lit(), twod.SortByY())
	m := twod.NewTileMap(tiles, twod.Cells([][]int{{1}}), twod.CellSize(32, 32), twod.TallIn(sorted), Unlit())
	ground := twod.NewLayer(0, Lit()).Add(m)
	win.Add(ground, sorted)
	w.AddEntities(ground, sorted)
	w.RunFrames(2)
	vs = draw.FrontFor(w, win).Vertices
	require.NotEmpty(t, vs)
	for _, v := range vs {
		assert.Equal(t, 0, v.Lit(), "sorted into a lit layer with the characters, still unlit")
	}
}

func TestAStaticLightLeavesTheLiveListForTilesWhoseVersionsFollowWhatChanges(t *testing.T) {
	space := collide.NewSpace()
	tiles := twod.NewTileset(picture(16, 16), twod.Cell(16, 16), twod.Solid(1))
	m := twod.NewTileMap(tiles, twod.Cells([][]int{{1, 0}}), twod.CellSize(16, 16), twod.At(40, 0))
	space.AddSolid(m)
	w, win, _ := scene(t, OccludersFrom(space))
	torch := NewPoint(100, color.White, At(0, 0), Static())
	live := NewPoint(50, color.White, At(-100, 0))
	layer := twod.NewLayer(0).Add(torch, live, m)
	win.Add(layer)
	w.AddEntities(layer)
	w.RunFrames(2)
	l := draw.LightingFor(w, win)
	require.Len(t, l.Lights, 1, "only the live light is drawn a frame")
	require.NotNil(t, l.Static)
	assert.Equal(t, 1, l.Static.Count)
	versions := func() map[[2]int]uint64 {
		v := map[[2]int]uint64{}
		for _, tile := range draw.LightingFor(w, win).Static.Tiles {
			v[[2]int{tile.X, tile.Y}] = tile.Version
		}
		return v
	}
	before := versions()
	assert.Len(t, before, 4, "the four tiles round the origin its 100 reach")
	b := l.Static.Tiles[0].Bake()
	require.Len(t, b.Lighting.Lights, 1)
	assert.Equal(
		t,
		geom.V2(b.Inner.X, b.Inner.Y).
			Add(geom.V2(-float32(l.Static.Tiles[0].X)*256, -float32(l.Static.Tiles[0].Y)*256)),
		b.Lighting.Lights[0].At,
		"the light in the frame's points: the tile's corner is Inner's",
	)
	assert.NotEmpty(t, b.Lighting.Occluders.Vertices, "the map's solid is in the frame")

	w.RunFrames(1)
	assert.Equal(t, before, versions(), "nothing changed, nothing to draw again")
	torch.SetPosition(geom.V2(200, 0))
	w.RunFrames(1)
	after := versions()
	for k, v := range before {
		if after[k] == v {
			t.Fatalf("tile %v, which the torch lit, is drawn again", k)
		}
	}
	m.Set(1, 0, 1)
	w.RunFrames(1)
	for k, v := range after {
		assert.NotEqual(t, v, versions()[k], "a map that changed draws every tile again")
	}
	torch.SetHidden(true)
	w.RunFrames(1)
	assert.Nil(t, draw.LightingFor(w, win).Static.Tiles, "a hidden static light reaches no tile")
}

func TestALayerShadedBySomeGroupsMarksWhatItDrawsAndItsChildrenFollow(t *testing.T) {
	w, win, _ := scene(t)
	monster := twod.NewSprite(picture(10, 10), twod.At(-50, 0))
	own := twod.NewSprite(picture(10, 10), twod.At(0, 0), ShadedBy(0, 1, 2))
	crowd := twod.NewLayer(1, Lit(), ShadedBy(0, 1)).Add(monster, own)
	hero := twod.NewSprite(picture(10, 10), twod.At(50, 0))
	world := twod.NewLayer(0, Lit()).Add(hero)
	win.Add(world, crowd)
	w.AddEntities(world, crowd)
	w.RunFrames(2)

	groups, set := ShadedByOf(crowd)
	assert.Equal(t, []int{0, 1}, groups)
	assert.True(t, set)
	_, set = ShadedByOf(monster)
	assert.False(t, set, "the monster follows its layer")

	front := draw.FrontFor(w, win)
	vs := front.Vertices
	require.Len(t, vs, 12)
	assert.Equal(t, uint8(0b111), vs[0].Shade(), "the hero, by every group")
	assert.Equal(t, uint8(0b011), vs[4].Shade(), "the monster, as its layer")
	assert.Equal(t, uint8(0b111), vs[8].Shade(), "one set back to every group")
	for _, v := range vs {
		assert.Equal(t, 1, v.Lit())
	}
	assert.Equal(t, draw.KeysOf(0, 0b100), front.Shades, "the hero's light picture and the crowd's")

	SetShadedBy(crowd)
	w.RunFrames(1)
	assert.Equal(t, uint8(0), draw.FrontFor(w, win).Vertices[4].Shade(), "shaded by none")
}
