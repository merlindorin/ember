package twod //nolint:testpackage // white-box: shares twod's test helpers, reaches unexported state

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/merlindorin/ember/geom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func pngOf(t *testing.T, w, h int) []byte {
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, w, h))))
	return buf.Bytes()
}

func zlibCells(t *testing.T, cells ...uint32) string {
	var raw bytes.Buffer
	z := zlib.NewWriter(&raw)
	require.NoError(t, binary.Write(z, binary.LittleEndian, cells))
	require.NoError(t, z.Close())
	return base64.StdEncoding.EncodeToString(raw.Bytes())
}

const tsj = `{"image": "../art/tiles.png", "tilewidth": 8, "tileheight": 8,
 "tiles": [
  {"id": 0, "properties": [{"name": "solid", "type": "bool", "value": true}]},
  {"id": 1, "properties": [{"name": "tall", "type": "bool", "value": true}, {"name": "sound",
     "type": "string", "value": "leaves"}]},
  {"id": 2, "animation": [{"tileid": 2, "duration": 250}, {"tileid": 3, "duration": 250}]}
 ]}`

func TestLoadTiledReadsLayersTilesetsAndObjects(t *testing.T) {
	walls := zlibCells(t, 10, 10, 0, 11)
	tmj := `{"orientation": "orthogonal", "tilewidth": 16, "tileheight": 16,
	 "tilesets": [{"firstgid": 10, "source": "tilesets/tiles.tsj"}],
	 "layers": [
	  {"type": "tilelayer", "name": "ground", "width": 2, "height": 2, "data": [12, 0, ` + "2147483660" + `, 13]},
	  {"type": "tilelayer", "name": "walls", "width": 2, "height": 2, "encoding": "base64",
	     "compression": "zlib", "data": "` + walls + `"},
	  {"type": "objectgroup", "name": "spawns", "objects": [{"name": "hero", "class": "spawn", "x": 4, "y": 6,
	     "width": 8, "height": 8, "properties": [{"name": "facing", "type": "string", "value": "left"}]}]}
	 ]}`
	fsys := fstest.MapFS{
		"levels/one.tmj":            {Data: []byte(tmj)},
		"levels/tilesets/tiles.tsj": {Data: []byte(tsj)},
		"levels/art/tiles.png":      {Data: pngOf(t, 32, 8)},
	}
	m, err := LoadTiled(fsys, "levels/one.tmj")
	require.NoError(t, err)
	require.Len(t, m.TileLayers(), 2)
	ground, ok := m.TileLayer("ground")
	require.True(t, ok)
	assert.Equal(t, []int{3, 0, 3, 4}, ground.cells, "global ids from the tileset's first, flip bits kept apart")
	wall, _ := m.TileLayer("walls")
	assert.Equal(t, []int{1, 1, 0, 2}, wall.cells, "base64 and zlib")
	assert.Equal(t, geom.V2(16, 16), wall.tileSize)
	tiles := m.Tilesets()[0]
	assert.True(t, solidOf(tiles, 1))
	assert.True(t, tiles.Tall(2))
	sound, _ := tiles.Property(2, "sound")
	assert.Equal(t, "leaves", sound)
	assert.Equal(t, frame(tiles, 4, 0), frame(tiles, 3, 0.25), "the animation runs at 4 fps")
	assert.Len(t, wall.TallSprites(), 1, "tall tiles stand as sprites")
	assert.Equal(
		t,
		[]TiledObject{
			{
				Name:       "hero",
				Type:       "spawn",
				Rect:       geom.Rect{X: 4, Y: 6, W: 8, H: 8},
				Visible:    true,
				Opacity:    1,
				Properties: TiledProperties{"facing": "left"},
			},
		},
		m.Objects("spawns"),
	)
}

