package tiles_test

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/lantervale/ember/collide/tiles"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/twod"
)

func endless(ts *twod.Tileset) *twod.InfiniteMap {
	return twod.NewInfiniteMap(ts, twod.ChunkSize(8), twod.KeepChunks(256), twod.CellSize(32, 32),
		twod.At(40, -30), twod.Generator(func(cx, cy int, cells [][]int) {
			for y := range cells {
				for x := range cells[y] {
					cells[y][x] = int((uint(cx*8+x)*73856093 ^ uint(cy*8+y)*19349663) % 6)
				}
			}
		}))
}

// endlessAreas are areas round the map's origin, on both sides of it.
func endlessAreas(seed uint64) []geom.Rect {
	rng := rand.New(rand.NewPCG(seed, 4))
	out := make([]geom.Rect, 300)
	for i := range out {
		out[i] = geom.Rect{
			X: rng.Float32()*2000 - 1000, Y: rng.Float32()*2000 - 1000,
			W: rng.Float32() * 120, H: rng.Float32() * 120,
		}
	}
	return out
}

// endlessReference is what a layer of m lists in area, worked out the slow
// way: the rects of every cell under area and one more round it, as no
// test tile reaches a whole cell past its own, that touch area.
func endlessReference(m *twod.InfiniteMap, area geom.Rect) []geom.Rect {
	x0, y0 := m.Cell(geom.V2(area.X, area.Y))
	x1, y1 := m.Cell(geom.V2(area.X+area.W, area.Y+area.H))
	var out []geom.Rect
	for y := y0 - 1; y <= y1+1; y++ {
		for x := x0 - 1; x <= x1+1; x++ {
			out = placedOn(out, m.Tileset(), m.At(x, y), m.CellOrigin(x, y), m.CellSize(), 0)
		}
	}
	return slices.DeleteFunc(out, func(r geom.Rect) bool { return !r.Overlaps(area) })
}

// sameEndless checks l against the reference.
func sameEndless(t *testing.T, what string, m *twod.InfiniteMap, l *tiles.Layer, areas []geom.Rect) {
	t.Helper()
	var dst []geom.Rect
	for _, a := range areas {
		dst = l.AppendSolidIn(dst[:0], a)
		if want := endlessReference(m, a); !sameSet(dst, want) {
			t.Fatalf("%s: in %v, %v, the reference's %v", what, a, dst, want)
		}
	}
}

// sameSet reports whether a and b hold the same rects as many times each:
// rows of cells across blocks come in another order than the reference's.
func sameSet(a, b []geom.Rect) bool {
	if len(a) != len(b) {
		return false
	}
	left := slices.Clone(b)
	for _, r := range a {
		i := slices.Index(left, r)
		if i < 0 {
			return false
		}
		left = slices.Delete(left, i, i+1)
	}
	return true
}

func TestAnEndlessMapsSolidsAreItsTilesPlaced(t *testing.T) {
	m := endless(newTileset())
	l := tiles.Of(m)
	areas := endlessAreas(1)
	sameEndless(t, "as made", m, l, areas)
	for y := -40; y < 40; y++ {
		for x := -40; x < 40; x++ {
			if l.Solid(x, y) != tiles.Stops(m.Tileset(), m.At(x, y)) {
				t.Fatalf("cell %d,%d: solid %v", x, y, l.Solid(x, y))
			}
		}
	}
	m.Set(2, -3, wall)
	m.Set(-7, 5, spilling)
	sameEndless(t, "after two Sets", m, l, areas)
	m.Reset()
	sameEndless(t, "after a Reset", m, l, areas)
	m.SetPosition(geom.V2(-300, 70))
	sameEndless(t, "moved", m, l, areas)
}

func TestAnAreasCellsAreItsMapsCells(t *testing.T) {
	m := endless(newTileset())
	l := tiles.Of(m)
	made := l.Area(-20, -15, 40, 30)
	check := func(what string) {
		t.Helper()
		if x, y := made.Origin(); made.Columns() != 40 || made.Rows() != 30 || x != -20 || y != -15 {
			t.Fatalf("%s: size or origin", what)
		}
		got := make([]bool, 40*30)
		made.FillSolid(got)
		for y := -1; y <= 30; y++ {
			for x := -1; x <= 40; x++ {
				inside := x >= 0 && y >= 0 && x < 40 && y < 30
				want := !inside || tiles.Stops(m.Tileset(), m.At(x-20, y-15))
				if made.Solid(x, y) != want || inside && got[y*40+x] != want {
					t.Fatalf("%s: cell %d,%d solid %v, want %v", what, x, y, made.Solid(x, y), want)
				}
				if c := made.Center(x, y); c != m.Center(x-20, y-15) {
					t.Fatalf("%s: cell %d,%d's centre %v", what, x, y, c)
				}
				if cx, cy, ok := made.Cell(m.Center(x-20, y-15)); cx != x || cy != y || ok != inside {
					t.Fatalf("%s: the cell under %d,%d's centre is %d,%d %v", what, x, y, cx, cy, ok)
				}
			}
		}
	}
	check("as made")
	v := made.Version()
	m.Set(0, 0, wall)
	if made.Version() == v {
		t.Fatal("a Set changes the area's version, so a field plans again")
	}
	check("after a Set")
	if x, y := l.AreaAround(geom.V2(140, 70), 9, 7).Origin(); x != -1 || y != 0 {
		t.Fatalf("AreaAround centres the area on the cell under the point, 3,3: origin %d,%d", x, y)
	}
}

