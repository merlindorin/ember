// Package nav finds paths around the solid cells of a grid, such as a
// twod.TileMap: A* with diagonal moves that never cut a wall's corner, the
// path pulled straight wherever the way is clear, and routes that plan
// again when the grid changes. It is pure Go, for games and servers alike.
//
//	m := nav.New(level)
//	route := m.Route(hero.Position(), click)
//	hero.Translate(route.Toward(hero.Position(), 4).Scale(speed * dt))
package nav

import (
	"math"

	"github.com/lantervale/ember/geom"
)

// Grid is cells, some of them solid, laid out in a plane: a twod.TileMap,
// or a server's own level.
type Grid interface {
	Columns() int
	Rows() int
	// Solid reports whether a cell blocks the way; cells outside count as
	// solid.
	Solid(x, y int) bool
	// Cell is the cell under p.
	Cell(p geom.Vec2) (x, y int, ok bool)
	// Center is the middle of cell x, y.
	Center(x, y int) geom.Vec2
}

// versioned is a grid that says when its cells change.
type versioned interface{ Version() uint64 }

// Map finds paths over a grid, keeping the memory its searches use.
type Map struct {
	grid      Grid
	diagonals bool

	cols, rows int
	g          []float32
	parent     []int32
	stamp      []uint32
	closed     []uint32
	search     uint32
	open       openSet
}

// Option sets up a Map.
type Option interface{ mapOption(*Map) }

// SharedOption sets up a Map or a FlowField.
type SharedOption interface {
	Option
	FlowOption
}

type noDiagonals struct{}

func (noDiagonals) mapOption(m *Map)        { m.diagonals = false }
func (noDiagonals) flowOption(f *FlowField) { f.diagonals = false }

// NoDiagonals keeps paths to the four straight directions.
func NoDiagonals() SharedOption { return noDiagonals{} }

// New returns a map finding paths over grid.
func New(grid Grid, opts ...Option) *Map {
	m := &Map{grid: grid, diagonals: true}
	for _, o := range opts {
		o.mapOption(m)
	}
	return m
}

// Diagonals reports whether paths may move diagonally.
func (m *Map) Diagonals() bool { return m.diagonals }

// Grid is the grid the map finds paths over.
func (m *Map) Grid() Grid { return m.grid }

func (m *Map) fit() {
	cols, rows := m.grid.Columns(), m.grid.Rows()
	if cols == m.cols && rows == m.rows {
		return
	}
	n := cols * rows
	m.cols, m.rows = cols, rows
	m.g, m.parent, m.stamp, m.closed = make([]float32, n), make([]int32, n), make([]uint32, n), make([]uint32, n)
	m.search = 0
}

// Find is the way from from to to, as waypoints after from, the last one
// to; ok is false when to is solid or out of reach, and the way then ends
// at the reachable cell nearest it.
func (m *Map) Find(from, to geom.Vec2) ([]geom.Vec2, bool) {
	cells, reached := m.cells(from, to)
	if len(cells) == 0 {
		return nil, false
	}
	points := make([]geom.Vec2, 0, len(cells)+1)
	points = append(points, from)
	for _, c := range cells[1:] {
		points = append(points, m.grid.Center(c%m.cols, c/m.cols))
	}
	if reached {
		if len(points) > 1 {
			points[len(points)-1] = to
		} else {
			points = append(points, to)
		}
	}
	return m.pull(points), reached
}

// cells is the A* path of cell indexes from from's cell towards to's,
// from's cell first, and whether it reaches to's cell.
func (m *Map) cells(from, to geom.Vec2) ([]int, bool) {
	m.fit()
	sx, sy, ok := m.grid.Cell(from)
	if !ok || m.cols == 0 {
		return nil, false
	}
	tx, ty, _ := m.grid.Cell(to)
	m.search++
	if m.search == 0 {
		clear(m.stamp)
		clear(m.closed)
		m.search = 1
	}
	start, goal := sy*m.cols+sx, ty*m.cols+tx
	goalIn := tx >= 0 && ty >= 0 && tx < m.cols && ty < m.rows && !m.grid.Solid(tx, ty)
	h := func(i int) float32 {
		dx, dy := abs(i%m.cols-tx), abs(i/m.cols-ty)
		if !m.diagonals {
			return float32(dx+dy) * tieBreak
		}
		return (float32(dx+dy) + (math.Sqrt2-2)*float32(min(dx, dy))) * tieBreak
	}
	m.open = m.open[:0]
	m.g[start], m.parent[start], m.stamp[start] = 0, -1, m.search
	m.open.push(node{start, h(start)})
	best, bestH := start, h(start)
	for len(m.open) > 0 {
		cur := m.open.pop().cell
		if m.closed[cur] == m.search {
			continue
		}
		m.closed[cur] = m.search
		if hc := h(cur); hc < bestH {
			best, bestH = cur, hc
		}
		if goalIn && cur == goal {
			best = goal
			break
		}
		cx, cy := cur%m.cols, cur/m.cols
		for _, d := range steps {
			if d.diagonal && !m.diagonals {
				continue
			}
			nx, ny := cx+d.x, cy+d.y
			if m.grid.Solid(nx, ny) || d.diagonal && (m.grid.Solid(cx+d.x, cy) || m.grid.Solid(cx, cy+d.y)) {
				continue
			}
			next := ny*m.cols + nx
			if m.closed[next] == m.search {
				continue
			}
			g := m.g[cur] + d.cost
			if m.stamp[next] == m.search && g >= m.g[next] {
				continue
			}
			//nolint:gosec // a cell index, as parent keeps them
			m.g[next], m.parent[next], m.stamp[next] = g, int32(cur), m.search
			m.open.push(node{next, g + h(next)})
		}
	}
	return m.trace(best), goalIn && best == goal
}