func TestLoadTiledReadsIsometricMapsAndRefusesTheRest(t *testing.T) {
	embedded := `"tilesets": [{"firstgid": 1, "image": "tiles.png", "tilewidth": 8, "tileheight": 8}]`
	layer := `{"type": "tilelayer", "name": "floor", "width": 1, "height": 1, "data": [1]}`
	fsys := fstest.MapFS{"tiles.png": {Data: pngOf(t, 16, 8)}}
	write := func(name, body string) { fsys[name] = &fstest.MapFile{Data: []byte(body)} }
	write(
		"iso.tmj",
		`{"orientation": "isometric", "tilewidth": 32, "tileheight": 16, `+embedded+`, "layers": [`+layer+`]}`,
	)
	m, err := LoadTiled(fsys, "iso.tmj")
	require.NoError(t, err)
	assert.True(t, m.TileLayers()[0].isometric)

	for name, body := range map[string]string{
		"hex.tmj":      `{"orientation": "hexagonal", "layers": []}`,
		"infinite.tmj": `{"orientation": "orthogonal", "infinite": true, "layers": []}`,
		"chunks.tmj":   `{"orientation": "orthogonal", "layers": [{"type": "chunks", "name": "sky"}]}`,
	} {
		write(name, body)
		_, loadErr := LoadTiled(fsys, name)
		assert.Error(t, loadErr, name)
	}
	_, err = LoadTiled(fsys, "missing.tmj")
	assert.Error(t, err)
}

func TestLoadTiledReadsTypedPropertiesAndDrawsFlippedCellsFlipped(t *testing.T) {
	const flipX, flipY, flipD = 0x80000000, 0x40000000, 0x20000000
	tmj := `{"orientation": "orthogonal", "tilewidth": 8, "tileheight": 8,
	 "properties": [{"name": "room", "type": "string", "value": "crypt"}, {"name": "tier", "type": "int", "value": 3},
	  {"name": "weight", "type": "float", "value": 0.25}, {"name": "boss", "type": "bool", "value": true}],
	 "tilesets": [{"firstgid": 1, "image": "tiles.png", "tilewidth": 8, "tileheight": 8}],
	 "layers": [
	  {"type": "tilelayer", "name": "floor", "width": 4, "height": 1, "properties": [{"name": "lit",
	     "type": "bool", "value": false}],
	   "data": [1, ` + fmt.Sprint(uint32(1|flipX)) + `,
	     ` + fmt.Sprint(uint32(1|flipY)) + `, ` + fmt.Sprint(uint32(1|flipD)) + `]},
	  {"type": "objectgroup", "name": "spawns", "objects": [{"name": "boss", "x": 0, "y": 0,
	     "properties": [{"name": "hp", "type": "int", "value": 40}]}]}
	 ]}`
	m, err := LoadTiled(fstest.MapFS{"room.tmj": {Data: []byte(tmj)}, "tiles.png": {Data: pngOf(t, 16, 8)}}, "room.tmj")
	require.NoError(t, err)
	p := m.Properties()
	room, _ := p.String("room")
	tier, _ := p.Int("tier")
	weight, _ := p.Float("weight")
	boss, _ := p.Bool("boss")
	assert.Equal(t, "crypt", room)
	assert.Equal(t, 3, tier, "an int stays an int")
	assert.Equal(t, 0.25, weight)
	assert.True(t, boss)
	lit, ok := m.LayerProperties("floor").Bool("lit")
	assert.True(t, ok && !lit, "a layer's own properties")
	hp, _ := m.Objects("spawns")[0].Properties.Int("hp")
	assert.Equal(t, 40, hp, "and an object's, typed too")

	floor, _ := m.TileLayer("floor")
	assert.Equal(t, []int{1, 1, 1, 1}, floor.cells, "the same tile, its flip bits apart")
	assert.Equal(
		t,
		[]TileFlip{0, TileFlipX, TileFlipY, TileFlipDiagonal},
		[]TileFlip{floor.Flip(0, 0), floor.Flip(1, 0), floor.Flip(2, 0), floor.Flip(3, 0)},
	)

	w, win, c := app()
	win.Add(floor)
	w.AddEntities(floor)
	w.RunFrames(2)
	require.Len(t, c.Unindexed(), 4*6)
	corners := func(cell int) [4]geom.Vec2 {
		v := c.Unindexed()[cell*6:]
		return [4]geom.Vec2{v[0].UV, v[1].UV, v[2].UV, v[5].UV}
	}
	tl, tr, br, bl := geom.V2(0, 0), geom.V2(0.5, 0), geom.V2(0.5, 1), geom.V2(0, 1)
	assert.Equal(t, [4]geom.Vec2{tl, tr, br, bl}, corners(0), "as it is")
	assert.Equal(t, [4]geom.Vec2{tr, tl, bl, br}, corners(1), "mirrored across")
	assert.Equal(t, [4]geom.Vec2{bl, br, tr, tl}, corners(2), "upside down")
	assert.Equal(t, [4]geom.Vec2{tl, bl, br, tr}, corners(3), "across its diagonal")
}

