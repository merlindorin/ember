package tiles_test

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/lantervale/ember/collide"
	"github.com/lantervale/ember/collide/tiles"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/internal/lit"
	"github.com/lantervale/ember/nav"
	"github.com/lantervale/ember/twod"
)

// The tiles of the tests: floor, a wall, a tile with a collision shape, one
// with a shape that stops bodies only, and one whose shape reaches past its
// cell.
const (
	floor, wall, shaped, body, spilling = 1, 2, 3, 4, 5
)

// newTileset is the tests' tiles.
func newTileset() *twod.Tileset {
	return twod.NewTileset(nil, twod.Cell(16, 16), tiles.Solid(wall),
		tiles.Collision(shaped, geom.Rect{X: 0, Y: 8, W: 16, H: 8}),
		tiles.BodyShapes(body, geom.Rect{X: 4, Y: 4, W: 8, H: 8}),
		tiles.Collision(spilling, geom.Rect{X: 10, Y: -6, W: 12, H: 10}))
}

func grid(rng *rand.Rand, columns, rows int) [][]int {
	g := make([][]int, rows)
	for y := range g {
		g[y] = make([]int, columns)
		for x := range g[y] {
			g[y][x] = rng.IntN(6)
		}
	}
	return g
}

func tileMap(ts *twod.Tileset, seed uint64) *twod.TileMap {
	rng := rand.New(rand.NewPCG(seed, 1))
	m := twod.NewTileMap(ts, twod.Cells(grid(rng, 30, 20)), twod.CellSize(32, 32), twod.CellOffset(5, 7),
		twod.At(100, 50))
	for range 40 {
		m.SetFlip(rng.IntN(30), rng.IntN(20), twod.TileFlip(rng.IntN(4)))
	}
	return m
}

// someAreas are areas of every size round and past the map's edges.
func someAreas(seed uint64) []geom.Rect {
	rng := rand.New(rand.NewPCG(seed, 2))
	out := make([]geom.Rect, 300)
	for i := range out {
		out[i] = geom.Rect{
			X: rng.Float32()*1200 - 100, Y: rng.Float32()*800 - 100,
			W: rng.Float32() * 120, H: rng.Float32() * 120,
		}
	}
	return out
}

// placedOn appends the rects tile id stops bodies with on the cell at
// origin of size cell, flipped by f: its shapes, else its whole cell.
func placedOn(dst []geom.Rect, ts *twod.Tileset, id int, origin, cell geom.Vec2, f twod.TileFlip) []geom.Rect {
	if !tiles.Stops(ts, id) {
		return dst
	}
	if rects := tiles.Rects(ts, id); len(rects) > 0 {
		return ts.Place(dst, id, rects, origin, cell, f)
	}
	return append(dst, geom.Rect{X: origin.X, Y: origin.Y, W: cell.X, H: cell.Y})
}

// reference is what a layer of m lists in area, worked out the slow way:
// every cell's rects, row by row, that touch area.
func reference(m *twod.TileMap, area geom.Rect) []geom.Rect {
	if m.Isometric() {
		return nil
	}
	var out []geom.Rect
	for y := range m.Rows() {
		for x := range m.Columns() {
			out = placedOn(out, m.Tileset(), m.At(x, y), m.CellOrigin(x, y), m.CellSize(), m.Flip(x, y))
		}
	}
	return slices.DeleteFunc(out, func(r geom.Rect) bool { return !r.Overlaps(area) })
}

// sameSolids checks that l answers as the reference, rectangle for
// rectangle, in order.
func sameSolids(t *testing.T, what string, m *twod.TileMap, got *tiles.Layer, areas []geom.Rect) {
	t.Helper()
	var dst []geom.Rect
	for _, a := range areas {
		dst = got.AppendSolidIn(dst[:0], a)
		if w := reference(m, a); !slices.Equal(dst, w) {
			t.Fatalf("%s: in %v, %v, the reference's %v", what, a, dst, w)
		}
		if s := got.SolidIn(a); !slices.Equal(s, dst) {
			t.Fatalf("%s: SolidIn and AppendSolidIn differ in %v", what, a)
		}
	}
}

// solidCell is whether cell x, y of m blocks paths: outside the map, or
// its tile stops bodies.
func solidCell(m *twod.TileMap, x, y int) bool {
	return x < 0 || y < 0 || x >= m.Columns() || y >= m.Rows() || tiles.Stops(m.Tileset(), m.At(x, y))
}

func TestAMapsSolidsAreItsTilesPlaced(t *testing.T) {
	m := tileMap(newTileset(), 1)
	l := tiles.Of(m)
	sameSolids(t, "as made", m, l, someAreas(1))
	for y := -1; y <= m.Rows(); y++ {
		for x := -1; x <= m.Columns(); x++ {
			if l.Solid(x, y) != solidCell(m, x, y) {
				t.Fatalf("cell %d,%d: solid %v", x, y, l.Solid(x, y))
			}
		}
	}
}

