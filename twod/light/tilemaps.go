package light

import (
	"math"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/internal/lit"
	"github.com/lantervale/ember/twod"
)

// tileSource is a map whose tiles block light: a TileMap or an
// InfiniteMap. Light finds them in the World itself, hidden or not: a map
// casts in the window it is in.
type tileSource interface {
	twod.Entity
	Tileset() *twod.Tileset
	CellSize() geom.Vec2
	CellOrigin(x, y int) geom.Vec2
	Version() uint64
	At(x, y int) int
	Position() geom.Vec2
}

// TileShadowRect is a rectangle a tile map's tile blocks light with, in the
// map's parent's space, in an occluder group, standing Height points tall,
// 0 for all the way.
type TileShadowRect struct {
	Rect   geom.Rect
	Group  int
	Height float32
}

// TileShadowsIn appends to dst what m's tiles block light with that
// touches area, in m's parent's space, as light casts it: for a game's
// rules that follow light, such as what blocks sight. It needs no window,
// and reads m as it is, NoShadows or not.
func TileShadowsIn[M *twod.TileMap | *twod.InfiniteMap](dst []TileShadowRect, m M, area geom.Rect) []TileShadowRect {
	src, _ := any(m).(tileSource)
	ts := src.Tileset()
	if ts == nil {
		return dst
	}
	if tiles, ok := src.(*twod.TileMap); ok && tiles.Isometric() {
		return dst
	}
	tb, cell, pos := tableOf(ts), src.CellSize(), src.Position()
	c := mapCache{key: mapKey{cell: cell}, reach: tableOf(ts).overhangAt(ts, cell)}
	corner := pos
	tiles, bounded := src.(*twod.TileMap)
	if bounded {
		corner = pos.Add(tiles.CellOffset())
	}
	x0, y0, x1, y1 := c.span(area, corner)
	if bounded {
		x0, y0, x1, y1 = max(x0, 0), max(y0, 0), min(x1, tiles.Columns()-1), min(y1, tiles.Rows()-1)
	}
	var scratch [8]shadow
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			var f twod.TileFlip
			if bounded {
				f = tiles.Flip(x, y)
			}
			for _, s := range place(scratch[:0], ts, tb, src.At(x, y), src.CellOrigin(x, y), cell, f) {
				if overlaps(s.rect, area) {
					dst = append(dst, TileShadowRect{s.rect, s.group, s.height})
				}
			}
		}
	}
	return dst
}

// fromMap reports whether src, a collide space's solid, is a tile map or
// collision's reading of one (collide/tiles' Layer, marked by
// lit.FromMap): light casts maps itself, from its own reading of their
// tiles.
func fromMap(src any) bool {
	if _, isMap := src.(tileSource); isMap {
		return true
	}
	_, read := src.(interface{ ReadsTileMap() lit.FromMap })
	return read
}

// mapCache is what a map's tiles block light with, placed once in the
// map's own space for each cell, made again when the map, its tileset or
// its cells' size or offset change.
type mapCache struct {
	key mapKey
	// start indexes each cell's first shadow in shadows, by y*columns+x,
	// one past the last at the end; chunks are an InfiniteMap's, by chunk.
	start   []int32
	shadows []shadow
	built   bool
	chunks  map[[2]int]*chunkCache
	reach   overhang
}

// chunkCache is an InfiniteMap chunk's shadows, placed in the map's space.
type chunkCache struct {
	start   []int32
	shadows []shadow
}

type mapKey struct {
	tileset         *twod.Tileset
	tilesetVersion  uint64
	version         uint64
	cell, offset    geom.Vec2
	columns, chunks int
}

// overhang is how far, in points, a tile's shadows reach past its cell,
// across and up or down, either side since tiles flip: a tall tile's
// shapes rise above it.
type overhang struct{ x, y float32 }

// keyOf is m's mapKey now.
func keyOf(m tileSource) mapKey {
	k := mapKey{tileset: m.Tileset(), version: m.Version(), cell: m.CellSize()}
	if k.tileset != nil {
		k.tilesetVersion = k.tileset.Version()
	}
	switch m := m.(type) {
	case *twod.TileMap:
		k.offset, k.columns = m.CellOffset(), m.Columns()
	case *twod.InfiniteMap:
		k.chunks = m.ChunkSize()
	}
	return k
}

// overhangOf is how far ts's tiles block light past cells of size cell,
// flipped either way.
func overhangOf(ts *twod.Tileset, tb *table, cell geom.Vec2) overhang {
	var r overhang
	var scratch []shadow
	for id := range tb.tiles {
		scratch = place(scratch[:0], ts, tb, id, geom.Vec2{}, cell, 0)
		for _, s := range scratch {
			r.x = max(r.x, -s.rect.X, s.rect.X+s.rect.W-cell.X)
			r.y = max(r.y, -s.rect.Y, s.rect.Y+s.rect.H-cell.Y)
		}
	}
	return r
}

