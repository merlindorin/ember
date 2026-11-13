package twod

import "github.com/merlindorin/ember/geom"

// TileShape is a shape drawn on a tile, in Tiled's collision editor or by
// TileShapes: a rectangle in the tile's pixels from its top-left corner,
// and its class, which says what reads it ("shadow" for light, "body" for
// collision, empty for both).
type TileShape struct {
	Rect  geom.Rect `json:"rect"`
	Class string    `json:"class"`
}

// Shapes are tile id's shapes, of every class.
func (t *Tileset) Shapes(id int) []TileShape { return t.shapes[id] }

// Version changes whenever a tile's property or shape does, so a table
// a system builds from the tileset knows to build again.
func (t *Tileset) Version() uint64 { return t.version }

// TileValue sets property name of tile id to v, as Tiled would: what a
// system reads with Value, for a tileset made in code.
func TileValue(id int, name string, v any) TilesetOption {
	return func(t *Tileset) { t.setProperty(id, name, v) }
}

// TileShapes gives tile id shapes of class, rectangles in its pixels from
// its top-left corner, for a tileset made in code.
func TileShapes(id int, class string, rects ...geom.Rect) TilesetOption {
	return func(t *Tileset) {
		for _, r := range rects {
			t.addShape(id, TileShape{r, class})
		}
	}
}

// addShape gives tile id shape s.
func (t *Tileset) addShape(id int, s TileShape) {
	t.shapes[id] = append(t.shapes[id], s)
	t.version++
}

// Place appends rects, in tile id's pixels, to dst as tile id stands on
// the cell at origin, of size cell, flipped by f: scaled to the cell's
// width, a tile taller than the cell rising above it from its foot.
func (t *Tileset) Place(dst []geom.Rect, id int, rects []geom.Rect, origin, cell geom.Vec2, f TileFlip) []geom.Rect {
	return t.place(dst, rects, id, origin, cell, f)
}

// Fill puts the tile ids of the columns×rows cells from x, y in dst, row
// by row, looking each chunk up once: what a system reading a wide area
// of an endless map asks rather than At per cell. dst holds at least
// columns×rows ids.
func (m *InfiniteMap) Fill(x, y, columns, rows int, dst []int) {
	for ry := range rows {
		row := dst[ry*columns : (ry+1)*columns]
		for rx := 0; rx < columns; {
			k, lx, ly := m.split(x+rx, y+ry)
			cells := m.chunk(k).cells[ly]
			rx += copy(row[rx:], cells[lx:])
		}
	}
}

// MaxID is the highest id of a tile the tileset says anything of: a
// picture, a property, a shape or a class. A tile past it is empty, so a
// system's table by id needs no more.
func (t *Tileset) MaxID() int {
	n := t.Count()
	for _, ids := range [][]int{
		keysOf(t.props), keysOf(t.shapes), keysOf(t.classes), keysOf(t.tall), keysOf(t.anims),
	} {
		for _, id := range ids {
			n = max(n, id)
		}
	}
	return n
}

func keysOf[V any](m map[int]V) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// Cached is what build made of the tileset for key: a system's own table
// of its tiles, made once and again only after the tileset's Version
// changed. Safe from several goroutines.
func (t *Tileset) Cached(key any, build func() any) any {
	t.cacheMu.Lock()
	defer t.cacheMu.Unlock()
	if c, ok := t.cached[key]; ok && c.version == t.version {
		return c.value
	}
	if t.cached == nil {
		t.cached = map[any]cachedTable{}
	}
	v := build()
	t.cached[key] = cachedTable{t.version, v}
	return v
}

// cachedTable is a system's table of a tileset, as it was at version.
type cachedTable struct {
	version uint64
	value   any
}

// CellOrigin is the top-left corner of cell x, y in the map's parent's
// space, the top corner of its diamond when isometric: where Place stands
// a tile of the cell.
func (m *TileMap) CellOrigin(x, y int) geom.Vec2 { return m.position.Add(m.origin(x, y)) }

// CellOrigin is the top-left corner of cell x, y in the map's parent's
// space: where Place stands a tile of the cell.
func (m *InfiniteMap) CellOrigin(x, y int) geom.Vec2 { return m.position.Add(m.origin(x, y)) }
