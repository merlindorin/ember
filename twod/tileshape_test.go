package twod //nolint:testpackage // white-box: shares twod's test helpers

import (
	"encoding/json"
	"image"
	"testing"
	"testing/fstest"

	"github.com/merlindorin/ember/geom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestATilesShapesKeepTheirClass(t *testing.T) {
	tsj := `{"image": "tiles.png", "tilewidth": 16, "tileheight": 16, "tiles": [{"id": 0, "objectgroup": {"objects": [
	  {"x": 0, "y": 8, "width": 16, "height": 8}, {"class": "shadow", "x": 2, "y": 2, "width": 4, "height": 4},
	  {"type": "body", "x": 0, "y": 0, "width": 2, "height": 2}, {"class": "glow", "x": 4, "y": 4, "width": 8,
	  "height": 8}]}}]}`
	ts, err := LoadTiledTileset(fstest.MapFS{"t.tsj": {Data: []byte(tsj)}, "tiles.png": {Data: pngOf(t, 16, 16)}},
		"t.tsj")
	require.NoError(t, err)
	assert.ElementsMatch(t, []TileShape{
		{geom.Rect{Y: 8, W: 16, H: 8}, ""},
		{geom.Rect{X: 2, Y: 2, W: 4, H: 4}, "shadow"},
		{geom.Rect{W: 2, H: 2}, "body"},
		{geom.Rect{X: 4, Y: 4, W: 8, H: 8}, "glow"},
	}, ts.Shapes(1), "a class of the game's own too")

	data, err := json.Marshal(ts)
	require.NoError(t, err)
	var back Tileset
	require.NoError(t, json.Unmarshal(data, &back))
	assert.ElementsMatch(t, ts.Shapes(1), back.Shapes(1), "saved with the tileset")
}

func TestATilesetMadeInCodeTakesValuesAndShapesByName(t *testing.T) {
	ts := NewTileset(nil, Cell(16, 16), TileValue(2, "durability", 7),
		TileShapes(2, "glow", geom.Rect{W: 4, H: 4}), TileShapes(2, "", geom.Rect{W: 16, H: 8}))
	v, ok := ts.Value(2, "durability")
	assert.True(t, ok)
	assert.Equal(t, 7, v.Int())
	assert.Equal(t, []TileShape{{geom.Rect{W: 4, H: 4}, "glow"}, {geom.Rect{W: 16, H: 8}, ""}}, ts.Shapes(2),
		"in the order given")
}

func TestATilesetsVersionChangesWithItsTiles(t *testing.T) {
	ts := NewTileset(nil, Cell(16, 16))
	before := ts.Version()
	TileValue(1, "x", 1)(ts)
	afterValue := ts.Version()
	TileShapes(1, "", geom.Rect{W: 1, H: 1})(ts)
	assert.Greater(t, afterValue, before, "a property")
	assert.Greater(t, ts.Version(), afterValue, "a shape")
}

func TestPlaceStandsATilesRectsOnItsCell(t *testing.T) {
	ts := NewTileset(nil, Cell(16, 16))
	got := ts.Place(nil, 1, []geom.Rect{{X: 0, Y: 8, W: 4, H: 8}}, geom.V2(100, 200), geom.V2(32, 32), TileFlipX)
	assert.Equal(t, []geom.Rect{{X: 124, Y: 216, W: 8, H: 16}}, got, "flipped across, at 2 points a pixel")
}

func TestFillReadsAnEndlessMapsIdsAChunkAtATime(t *testing.T) {
	m := NewInfiniteMap(NewTileset(nil, Cell(16, 16)), ChunkSize(4), Generator(func(cx, cy int, cells [][]int) {
		for y := range cells {
			for x := range cells[y] {
				cells[y][x] = (cx*4+x)*100 + cy*4 + y + 1000
			}
		}
	}))
	ids := make([]int, 6*3)
	m.Fill(-3, 2, 6, 3, ids)
	for y := range 3 {
		for x := range 6 {
			assert.Equal(t, m.At(-3+x, 2+y), ids[y*6+x], "cell %d, %d", -3+x, 2+y)
		}
	}
}

func TestAFlipChangesTheMapsVersion(t *testing.T) {
	m := NewTileMap(NewTileset(nil, Cell(16, 16)), Cells([][]int{{1}}))
	v := m.Version()
	m.SetFlip(0, 0, TileFlipX)
	assert.Greater(t, m.Version(), v, "what's built from the map's cells knows to build again")
}

func TestMaxIDIsTheLastTileTheTilesetSaysAnythingOf(t *testing.T) {
	assert.Equal(t, 0, NewTileset(nil, Cell(16, 16)).MaxID())
	assert.Equal(t, 9, NewTileset(nil, Cell(16, 16), TileShapes(3, "", geom.Rect{W: 1, H: 1}), TileValue(9, "x", 1),
		Tall(4)).MaxID())
	assert.Equal(t, 8, NewTileset(NewTexture(image.NewRGBA(image.Rect(0, 0, 64, 32))), Cell(16, 16)).MaxID(),
		"its atlas's last cell")
}

func TestDeclaredListsWhatOptionsDeclare(t *testing.T) {
	a, b := Property{Name: "a", Kind: KindInt}, Property{Name: "b", Kind: KindBool}
	assert.Equal(t, []Property{a, b}, Declared(TileProperties(a), Classes(Class{Name: "c"}), TileProperties(b)))
}

func TestCachedIsBuiltOnceAVersion(t *testing.T) {
	ts := NewTileset(nil, Cell(16, 16))
	builds := 0
	build := func() any { builds++; return builds }
	assert.Equal(t, 1, ts.Cached("k", build))
	assert.Equal(t, 1, ts.Cached("k", build), "kept")
	TileValue(1, "x", 1)(ts)
	assert.Equal(t, 2, ts.Cached("k", build), "made again once the tileset changed")
}

func TestCellOriginIsInTheMapsParentsSpace(t *testing.T) {
	m := NewTileMap(NewTileset(nil, Cell(16, 16)), Cells([][]int{{1, 1}}), CellSize(32, 32), CellOffset(3, 4),
		At(100, 200))
	assert.Equal(t, geom.V2(135, 204), m.CellOrigin(1, 0), "position, cell offset and the cell's place")
	inf := NewInfiniteMap(NewTileset(nil, Cell(16, 16)), CellSize(32, 32))
	inf.SetPosition(geom.V2(-10, 5))
	assert.Equal(t, geom.V2(-10-64, 5+96), inf.CellOrigin(-2, 3))
}
