package twod //nolint:testpackage // white-box: shares twod's test helpers, reaches unexported state

import (
	"math"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// wholeMap is a map holding a tile layer, an object layer and another tile
// layer, interleaved, each with its own settings, for the tests of what a
// Tiled says of itself.
const wholeMap = `{"orientation": "orthogonal", "class": "level", "width": 3, "height": 2,
 "tilewidth": 16, "tileheight": 16, "backgroundcolor": "#80ff0000",
 "parallaxoriginx": 12, "parallaxoriginy": -4,
 "tilesets": [{"firstgid": 1, "image": "tiles.png", "tilewidth": 16, "tileheight": 16,
  "objectalignment": "%s"}],
 "layers": [
  {"id": 4, "type": "tilelayer", "name": "ground", "class": "floor", "width": 3, "height": 2,
   "data": [1, 1, 1, 1, 1, 1]},
  {"id": 7, "type": "objectgroup", "name": "props", "draworder": "index", "visible": false, "opacity": 0.5,
   "offsetx": 3, "offsety": -2, "tintcolor": "#ff00ff", "parallaxx": 0.5, "parallaxy": 2,
   "properties": [{"name": "kind", "type": "string", "value": "decor"}],
   "objects": [
    {"id": 12, "name": "crate", "gid": 2, "x": 16, "y": 32, "width": 32, "height": 16, "rotation": 90,
     "visible": false},
    {"id": 15, "name": "door", "x": 0, "y": 0, "width": 16, "height": 16,
     "properties": [{"name": "leads", "type": "object", "value": 12}]}]},
  {"id": 9, "type": "tilelayer", "name": "walls", "width": 3, "height": 2, "data": [0, 2, 0, 0, 0, 0]}
 ]}`

func wholeFS(t *testing.T, alignment string) fstest.MapFS {
	return fstest.MapFS{
		"one.tmj":   {Data: []byte(strings.Replace(wholeMap, "%s", alignment, 1))},
		"tiles.png": {Data: pngOf(t, 32, 16)},
	}
}

func TestTiledSaysWhatTheMapSaysOfItself(t *testing.T) {
	m, err := LoadTiled(wholeFS(t, ""), "one.tmj", CellSize(32, 32))
	require.NoError(t, err)
	assert.Equal(t, "level", m.Class())
	assert.Equal(t, 3, m.Columns())
	assert.Equal(t, 2, m.Rows())
	w, h := m.TilePixels()
	assert.Equal(t, [2]int{16, 16}, [2]int{w, h})
	assert.InDelta(t, 2, m.Scale(), 1e-6, "32-point cells of 16 pixels: two points a pixel")
	want, err := color.ParseHex("ff000080")
	require.NoError(t, err)
	assert.Equal(t, want, m.Background(), "Tiled's #AARRGGBB")
	assert.Equal(t, geom.V2(12, -4), m.ParallaxOrigin())
	plain, err := LoadTiled(wholeFS(t, ""), "one.tmj")
	require.NoError(t, err)
	assert.InDelta(t, 1, plain.Scale(), 1e-6, "without CellSize a pixel is a point")
}

func TestTiledKeepsItsLayersInTheFilesOrderWithTheirSettings(t *testing.T) {
	m, err := LoadTiled(wholeFS(t, ""), "one.tmj")
	require.NoError(t, err)
	var names []string
	var kinds []TiledLayerKind
	for _, l := range m.Layers() {
		names, kinds = append(names, l.Name), append(kinds, l.Kind)
	}
	assert.Equal(t, []string{"ground", "props", "walls"}, names, "tile and object layers interleaved as in the file")
	assert.Equal(t, []TiledLayerKind{TiledTiles, TiledObjects, TiledTiles}, kinds)
	var walked []string
	m.Walk(func(l *TiledLayer) bool { walked = append(walked, l.Name); return true })
	assert.Equal(t, names, walked, "Walk visits them in draw order")
	m.Walk(func(l *TiledLayer) bool { walked = append(walked[:0], l.Name); return false })
	assert.Equal(t, []string{"ground"}, walked, "and stops when asked")

	ground, ok := m.Layer("ground")
	require.True(t, ok)
	assert.Equal(t, 4, ground.ID)
	assert.Equal(t, "floor", ground.Class)
	assert.True(t, ground.Visible)
	assert.InDelta(t, 1, ground.Opacity, 1e-6)
	assert.Equal(t, color.White, ground.Tint)
	assert.Equal(t, geom.V2(1, 1), ground.Parallax)
	assert.NotNil(t, ground.TileMap())
	assert.Nil(t, ground.Parent())
	assert.Empty(t, ground.Layers())

	props, _ := m.Layer("props")
	assert.Nil(t, props.TileMap())
	assert.False(t, props.Visible)
	assert.InDelta(t, 0.5, props.Opacity, 1e-6)
	assert.Equal(t, geom.V2(3, -2), props.Offset)
	assert.Equal(t, color.Hex("#ff00ff"), props.Tint)
	assert.Equal(t, geom.V2(0.5, 2), props.Parallax)
	assert.Equal(t, TiledIndex, props.DrawOrder)
	assert.Equal(t, TiledProperties{"kind": "decor"}, m.LayerProperties("props"))
	require.Len(t, props.Objects(), 2)
	assert.Equal(t, props.Objects(), m.Objects("props"))

	tiles := m.TileLayers()
	require.Len(t, tiles, 2)
	assert.Same(t, ground.TileMap(), tiles[0])
	walls, ok := m.TileLayer("walls")
	assert.True(t, ok)
	assert.Same(t, tiles[1], walls)
	_, ok = m.TileLayer("props")
	assert.False(t, ok, "an object layer has no map")
}

func TestTiledObjectsKeepTheirIDsVisibilityAndRotation(t *testing.T) {
	m, err := LoadTiled(wholeFS(t, ""), "one.tmj")
	require.NoError(t, err)
	crate, ok := m.Object(12)
	require.True(t, ok)
	assert.Equal(t, "crate", crate.Name)
	assert.Equal(t, 12, crate.ID)
	assert.False(t, crate.Visible)
	assert.InDelta(t, math.Pi/2, crate.Rotation, 1e-6, "90 degrees, in radians")
	door, ok := m.Object(15)
	require.True(t, ok)
	assert.True(t, door.Visible, "visible unless the file says otherwise")
	leads, ok := door.Properties.Int("leads")
	assert.True(t, ok, "a property pointing to an object is the object's id")
	assert.Equal(t, 12, leads)
	_, ok = m.Object(99)
	assert.False(t, ok)
}

func TestTiledSpriteStandsATileObjectAsTiledDrawsIt(t *testing.T) {
	for _, c := range []struct {
		alignment string
		pivot     geom.Vec2
	}{
		{"", geom.V2(0, 1)},
		{"unspecified", geom.V2(0, 1)},
		{"bottomleft", geom.V2(0, 1)},
		{"bottom", geom.V2(0.5, 1)},
		{"center", geom.V2(0.5, 0.5)},
		{"topright", geom.V2(1, 0)},
	} {
		m, err := LoadTiled(wholeFS(t, c.alignment), "one.tmj", CellSize(32, 32))
		require.NoError(t, err)
		crate, _ := m.Object(12)
		s := m.Sprite(crate)
		assert.Equal(t, c.pivot, s.Pivot(), "%q: anchored by the tileset's object alignment", c.alignment)
		assert.Equal(t, geom.V2(38, 60), s.Position(), "%q: at the object's x, y moved by its layer's offset, in points",
			c.alignment)
		assert.Equal(t, geom.V2(64, 32), s.Size(), "%q: the object's size, resized in Tiled", c.alignment)
		assert.InDelta(t, math.Pi/2, s.Rotation(), 1e-6, "%q: turned as Tiled turns it", c.alignment)
		assert.True(t, s.Hidden(), "%q: hidden as the object is", c.alignment)
		_, region := s.Picture()
		assert.Equal(t, geom.Rect{X: 16, W: 16, H: 16}, region, "%q: the tile's own picture", c.alignment)
	}
	m, err := LoadTiled(wholeFS(t, ""), "one.tmj")
	require.NoError(t, err)
	crate, _ := m.Object(12)
	crate.Tile.Flip = TileFlipX
	s := m.Sprite(crate, Pivot(0.5, 0.5))
	x, _ := s.Flip()
	assert.True(t, x, "flipped as the object is")
	assert.Equal(t, geom.V2(0.5, 0.5), s.Pivot(), "opts come last")
	door, _ := m.Object(15)
	assert.Panics(t, func() { m.Sprite(door) }, "only a tile object has a picture")
}

func TestTiledRefusesWhatItCantReadNamingIt(t *testing.T) {
	layer := func(extra string) string {
		return `{"type": "objectgroup", "name": "things", "objects": [{"id": 3, "name": "it", "x": 0, "y": 0` +
			extra + `}]}`
	}
	tileset := func(extra string) string {
		return `{"firstgid": 1, "image": "tiles.png", "tilewidth": 16, "tileheight": 16` + extra + `}`
	}
	tiles := `{"type": "tilelayer", "name": "floor", "width": 1, "height": 1, "data": [1]}`
	for _, c := range []struct {
		name, tileset, layers, want string
	}{
		{"two layers of one name", tileset(""), tiles + `,` + tiles, `both called "floor"`},
		{"a group", tileset(""), `{"type": "group", "name": "outside", "layers": []}`, "group layers are not supported"},
		{"an image layer", tileset(""), `{"type": "imagelayer", "name": "sky", "image": "sky.png"}`,
			"imagelayer layers are not supported"},
		{"an XML template", tileset(""), layer(`, "template": "crate.tx"`), "only templates saved as JSON"},
		{"a render size", tileset(`, "tilerendersize": "grid"`), tiles, "render size grid"},
		{"a fill mode", tileset(`, "fillmode": "preserve-aspect-fit"`), tiles, "fill mode"},
		{"a transparent colour", tileset(`, "transparentcolor": "#ff00ff"`), tiles, "transparent colour"},
		{"a tileset class", tileset(`, "class": "walls"`), tiles, "tileset's class"},
		{"tileset properties", tileset(`, "properties": [{"name": "x", "type": "int", "value": 1}]`), tiles,
			"tileset's own properties"},
		{"a tile class", tileset(`, "tiles": [{"id": 0, "type": "wall"}]`), tiles, "tile 0: a tile's class"},
		{"a turned shape", tileset(`, "tiles": [{"id": 0, "objectgroup": {"objects": [{"x": 0, "y": 0, "width": 4,
			"height": 4, "rotation": 45}]}}]`), tiles, "tile 0: a turned collision shape"},
	} {
		tmj := `{"orientation": "orthogonal", "width": 1, "height": 1, "tilewidth": 16, "tileheight": 16,
		 "tilesets": [` + c.tileset + `], "layers": [` + c.layers + `]}`
		_, err := LoadTiled(fstest.MapFS{"one.tmj": {Data: []byte(tmj)}, "tiles.png": {Data: pngOf(t, 16, 16)}},
			"one.tmj")
		require.Error(t, err, c.name)
		assert.Contains(t, err.Error(), "one.tmj", "%s: names the file", c.name)
		assert.Contains(t, err.Error(), c.want, c.name)
	}
}
