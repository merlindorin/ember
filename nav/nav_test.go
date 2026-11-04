package nav //nolint:testpackage // white-box: abs and the A* cells

import (
	"math/rand/v2"
	"testing"

	"github.com/merlindorin/ember/geom"
)

// level is a grid drawn in text, # solid, of 10-point cells.
type level struct {
	cells   [][]byte
	version uint64
}

func newLevel(rows ...string) *level {
	l := &level{}
	for _, r := range rows {
		l.cells = append(l.cells, []byte(r))
	}
	return l
}

func (l *level) Columns() int { return len(l.cells[0]) }
func (l *level) Rows() int    { return len(l.cells) }
func (l *level) Solid(x, y int) bool {
	return x < 0 || y < 0 || y >= len(l.cells) || x >= len(l.cells[0]) || l.cells[y][x] == '#'
}
func (l *level) Cell(p geom.Vec2) (int, int, bool) {
	x, y := int(geom.Floor(p.X/10)), int(geom.Floor(p.Y/10))
	return x, y, x >= 0 && y >= 0 && y < len(l.cells) && x < len(l.cells[0])
}
func (l *level) Center(x, y int) geom.Vec2 { return geom.V2(float32(x)*10+5, float32(y)*10+5) }
func (l *level) Version() uint64           { return l.version }
func (l *level) set(x, y int, c byte)      { l.cells[y][x] = c; l.version++ }

func at(x, y int) geom.Vec2 { return geom.V2(float32(x)*10+5, float32(y)*10+5) }

// walkable checks every leg of a path is clear.
func walkable(t *testing.T, m *Map, from geom.Vec2, path []geom.Vec2) {
	t.Helper()
	for _, p := range path {
		if !m.Clear(from, p) {
			t.Fatalf("the leg %v → %v crosses a wall: %v", from, p, path)
		}
		from = p
	}
}

func TestAnOpenWayIsOneStraightLeg(t *testing.T) {
	m := New(newLevel(
		"..........",
		"..........",
		"..........",
	))
	path, ok := m.Find(at(0, 0), at(9, 2))
	if !ok || len(path) != 1 || path[0] != at(9, 2) {
		t.Fatalf("straight there: %v %v", path, ok)
	}
}

func TestAPathGoesAroundWallsAndIsPulledStraight(t *testing.T) {
	m := New(newLevel(
		"..........",
		"..######..",
		"..#....#..",
		"..#.....#.",
		"..........",
	))
	from, to := at(4, 2), at(4, 0)
	path, ok := m.Find(from, to)
	if !ok || path[len(path)-1] != to {
		t.Fatalf("out of the room and round: %v %v", path, ok)
	}
	walkable(t, m, from, path)
	if len(path) > 4 {
		t.Fatalf("pulled to a few corners, not a cell each: %d waypoints %v", len(path), path)
	}
}

func TestDiagonalsNeverCutACorner(t *testing.T) {
	m := New(newLevel(
		"...",
		".#.",
		"#..",
	))
	if m.Clear(at(0, 1), at(1, 2)) {
		t.Fatal("slipping between two diagonal walls is not clear")
	}
	if m.Clear(at(0, 0), at(2, 2)) {
		t.Fatal("the diagonal through the middle wall is not clear")
	}
	path, ok := m.Find(at(0, 1), at(1, 2))
	if !ok {
		t.Fatal("reachable the long way")
	}
	walkable(t, m, at(0, 1), path)
}

func TestNoDiagonalsMovesInFourDirections(t *testing.T) {
	m := New(newLevel(
		".....",
		".....",
		".....",
	), NoDiagonals())
	cells, ok := m.cells(at(0, 0), at(4, 2))
	if !ok || len(cells) != 7 {
		t.Fatalf("six straight steps: %v", cells)
	}
	for i := 1; i < len(cells); i++ {
		dx, dy := abs(cells[i]%5-cells[i-1]%5), abs(cells[i]/5-cells[i-1]/5)
		if dx+dy != 1 {
			t.Fatalf("a diagonal step: %v", cells)
		}
	}
}

