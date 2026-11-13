package tiles

import (
	"sync"
	"sync/atomic"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/internal/lit"
	"github.com/merlindorin/ember/twod"
)

// Layer is a tile map as collision and paths see it: a collide.Solid, a
// nav grid over a TileMap's cells, and for an InfiniteMap the Areas a
// flow field follows. It keeps the rects its tiles stop bodies with placed
// in the map's space, redone cell by cell as the map changes.
type Layer struct {
	flat    *twod.TileMap
	endless *twod.InfiniteMap

	mu     sync.Mutex
	tab    table
	solids atomic.Pointer[solids]
	cells  placed
	blocks blocks
	// scratch holds the rects Room and Pass test against.
	scratch []geom.Rect
	// window holds the tiles an Area's FillSolid reads, for every Area.
	window []int
	// rooms are tiles' shapes in their cell, as of roomsVersion.
	rooms        map[roomKey]shaped
	roomsVersion uint64
}

// Of is the layer of a *twod.TileMap or a *twod.InfiniteMap. Keep it, and
// give the same layer to the collide.Space and to nav, so they share its
// tables.
func Of[M *twod.TileMap | *twod.InfiniteMap](m M) *Layer {
	switch m := any(m).(type) {
	case *twod.TileMap:
		return &Layer{flat: m}
	case *twod.InfiniteMap:
		return &Layer{endless: m}
	}
	return nil
}

// ReadsTileMap marks the layer as a solid that reads a tile map's cells:
// light, which finds the maps itself, skips it in a collide.Space so as not
// to cast the map twice. Only engine packages can name lit.FromMap, so no
// game solid matches it by accident.
func (l *Layer) ReadsTileMap() lit.FromMap { return lit.FromMap{} }

// tileset is the map's tileset.
func (l *Layer) tileset() *twod.Tileset {
	if l.flat != nil {
		return l.flat.Tileset()
	}
	return l.endless.Tileset()
}

// at is the tile id in cell x, y.
func (l *Layer) at(x, y int) int {
	if l.flat != nil {
		return l.flat.At(x, y)
	}
	return l.endless.At(x, y)
}

// Version changes whenever a cell or a tile's collision does, so a path
// planned over the layer knows to plan again.
func (l *Layer) Version() uint64 {
	ts := l.tileset()
	var v uint64
	if ts != nil {
		v = ts.Version()
	}
	if l.flat != nil {
		return l.flat.Version() + v
	}
	return l.endless.Version() + v
}

