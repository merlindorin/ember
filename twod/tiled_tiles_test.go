package twod //nolint:testpackage // white-box: reads a tile's frame at a time

import (
	"encoding/json"
	"testing"
	"testing/fstest"

	"github.com/lantervale/ember/geom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tilesMap uses an atlas of four tiles: the first plays the next three for
// 100, 300 and 100 ms, the second is an old-style typed wall, the third a
// classed door; a second tileset in the same layer makes the map merge
// them.
const tilesMap = `{"orientation": "orthogonal", "width": 2, "height": 1, "tilewidth": 16, "tileheight": 16,
 "tilesets": [{"firstgid": 1, "image": "tiles.png", "tilewidth": 16, "tileheight": 16,
   "tiles": [{"id": 0, "animation": [{"tileid": 1, "duration": 100}, {"tileid": 2, "duration": 300},
     {"tileid": 3, "duration": 100}]}, {"id": 1, "type": "wall"}, {"id": 2, "class": "door"}]},
  {"firstgid": 5, "image": "more.png", "tilewidth": 16, "tileheight": 16}],
 "layers": [{"type": "tilelayer", "name": "floor", "width": 2, "height": 1, "data": [1, 5]}]}`

func loadTiles(t *testing.T, opts ...TileMapOption) *Tiled {
	t.Helper()
	m, err := LoadTiled(fstest.MapFS{
		"one.tmj":   {Data: []byte(tilesMap)},
		"tiles.png": {Data: pngOf(t, 64, 16)},
		"more.png":  {Data: pngOf(t, 16, 16)},
	}, "one.tmj", opts...)
	require.NoError(t, err)
	return m
}

func TestATiledAnimationPlaysEachFrameForItsOwnDuration(t *testing.T) {
	m := loadTiles(t)
	tileset := m.Tilesets()[0]
	frameAt := func(ts *Tileset, id int, time float32) geom.Rect {
		_, r := ts.image(id, time)
		return r
	}
	for _, c := range []struct {
		time float32
		want int
	}{{0.05, 2}, {0.2, 3}, {0.39, 3}, {0.45, 4}, {0.55, 2}, {-0.05, 4}} {
		assert.Equal(t, tileset.region(c.want), frameAt(tileset, 1, c.time), "at %.2f s", c.time)
	}

	floor, _ := m.TileLayer("floor")
	merged := floor.Tileset()
	require.NotSame(t, tileset, merged, "two tilesets in one layer are merged")
	_, want := tileset.image(1, 0.2)
	_, got := merged.image(1, 0.2)
	assert.Equal(t, want, got, "and the merged one plays the same")

	data, err := json.Marshal(tileset)
	require.NoError(t, err)
	var back Tileset
	require.NoError(t, json.Unmarshal(data, &back))
	assert.Equal(t, tileset.anims, back.anims, "the durations saved with the tileset")
	assert.Equal(t, "door", back.Class(3), "and the classes")
}

func TestATilesClassIsExposed(t *testing.T) {
	m := loadTiles(t)
	tileset := m.Tilesets()[0]
	assert.Equal(t, "wall", tileset.Class(2), "a type, as Tiled named it before 1.9")
	assert.Equal(t, "door", tileset.Class(3))
	assert.Empty(t, tileset.Class(1))
	floor, _ := m.TileLayer("floor")
	assert.Equal(t, "door", floor.Tileset().Class(3), "kept by the merged tileset")
}

func TestPixelScaleSizesTheCellsFromTheMapsTiles(t *testing.T) {
	for _, c := range []struct {
		name  string
		opts  []TileMapOption
		scale float32
	}{
		{"alone", []TileMapOption{PixelScale(2)}, 2},
		{"before a CellSize", []TileMapOption{PixelScale(2), CellSize(48, 48)}, 3},
		{"after one", []TileMapOption{CellSize(48, 48), PixelScale(2)}, 2},
	} {
		m := loadTiles(t, c.opts...)
		assert.InDelta(t, c.scale, m.Scale(), 1e-6, c.name)
		floor, _ := m.TileLayer("floor")
		assert.Equal(t, geom.V2(16*c.scale, 16*c.scale), floor.tileSize, c.name)
	}
}