func TestASolidOrUnreachableGoalEndsAsNearAsItCan(t *testing.T) {
	m := New(newLevel(
		".....#...",
		".....#.#.",
		".....#...",
	))
	path, ok := m.Find(at(0, 1), at(7, 1))
	if ok {
		t.Fatal("the goal is a wall")
	}
	if last := path[len(path)-1]; last != at(4, 1) {
		t.Fatalf("it walks up to the wall between: %v", path)
	}
}

func TestARoutePlansAgainWhenADoorOpensOrCloses(t *testing.T) {
	l := newLevel(
		"..........",
		"#########.",
		"..........",
	)
	m := New(l)
	from, to := at(0, 0), at(0, 2)
	route := m.Route(from, to)
	long := len(route.Waypoints())
	if !route.Reaches() || route.Waypoints()[0] != at(9, 0) && long < 2 {
		t.Fatalf("round by the far end: %v", route.Waypoints())
	}

	l.set(1, 1, '.')
	dir := route.Toward(from, 1)
	if w := route.Waypoints(); len(w) != 3 || w[0] != at(1, 0) || w[1] != at(1, 2) || dir != geom.V2(1, 0) {
		t.Fatalf("an opened door is the new way, along to it and through: %v, heading %v", w, dir)
	}
	walkable(t, m, from, route.Waypoints())

	l.set(1, 1, '#')
	route.Toward(from, 1)
	walkable(t, m, from, route.Waypoints())
	if !route.Reaches() || route.Waypoints()[0] == at(1, 0) && len(route.Waypoints()) == 3 {
		t.Fatalf("a closed door sends it round again: %v", route.Waypoints())
	}

	l.set(1, 1, '.')
	route.Toward(from, 1)
	l.cells[1][1] = '#'
	route.Toward(at(1, 0), 1)
	walkable(t, m, at(1, 0), route.Waypoints())
}

func TestARouteIsWalkedWaypointByWaypoint(t *testing.T) {
	m := New(newLevel(
		".....",
		"###..",
		".....",
	))
	route := m.Route(at(0, 0), at(0, 2))
	p := at(0, 0)
	for range 200 {
		if route.Done() {
			break
		}
		p = p.Add(route.Toward(p, 1).Scale(0.5))
	}
	if !route.Done() || p.Dist(at(0, 2)) > 1.5 {
		t.Fatalf("walking it gets there: %v", p)
	}
}

func bigLevel() *level {
	random := rand.New(rand.NewPCG(3, 4))
	rows := make([]string, 128)
	for y := range rows {
		b := make([]byte, 128)
		for x := range b {
			b[x] = '.'
			if random.Float32() < 0.25 && x > 1 && y > 1 && x < 126 && y < 126 {
				b[x] = '#'
			}
		}
		rows[y] = string(b)
	}
	return newLevel(rows...)
}

func BenchmarkWorstPath128(b *testing.B) {
	m := New(bigLevel())
	for b.Loop() {
		m.Find(at(0, 0), at(127, 127))
	}
}

// TestTheWorstPathOn128By128StaysFast guards against a search gone
// quadratic, not a timing: about a millisecond alone, it has a generous
// ceiling so a loaded machine running every package doesn't fail it.
// BenchmarkWorstPath128 measures it.
func TestTheWorstPathOn128By128StaysFast(t *testing.T) {
	if testing.Short() {
		t.Skip("a timing check")
	}
	m := New(bigLevel())
	path, ok := m.Find(at(0, 0), at(127, 127))
	if !ok {
		t.Fatal("the random level is crossable")
	}
	walkable(t, m, at(0, 0), path)
	if r := testing.Benchmark(BenchmarkWorstPath128); r.NsPerOp() > 20_000_000 {
		t.Fatalf("%.2f ms a search", float64(r.NsPerOp())/1e6)
	}
}
