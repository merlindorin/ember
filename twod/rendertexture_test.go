package twod //nolint:testpackage // white-box: shares twod's test helpers, reaches unexported state

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/internal/draw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestACameraDrawsIntoARenderTextureSpritesShow(t *testing.T) {
	w, win, c := app()
	minimap := NewRenderTexture(160, 90, Pixelated())
	hero := NewSprite(atlas(16, 16), At(400, 300))
	overview := NewCamera(Into(minimap), Zoom(0.25), Follow(hero), Priority(9))
	hud := NewLayer(1, Fixed())
	hud.Add(NewSprite(minimap, At(700, 60)))
	win.AppendChild(hero)
	win.AppendChild(overview)
	win.AppendChild(hud)
	w.AddEntities(hero, overview, hud)
	w.RunFrames(1)

	assert.Equal(t, geom.V2(160, 90), minimap.Size())
	passes := draw.PassesFor(w, win)
	require.Len(t, passes, 1)
	p := passes[0]
	assert.Equal(t, &minimap.Texture, p.Target, "the camera draws into the texture")
	assert.Equal(t, 160, p.Width)
	require.Len(t, p.List.Unindexed(), 6, "the world, not the HUD showing it")
	lo, hi := corners(p.List.Unindexed())
	assert.Equal(t, geom.V2(78, 43), lo, "the hero in the middle, a quarter of its size")
	assert.Equal(t, geom.V2(82, 47), hi)

	require.Len(t, c.Unindexed(), 12, "the window still shows the world without that camera, and the HUD")
	lo, _ = corners(c.Unindexed()[:6])
	assert.Equal(t, geom.V2(392, 292), lo, "a camera Into a texture is not the window's")
	assert.Equal(t, &minimap.Texture, c.Batches[len(c.Batches)-1].Image(), "the HUD sprite shows the texture")
	lo, hi = corners(c.Unindexed()[6:])
	assert.Equal(t, geom.V2(620, 15), lo, "at the texture's size")
	assert.Equal(t, geom.V2(780, 105), hi)
}

func TestARenderTextureDrawsPicturedLayersThroughPassesOfItsOwn(t *testing.T) {
	w, win, _ := app()
	portrait := NewRenderTexture(64, 64)
	disc := NewLayer(3, AsMask())
	disc.Add(NewSprite(atlas(8, 8)))
	inside := NewLayer(2, MaskedBy(disc))
	inside.Add(NewSprite(atlas(8, 8)))
	win.AppendChild(disc)
	win.AppendChild(inside)
	cam := NewCamera(Into(portrait))
	win.AppendChild(cam)
	w.AddEntities(disc, inside)
	w.RunFrames(1)
	passes := draw.PassesFor(w, win)
	into := slices.IndexFunc(passes, func(p *draw.Pass) bool { return p.Target == &portrait.Texture })
	require.GreaterOrEqual(t, into, 0, "the texture is drawn")
	own, mask := cam.pictures[inside], cam.pictures[disc]
	require.NotNil(t, own, "the masked layer is a picture in the texture")
	require.NotNil(t, mask, "and so is its mask")
	assert.NotEmpty(t, mask.List.Vertices, "which is drawn: a mask layer isn't culled from a render texture")
	assert.NotSame(t, own, inside.pass, "through a pass of the camera's own, never the window's")
	assert.Less(t, slices.Index(passes, own), into, "drawn before the texture shows it")
	assert.Less(t, slices.Index(passes, mask), into)
	require.Len(t, passes[into].List.Batches, 1, "the texture shows the picture, the mask itself never")
	assert.Equal(t, mask.Target, passes[into].List.Batches[0].Mask, "through its mask")
}

func TestARenderTextureKeepsItsSizeInScenes(t *testing.T) {
	data, err := json.Marshal(NewRenderTexture(32, 16))
	require.NoError(t, err)
	var back Texture
	require.NoError(t, json.Unmarshal(data, &back))
	assert.Equal(t, geom.V2(32, 16), back.Size())
}

func TestACameraDrawsOnlyItsLayers(t *testing.T) {
	w, win, c := app()
	minimap := NewRenderTexture(100, 100)
	ground, markers, hud := NewLayer(0), NewLayer(1), NewLayer(2, Fixed())
	ground.Add(NewSprite(atlas(16, 16), At(400, 300)))
	dot := NewSprite(atlas(4, 4), At(400, 300))
	markers.Add(dot)
	hud.Add(NewSprite(minimap, At(700, 60)))
	overview := NewCamera(Into(minimap), At(400, 300), Layers(markers))
	main := NewCamera(At(400, 300))
	win.Add(ground, markers, hud, overview, main)
	w.AddEntities(ground, markers, hud, overview, main)
	w.RunFrames(1)

	into := draw.PassesFor(w, win)[0]
	require.Len(t, into.List.Unindexed(), 6, "the minimap draws its markers and nothing else")
	lo, hi := corners(into.List.Unindexed())
	assert.Equal(t, geom.V2(4, 4), hi.Sub(lo))
	assert.Len(t, c.Unindexed(), 18, "the main camera, unfiltered, draws the ground, the marker and the HUD")

	main.SetLayers(ground)
	w.RunFrames(1)
	assert.Len(t, c.Unindexed(), 12, "filtered to the ground, it leaves the marker out but keeps the fixed HUD")
	assert.Equal(t, []*Layer{ground}, main.Layers())
	nested := NewLayer(0)
	ground.Add(nested.Add(NewSprite(atlas(8, 8), At(400, 300))))
	w.AddEntities(nested)
	w.RunFrames(1)
	assert.Len(t, c.Unindexed(), 18, "a layer inside a shown one shows too")
	main.SetLayers()
	w.RunFrames(1)
	assert.Len(t, c.Unindexed(), 24, "no layers, every layer")
}