// shadowsIn appends the shadows of m's tiles within area, in m's
// parent's space, to dst, cache made again first when it is stale.
func (c *mapCache) shadowsIn(dst []shadow, m tileSource, area geom.Rect) []shadow {
	ts := m.Tileset()
	if ts == nil {
		return dst
	}
	if k := keyOf(m); k != c.key {
		if k.tileset != c.key.tileset || k.tilesetVersion != c.key.tilesetVersion || k.cell != c.key.cell {
			c.reach = tableOf(ts).overhangAt(ts, k.cell)
		}
		c.key, c.built, c.start, c.shadows = k, false, c.start[:0], c.shadows[:0]
		clear(c.chunks)
	}
	pos := m.Position()
	switch m := m.(type) {
	case *twod.TileMap:
		if m.Isometric() {
			return dst
		}
		if !c.built {
			c.start, c.shadows = c.build(c.start, c.shadows, m, ts, 0, 0, m.Columns(), m.Rows())
			c.built = true
		}
		x0, y0, x1, y1 := c.span(area, pos.Add(m.CellOffset()))
		return c.collect(dst, c.start, c.shadows, m.Columns(), pos, area,
			max(x0, 0), max(y0, 0), min(x1, m.Columns()-1), min(y1, m.Rows()-1), 0, 0)
	case *twod.InfiniteMap:
		return c.chunksIn(dst, m, ts, pos, area)
	}
	return dst
}

// span is the cells area covers, grown by the cache's reach, on a grid
// whose cell 0, 0 is at corner.
func (c *mapCache) span(area geom.Rect, corner geom.Vec2) (int, int, int, int) {
	cell := c.key.cell
	f := func(v, size float32) int { return int(math.Floor(float64(v / size))) }
	return f(area.X-corner.X-c.reach.x, cell.X), f(area.Y-corner.Y-c.reach.y, cell.Y),
		f(area.X+area.W-corner.X+c.reach.x, cell.X), f(area.Y+area.H-corner.Y+c.reach.y, cell.Y)
}

// build places the shadows of the columns×rows cells of m from x0, y0 in
// m's own space, onto start and shadows.
func (c *mapCache) build(
	start []int32, shadows []shadow, m tileSource, ts *twod.Tileset, x0, y0, columns, rows int,
) ([]int32, []shadow) {
	tb := tableOf(ts)
	pos, cell := m.Position(), c.key.cell
	tiles, _ := m.(*twod.TileMap)
	for y := y0; y < y0+rows; y++ {
		for x := x0; x < x0+columns; x++ {
			start = append(start, int32(len(shadows))) //nolint:gosec // a map's shadows are far fewer than 2³¹
			var f twod.TileFlip
			if tiles != nil {
				f = tiles.Flip(x, y)
			}
			shadows = place(shadows, ts, tb, m.At(x, y), m.CellOrigin(x, y).Sub(pos), cell, f)
		}
	}
	return append(start, int32(len(shadows))), shadows //nolint:gosec // as above
}

// collect appends the shadows of cells x0..x1, y0..y1, of a block of
// columns cells a row starting at bx, by, that overlap area, moved by pos.
func (c *mapCache) collect(
	dst []shadow, start []int32, shadows []shadow, columns int, pos geom.Vec2, area geom.Rect,
	x0, y0, x1, y1, bx, by int,
) []shadow {
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			i := (y-by)*columns + x - bx
			for _, s := range shadows[start[i]:start[i+1]] {
				s.rect.X, s.rect.Y = s.rect.X+pos.X, s.rect.Y+pos.Y
				if overlaps(s.rect, area) {
					dst = append(dst, s)
				}
			}
		}
	}
	return dst
}

// chunksIn appends the shadows of m's chunks within area, each placed the
// first time asked.
func (c *mapCache) chunksIn(
	dst []shadow, m *twod.InfiniteMap, ts *twod.Tileset, pos geom.Vec2, area geom.Rect,
) []shadow {
	n := m.ChunkSize()
	x0, y0, x1, y1 := c.span(area, pos)
	div := func(v int) int { return int(math.Floor(float64(v) / float64(n))) }
	if c.chunks == nil {
		c.chunks = map[[2]int]*chunkCache{}
	}
	for cy := div(y0); cy <= div(y1); cy++ {
		for cx := div(x0); cx <= div(x1); cx++ {
			ch := c.chunks[[2]int{cx, cy}]
			if ch == nil {
				if len(c.chunks) >= keptChunks {
					clear(c.chunks)
				}
				ch = &chunkCache{}
				ch.start, ch.shadows = c.build(nil, nil, m, ts, cx*n, cy*n, n, n)
				c.chunks[[2]int{cx, cy}] = ch
			}
			dst = c.collect(dst, ch.start, ch.shadows, n, pos, area,
				max(x0, cx*n), max(y0, cy*n), min(x1, cx*n+n-1), min(y1, cy*n+n-1), cx*n, cy*n)
		}
	}
	return dst
}

// keptChunks is how many chunks of an InfiniteMap light keeps placed
// before starting again.
const keptChunks = 256
