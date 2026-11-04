package twod //nolint:testpackage // white-box: shares twod's test helpers, reaches unexported state

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
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
	require.Len(t, m.Layers(), 2)
	ground, ok := m.Layer("ground")
	require.True(t, ok)
	assert.Equal(t, []int{3, 0, 3, 4}, ground.cells, "global ids from the tileset's first, flip bits kept apart")
	wall, _ := m.Layer("walls")
	assert.Equal(t, []int{1, 1, 0, 2}, wall.cells, "base64 and zlib")
	assert.Equal(t, geom.V2(16, 16), wall.tileSize)
	tiles := m.Tilesets()[0]
	assert.True(t, tiles.Solid(1))
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
	assert.True(t, m.Layers()[0].isometric)

	for name, body := range map[string]string{
		"hex.tmj":      `{"orientation": "hexagonal", "layers": []}`,
		"infinite.tmj": `{"orientation": "orthogonal", "infinite": true, "layers": []}`,
		"image.tmj":    `{"orientation": "orthogonal", ` + embedded + `, "layers": [{"type": "imagelayer", "name": "sky"}]}`,
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

	floor, _ := m.Layer("floor")
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
	assert.True(t, high.Solid(pillar))
	assert.Equal(t, geom.Rect{W: 16, H: 32}, frame(high, 5, 0), "a picture of its own, whole")
	cells, _ := high.Properties(5).Int("cells")
	assert.Equal(t, 2, cells)
	assert.Equal(
		t,
		[]geom.Rect{{Y: 16, W: 5, H: 16}, {X: 8, Y: 20, W: 4, H: 6}},
		high.Collision(5),
		"rectangles, a polygon by its bounds, no points",
	)
	assert.True(t, high.Solid(5))
	assert.False(t, high.Solid(pillar+1))
	assert.Equal(t, 5, high.Count())

	m, err := LoadTiled(fsys, "map.tmj")
	require.NoError(t, err)
	things, _ := m.Layer("things")
	assert.Equal(t, []int{2, 5, 9}, things.cells, "tiles of several tilesets keep their global ids")
	merged := things.Tileset()
	assert.Equal(t, geom.Rect{X: 16, W: 16, H: 16}, frame(merged, 2, 0), "the floor's second cell")
	assert.Equal(t, geom.Rect{X: 16, W: 32, H: 48}, frame(merged, 5, 0), "the pillar")
	assert.True(t, merged.Solid(5))
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
	a := NewTileset(NewTexture(image.NewRGBA(image.Rect(0, 0, 16, 8))), Cell(8, 8), Solid(2), TileAnimation(1, 4, 1, 2))
	b := NewTileset(NewTexture(image.NewRGBA(image.Rect(0, 0, 8, 8))), Cell(8, 8), Tall(1))
	m := MergeTilesets(a, b)
	assert.Equal(t, 3, m.Count())
	assert.True(t, m.Solid(2))
	assert.True(t, m.Tall(3), "b's first tile is the merged third")
	assert.Equal(t, frame(m, 2, 0), frame(m, 1, 0.25), "animations follow their frames")
	assert.Equal(t, geom.Rect{X: 8, W: 8, H: 8}, frame(m, 2, 0))
}

func TestMergeTilesetsKeepsShadowShapesNoShadowAndGroups(t *testing.T) {
	a := NewTileset(NewTexture(image.NewRGBA(image.Rect(0, 0, 16, 8))), Cell(8, 8), Solid(1, 2), CastShadow(false, 2))
	b := NewTileset(
		NewTexture(image.NewRGBA(image.Rect(0, 0, 16, 8))),
		Cell(8, 8),
		Solid(2),
		ShadowShapes(1, geom.Rect{W: 8, H: 4}),
		OccluderGroup(1, 2),
	)
	m := MergeTilesets(a, b)
	assert.False(t, m.CastShadow(2), "a's No shadow tile stays No shadow")
	assert.Equal(t, []geom.Rect{{W: 8, H: 4}}, m.ShadowShapes(3), "b's first tile keeps its shadow shape")
	assert.Equal(t, 1, m.OccluderGroup(4), "and b's second its group")
}