func TestLoadTiledReadsCollectionsAndLayersOfSeveralTilesets(t *testing.T) {
	collection := `{"tilewidth": 32, "tileheight": 48, "grid": {"width": 16, "height": 16},
	 "tiles": [
	  {"id": 0, "image": "walls.png", "x": 16, "y": 0, "width": 32, "height": 48, "properties": [{"name": "name",
	     "type": "string", "value": "pillar"}, {"name": "solid", "type": "bool", "value": true}]},
	  {"id": 4, "image": "lamp.png", "properties": [{"name": "cells", "type": "int", "value": 2}],
	   "objectgroup": {"objects": [{"x": 0, "y": 16, "width": 5, "height": 16}, {"x": 8, "y": 20,
	      "polygon": [{"x": 0, "y": 0}, {"x": 4, "y": 2}, {"x": 1, "y": 6}]}, {"x": 3, "y": 3, "point": true}]}}
	 ]}`
	tmj := `{"orientation": "orthogonal", "tilewidth": 16, "tileheight": 16,
	 "tilesets": [{"firstgid": 1, "image": "floor.png", "tilewidth": 16, "tileheight": 16}, {"firstgid": 5,
	    "source": "high.tsj"}],
	 "layers": [{"type": "tilelayer", "name": "things", "width": 3, "height": 1, "data": [2, 5, 9]}]}`
	fsys := fstest.MapFS{
		"map.tmj":   {Data: []byte(tmj)},
		"high.tsj":  {Data: []byte(collection)},
		"floor.png": {Data: pngOf(t, 32, 32)},
		"walls.png": {Data: pngOf(t, 64, 48)},
		"lamp.png":  {Data: pngOf(t, 16, 32)},
	}
	high, err := LoadTiledTileset(fsys, "high.tsj")
	require.NoError(t, err)
	pillar, ok := high.ID("pillar")
	require.True(t, ok)
	assert.Equal(t, geom.Rect{X: 16, W: 32, H: 48}, frame(high, pillar, 0), "a rectangle of a shared picture")
	assert.True(t, solidOf(high, pillar))
	assert.Equal(t, geom.Rect{W: 16, H: 32}, frame(high, 5, 0), "a picture of its own, whole")
	cells, _ := high.Properties(5).Int("cells")
	assert.Equal(t, 2, cells)
	assert.Equal(
		t,
		[]TileShape{{geom.Rect{Y: 16, W: 5, H: 16}, ""}, {geom.Rect{X: 8, Y: 20, W: 4, H: 6}, ""}},
		high.Shapes(5),
		"rectangles, a polygon by its bounds, no points",
	)
	assert.False(t, solidOf(high, pillar+1))
	assert.Equal(t, 5, high.Count())

	m, err := LoadTiled(fsys, "map.tmj")
	require.NoError(t, err)
	things, _ := m.TileLayer("things")
	assert.Equal(t, []int{2, 5, 9}, things.cells, "tiles of several tilesets keep their global ids")
	merged := things.Tileset()
	assert.Equal(t, geom.Rect{X: 16, W: 16, H: 16}, frame(merged, 2, 0), "the floor's second cell")
	assert.Equal(t, geom.Rect{X: 16, W: 32, H: 48}, frame(merged, 5, 0), "the pillar")
	assert.True(t, solidOf(merged, 5))
	assert.Equal(t, geom.Rect{W: 16, H: 32}, frame(merged, 9, 0), "the lamp")
	id, _ := merged.ID("pillar")
	assert.Equal(t, 5, id)
	assert.Equal(
		t,
		[]int{16, 16},
		func() []int { w, h := merged.TilePixels(); return []int{w, h} }(),
		"drawn at the map's scale",
	)
}