func TestAFlippedShapeSpillingPastItsCellIsFoundFromTheCellItSpillsInto(t *testing.T) {
	m := twod.NewTileMap(newTileset(), twod.Cells([][]int{{floor, floor}, {floor, spilling}}), twod.CellSize(32, 32))
	m.SetFlip(1, 1, twod.TileFlipX)
	want := []geom.Rect{{X: 20, Y: 20, W: 24, H: 20}}
	if got := tiles.Of(m).SolidIn(geom.Rect{X: 20, Y: 20, W: 4, H: 4}); !slices.Equal(got, want) {
		t.Fatalf("mirrored, the shape reaches into the cells left of and above its own: %v, want %v", got, want)
	}
	if got := tiles.Of(m).SolidIn(geom.Rect{X: 40, Y: 36, W: 2, H: 2}); !slices.Equal(got, want) {
		t.Fatalf("from its own cell: %v, want %v", got, want)
	}
}

func TestASetOrAMoveIsSeenAtOnce(t *testing.T) {
	m := tileMap(newTileset(), 2)
	l := tiles.Of(m)
	areas := someAreas(2)
	sameSolids(t, "as made", m, l, areas)
	before := l.Version()
	m.Set(3, 4, wall)
	m.Set(10, 7, spilling)
	m.Set(0, 0, 0)
	if l.Version() == before {
		t.Fatal("a Set changes the layer's version, so routes plan again")
	}
	sameSolids(t, "after three Sets", m, l, areas)
	m.SetFlip(10, 7, twod.TileFlipX)
	m.SetFlip(5, 5, twod.TileFlipX|twod.TileFlipY)
	sameSolids(t, "after two SetFlips", m, l, areas)
	m.SetPosition(geom.V2(-40, 220))
	sameSolids(t, "moved", m, l, areas)
	m.SetHidden(true)
	sameSolids(t, "hidden: a hidden map still blocks", m, l, areas)
}

func TestATileWithShapesIsSolidWhateverItsProperty(t *testing.T) {
	ts := twod.NewTileset(nil, twod.Cell(16, 16), tiles.BodyShapes(body, geom.Rect{W: 2, H: 2}),
		twod.TileValue(body, tiles.PropSolid, false))
	m := twod.NewTileMap(ts, twod.Cells([][]int{{body}}), twod.CellSize(16, 16))
	if !tiles.Of(m).Solid(0, 0) {
		t.Fatal("a tile with body shapes is solid, its solid property false or not")
	}
}

func TestAppendSolidInAllocatesNothing(t *testing.T) {
	m := tileMap(newTileset(), 4)
	l := tiles.Of(m)
	areas := someAreas(4)
	dst := make([]geom.Rect, 0, 256)
	for _, a := range areas {
		dst = l.AppendSolidIn(dst[:0], a)
	}
	i := 0
	if n := testing.AllocsPerRun(200, func() {
		dst = l.AppendSolidIn(dst[:0], areas[i%len(areas)])
		i++
	}); n != 0 {
		t.Fatalf("%v allocations a query", n)
	}
}

func TestAnIsometricMapStopsNothingButKeepsItsCells(t *testing.T) {
	m := twod.NewTileMap(newTileset(), twod.Cells([][]int{{wall, floor}}), twod.CellSize(32, 16), twod.Isometric())
	l := tiles.Of(m)
	if s := l.SolidIn(geom.Rect{X: -100, Y: -100, W: 400, H: 400}); len(s) != 0 {
		t.Fatalf("an isometric map stops nothing: %v", s)
	}
	if !l.Solid(0, 0) || l.Solid(1, 0) {
		t.Fatal("its cells are solid as their tiles are")
	}
}

// cells is m as a nav grid read the slow way, cell by cell.
type cells struct{ m *twod.TileMap }

func (c cells) Columns() int                      { return c.m.Columns() }
func (c cells) Rows() int                         { return c.m.Rows() }
func (c cells) Solid(x, y int) bool               { return solidCell(c.m, x, y) }
func (c cells) Cell(p geom.Vec2) (int, int, bool) { return c.m.Cell(p) }
func (c cells) Center(x, y int) geom.Vec2         { return c.m.Center(x, y) }

func TestPathsOverTheLayerFollowItsCells(t *testing.T) {
	m := tileMap(newTileset(), 5)
	old, made := nav.New(cells{m}), nav.New(tiles.Of(m))
	rng := rand.New(rand.NewPCG(5, 3))
	for range 50 {
		from := m.Center(rng.IntN(30), rng.IntN(20))
		to := m.Center(rng.IntN(30), rng.IntN(20))
		a, aok := old.Find(from, to)
		b, bok := made.Find(from, to)
		if aok != bok || !slices.Equal(a, b) {
			t.Fatalf("from %v to %v: %v %v, cell by cell %v %v", from, to, b, bok, a, aok)
		}
	}
}

func TestALayerSaysItReadsATileMap(t *testing.T) {
	var solid collide.Solid = tiles.Of(tileMap(newTileset(), 6))
	if _, ok := solid.(interface{ ReadsTileMap() lit.FromMap }); !ok {
		t.Fatal("a layer is marked as reading a tile map, so light skips it in a collide space")
	}
}

func TestPropertiesDeclareSolid(t *testing.T) {
	defs := twod.Declared(tiles.Properties())
	if len(defs) != 1 || defs[0].Name != tiles.PropSolid || defs[0].Kind != twod.KindBool || defs[0].Default != false {
		t.Fatalf("collide/tiles declares solid, a bool, false by default: %+v", defs)
	}
}
