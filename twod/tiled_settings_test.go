package twod //nolint:testpackage // white-box: shares twod's test helpers, reaches unexported state

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// wallsMap is a map of one tile layer, a solid wall's tile standing tall
// at its first cell, the layer's settings and the tileset's drawing offset
// set by the test.
const wallsMap = `{"orientation": "orthogonal", "width": 2, "height": 1, "tilewidth": 16, "tileheight": 16,
 "tilesets": [{"firstgid": 1, "image": "tiles.png", "tilewidth": 16, "tileheight": 16, "tileoffset": %OFFSET%,
   "tiles": [{"id": 0, "properties": [{"name": "solid", "type": "bool", "value": true},
     {"name": "tall", "type": "bool", "value": true}]}]}],
 "layers": [{"id": 1, "type": "tilelayer", "name": "walls", "width": 2, "height": 1, "data": [1, 0] %LAYER%}]}`

func wallsFS(t *testing.T, layer, offset string) fstest.MapFS {
	tmj := strings.NewReplacer("%LAYER%", layer, "%OFFSET%", offset).Replace(wallsMap)
	return fstest.MapFS{"one.tmj": {Data: []byte(tmj)}, "tiles.png": {Data: pngOf(t, 16, 16)}}
}

func TestAHiddenTintedMovedLayerTakesItsTallTilesAlong(t *testing.T) {
	still := NewLayer(1)
	plain, err := LoadTiled(wallsFS(t, "", `{"x": 0, "y": 0}`), "one.tmj", CellSize(32, 32), TallIn(still))
	require.NoError(t, err)
	moved := NewLayer(1)
	m, err := LoadTiled(wallsFS(t, `, "visible": false, "opacity": 0.5, "tintcolor": "#ff8000", "offsetx": 4,
	 "offsety": -3`, `{"x": 0, "y": 0}`), "one.tmj", CellSize(32, 32), TallIn(moved))
	require.NoError(t, err)
	walls, _ := m.TileLayer("walls")
	want := color.Hex("#ff8000")
	want.A = 0.5
	assert.True(t, walls.Hidden(), "not drawn")
	assert.Equal(t, want, walls.Tint(), "tinted by its tint and opacity")
	assert.Equal(t, geom.V2(8, -6), walls.CellOffset(), "its cells moved by its offset, 2 points a pixel")
	assert.Equal(t, geom.Vec2{}, walls.Position(), "the map itself left for the game to place")
	talls, before := walls.TallSprites(), func() []*Sprite { w, _ := plain.TileLayer("walls"); return w.TallSprites() }()
	require.Len(t, talls, 1)
	require.Len(t, before, 1)
	assert.True(t, talls[0].Hidden(), "its tall tile, standing in another layer, hidden too")
	assert.Equal(t, want, talls[0].Tint(), "and tinted")
	assert.Equal(t, before[0].Position().Add(geom.V2(8, -6)), talls[0].Position(), "and moved")
	assert.NotZero(t, walls.At(0, 0), "a hidden layer keeps its cells for collision and light to read")

	walls.SetHidden(false)
	walls.SetTint(color.White)
	assert.False(t, talls[0].Hidden(), "shown again, its tall tile with it")
	assert.Equal(t, color.White, talls[0].Tint())
}

func TestALayersOffsetSurvivesTheGamePlacingItsMap(t *testing.T) {
	load := func(layer string) *TileMap {
		m, err := LoadTiled(wallsFS(t, layer, `{"x": 0, "y": 0}`), "one.tmj", CellSize(32, 32), TallIn(NewLayer(1)))
		require.NoError(t, err)
		walls, _ := m.TileLayer("walls")
		walls.SetPosition(geom.V2(100, 200))
		return walls
	}
	plain, moved := load(""), load(`, "offsetx": 4, "offsety": -3`)
	offset := geom.V2(8, -6)
	assert.Equal(t, plain.Center(0, 0).Add(offset), moved.Center(0, 0), "its cells at the position plus the offset")
	x, y, ok := moved.Cell(geom.V2(100+8+1, 200-6+1))
	assert.True(t, ok)
	assert.Equal(t, [2]int{0, 0}, [2]int{x, y}, "and found there")
	want := plain.TallSprites()[0].Position().Add(offset)
	assert.Equal(t, want, moved.TallSprites()[0].Position(), "its tall tile too, standing in another layer")
	assert.Equal(t, geom.V2(108, 194), moved.CellOrigin(0, 0), "and its cell, where collision and light place it")

	moved.Translate(geom.V2(-10, 5))
	assert.Equal(t, want.Add(geom.V2(-10, 5)), moved.TallSprites()[0].Position(), "the tall tile following a move")
}