func TestTiledShadowShapesAndPropertiesTellLightFromCollision(t *testing.T) {
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
	assert.Equal(t, []geom.Rect{{X: 4, Y: 8, W: 8, H: 8}}, ts.Collision(1), "an unclassed shape collides")
	assert.Equal(t, []geom.Rect{{W: 16, H: 8}}, ts.ShadowShapes(1), "a shadow shape only shadows")
	assert.True(t, ts.Solid(1) && ts.CastShadow(1))
	assert.False(t, ts.Solid(2), "only shadow shapes: it blocks light, not bodies")
	assert.True(t, ts.CastShadow(2))
	assert.True(t, ts.Solid(3), "shadow none still blocks bodies")
	assert.False(t, ts.CastShadow(3), "but no light")
	assert.Equal(t, 2, ts.OccluderGroup(4))

	m := NewTileMap(ts, Cells([][]int{{1, 2, 3, 4}}), CellSize(16, 16))
	everything := geom.Rect{X: -100, Y: -100, W: 400, H: 400}
	assert.ElementsMatch(
		t,
		[]geom.Rect{{X: 4, Y: 8, W: 8, H: 8}, {X: 48, Y: 0, W: 16, H: 16}, {X: 32, W: 16, H: 16}},
		m.SolidIn(everything),
		"bodies meet tile 1's shape and the solid cells of 3 and 4, not tile 2",
	)
	shadows := m.ShadowsIn(everything)
	assert.ElementsMatch(t, []ShadowRect{
		{Rect: geom.Rect{X: 4, Y: 8, W: 8, H: 8}, Height: 16}, {Rect: geom.Rect{W: 16, H: 8}, Height: 16},
		{Rect: geom.Rect{X: 18, Y: 2, W: 12, H: 12}},
		{Rect: geom.Rect{X: 48, W: 16, H: 16}, Group: 2, Height: 16},
	}, shadows, "light meets tile 1's both shapes, tile 2's shadow, tile 4 in group 2, not tile 3; "+
		"solid tiles stand their cell high, tile 2's shadow shape (not solid) all the way")
}

func TestABodyShapeStopsBodiesAndCastsNoShadow(t *testing.T) {
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
	assert.Equal(t, []geom.Rect{{Y: 12, W: 16, H: 4}}, ts.BodyShapes(1), "class body")
	assert.Equal(t, []geom.Rect{{W: 16, H: 12}}, ts.Collision(1))
	assert.True(
		t,
		ts.Solid(2) && !ts.CastShadow(2),
		"a solid tile whose only shape is a body shape casts nothing, not its whole cell",
	)
	assert.False(t, ts.CastShadow(3), "shadow none silences the whole tile, shadow shapes included")

	m := NewTileMap(ts, Cells([][]int{{1, 2, 3}}), CellSize(16, 16))
	everything := geom.Rect{X: -100, Y: -100, W: 400, H: 400}
	assert.ElementsMatch(
		t,
		[]geom.Rect{{W: 16, H: 12}, {Y: 12, W: 16, H: 4}, {X: 16, Y: 8, W: 16, H: 8}},
		m.SolidIn(everything),
		"bodies meet the collision and body shapes",
	)
	var shadows []geom.Rect
	for _, s := range m.ShadowsIn(everything) {
		shadows = append(shadows, s.Rect)
	}
	assert.Equal(t, []geom.Rect{{W: 16, H: 12}}, shadows, "light meets the collision shape only")

	again := NewTileset(
		NewTexture(image.NewRGBA(image.Rect(0, 0, 16, 16))),
		Cell(16, 16),
		BodyShapes(1, geom.Rect{W: 16, H: 4}),
	)
	assert.Equal(t, []geom.Rect{{W: 16, H: 4}}, again.BodyShapes(1))
}

func TestABodyShapeReachingPastItsCellIsFoundFromTheNextCell(t *testing.T) {
	ts := NewTileset(
		NewTexture(image.NewRGBA(image.Rect(0, 0, 16, 16))),
		Cell(16, 16),
		BodyShapes(1, geom.Rect{W: 16, H: 20}),
	)
	m := NewTileMap(ts, Cells([][]int{{1}, {0}}), CellSize(16, 16))
	below := geom.Rect{X: 2, Y: 17, W: 4, H: 4}
	assert.Equal(
		t,
		[]geom.Rect{{W: 16, H: 20}},
		m.SolidIn(below),
		"its 4 points below its cell, asked from the cell under it",
	)
}