func TestANewAreaReadsItsCellsWithoutAllocating(t *testing.T) {
	m := twod.NewInfiniteMap(newTileset(), twod.ChunkSize(16), twod.KeepChunks(256), twod.CellSize(32, 32),
		twod.Generator(meadow(1)))
	l := tiles.Of(m)
	dst := make([]bool, 128*128)
	at := [2]geom.Vec2{geom.V2(0, 0), geom.V2(5000, 0)}
	for _, p := range at {
		l.AreaAround(p, 128, 128).FillSolid(dst)
	}
	i := 0
	if n := testing.AllocsPerRun(20, func() {
		l.AreaAround(at[i%2], 128, 128).FillSolid(dst)
		i++
	}); n > 1 {
		t.Fatalf("a field's re-cut allocates its Area alone, not a copy of its tiles: %v allocations", n)
	}
}

// room is an endless map of floor in 32-point cells from 0, 0, but for a
// shaped tile at 1,1, whose shape fills its cell's lower half, a wall
// under it, and a spilling tile at 5,1, whose shape reaches 12 points into
// the cell right of it and 12 up.
func room() *twod.InfiniteMap {
	return twod.NewInfiniteMap(newTileset(), twod.ChunkSize(8), twod.CellSize(32, 32),
		twod.Generator(func(cx, cy int, cells [][]int) {
			for y := range cells {
				for x := range cells[y] {
					cells[y][x] = floor
					switch [2]int{cx*8 + x, cy*8 + y} {
					case [2]int{1, 1}:
						cells[y][x] = shaped
					case [2]int{1, 2}:
						cells[y][x] = wall
					case [2]int{5, 1}:
						cells[y][x] = spilling
					}
				}
			}
		}))
}

func TestAnAreaFindsRoomWhereTilesPartlyFillACell(t *testing.T) {
	a := tiles.Of(room()).Area(0, 0, 8, 4)
	for _, c := range []struct {
		what      string
		x, y      int
		body, at  geom.Vec2
		hasRoomIn bool
	}{
		{"open floor: its middle", 0, 0, geom.V2(8, 8), geom.V2(16, 16), true},
		{"a wall: none", 1, 2, geom.V2(8, 8), geom.V2(48, 80), false},
		{"the shaped cell: the nearest eighth above its shape", 1, 1, geom.V2(8, 8), geom.V2(48, 44), true},
		{"a body too tall for the half left", 1, 1, geom.V2(8, 40), geom.V2(48, 48), false},
		{"beside a spilling tile, a small body fits its middle", 6, 1, geom.V2(8, 8), geom.V2(208, 48), true},
		{"beside a spilling tile, a big one moves down off its shape", 6, 1, geom.V2(20, 20), geom.V2(208, 52), true},
	} {
		at, ok := a.Room(c.x, c.y, c.body)
		if at != c.at || ok != c.hasRoomIn {
			t.Errorf("%s: %v %v, want %v %v", c.what, at, ok, c.at, c.hasRoomIn)
		}
	}
}

func TestAnAreaPassesWhereNoShapeIsInTheWay(t *testing.T) {
	a := tiles.Of(room()).Area(0, 0, 8, 4)
	small := geom.V2(8, 8)
	for _, c := range []struct {
		what     string
		from, to geom.Vec2
		body     geom.Vec2
		passes   bool
	}{
		{"along the open row above", geom.V2(16, 16), geom.V2(80, 16), small, true},
		{"across the shaped cell's middle", geom.V2(16, 48), geom.V2(80, 48), small, false},
		{"across the shaped cell above its shape", geom.V2(16, 40), geom.V2(80, 40), small, true},
		{"into the wall", geom.V2(16, 80), geom.V2(48, 80), small, false},
		{"past the spilling tile's reach, right of it", geom.V2(208, 112), geom.V2(240, 112), small, true},
		{"into the spilling tile's reach", geom.V2(240, 40), geom.V2(200, 40), small, false},
	} {
		if got := a.Pass(c.from, c.to, c.body); got != c.passes {
			t.Errorf("%s: %v, want %v", c.what, got, c.passes)
		}
	}
}