func TestATilesetsDrawingOffsetMovesItsPictures(t *testing.T) {
	plain, err := LoadTiled(wallsFS(t, "", `{"x": 0, "y": 0}`), "one.tmj", CellSize(32, 32))
	require.NoError(t, err)
	moved, err := LoadTiled(wallsFS(t, "", `{"x": 3, "y": -2}`), "one.tmj", CellSize(32, 32))
	require.NoError(t, err)
	a, _ := plain.TileLayer("walls")
	b, _ := moved.TileLayer("walls")
	assert.Equal(t, a.TallSprites()[0].Position().Add(geom.V2(6, -4)), b.TallSprites()[0].Position(),
		"3, -2 pixels at 2 points a pixel")
	assert.Equal(t, a.CellOrigin(0, 0), b.CellOrigin(0, 0), "the cell it stands on doesn't move")
}

func TestAnObjectLayersOffsetMovesItsObjects(t *testing.T) {
	m, err := LoadTiled(wholeFS(t, ""), "one.tmj")
	require.NoError(t, err)
	door, _ := m.Object(15)
	assert.Equal(t, geom.Rect{X: 3, Y: -2, W: 16, H: 16}, door.Rect, "the props layer's offset, 3, -2")
}

func TestTextObjectsAndPropertiesOfEveryKind(t *testing.T) {
	tmj := `{"orientation": "orthogonal", "width": 1, "height": 1, "tilewidth": 16, "tileheight": 16,
	 "properties": [{"name": "music", "type": "file", "value": "../sounds/crypt.ogg"},
	   {"name": "look", "type": "class", "propertytype": "look",
	    "value": {"scale": 2, "glow": 0.5, "rim": {"width": 3, "color": "#ff00ff00"}}}],
	 "tilesets": [],
	 "layers": [{"type": "objectgroup", "name": "signs", "objects": [
	   {"id": 1, "name": "sign", "x": 4, "y": 8, "width": 64, "height": 16,
	    "text": {"text": "Turn back", "fontfamily": "Serif", "pixelsize": 12, "color": "#ffcc0000", "bold": true,
	     "wrap": true, "kerning": false, "halign": "center"}},
	   {"id": 2, "name": "plain", "x": 0, "y": 0, "width": 8, "height": 8, "text": {"text": "hi"}}]}]}`
	m, err := LoadTiled(fstest.MapFS{"levels/one.tmj": {Data: []byte(tmj)}}, "levels/one.tmj")
	require.NoError(t, err)
	sign, _ := m.Object(1)
	assert.Equal(t, TiledText, sign.Shape)
	assert.Equal(t, "Turn back", sign.Text)
	require.NotNil(t, sign.TextStyle)
	assert.Equal(t, TiledTextStyle{
		FontFamily: "Serif", PixelSize: 12, Color: color.Hex("#cc0000ff"), Wrap: true, Bold: true,
		HAlign: "center", VAlign: "top",
	}, *sign.TextStyle)
	plain, _ := m.Object(2)
	assert.Equal(t, TiledTextStyle{PixelSize: 16, Color: color.Black, Kerning: true, HAlign: "left", VAlign: "top"},
		*plain.TextStyle, "Tiled's defaults")

	music, _ := m.Properties().String("music")
	assert.Equal(t, "sounds/crypt.ogg", music, "a file's path from the file system's root")
	look, ok := m.Properties().Class("look")
	require.True(t, ok, "a class property's members")
	scale, ok := look.Int("scale")
	assert.True(t, ok)
	assert.Equal(t, 2, scale, "a whole number reads as an int")
	glow, _ := look.Float("glow")
	assert.InDelta(t, 0.5, glow, 1e-9)
	rim, ok := look.Class("rim")
	require.True(t, ok, "a nested class")
	width, _ := rim.Int("width")
	assert.Equal(t, 3, width)
}

