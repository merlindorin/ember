package tiles

import (
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/twod"
)

// blockSide is how many cells a side a block of an InfiniteMap's placed
// rects holds.
const blockSide = 16

// mostBlocks is how many blocks a layer keeps before it forgets them all,
// so an endless walk doesn't grow it without end.
const mostBlocks = 1024

// blocks are an InfiniteMap's rects, placed block by block as queries
// reach them, as of the map's position and tileset; a block redoes the
// cells that changed when the map's version moves.
type blocks struct {
	ts        *twod.Tileset
	tsVersion uint64
	position  geom.Vec2
	cell      geom.Vec2
	of        map[[2]int]*block
	reach     reach
	// last is the block get gave last, and its key, read again first.
	last    *block
	lastKey [2]int
}

// block is blockSide by blockSide cells' tiles and rects, from cell x0, y0.
type block struct {
	version             uint64
	x0, y0              int
	ids                 [blockSide * blockSide]int
	start, count, holds [blockSide * blockSide]int
	rects               []geom.Rect
}

// fresh forgets every block when the map moved, its cells changed size,
// or its tileset changed.
func (b *blocks) fresh(l *Layer) bool {
	m := l.endless
	ts := m.Tileset()
	if ts == nil {
		return false
	}
	if b.of == nil || b.ts != ts || b.tsVersion != ts.Version() || b.position != m.Position() ||
		b.cell != m.CellSize() || len(b.of) > mostBlocks {
		b.ts, b.tsVersion, b.position, b.cell = ts, ts.Version(), m.Position(), m.CellSize()
		b.of, b.reach, b.last = map[[2]int]*block{}, reach{}, nil
	}
	return true
}

// get is the block holding cell x, y, made or redone as the map stands.
func (b *blocks) get(l *Layer, x, y int) *block {
	m := l.endless
	k := [2]int{blockOf(x), blockOf(y)}
	if b.last != nil && k == b.lastKey && b.last.version == m.Version() {
		return b.last
	}
	bl, ok := b.of[k]
	switch {
	case !ok:
		bl = &block{x0: k[0] * blockSide, y0: k[1] * blockSide}
		for i := range bl.ids {
			bl.ids[i] = -1
		}
		b.of[k] = bl
		b.redo(l, bl)
	case bl.version != m.Version():
		b.redo(l, bl)
	}
	b.last, b.lastKey = bl, k
	return bl
}

// redo places again the block's cells whose tile changed.
func (b *blocks) redo(l *Layer, bl *block) {
	m := l.endless
	bl.version = m.Version()
	var fresh []geom.Rect
	for cy := range blockSide {
		for cx := range blockSide {
			i := cy*blockSide + cx
			x, y := bl.x0+cx, bl.y0+cy
			id := m.At(x, y)
			if id == bl.ids[i] {
				continue
			}
			bl.ids[i] = id
			origin := m.CellOrigin(x, y)
			fresh = l.tab.place(fresh[:0], b.ts, id, origin, b.cell, 0)
			if len(fresh) > bl.holds[i] {
				bl.start[i], bl.holds[i] = len(bl.rects), len(fresh)
				bl.rects = append(bl.rects, fresh...)
			} else {
				copy(bl.rects[bl.start[i]:], fresh)
			}
			bl.count[i] = len(fresh)
			for _, r := range fresh {
				b.reach.grow(r, origin, b.cell)
			}
		}
	}
}

// appendIn appends the placed rects that touch area, making the blocks it
// covers; the cells it visits widen as blocks show rects reaching further.
func (b *blocks) appendIn(dst []geom.Rect, l *Layer, area geom.Rect) []geom.Rect {
	if !b.fresh(l) {
		return dst
	}
	local := geom.Rect{X: area.X - b.position.X, Y: area.Y - b.position.Y, W: area.W, H: area.H}
	for {
		before := b.reach
		x0, y0, x1, y1 := b.reach.span(local, b.cell)
		for by := blockOf(y0); by <= blockOf(y1); by++ {
			for bx := blockOf(x0); bx <= blockOf(x1); bx++ {
				b.get(l, bx*blockSide, by*blockSide)
			}
		}
		if b.reach != before {
			continue
		}
		for y := y0; y <= y1; y++ {
			for x := x0; x <= x1; {
				bl := b.get(l, x, y)
				row := (y - bl.y0) * blockSide
				for ; x <= x1 && x < bl.x0+blockSide; x++ {
					i := row + x - bl.x0
					for _, r := range bl.rects[bl.start[i] : bl.start[i]+bl.count[i]] {
						if r.Overlaps(area) {
							dst = append(dst, r)
						}
					}
				}
			}
		}
		return dst
	}
}

// blockOf is the block of cell a along one axis: a / blockSide rounded
// down.
func blockOf(a int) int {
	q := a / blockSide
	if a%blockSide != 0 && a < 0 {
		q--
	}
	return q
}
