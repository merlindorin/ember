package tiles

import (
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/twod"
)

// tile is what a tile is to collision: solid or not, and the shapes it
// stops bodies with, in its pixels, none for its whole cell.
type tile struct {
	solid bool
	rects []geom.Rect
}

// table is a tileset's tiles by id, read from its properties and shapes the
// first time each is asked for, again once the tileset's Version moves.
type table struct {
	ts      *twod.Tileset
	version uint64
	tiles   []tile
	known   []bool
}

// of is tile id of ts.
func (tb *table) of(ts *twod.Tileset, id int) *tile {
	if tb.ts != ts || tb.version != ts.Version() {
		tb.ts, tb.version = ts, ts.Version()
		clear(tb.known)
	}
	if id >= len(tb.tiles) {
		n := max(id+1, 2*len(tb.tiles), ts.Count()+1)
		tb.tiles = append(tb.tiles, make([]tile, n-len(tb.tiles))...)
		tb.known = append(tb.known, make([]bool, n-len(tb.known))...)
	}
	if !tb.known[id] {
		tb.tiles[id], tb.known[id] = read(ts, id), true
	}
	return &tb.tiles[id]
}

// read is tile id of ts: its shapes of class "" then body, and solid when
// it has any or its solid property is true.
func read(ts *twod.Tileset, id int) tile {
	var t tile
	for _, s := range ts.Shapes(id) {
		if s.Class == "" {
			t.rects = append(t.rects, s.Rect)
		}
	}
	for _, s := range ts.Shapes(id) {
		if s.Class == Body {
			t.rects = append(t.rects, s.Rect)
		}
	}
	v, _ := ts.Value(id, PropSolid)
	t.solid = len(t.rects) > 0 || v.Bool()
	return t
}

// place appends the rects tile id of ts stops bodies with, standing on the
// cell at origin of size cell, flipped by f.
func (tb *table) place(dst []geom.Rect, ts *twod.Tileset, id int, origin, cell geom.Vec2, f twod.TileFlip) []geom.Rect {
	t := tb.of(ts, id)
	switch {
	case !t.solid:
		return dst
	case len(t.rects) == 0:
		return append(dst, geom.Rect{X: origin.X, Y: origin.Y, W: cell.X, H: cell.Y})
	}
	return ts.Place(dst, id, t.rects, origin, cell, f)
}

// solids is which ids of a tileset are solid, as of its version: what a
// path asks of a cell, read without a lock.
type solids struct {
	ts      *twod.Tileset
	version uint64
	solid   []bool
}

// reach is how far the rects placed on cells reach past them, for the
// cells a query must visit.
type reach struct{ left, up, right, down float32 }

// grow widens r to hold rect, placed on the cell at origin of size cell.
func (r *reach) grow(rect geom.Rect, origin, cell geom.Vec2) {
	r.left = max(r.left, origin.X-rect.X)
	r.up = max(r.up, origin.Y-rect.Y)
	r.right = max(r.right, rect.X+rect.W-origin.X-cell.X)
	r.down = max(r.down, rect.Y+rect.H-origin.Y-cell.Y)
}

// span is the cells, x0 to x1 and y0 to y1, whose rects may touch area,
// in the space where cell x, y has its top-left corner at x×cell, y×cell.
func (r reach) span(area geom.Rect, cell geom.Vec2) (int, int, int, int) {
	x0 := int(geom.Floor((area.X - r.right) / cell.X))
	y0 := int(geom.Floor((area.Y - r.down) / cell.Y))
	x1 := int(geom.Floor((area.X + area.W + r.left) / cell.X))
	y1 := int(geom.Floor((area.Y + area.H + r.up) / cell.Y))
	return x0, y0, x1, y1
}