func TestMergeTilesetsNumbersEachSetOnFromTheLast(t *testing.T) {
	a := NewTileset(NewTexture(image.NewRGBA(image.Rect(0, 0, 16, 8))), Cell(8, 8), TileValue(2, "solid", true),
		TileAnimation(1, 4, 1, 2))
	b := NewTileset(NewTexture(image.NewRGBA(image.Rect(0, 0, 8, 8))), Cell(8, 8), Tall(1))
	m := MergeTilesets(a, b)
	assert.Equal(t, 3, m.Count())
	assert.True(t, solidOf(m, 2))
	assert.True(t, m.Tall(3), "b's first tile is the merged third")
	assert.Equal(t, frame(m, 2, 0), frame(m, 1, 0.25), "animations follow their frames")
	assert.Equal(t, geom.Rect{X: 8, W: 8, H: 8}, frame(m, 2, 0))
}

func TestMergeTilesetsKeepsEveryTilesShapesAndValues(t *testing.T) {
	a := NewTileset(NewTexture(image.NewRGBA(image.Rect(0, 0, 16, 8))), Cell(8, 8), TileValue(2, "shadow", "none"))
	b := NewTileset(
		NewTexture(image.NewRGBA(image.Rect(0, 0, 16, 8))),
		Cell(8, 8),
		TileShapes(1, "shadow", geom.Rect{W: 8, H: 4}),
		TileValue(2, "occluder_group", 1),
	)
	m := MergeTilesets(a, b)
	shadow, _ := m.Value(2, "shadow")
	group, _ := m.Value(4, "occluder_group")
	assert.Equal(t, "none", shadow.String(), "a's values")
	assert.Equal(t, []TileShape{{geom.Rect{W: 8, H: 4}, "shadow"}}, m.Shapes(3), "b's first tile keeps its shape")
	assert.Equal(t, 1, group.Int(), "and b's second its value")
}

// solidOf is whether tile id of ts says it is solid, as collide/tiles
// reads it.
func solidOf(ts *Tileset, id int) bool {
	v, ok := ts.Value(id, "solid")
	return ok && v.Bool()
}