// templateFS is a map using tiles.tsj, a crate template from that tileset
// and a lamp template from lamps.tsj, which the map doesn't name.
func templateFS(t *testing.T, objects string) fstest.MapFS {
	tmj := `{"orientation": "orthogonal", "width": 1, "height": 1, "tilewidth": 16, "tileheight": 16,
	 "tilesets": [{"firstgid": 1, "source": "tiles.tsj"}],
	 "layers": [{"type": "tilelayer", "name": "floor", "width": 1, "height": 1, "data": [1]},
	  {"type": "objectgroup", "name": "props", "objects": [` + objects + `]}]}`
	return fstest.MapFS{
		"maps/one.tmj":    {Data: []byte(tmj)},
		"maps/tiles.tsj":  {Data: []byte(`{"image": "tiles.png", "tilewidth": 16, "tileheight": 16}`)},
		"maps/tiles.png":  {Data: pngOf(t, 32, 16)},
		"lamps/lamps.tsj": {Data: []byte(`{"image": "lamps.png", "tilewidth": 16, "tileheight": 32}`)},
		"lamps/lamps.png": {Data: pngOf(t, 48, 32)},
		"maps/crate.tj": {Data: []byte(`{"type": "template",
		 "tileset": {"firstgid": 1, "source": "tiles.tsj"},
		 "object": {"name": "crate", "type": "prop", "gid": 2, "width": 16, "height": 16, "rotation": 0,
		  "visible": true, "properties": [{"name": "loot", "type": "string", "value": "coins"},
		   {"name": "hp", "type": "int", "value": 3}]}}`)},
		"maps/lamp.tj": {Data: []byte(`{"type": "template",
		 "tileset": {"firstgid": 1, "source": "../lamps/lamps.tsj"},
		 "object": {"name": "lamp", "gid": 3, "width": 16, "height": 32}}`)},
		"maps/sign.tj": {Data: []byte(`{"type": "template",
		 "object": {"name": "sign", "width": 8, "height": 8,
		  "properties": [{"name": "art", "type": "file", "value": "sign.png"}]}}`)},
	}
}

func TestAnObjectMadeFromATemplateIsTheTemplateWithItsOwnValuesOver(t *testing.T) {
	m, err := LoadTiled(templateFS(t, `{"id": 7, "template": "crate.tj", "x": 32, "y": 48,
	  "properties": [{"name": "hp", "type": "int", "value": 9}]},
	 {"id": 8, "template": "lamp.tj", "x": 0, "y": 16, "name": "porch"}`), "maps/one.tmj")
	require.NoError(t, err)
	crate, ok := m.Object(7)
	require.True(t, ok)
	assert.Equal(t, "crate", crate.Name, "the template's name")
	assert.Equal(t, "prop", crate.Type)
	assert.Equal(t, geom.Rect{X: 32, Y: 48, W: 16, H: 16}, crate.Rect, "where the map places it, the template's size")
	hp, _ := crate.Properties.Int("hp")
	loot, _ := crate.Properties.String("loot")
	assert.Equal(t, 9, hp, "its own value over the template's")
	assert.Equal(t, "coins", loot, "the template's value it leaves alone")
	require.NotNil(t, crate.Tile)
	assert.Same(t, m.Tilesets()[0], crate.Tile.Tileset, "its tile from the map's own tileset of the same file")
	assert.Equal(t, 2, crate.Tile.ID)

	lamp, _ := m.Object(8)
	assert.Equal(t, "porch", lamp.Name, "its own name over the template's")
	require.NotNil(t, lamp.Tile)
	require.Len(t, m.Tilesets(), 2, "the template's tileset added to the map's")
	assert.Same(t, m.Tilesets()[1], lamp.Tile.Tileset)
	assert.Equal(t, "lamps/lamps.tsj", lamp.Tile.Tileset.Source())
	assert.Equal(t, 3, lamp.Tile.ID)
	assert.Contains(t, m.UsedTilesets(), lamp.Tile.Tileset, "and loaded, since the map uses it")
	assert.Equal(t, geom.V2(16, 32), m.Sprite(lamp).Size(), "its picture drawn")
}

func TestATileObjectsOpacityMakesItsSpriteSeeThrough(t *testing.T) {
	m, err := LoadTiled(templateFS(t, `{"id": 9, "gid": 1, "x": 0, "y": 16, "width": 16, "height": 16,
	 "opacity": 0.5}, {"id": 10, "gid": 1, "x": 0, "y": 16, "width": 16, "height": 16}`), "maps/one.tmj")
	require.NoError(t, err)
	faded, _ := m.Object(9)
	plain, _ := m.Object(10)
	assert.InDelta(t, 0.5, faded.Opacity, 1e-6)
	assert.InDelta(t, 1, plain.Opacity, 1e-6, "opaque unless Tiled says otherwise")
	assert.Equal(t, color.RGBA(1, 1, 1, 0.5), m.Sprite(faded).Tint())
	assert.Equal(t, color.White, m.Sprite(plain).Tint())
}

func TestATemplateTheLoaderCantReadIsAnError(t *testing.T) {
	for _, c := range []struct{ objects, want string }{
		{`{"id": 1, "template": "sign.tj", "x": 0, "y": 0}`, "property art: a file property in a template"},
		{`{"id": 1, "template": "missing.tj", "x": 0, "y": 0}`, "missing.tj"},
	} {
		_, err := LoadTiled(templateFS(t, c.objects), "maps/one.tmj")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "object 1", c.want)
		assert.Contains(t, err.Error(), c.want)
	}
}
