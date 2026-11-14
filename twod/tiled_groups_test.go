package twod //nolint:testpackage // white-box: shares twod's test helpers

import (
	"testing"
	"testing/fstest"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// groupsMap holds a ground at the top and an outside group, moved, half
// opaque and red, holding walls of a tall tile and a hidden inner group
// with markers, another layer called walls, and a sky picture.
const groupsMap = `{"orientation": "orthogonal", "width": 2, "height": 1, "tilewidth": 16, "tileheight": 16,
 "tilesets": [{"firstgid": 1, "image": "tiles.png", "tilewidth": 16, "tileheight": 16,
   "tiles": [{"id": 0, "properties": [{"name": "solid", "type": "bool", "value": true},
     {"name": "tall", "type": "bool", "value": true}]}]}],
 "layers": [
  {"id": 1, "type": "tilelayer", "name": "ground", "width": 2, "height": 1, "data": [0, 0]},
  {"id": 2, "type": "group", "name": "outside", "offsetx": 2, "offsety": 1, "opacity": 0.5, "tintcolor": "#ff0000",
   "properties": [{"name": "zone", "type": "string", "value": "yard"}], "layers": [
    {"id": 3, "type": "tilelayer", "name": "walls", "width": 2, "height": 1, "data": [1, 0], "offsetx": 1,
     "offsety": 1, "opacity": 0.5},
    {"id": 4, "type": "group", "name": "inner", "visible": false, "layers": [
     {"id": 5, "type": "objectgroup", "name": "markers", "offsetx": 1, "objects": [
      {"id": 9, "name": "spawn", "x": 10, "y": 20, "point": true}]},
     {"id": 6, "type": "objectgroup", "name": "walls", "objects": []},
     {"id": 7, "type": "imagelayer", "name": "sky", "image": "sky.png", "offsetx": 4}]}]}]}`

func loadGroups(t *testing.T, opts ...TileMapOption) *Tiled {
	t.Helper()
	m, err := LoadTiled(fstest.MapFS{
		"one.tmj":   {Data: []byte(groupsMap)},
		"tiles.png": {Data: pngOf(t, 16, 16)},
		"sky.png":   {Data: pngOf(t, 40, 10)},
	}, "one.tmj", append([]TileMapOption{CellSize(32, 32)}, opts...)...)
	require.NoError(t, err)
	return m
}

func TestAGroupIsATreeOfItsLayers(t *testing.T) {
	m := loadGroups(t)
	var top []string
	for _, l := range m.Layers() {
		top = append(top, l.Name)
	}
	assert.Equal(t, []string{"ground", "outside"}, top, "the map's top layers")
	outside, ok := m.Layer("outside")
	require.True(t, ok)
	assert.Equal(t, TiledGroup, outside.Kind)
	zone, _ := outside.Properties.String("zone")
	assert.Equal(t, "yard", zone, "a group's own properties")
	inner, ok := outside.Layer("inner")
	require.True(t, ok)
	assert.Same(t, outside, inner.Parent())
	assert.Nil(t, outside.Parent(), "at the top")
	walls, _ := outside.Layer("walls")
	innerWalls, _ := inner.Layer("walls")
	assert.Equal(t, TiledTiles, walls.Kind)
	assert.Equal(t, TiledObjects, innerWalls.Kind, "the same name in two groups is two layers")

	var order []int
	m.Walk(func(l *TiledLayer) bool { order = append(order, l.ID); return true })
	assert.Equal(t, []int{1, 2, 3, 4, 5, 6, 7}, order, "a group before what it holds, in draw order")
	assert.Equal(t, []*TileMap{m.Layers()[0].TileMap(), walls.TileMap()}, m.TileLayers(), "groups' tile layers too")
	_, top0 := m.TileLayer("walls")
	assert.False(t, top0, "TileLayer names a top layer")
}

func TestAGroupsSettingsReachWhatItHolds(t *testing.T) {
	tall := NewLayer(1)
	m := loadGroups(t, TallIn(tall))
	outside, _ := m.Layer("outside")
	walls, _ := outside.Layer("walls")
	inner, _ := outside.Layer("inner")
	markers, _ := inner.Layer("markers")

	assert.Equal(t, geom.V2(1, 1), walls.Offset, "each layer keeps its own values")
	assert.InDelta(t, 0.5, walls.Opacity, 1e-6)
	assert.True(t, walls.Shown())
	assert.InDelta(t, 0.25, walls.FoldedOpacity(), 1e-6, "opacities multiplied")
	assert.Equal(t, color.Hex("#ff0000"), walls.FoldedTint(), "tints multiplied")
	assert.Equal(t, geom.V2(3, 2), walls.FoldedOffset(), "offsets added")
	assert.False(t, markers.Shown(), "hidden when a group above is")
	assert.True(t, markers.Visible)

	want := color.Hex("#ff0000")
	want.A = 0.25
	tiles := walls.TileMap()
	assert.Equal(t, geom.V2(6, 4), tiles.CellOffset(), "its map moved by the folded offset, 2 points a pixel")
	assert.Equal(t, want, tiles.Tint(), "and tinted by the folded tint and opacity")
	assert.False(t, tiles.Hidden())
	require.Len(t, tiles.TallSprites(), 1)
	assert.Equal(t, want, tiles.TallSprites()[0].Tint(), "its tall tile too")

	spawn, ok := m.Object(9)
	require.True(t, ok, "a group's objects are the map's")
	assert.Equal(t, geom.Rect{X: 13, Y: 21}, spawn.Rect, "moved by the folded offset, in map pixels")
	assert.Equal(t, []TiledObject{spawn}, markers.Objects(), "listed though a group above is hidden")
}

func TestAnImageLayerIsAPicture(t *testing.T) {
	m := loadGroups(t)
	var sky *TiledLayer
	m.Walk(func(l *TiledLayer) bool {
		if l.Kind == TiledImage {
			sky = l
		}
		return sky == nil
	})
	require.NotNil(t, sky)
	pic := sky.Picture()
	require.NotNil(t, pic)
	assert.Equal(t, geom.V2(12, 2), pic.Position(), "its top-left corner at the folded offset, 6, 1, in points")
	assert.Equal(t, geom.V2(0, 0), pic.Pivot())
	assert.Equal(t, geom.V2(80, 20), pic.Size(), "its picture at 2 points a pixel")
	assert.True(t, pic.Hidden(), "hidden by the inner group")
	want := color.Hex("#ff0000")
	want.A = 0.5
	assert.Equal(t, want, pic.Tint(), "tinted by the outside group")

	empty, err := LoadTiled(fstest.MapFS{"one.tmj": {Data: []byte(`{"orientation": "orthogonal", "tilesets": [],
	 "layers": [{"type": "imagelayer", "name": "sky", "image": ""}]}`)}}, "one.tmj")
	require.NoError(t, err)
	assert.Nil(t, empty.Layers()[0].Picture(), "an image layer naming no picture has none")
}