func TestATiledTilesetsShapesKeepTheirClassAndItsPropertiesTheirValues(t *testing.T) {
	tsj := `{"image": "tiles.png", "tilewidth": 16, "tileheight": 16, "columns": 4, "tilecount": 4,
	 "tiles": [
	  {"id": 0, "objectgroup": {"objects": [{"x": 4, "y": 8, "width": 8, "height": 8}, {"class": "shadow",
	     "x": 0, "y": 0, "width": 16, "height": 8}]}},
	  {"id": 1, "objectgroup": {"objects": [{"type": "shadow", "x": 2, "y": 2, "width": 12, "height": 12}]}},
	  {"id": 2, "properties": [{"name": "solid", "type": "bool", "value": true}, {"name": "shadow",
	     "type": "string", "value": "none"}]},
	  {"id": 3, "properties": [{"name": "solid", "type": "bool", "value": true}, {"name": "occluder_group",
	     "type": "int", "value": 2}]}
	 ]}`
	fsys := fstest.MapFS{"walls.tsj": {Data: []byte(tsj)}, "tiles.png": {Data: pngOf(t, 64, 16)}}
	ts, err := LoadTiledTileset(fsys, "walls.tsj")
	require.NoError(t, err)
	assert.Equal(t, []TileShape{{geom.Rect{X: 4, Y: 8, W: 8, H: 8}, ""}, {geom.Rect{W: 16, H: 8}, "shadow"}},
		ts.Shapes(1), "each shape with its class, for collide/tiles and light to read")
	assert.Equal(t, []TileShape{{geom.Rect{X: 2, Y: 2, W: 12, H: 12}, "shadow"}}, ts.Shapes(2), "a type, as before 1.9")
	shadow, _ := ts.Value(3, "shadow")
	group, _ := ts.Value(4, "occluder_group")
	assert.True(t, solidOf(ts, 3))
	assert.Equal(t, "none", shadow.String())
	assert.Equal(t, 2, group.Int())
}

func TestABodyShapeKeepsItsClass(t *testing.T) {
	tsj := `{"image": "tiles.png", "tilewidth": 16, "tileheight": 16, "columns": 4, "tilecount": 4,
	 "tiles": [
	  {"id": 0, "objectgroup": {"objects": [{"x": 0, "y": 0, "width": 16, "height": 12}, {"class": "body",
	     "x": 0, "y": 12, "width": 16, "height": 4}]}},
	  {"id": 1, "properties": [{"name": "solid", "type": "bool", "value": true}],
	     "objectgroup": {"objects": [{"type": "body", "x": 0, "y": 8, "width": 16, "height": 8}]}},
	  {"id": 2, "properties": [{"name": "shadow", "type": "string", "value": "none"}],
	     "objectgroup": {"objects": [{"class": "shadow", "x": 0, "y": 0, "width": 16, "height": 8}]}}
	 ]}`
	fsys := fstest.MapFS{"bars.tsj": {Data: []byte(tsj)}, "tiles.png": {Data: pngOf(t, 64, 16)}}
	ts, err := LoadTiledTileset(fsys, "bars.tsj")
	require.NoError(t, err)
	assert.Equal(t, []TileShape{{geom.Rect{W: 16, H: 12}, ""}, {geom.Rect{Y: 12, W: 16, H: 4}, "body"}}, ts.Shapes(1))
	assert.Equal(t, []TileShape{{geom.Rect{Y: 8, W: 16, H: 8}, "body"}}, ts.Shapes(2))
	shadow, _ := ts.Value(3, "shadow")
	assert.Equal(t, "none", shadow.String())

	again := NewTileset(
		NewTexture(image.NewRGBA(image.Rect(0, 0, 16, 16))),
		Cell(16, 16),
		TileShapes(1, "body", geom.Rect{W: 16, H: 4}),
	)
	assert.Equal(t, []TileShape{{geom.Rect{W: 16, H: 4}, "body"}}, again.Shapes(1))
}