// trace is the path of cell indexes the search took to end, start first.
func (m *Map) trace(end int) []int {
	var out []int
	for c := end; c >= 0; c = int(m.parent[c]) {
		out = append(out, c)
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

// pull keeps only the waypoints the straight way can't skip, going as far
// ahead from each as is clear, and drops points[0], where the walk starts.
func (m *Map) pull(points []geom.Vec2) []geom.Vec2 {
	var out []geom.Vec2
	at := 0
	for at < len(points)-1 {
		next := at + 1
		for next+1 < len(points) && m.Clear(points[at], points[next+1]) {
			next++
		}
		out = append(out, points[next])
		at = next
	}
	return out
}

// Clear reports whether the straight way from a to b crosses no solid
// cell, nor slips between two diagonal ones.
func (m *Map) Clear(a, b geom.Vec2) bool {
	ax, ay, ok := m.grid.Cell(a)
	if !ok {
		return false
	}
	o := m.grid.Center(0, 0)
	size := m.grid.Center(1, 1).Sub(o)
	if size.X == 0 || size.Y == 0 {
		return false
	}
	// u, v are a and b in cells, the centre of cell 0, 0 at 0.5, 0.5.
	ua, va := (a.X-o.X)/size.X+0.5, (a.Y-o.Y)/size.Y+0.5
	ub, vb := (b.X-o.X)/size.X+0.5, (b.Y-o.Y)/size.Y+0.5
	x, y := ax, ay
	bx, by, _ := m.grid.Cell(b)
	du, dv := ub-ua, vb-va
	stepX, stepY := sign(du), sign(dv)
	tMaxX, tMaxY := boundary(ua, du), boundary(va, dv)
	tDeltaX, tDeltaY := inverse(du), inverse(dv)
	for range 2*(m.cols+m.rows) + 4 {
		if m.grid.Solid(x, y) {
			return false
		}
		if x == bx && y == by || min(tMaxX, tMaxY) > 1 {
			return true
		}
		switch {
		case tMaxX < tMaxY-1e-6:
			x += stepX
			tMaxX += tDeltaX
		case tMaxY < tMaxX-1e-6:
			y += stepY
			tMaxY += tDeltaY
		default:
			if m.grid.Solid(x+stepX, y) || m.grid.Solid(x, y+stepY) {
				return false
			}
			x, y = x+stepX, y+stepY
			tMaxX += tDeltaX
			tMaxY += tDeltaY
		}
	}
	return false
}

// boundary is the fraction of the way, d long from u, at which the first
// cell boundary is crossed.
func boundary(u, d float32) float32 {
	switch {
	case d > 0:
		return (float32(math.Floor(float64(u))) + 1 - u) / d
	case d < 0:
		return (u - float32(math.Floor(float64(u)))) / -d
	}
	return float32(math.Inf(1))
}

func inverse(d float32) float32 {
	if d == 0 {
		return float32(math.Inf(1))
	}
	return 1 / geom.Abs(d)
}

func sign(v float32) int {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	}
	return 0
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// tieBreak nudges the heuristic so that among cells as promising, the one
// nearer the goal is opened first, and fewer cells are opened.
const tieBreak = 1 + 1.0/1024

type step struct {
	x, y     int
	cost     float32
	diagonal bool
}

//nolint:gochecknoglobals // a lookup table: the eight moves, read in the search's inner loop
var steps = [...]step{
	{1, 0, 1, false}, {-1, 0, 1, false}, {0, 1, 1, false}, {0, -1, 1, false},
	{1, 1, math.Sqrt2, true}, {1, -1, math.Sqrt2, true}, {-1, 1, math.Sqrt2, true}, {-1, -1, math.Sqrt2, true},
}

type node struct {
	cell int
	f    float32
}

// openSet is a binary min-heap of cells by their f score.
type openSet []node

func (s *openSet) push(n node) {
	*s = append(*s, n)
	h := *s
	for i := len(h) - 1; i > 0; {
		p := (i - 1) / 2
		if h[p].f <= h[i].f {
			break
		}
		h[p], h[i] = h[i], h[p]
		i = p
	}
}

func (s *openSet) pop() node {
	h := *s
	top := h[0]
	last := len(h) - 1
	h[0] = h[last]
	h = h[:last]
	for i := 0; ; {
		l, r, small := 2*i+1, 2*i+2, i
		if l < len(h) && h[l].f < h[small].f {
			small = l
		}
		if r < len(h) && h[r].f < h[small].f {
			small = r
		}
		if small == i {
			break
		}
		h[i], h[small] = h[small], h[i]
		i = small
	}
	*s = h
	return top
}
