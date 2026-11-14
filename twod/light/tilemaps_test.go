package light //nolint:testpackage // white-box: shares light's test scene, reaches unexported state

import (
	"testing"

	"github.com/lantervale/ember/collide/tiles"
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/internal/draw"
	"github.com/lantervale/ember/internal/lit"
	"github.com/lantervale/ember/twod"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// occluding is how many occluder quads light drew in win after two frames
// of a map of tiles, every tile solid, shown in it.
func occluding(t *testing.T, opts ...twod.TileMapOption) int {
	t.Helper()
	set := twod.NewTileset(picture(16, 16), twod.Cell(16, 16), twod.TileValue(1, "solid", true))
	m := twod.NewTileMap(set, append([]twod.TileMapOption{twod.Cells([][]int{{1, 0, 1}}),
		twod.CellSize(16, 16)}, opts...)...)
	w, win, _ := scene(t)
	layer := twod.NewLayer(0).Add(m, NewPoint(80, color.White))
	win.Add(layer)
	w.AddEntities(layer)
	w.RunFrames(2)
	l := draw.LightingFor(w, win)
	require.NotNil(t, l)
	return len(l.Occluders.Vertices) / 4
}

func TestLightFindsTheTileMapsInItsWindowItself(t *testing.T) {
	assert.Equal(t, 2, occluding(t), "two solid tiles, with no collide space given")
	assert.Equal(t, 2, occluding(t, twod.Hidden()), "a hidden map still casts")
	assert.Equal(t, 0, occluding(t, NoShadows()), "a map made with NoShadows casts nothing")
}

func TestTileShadowsInIsWhatLightCastsWithoutAWindow(t *testing.T) {
	ts := twod.NewTileset(picture(16, 16), twod.Cell(16, 16), twod.TileValue(1, "solid", true), TileOccluderGroup(1, 1))
	m := twod.NewTileMap(ts, twod.Cells([][]int{{1, 0, 1}}), twod.CellSize(16, 16), twod.At(100, 0), NoShadows())
	got := TileShadowsIn(nil, m, geom.Rect{X: 100, W: 20, H: 16})
	assert.Equal(t, []TileShadowRect{{geom.Rect{X: 100, W: 16, H: 16}, 1, 16}}, got, "the cells touching the area")
	assert.True(t, NoShadowsOf(m))
	inf := twod.NewInfiniteMap(ts, twod.CellSize(16, 16), twod.Generator(func(_, _ int, cells [][]int) {
		cells[0][0] = 1
	}))
	assert.Len(t, TileShadowsIn(nil, inf, geom.Rect{X: -40, Y: -40, W: 400, H: 400}), 4, "an endless map's too")
}

type readsMap struct{ walls }

func (readsMap) ReadsTileMap() lit.FromMap { return lit.FromMap{} }

func TestLightSkipsTheSolidsThatReadATileMap(t *testing.T) {
	assert.True(t, fromMap(readsMap{}), "collision's reading of a map: light reads the map itself")
	assert.True(t, fromMap(twod.NewTileMap(nil)))
	assert.False(t, fromMap(walls{}), "a game's own solid still casts")
}

func TestAStaticLightIsBakedAgainWhenAMapComesOrGoes(t *testing.T) {
	solid := func() *twod.TileMap {
		ts := twod.NewTileset(picture(16, 16), twod.Cell(16, 16), twod.TileValue(1, "solid", true))
		return twod.NewTileMap(ts, twod.Cells([][]int{{1}}), twod.CellSize(16, 16), twod.At(40, 0))
	}
	w, win, _ := scene(t)
	a := solid()
	layer := twod.NewLayer(0).Add(NewPoint(100, color.White, At(0, 0), Static()), a)
	win.Add(layer)
	w.AddEntities(layer)
	w.RunFrames(2)
	version := func() uint64 { return draw.LightingFor(w, win).Static.Tiles[0].Version }
	before := version()
	w.RemoveEntity(a)
	w.RunFrames(1)
	gone := version()
	assert.NotEqual(t, before, gone, "a map gone")
	b := solid()
	layer.Add(b)
	w.AddEntities(b)
	w.RunFrames(1)
	assert.NotEqual(t, gone, version(), "a map come, as a room loaded")
}

func TestATileMapsShadowsFollowItsCells(t *testing.T) {
	set := twod.NewTileset(picture(16, 16), twod.Cell(16, 16), twod.TileValue(1, "solid", true))
	m := twod.NewTileMap(set, twod.Cells([][]int{{1, 0, 0}}), twod.CellSize(16, 16))
	c := &mapCache{}
	area := geom.Rect{X: -100, Y: -100, W: 300, H: 300}
	assert.Len(t, c.shadowsIn(nil, m, area), 1)
	m.Set(2, 0, 1)
	assert.Len(t, c.shadowsIn(nil, m, area), 2, "made again after a Set")
	m.SetPosition(geom.V2(1000, 0))
	assert.Empty(t, c.shadowsIn(nil, m, area), "moved with the map, nothing placed again")
	got := c.shadowsIn(nil, m, geom.Rect{X: 1000, W: 64, H: 16})
	require.Len(t, got, 2)
	assert.Equal(t, geom.Rect{X: 1000, W: 16, H: 16}, got[0].rect)
}

func TestLightsMapShadowsOnAMapOfEveryKindOfTile(t *testing.T) {
	ts := twod.NewTileset(picture(64, 32), twod.Cell(16, 16), tiles.Solid(1, 2), twod.Tall(3),
		tiles.Collision(2, geom.Rect{Y: 8, W: 16, H: 8}), TileShadowShapes(3, geom.Rect{X: 2, Y: 4, W: 10, H: 12}),
		tiles.BodyShapes(4, geom.Rect{W: 4, H: 4}), tiles.Solid(4), tiles.Solid(5), TileShadow(false, 5),
		TileOccluderGroup(2, 2), TileHeight(20, 1))
	grid := [][]int{{1, 2, 3, 4, 5, 0}, {0, 3, 2, 1, 2, 3}, {2, 2, 0, 3, 1, 4}}
	m := twod.NewTileMap(ts, twod.Cells(grid), twod.CellSize(32, 32), twod.CellOffset(5, -3), twod.At(40, 70))
	m.SetFlip(1, 0, twod.TileFlipX)
	m.SetFlip(3, 1, twod.TileFlipY)
	area := geom.Rect{X: 50, Y: 60, W: 150, H: 80}
	assert.ElementsMatch(t, []shadow{
		{geom.Rect{X: 45, Y: 67, W: 32, H: 32}, 0, 40}, {geom.Rect{X: 77, Y: 83, W: 32, H: 16}, 2, 32},
		{geom.Rect{X: 113, Y: 75, W: 20, H: 24}, 0, 32}, {geom.Rect{X: 81, Y: 107, W: 20, H: 24}, 0, 32},
		{geom.Rect{X: 109, Y: 115, W: 32, H: 16}, 2, 32}, {geom.Rect{X: 141, Y: 99, W: 32, H: 32}, 0, 40},
		{geom.Rect{X: 173, Y: 115, W: 32, H: 16}, 2, 32}, {geom.Rect{X: 145, Y: 139, W: 20, H: 24}, 0, 32},
		{geom.Rect{X: 173, Y: 131, W: 32, H: 32}, 0, 40},
	}, (&mapCache{}).shadowsIn(nil, m, area), "the answer twod's ShadowsIn gave for the rects touching the area, "+
		"before light read tiles itself: solid cells, a flipped shape, shadow shapes, no body shape, nothing "+
		"from a tile whose shadow is none")
}

func TestAFlippedTilesShadowSpillingPastItsCellIsFoundFromEitherSide(t *testing.T) {
	ts := twod.NewTileset(picture(16, 16), twod.Cell(16, 16), TileShadowShapes(1, geom.Rect{X: 12, W: 12, H: 4}))
	m := twod.NewTileMap(ts, twod.Cells([][]int{{0, 1, 0}}), twod.CellSize(16, 16))
	m.SetFlip(1, 0, twod.TileFlipX)
	left := geom.Rect{X: 0, Y: 0, W: 10, H: 16}
	got := (&mapCache{}).shadowsIn(nil, m, left)
	require.Len(t, got, 1, "mirrored, it spills into the cell on the left, which asks alone")
	assert.Equal(t, geom.Rect{X: 8, W: 12, H: 4}, got[0].rect)
}

func TestAnInfiniteMapsShadowsArePlacedByChunk(t *testing.T) {
	set := twod.NewTileset(picture(16, 16), twod.Cell(16, 16), TileHeight(8, 1), twod.TileValue(1, "solid", true))
	m := twod.NewInfiniteMap(set, twod.CellSize(16, 16), twod.ChunkSize(4),
		twod.Generator(func(cx, cy int, cells [][]int) {
			if cx == -1 && cy == 0 {
				cells[0][3] = 1
			}
		}))
	c := &mapCache{}
	got := c.shadowsIn(nil, m, geom.Rect{X: -64, Y: 0, W: 120, H: 60})
	require.Len(t, got, 1, "cell -1, 0, in chunk -1, 0")
	assert.Equal(t, shadow{geom.Rect{X: -16, W: 16, H: 16}, 0, 8}, got[0], "its own height")
	assert.Len(t, c.chunks, 2, "the chunks the area covers, placed once")
}

func TestLightsTileOptionsAreValuesAnySystemReads(t *testing.T) {
	ts := twod.NewTileset(nil, twod.Cell(16, 16), TileHeight(12, 1), TileOccluderGroup(5, 1), TileShadow(false, 2),
		TileShadowShapes(3, geom.Rect{W: 4, H: 4}))
	h, _ := ts.Value(1, "height")
	g, _ := ts.Value(1, "occluder_group")
	s, _ := ts.Value(2, "shadow")
	assert.InDelta(t, 12, h.Float(), 0)
	assert.Equal(t, 2, g.Int(), "kept within 0–2")
	assert.Equal(t, "none", s.String())
	assert.Equal(t, []twod.TileShape{{Rect: geom.Rect{W: 4, H: 4}, Class: "shadow"}}, ts.Shapes(3))
	assert.Equal(t, caster{rects: []geom.Rect{{W: 4, H: 4}}}, casterOf(ts, 3), "a shadow shape alone casts")
}