func TestLoadTiledKeepsATileObjectsTileAndItsPictureFile(t *testing.T) {
	const flipX = 0x80000000
	rocks := `{"tilewidth": 32, "tileheight": 24, "tiles": [
	  {"id": 0, "image": "../art/pebble.png", "imagewidth": 8, "imageheight": 8},
	  {"id": 2, "image": "../art/rock.png", "imagewidth": 32, "imageheight": 24,
	   "properties": [{"name": "kind", "type": "string", "value": "rock"}]}]}`
	tmj := `{"orientation": "orthogonal", "tilewidth": 16, "tileheight": 16,
	 "tilesets": [{"firstgid": 1, "source": "tilesets/tiles.tsj"}, {"firstgid": 10, "source": "tilesets/rocks.tsj"}],
	 "layers": [
	  {"type": "tilelayer", "name": "floor", "width": 1, "height": 1, "data": [1]},
	  {"type": "objectgroup", "name": "props", "objects": [
	   {"name": "boulder", "type": "prop", "gid": ` + fmt.Sprint(flipX|12) + `, "x": 40, "y": 64, "width": 32,
	    "height": 24},
	   {"name": "spot", "x": 4, "y": 6, "width": 8, "height": 8}]}
	 ]}`
	fsys := fstest.MapFS{
		"levels/one.tmj":            {Data: []byte(tmj)},
		"levels/tilesets/tiles.tsj": {Data: []byte(tsj)},
		"levels/tilesets/rocks.tsj": {Data: []byte(rocks)},
		"levels/art/tiles.png":      {Data: pngOf(t, 32, 8)},
		"levels/art/pebble.png":     {Data: pngOf(t, 8, 8)},
		"levels/art/rock.png":       {Data: pngOf(t, 32, 24)},
	}
	m, err := LoadTiled(fsys, "levels/one.tmj")
	require.NoError(t, err)
	objects := m.Objects("props")
	require.Len(t, objects, 2)
	boulder := objects[0]
	require.NotNil(t, boulder.Tile)
	assert.Equal(
		t,
		TiledTile{GID: flipX | 12, Tileset: m.Tilesets()[1], ID: 3, Flip: TileFlipX},
		*boulder.Tile,
		"Tiled's tile 2 of the set starting at 10, flipped across",
	)
	assert.Equal(t, geom.Rect{X: 40, Y: 64, W: 32, H: 24}, boulder.Rect, "Tiled's rule: y is the picture's bottom")
	assert.Equal(t, TiledRectangle, boulder.Shape)
	kind, _ := boulder.Tile.Tileset.Property(boulder.Tile.ID, "kind")
	assert.Equal(t, "rock", kind)
	file, ok := boulder.Tile.Tileset.ImageFile(boulder.Tile.ID)
	assert.True(t, ok)
	assert.Equal(t, "levels/art/rock.png", file, "the picture's path in the file system, from the tileset's folder")
	atlas, _ := m.Tilesets()[0].ImageFile(2)
	assert.Equal(t, "levels/art/tiles.png", atlas, "an atlas's tiles share its file")
	tex, _ := boulder.Tile.Tileset.image(boulder.Tile.ID, 0)
	assert.Equal(t, geom.V2(32, 24), tex.Size(), "the picture is loaded: the set is used by an object")
	assert.Nil(t, objects[1].Tile, "a plain object has no tile")
}

func TestLoadTiledKeepsEachObjectsShape(t *testing.T) {
	tmj := `{"orientation": "orthogonal", "tilewidth": 16, "tileheight": 16, "layers": [
	  {"type": "objectgroup", "name": "markers", "objects": [
	   {"name": "box", "x": 1, "y": 2, "width": 3, "height": 4},
	   {"name": "ring", "ellipse": true, "x": 10, "y": 20, "width": 30, "height": 40},
	   {"name": "wake", "point": true, "x": 5, "y": 6},
	   {"name": "pool", "x": 100, "y": 200, "polygon": [{"x": 0, "y": 0}, {"x": 10, "y": 0}, {"x": 5, "y": -8}]},
	   {"name": "path", "x": 50, "y": 60, "polyline": [{"x": 0, "y": 0}, {"x": 16, "y": 16}]}]}
	 ]}`
	m, err := LoadTiled(fstest.MapFS{"m.tmj": {Data: []byte(tmj)}}, "m.tmj")
	require.NoError(t, err)
	byName := map[string]TiledObject{}
	for _, o := range m.Objects("markers") {
		byName[o.Name] = o
	}
	assert.Equal(t, TiledRectangle, byName["box"].Shape)
	assert.Equal(t, TiledEllipse, byName["ring"].Shape)
	assert.Equal(t, geom.Rect{X: 10, Y: 20, W: 30, H: 40}, byName["ring"].Rect, "an ellipse by its box")
	assert.Equal(t, TiledPoint, byName["wake"].Shape)
	assert.Equal(t, geom.Rect{X: 5, Y: 6}, byName["wake"].Rect, "a point at its x, y, sized 0")
	assert.Equal(t, TiledPolygon, byName["pool"].Shape)
	assert.Equal(
		t,
		[]geom.Vec2{geom.V2(100, 200), geom.V2(110, 200), geom.V2(105, 192)},
		byName["pool"].Points,
		"points in the map's pixels",
	)
	assert.Equal(t, TiledPolyline, byName["path"].Shape)
	assert.Equal(t, []geom.Vec2{geom.V2(50, 60), geom.V2(66, 76)}, byName["path"].Points)
	for _, o := range byName {
		assert.Nil(t, o.Tile, o.Name)
	}
}