// isSolid reports whether tile id stops bodies, read without a lock from
// the table of solid ids, made again under the lock when the tileset
// changed or holds an id the table doesn't.
func (l *Layer) isSolid(id int) bool {
	if id == 0 {
		return false
	}
	ts := l.tileset()
	if ts == nil {
		return false
	}
	if s := l.solids.Load(); s != nil && s.ts == ts && s.version == ts.Version() && id < len(s.solid) {
		return s.solid[id]
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.solidLocked(ts, id)
}

// solidTable makes the table of solid ids of ts up to id at least, and
// publishes it.
func (l *Layer) solidTable(ts *twod.Tileset, id int) *solids {
	n := max(id+1, ts.Count()+1)
	if old := l.solids.Load(); old != nil && old.ts == ts {
		n = max(n, len(old.solid))
	}
	s := &solids{ts: ts, version: ts.Version(), solid: make([]bool, n)}
	for i := range s.solid {
		s.solid[i] = i != 0 && l.tab.of(ts, i).solid
	}
	l.solids.Store(s)
	return s
}

// solidLocked is isSolid for a caller holding l.mu.
func (l *Layer) solidLocked(ts *twod.Tileset, id int) bool {
	if id == 0 || ts == nil {
		return false
	}
	s := l.solids.Load()
	if s == nil || s.ts != ts || s.version != ts.Version() || id >= len(s.solid) {
		s = l.solidTable(ts, id)
	}
	return s.solid[id]
}

// Solid reports whether cell x, y holds a solid tile, which blocks it for
// paths whatever its shapes; outside a TileMap counts as solid.
func (l *Layer) Solid(x, y int) bool {
	if l.flat != nil && (x < 0 || y < 0 || x >= l.flat.Columns() || y >= l.flat.Rows()) {
		return true
	}
	return l.isSolid(l.at(x, y))
}

// Columns and Rows are a TileMap's size in cells, 0 for an InfiniteMap,
// whose paths go over an Area.
func (l *Layer) Columns() int {
	if l.flat == nil {
		return 0
	}
	return l.flat.Columns()
}

func (l *Layer) Rows() int {
	if l.flat == nil {
		return 0
	}
	return l.flat.Rows()
}

// Cell is the cell under p, in the map's parent's space; ok is false
// outside a TileMap.
func (l *Layer) Cell(p geom.Vec2) (int, int, bool) {
	if l.flat != nil {
		return l.flat.Cell(p)
	}
	x, y := l.endless.Cell(p)
	return x, y, true
}

// Center is the middle of cell x, y in the map's parent's space.
func (l *Layer) Center(x, y int) geom.Vec2 {
	if l.flat != nil {
		return l.flat.Center(x, y)
	}
	return l.endless.Center(x, y)
}

// SolidIn lists the rectangles the map's tiles stop bodies with, whole
// cells or their shapes, that area, in the map's parent's space, touches:
// the layer is a collide.Solid.
func (l *Layer) SolidIn(area geom.Rect) []geom.Rect { return l.AppendSolidIn(nil, area) }

// AppendSolidIn appends to dst what SolidIn lists, allocating nothing once
// the map's rects are placed and dst holds them.
func (l *Layer) AppendSolidIn(dst []geom.Rect, area geom.Rect) []geom.Rect {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.flat != nil {
		return l.cells.appendIn(dst, l, area)
	}
	return l.blocks.appendIn(dst, l, area)
}

// placed is a TileMap's rects, placed in the map's parent's space cell by
// cell, as of the map's version, position, cells' offset and tileset.
type placed struct {
	built                bool
	version, tsVersion   uint64
	ts                   *twod.Tileset
	position, offset     geom.Vec2
	cell                 geom.Vec2
	columns, rows        int
	ids                  []int
	flips                []twod.TileFlip
	start, count, holds  []int
	rects                []geom.Rect
	reach                reach
	isometric, noTileset bool
}

// fresh makes p hold m's rects: all again when the map moved, its cells
// changed size or offset, or its tileset changed; else only the cells
// whose tile or flip changed since.
func (p *placed) fresh(l *Layer) {
	m := l.flat
	ts := m.Tileset()
	switch {
	case ts == nil || m.Isometric():
		p.built, p.isometric, p.noTileset = false, m.Isometric(), ts == nil
		return
	case !p.built || p.ts != ts || p.tsVersion != ts.Version() || p.position != m.Position() ||
		p.offset != m.CellOffset() || p.cell != m.CellSize() || p.columns != m.Columns() || p.rows != m.Rows():
		p.build(l, ts)
	case p.version != m.Version():
		p.redo(l, ts)
	}
	p.isometric, p.noTileset = false, false
}

// build places every cell.
func (p *placed) build(l *Layer, ts *twod.Tileset) {
	m := l.flat
	p.built, p.ts, p.tsVersion, p.version = true, ts, ts.Version(), m.Version()
	p.position, p.offset, p.cell, p.columns, p.rows = m.Position(), m.CellOffset(), m.CellSize(), m.Columns(), m.Rows()
	n := p.columns * p.rows
	p.ids, p.flips = resize(p.ids, n), resize(p.flips, n)
	p.start, p.count, p.holds = resize(p.start, n), resize(p.count, n), resize(p.holds, n)
	p.rects, p.reach = p.rects[:0], reach{}
	for y := range p.rows {
		for x := range p.columns {
			i := y*p.columns + x
			p.ids[i], p.flips[i] = m.At(x, y), m.Flip(x, y)
			start := len(p.rects)
			p.rects = l.tab.place(p.rects, ts, p.ids[i], m.CellOrigin(x, y), p.cell, p.flips[i])
			p.start[i], p.count[i] = start, len(p.rects)-start
			p.holds[i] = p.count[i]
			p.grow(i, m.CellOrigin(x, y))
		}
	}
}

// redo places again the cells whose tile or flip changed: in their own
// room when the new rects fit, else at the end.
func (p *placed) redo(l *Layer, ts *twod.Tileset) {
	m := l.flat
	p.version = m.Version()
	var fresh []geom.Rect
	for y := range p.rows {
		for x := range p.columns {
			i := y*p.columns + x
			id, f := m.At(x, y), m.Flip(x, y)
			if id == p.ids[i] && f == p.flips[i] {
				continue
			}
			p.ids[i], p.flips[i] = id, f
			fresh = l.tab.place(fresh[:0], ts, id, m.CellOrigin(x, y), p.cell, f)
			if len(fresh) > p.holds[i] {
				p.start[i], p.holds[i] = len(p.rects), len(fresh)
				p.rects = append(p.rects, fresh...)
			} else {
				copy(p.rects[p.start[i]:], fresh)
			}
			p.count[i] = len(fresh)
			p.grow(i, m.CellOrigin(x, y))
		}
	}
}

// grow widens the reach to hold cell i's rects.
func (p *placed) grow(i int, origin geom.Vec2) {
	for _, r := range p.rects[p.start[i] : p.start[i]+p.count[i]] {
		p.reach.grow(r, origin, p.cell)
	}
}

// appendIn appends the placed rects that touch area.
func (p *placed) appendIn(dst []geom.Rect, l *Layer, area geom.Rect) []geom.Rect {
	p.fresh(l)
	if !p.built {
		return dst
	}
	local := geom.Rect{X: area.X - p.position.X - p.offset.X, Y: area.Y - p.position.Y - p.offset.Y, W: area.W, H: area.H}
	x0, y0, x1, y1 := p.reach.span(local, p.cell)
	x0, y0, x1, y1 = max(x0, 0), max(y0, 0), min(x1, p.columns-1), min(y1, p.rows-1)
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			i := y*p.columns + x
			for _, r := range p.rects[p.start[i] : p.start[i]+p.count[i]] {
				if r.Overlaps(area) {
					dst = append(dst, r)
				}
			}
		}
	}
	return dst
}

// resize is s with n elements, reusing its memory.
func resize[T any](s []T, n int) []T {
	if cap(s) < n {
		return make([]T, n)
	}
	return s[:n]
}
