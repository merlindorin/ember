package twod //nolint:testpackage // white-box: shares twod's test helpers, reaches unexported state

import (
	"encoding/json"
	"image"
	stdcolor "image/color"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/internal/draw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tiles is a 4×2 atlas of 8×8 tiles: 1 floor, 2 wall (solid), 3 tree
// (tall), 4 and 5 water frames, played as tile 4.
func tiles() *Tileset {
	return NewTileset(NewTexture(image.NewRGBA(image.Rect(0, 0, 32, 16))), Cell(8, 8),
		TileValue(2, "solid", true), Tall(3), TileAnimation(4, 2, 4, 5))
}

const room = `
####
#.T#
#..#
####`

//nolint:gochecknoglobals // the tests' fixed values
var legend = map[rune]int{'#': 2, '.': 1, 'T': 3}

func TestAMapIsFilledFromAPictureACSVOrAGrid(t *testing.T) {
	m := NewTileMap(tiles(), ASCII(room, legend))
	columns, rows := m.Columns(), m.Rows()
	assert.Equal(t, [2]int{4, 4}, [2]int{columns, rows})
	assert.Equal(t, 2, m.At(0, 0))
	assert.Equal(t, 3, m.At(2, 1))
	assert.Equal(t, 0, m.At(9, 9), "outside the map is empty")
	csv := NewTileMap(tiles(), CSV("2,2,2,2\n2,1,3,2\n2,1,1,2\n2,2,2,2"))
	assert.Equal(t, m.cells, csv.cells)
	ragged := NewTileMap(tiles(), Cells([][]int{{1}, {1, 1, 1}}))
	assert.Equal(t, []int{1, 0, 0, 1, 1, 1}, ragged.cells, "short rows are padded")
	m.Set(1, 1, 2)
	m.Set(-1, 0, 2)
	assert.Equal(t, 2, m.At(1, 1))
}

func TestCellsAndPointsConvertBothWays(t *testing.T) {
	m := NewTileMap(tiles(), ASCII(room, legend), CellSize(32, 32), At(100, 50))
	assert.Equal(t, geom.V2(100+32+16, 50+64+16), m.Center(1, 2))
	x, y, ok := m.Cell(geom.V2(100+32+16, 50+64+16))
	assert.True(t, ok)
	assert.Equal(t, [2]int{1, 2}, [2]int{x, y})
	_, _, ok = m.Cell(geom.V2(0, 0))
	assert.False(t, ok)

	iso := NewTileMap(tiles(), ASCII(room, legend), CellSize(32, 16), Isometric())
	for _, c := range [][2]int{{0, 0}, {2, 1}, {3, 3}} {
		cx, cy, inside := iso.Cell(iso.Center(c[0], c[1]))
		assert.True(t, inside)
		assert.Equal(t, c, [2]int{cx, cy}, "an isometric cell's centre is in it")
	}
}

func TestTallTilesStandAsSpritesAtTheirFeet(t *testing.T) {
	m := NewTileMap(tiles(), ASCII(room, legend), CellSize(32, 32))
	talls := m.TallSprites()
	require.Len(t, talls, 1)
	assert.Equal(t, geom.V2(2*32+16, 2*32), talls[0].Position(), "bottom centre of its cell")
	assert.Equal(t, geom.V2(4, 4), talls[0].Scale(), "scaled to the cell")
	m.Set(2, 1, 1)
	assert.Empty(t, m.TallSprites(), "replacing a tall tile removes its sprite")

	high := NewTileMap(
		NewTileset(NewTexture(image.NewRGBA(image.Rect(0, 0, 16, 32))), Cell(16, 32), Tall(1)),
		Cells([][]int{{1}}),
		CellSize(32, 32),
	)
	require.Len(t, high.TallSprites(), 1)
	assert.Equal(
		t,
		geom.V2(2, 2),
		high.TallSprites()[0].Scale(),
		"a tile taller than wide keeps its shape and rises above its cell",
	)

	world2 := NewLayer(0, SortByY())
	in := NewTileMap(tiles(), ASCII(room, legend), CellSize(32, 32), At(100, 0), TallIn(world2))
	require.Len(t, in.TallSprites(), 1)
	assert.Equal(
		t,
		world2.GetSceneNode(),
		in.TallSprites()[0].GetSceneNode().ParentNode(),
		"TallIn stands them in another parent",
	)
	assert.Equal(t, geom.V2(100+2*32+16, 2*32), in.TallSprites()[0].Position(), "placed in that parent's space")
}

func TestTheMapDrawsOnlyTheCellsInView(t *testing.T) {
	w, win, c := app()
	var row []int
	for range 1000 {
		row = append(row, 1)
	}
	grid := make([][]int, 1000)
	for i := range grid {
		grid[i] = row
	}
	big := NewTileMap(tiles(), Cells(grid), CellSize(16, 16))
	win.AppendChild(big)
	w.AddEntities(big)
	w.RunFrames(1)
	cells := len(c.Unindexed()) / 6
	assert.LessOrEqual(t, cells, 51*38, "an 800×600 window shows about 50×38 cells, not a million")
	assert.Greater(t, cells, 50*37)
}

func TestAnimatedTilesPlayInStep(t *testing.T) {
	ts := tiles()
	assert.Equal(t, frame(ts, 4, 0), frame(ts, 4, 0.4))
	assert.Equal(t, frame(ts, 5, 0), frame(ts, 4, 0.5), "at 2 fps, the second frame half a second in")
	assert.Equal(t, geom.Rect{X: 8, W: 8, H: 8}, frame(ts, 2, 0))
	assert.Equal(t, 8, ts.Count())
}

func TestTilesetsWithGuttersAndInScenes(t *testing.T) {
	ts := NewTileset(
		NewTexture(image.NewRGBA(image.Rect(0, 0, 20, 11))),
		Cell(8, 8),
		Margin(1),
		Spacing(2, 2),
		TileValue(2, "solid", true),
	)
	assert.Equal(t, geom.Rect{X: 11, Y: 1, W: 8, H: 8}, frame(ts, 2, 0), "margin and spacing")
	ts.setProperty(2, "sound", "stone")
	data, err := json.Marshal(ts)
	require.NoError(t, err)
	var back Tileset
	require.NoError(t, json.Unmarshal(data, &back))
	solid, _ := back.Value(2, "solid")
	assert.True(t, solid.Bool())
	assert.Equal(t, frame(ts, 2, 0), frame(&back, 2, 0))
	v, ok := back.Property(2, "sound")
	assert.True(t, ok)
	assert.Equal(t, "stone", v)
}

func TestShapesSurviveTheTilesetsJSON(t *testing.T) {
	ts := NewTileset(atlas(16, 16), Cell(16, 16), TileShapes(1, "body", geom.Rect{Y: 12, W: 16, H: 4}))
	data, err := json.Marshal(ts)
	require.NoError(t, err)
	var back Tileset
	require.NoError(t, json.Unmarshal(data, &back))
	assert.Equal(t, []TileShape{{geom.Rect{Y: 12, W: 16, H: 4}, "body"}}, back.Shapes(1))
}

func TestEveryTileDrawsWholeStandingOnItsCell(t *testing.T) {
	// Two 16×48 pieces: 1 a wall piece drawn flat, 2 a column standing tall.
	walls := NewTileset(NewTexture(image.NewRGBA(image.Rect(0, 0, 32, 48))), Cell(16, 48), Tall(2))
	for _, m := range []Entity{
		NewTileMap(walls, Cells([][]int{{1, 2}}), CellSize(32, 32)),
		NewInfiniteMap(walls, CellSize(32, 32), Generator(func(cx, cy int, cells [][]int) {
			if cx == 0 && cy == 0 {
				cells[0][0], cells[0][1] = 1, 2
			}
		})),
	} {
		w, win, c := app()
		win.Add(m, NewCamera(At(32, 16)))
		w.AddEntities(m)
		w.RunFrames(2)
		var flat []draw.Vertex
		for i := 0; i+6 <= len(c.Unindexed()); i += 6 {
			if lo, _ := corners(c.Unindexed()[i : i+6]); lo.X == 400-32 && lo.Y == 300-16-64 {
				flat = c.Unindexed()[i : i+6]
			}
		}
		require.NotNil(t, flat, "%T: the flat tile's quad", m)
		lo, hi := corners(flat)
		assert.Equal(
			t,
			geom.V2(32, 96),
			hi.Sub(lo),
			"%T: the whole piece at the map's scale, rising above its cell, never cut",
			m,
		)
		uvLo, uvHi := flat[0].UV, flat[0].UV
		for _, v := range flat {
			uvLo, uvHi = geom.V2(
				min(uvLo.X, v.UV.X),
				min(uvLo.Y, v.UV.Y),
			), geom.V2(
				max(uvHi.X, v.UV.X),
				max(uvHi.Y, v.UV.Y),
			)
		}
		assert.InDelta(t, 0, uvLo.X, 1e-5, "%T: the piece's own column", m)
		assert.InDelta(t, 0.5, uvHi.X, 1e-5)
		assert.InDelta(t, 0, uvLo.Y, 1e-5, "%T: all of its 16×48", m)
		assert.InDelta(t, 1, uvHi.Y, 1e-5)
	}
	m := NewTileMap(walls, Cells([][]int{{1, 2}}), CellSize(32, 32))
	require.Len(t, m.TallSprites(), 1)
	assert.Equal(
		t,
		geom.Rect{X: 16, W: 16, H: 48},
		m.TallSprites()[0].Region(),
		"the tall one stands its whole picture, as before",
	)
}

// pieces is an atlas of three 16×48 pieces: 1 a crate opaque in x 3–12,
// y 24–47; 2 a column opaque all over; 3 a flat tile opaque in x 2–13,
// y 36–47.
func pieces() *Tileset {
	img := image.NewRGBA(image.Rect(0, 0, 48, 48))
	fill := func(x0, y0, x1, y1 int) {
		for y := y0; y <= y1; y++ {
			for x := x0; x <= x1; x++ {
				img.Pix[img.PixOffset(x, y)+3] = 255
			}
		}
	}
	fill(3, 24, 12, 47)
	fill(16, 0, 31, 47)
	fill(32+2, 36, 32+13, 47)
	return NewTileset(NewTexture(img), Cell(16, 48), Tall(1, 2))
}

func TestPictureIsWhatACellReallyShows(t *testing.T) {
	ts := pieces()
	assert.Equal(t, geom.Rect{X: 3, Y: 24, W: 10, H: 24}, ts.Bounds(1), "the crate's opaque pixels")
	assert.Equal(t, geom.Rect{W: 16, H: 48}, ts.Bounds(2))
	grid := map[[2]int]int{{0, 0}: 1, {1, 0}: 2, {2, 0}: 3}
	tileMap := NewTileMap(ts, Cells([][]int{{1, 2, 3, 0}}), CellSize(32, 32), At(100, 50))
	endless := NewInfiniteMap(ts, CellSize(32, 32), At(100, 50), Generator(func(cx, cy int, cells [][]int) {
		if cx == 0 && cy == 0 {
			for k, id := range grid {
				cells[k[1]][k[0]] = id
			}
		}
	}))
	for _, m := range []interface {
		Picture(x, y int) (geom.Rect, bool)
	}{tileMap, endless} {
		crate, ok := m.Picture(0, 0)
		require.True(t, ok)
		assert.Equal(
			t,
			geom.Rect{X: 106, Y: 34, W: 20, H: 48},
			crate,
			"%T: the crate trimmed, standing on its cell's bottom at 82",
			m,
		)
		column, ok := m.Picture(1, 0)
		require.True(t, ok)
		assert.Equal(t, geom.Rect{X: 132, Y: -14, W: 32, H: 96}, column, "%T: a column stands above its cell", m)
		flat, ok := m.Picture(2, 0)
		require.True(t, ok)
		assert.Equal(t, geom.Rect{X: 164 + 4, Y: 50 + 8, W: 24, H: 24}, flat, "%T: a flat tile inside its cell", m)
		_, ok = m.Picture(3, 0)
		assert.False(t, ok, "%T: an empty cell shows nothing", m)
	}
	tall := tileMap.TallSprites()[0]
	assert.Equal(
		t,
		geom.V2(116, 82),
		tileMap.Position().Add(tall.Position()),
		"where standTalls stood the crate: bottom-centre of the cell",
	)
}

// frame is the rectangle tile id of ts shows at time seconds.
func frame(ts *Tileset, id int, time float32) geom.Rect {
	_, r := ts.image(id, time)
	return r
}

func TestATileRisingOrSpreadingIntoViewIsDrawnFromItsCellOutside(t *testing.T) {
	ts := NewTileset(nil, Cell(16, 16))
	ts.tiles[1] = tileImage{atlas(16, 48), geom.Rect{W: 16, H: 48}}
	ts.tiles[2] = tileImage{atlas(48, 16), geom.Rect{W: 48, H: 16}}
	for _, c := range []struct {
		name string
		id   int
		at   geom.Vec2
		want geom.Rect
	}{
		{
			"a pillar standing just below the window rises into it", 1,
			geom.V2(100, 600), geom.Rect{X: 100, Y: 568, W: 16, H: 48},
		},
		{"a beam lying just left of it spreads into it", 2, geom.V2(-32, 100), geom.Rect{X: -32, Y: 100, W: 48, H: 16}},
	} {
		w, win, list := app()
		m := NewTileMap(ts, Cells([][]int{{c.id}}), CellSize(16, 16), At(c.at.X, c.at.Y))
		win.Add(m)
		w.AddEntities(m)
		w.RunFrames(1)
		require.Len(t, list.Unindexed(), 6, c.name)
		lo, hi := corners(list.Unindexed())
		assert.Equal(t, c.want, geom.Rect{X: lo.X, Y: lo.Y, W: hi.X - lo.X, H: hi.Y - lo.Y}, c.name)
	}
}

func TestATilesPictureIsItsTextureAndRegion(t *testing.T) {
	ts := tiles()
	tex, r := ts.Picture(2)
	assert.Same(t, ts.Texture(), tex)
	assert.Equal(t, geom.Rect{X: 8, W: 8, H: 8}, r, "the grid's second cell")
	own := atlas(16, 32)
	ts.tiles[9] = tileImage{own, geom.Rect{X: 0, Y: 0, W: 16, H: 32}}
	tex, r = ts.Picture(9)
	assert.Same(t, own, tex)
	assert.Equal(t, geom.Rect{W: 16, H: 32}, r, "or its own")
}

func TestATallTileSortsByTheFootOfItsArt(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 16, 48))
	for y := 4; y < 39; y++ {
		for x := range 16 {
			img.Set(x, y, stdcolor.White)
		}
	}
	ts := NewTileset(nil, Cell(16, 16), Tall(1))
	ts.tiles[1] = tileImage{NewTexture(img), geom.Rect{W: 16, H: 48}}
	m := NewTileMap(ts, Cells([][]int{{1}}), CellSize(32, 32))
	require.Len(t, m.TallSprites(), 1)
	s := m.TallSprites()[0]
	assert.Equal(
		t,
		geom.V2(16, 32-(48-39)*2),
		s.Position(),
		"placed at its art's foot, 9 pixels above its cell's bottom",
	)
	drawn, _ := m.Picture(0, 0)
	assert.Equal(t, float32(32-(48-39)*2), drawn.Y+drawn.H, "and drawn where it always was")
	assert.Equal(
		t,
		geom.Rect{X: -8, Y: -39, W: 16, H: 48},
		s.Rect(),
		"the whole tile round its pivot at the foot, in pixels before its scale",
	)
}