func TestLoadTiledLoadsOnlyTheTilesetsItUses(t *testing.T) {
	// The library's .tsj and the embedded atlas's picture are not in the
	// file system: reading or loading either would fail.
	rocks := `{"tilewidth": 16, "tileheight": 16, "tiles": [{"id": 0, "image": "../art/rock.png"}]}`
	tmj := `{"orientation": "orthogonal", "tilewidth": 16, "tileheight": 16,
	 "tilesets": [{"firstgid": 1, "source": "tilesets/tiles.tsj"}, {"firstgid": 10, "source": "tilesets/library.tsj"},
	  {"firstgid": 20, "image": "art/missing-atlas.png", "tilewidth": 8, "tileheight": 8,
	   "tiles": [{"id": 0, "properties": [{"name": "solid", "type": "bool", "value": true}]}]},
	  {"firstgid": 100, "source": "tilesets/rocks.tsj"}],
	 "layers": [
	  {"type": "tilelayer", "name": "floor", "width": 2, "height": 1, "data": [1, 0]},
	  {"type": "objectgroup", "name": "props", "objects": [{"gid": 100, "x": 0, "y": 16, "width": 16, "height": 16}]}
	 ]}`
	files := func(level string) fstest.MapFS {
		return fstest.MapFS{
			"levels/one.tmj":            {Data: []byte(level)},
			"levels/tilesets/tiles.tsj": {Data: []byte(tsj)},
			"levels/tilesets/rocks.tsj": {Data: []byte(rocks)},
			"levels/art/tiles.png":      {Data: pngOf(t, 32, 8)},
			"levels/art/rock.png":       {Data: pngOf(t, 16, 16)},
		}
	}
	m, err := LoadTiled(files(tmj), "levels/one.tmj")
	require.NoError(t, err, "the unused library's file and the unused atlas's picture are never opened")
	sets := m.Tilesets()
	require.Len(t, sets, 4, "every tileset is listed, in order")
	assert.Equal(t, []*Tileset{sets[0], sets[3]}, m.UsedTilesets(), "a tile layer's set and a tile object's")
	library := sets[1]
	assert.Equal(t, "levels/tilesets/library.tsj", library.Source(), "an unread tileset knows its file")
	assert.Zero(t, library.Count(), "and holds no tiles")
	assert.Equal(t, "levels/tilesets/rocks.tsj", sets[3].Source())
	assert.Empty(t, sets[2].Source(), "an embedded tileset has no file of its own")
	assert.True(t, solidOf(sets[2], 1), "an unused embedded tileset keeps its tiles' properties")
	assert.Nil(t, sets[2].texture, "but loads no picture")

	_, err = LoadTiled(files(strings.Replace(tmj, `"data": [1, 0]`, `"data": [1, 10]`, 1)), "levels/one.tmj")
	require.Error(t, err, "once a cell takes a library tile, the library is read, and a missing file is an error")
	assert.Contains(t, err.Error(), "library.tsj")
}
