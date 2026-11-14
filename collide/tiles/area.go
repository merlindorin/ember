package tiles

import (
	"slices"
	"sync"

	"github.com/lantervale/ember/geom"
)

// Area is a window of an InfiniteMap layer's cells, columns by rows from
// cell x, y, numbered from its own top-left: a finite grid nav finds paths
// over, and knows where in a solid cell a body still fits (nav.Roomy).
type Area struct {
	l                   *Layer
	x, y, columns, rows int
	// ids are the area's tiles and a border round it, as of idsVersion.
	ids        []int
	idsVersion uint64
	idsFilled  bool
}

// Area is the layer's window of columns by rows cells from cell x, y.
func (l *Layer) Area(x, y, columns, rows int) *Area {
	return &Area{l: l, x: x, y: y, columns: columns, rows: rows}
}

// AreaAround is the Area of columns by rows cells centred on the cell
// under at: what a nav.FlowField follows the hero with, through
// nav.Around(layer.AreaAround, columns, rows).
func (l *Layer) AreaAround(at geom.Vec2, columns, rows int) *Area {
	x, y, _ := l.Cell(at)
	return l.Area(x-columns/2, y-rows/2, columns, rows)
}

func (a *Area) Columns() int    { return a.columns }
func (a *Area) Rows() int       { return a.rows }
func (a *Area) Version() uint64 { return a.l.Version() }

// Origin is the map cell of the area's cell 0, 0.
func (a *Area) Origin() (int, int) { return a.x, a.y }

// Solid reports whether the area's cell x, y is solid; outside the area
// counts as solid.
func (a *Area) Solid(x, y int) bool {
	if x < 0 || y < 0 || x >= a.columns || y >= a.rows {
		return true
	}
	return a.l.Solid(a.x+x, a.y+y)
}

// FillSolid sets dst, columns by rows row by row, to which of the area's
// cells are solid, reading its tiles into a buffer the layer keeps, so a
// new area a field re-cuts to copies nothing.
func (a *Area) FillSolid(dst []bool) {
	l := a.l
	l.mu.Lock()
	defer l.mu.Unlock()
	n := a.columns * a.rows
	l.window = resize(l.window, n)
	l.fill(a.x, a.y, a.columns, a.rows, l.window)
	ts := l.tileset()
	for i, id := range l.window {
		dst[i] = l.solidLocked(ts, id)
	}
}

// Cell is the area's cell under p.
func (a *Area) Cell(p geom.Vec2) (int, int, bool) {
	mx, my, _ := a.l.Cell(p)
	x, y := mx-a.x, my-a.y
	return x, y, x >= 0 && y >= 0 && x < a.columns && y < a.rows
}

// Center is the middle of the area's cell x, y.
func (a *Area) Center(x, y int) geom.Vec2 { return a.l.Center(a.x+x, a.y+y) }

// idBorder is how many cells round an area idAt keeps.
const idBorder = 2

// idAt is the tile at map cell x, y, read from the area's copy of its
// tiles and a border round it, made when Room or Pass first asks and kept
// until the map changes.
func (a *Area) idAt(x, y int) int {
	l := a.l
	w, h := a.columns+2*idBorder, a.rows+2*idBorder
	if v := l.Version(); !a.idsFilled || a.idsVersion != v {
		a.ids = resize(a.ids, w*h)
		l.fill(a.x-idBorder, a.y-idBorder, w, h, a.ids)
		a.idsFilled, a.idsVersion = true, v
	}
	lx, ly := x-a.x+idBorder, y-a.y+idBorder
	if lx < 0 || ly < 0 || lx >= w || ly >= h {
		return l.at(x, y)
	}
	return a.ids[ly*w+lx]
}

// fill sets dst, columns by rows row by row, to the tiles from cell x, y,
// an InfiniteMap's read chunk by chunk.
func (l *Layer) fill(x, y, columns, rows int, dst []int) {
	if l.endless != nil {
		l.endless.Fill(x, y, columns, rows, dst)
		return
	}
	for ry := range rows {
		for rx := range columns {
			dst[ry*columns+rx] = l.at(x+rx, y+ry)
		}
	}
}

// shaped is a tile's rects in a cell from its top-left corner, and whether
// one fills the cell and whether any reaches past it.
type shaped struct {
	rects        []geom.Rect
	full, spills bool
}

// roomKey is a tile in a cell of a size.
type roomKey struct {
	id   int
	cell geom.Vec2
}

// cellShapes is tile id's shaped in a cell of size cell, read once per tile
// and cell size until the tileset changes; the caller holds l.mu.
func (l *Layer) cellShapes(id int, cell geom.Vec2) shaped {
	ts := l.tileset()
	if l.rooms == nil || l.roomsVersion != ts.Version() {
		l.rooms, l.roomsVersion = map[roomKey]shaped{}, ts.Version()
	}
	if s, ok := l.rooms[roomKey{id, cell}]; ok {
		return s
	}
	s := shaped{rects: l.tab.place(nil, ts, id, geom.Vec2{}, cell, 0)}
	for _, r := range s.rects {
		s.full = s.full || r.X <= 0 && r.Y <= 0 && r.X+r.W >= cell.X && r.Y+r.H >= cell.Y
		s.spills = s.spills || r.X < 0 || r.Y < 0 || r.X+r.W > cell.X || r.Y+r.H > cell.Y
	}
	l.rooms[roomKey{id, cell}] = s
	return s
}

// Room is the point of the area's cell x, y nearest its middle where a
// body of size body, centred there, overlaps none of the shapes of the
// cell's tile and its neighbours', false where there is none: what
// routing over the map asks of a cell its tiles partly fill (nav.Roomy).
func (a *Area) Room(x, y int, body geom.Vec2) (geom.Vec2, bool) {
	l := a.l
	mx, my := a.x+x, a.y+y
	if l.tileset() == nil {
		return l.Center(mx, my), true
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	cell := l.cellSize()
	if own := a.idAt(mx, my); own != 0 && l.cellShapes(own, cell).full {
		return l.Center(mx, my), false
	}
	l.scratch = l.scratch[:0]
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			id := a.idAt(mx+dx, my+dy)
			if id == 0 {
				continue
			}
			s := l.cellShapes(id, cell)
			if (dx != 0 || dy != 0) && !s.spills {
				continue
			}
			for _, r := range s.rects {
				r.X, r.Y = r.X+float32(dx)*cell.X, r.Y+float32(dy)*cell.Y
				l.scratch = append(l.scratch, r)
			}
		}
	}
	if len(l.scratch) == 0 {
		return l.Center(mx, my), true
	}
	corner := l.cellOrigin(mx, my)
	for _, p := range roomSamples(cell) {
		b := geom.Rect{X: p.X - body.X/2, Y: p.Y - body.Y/2, W: body.X, H: body.Y}.Inset(0.01)
		free := true
		for _, r := range l.scratch {
			if b.Overlaps(r) {
				free = false
				break
			}
		}
		if free {
			return corner.Add(p), true
		}
	}
	return l.Center(mx, my), false
}

// Pass reports whether a body of size body walks straight from from to
// to, its centre on the line, overlapping no shape of the tiles round
// either end: what routing asks of a step between cells partly filled.
func (a *Area) Pass(from, to, body geom.Vec2) bool {
	l := a.l
	if l.tileset() == nil {
		return true
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	cell := l.cellSize()
	l.scratch = l.scratch[:0]
	var seen [18][2]int
	n := 0
	for _, p := range [2]geom.Vec2{from, to} {
		cx, cy, _ := l.Cell(p)
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				k := [2]int{cx + dx, cy + dy}
				if slices.Contains(seen[:n], k) {
					continue
				}
				seen[n] = k
				n++
				id := a.idAt(k[0], k[1])
				if id == 0 {
					continue
				}
				corner := l.cellOrigin(k[0], k[1])
				for _, r := range l.cellShapes(id, cell).rects {
					r.X, r.Y = r.X+corner.X, r.Y+corner.Y
					l.scratch = append(l.scratch, r)
				}
			}
		}
	}
	if len(l.scratch) == 0 {
		return true
	}
	steps := max(int(to.Sub(from).Len()/(min(body.X, body.Y)/2)), 1)
	for i := 0; i <= steps; i++ {
		c := from.Lerp(to, float32(i)/float32(steps))
		b := geom.Rect{X: c.X - body.X/2, Y: c.Y - body.Y/2, W: body.X, H: body.Y}.Inset(0.01)
		for _, r := range l.scratch {
			if b.Overlaps(r) {
				return false
			}
		}
	}
	return true
}

// cellSize is the map's cells' size.
func (l *Layer) cellSize() geom.Vec2 {
	if l.flat != nil {
		return l.flat.CellSize()
	}
	return l.endless.CellSize()
}

// cellOrigin is the top-left corner of cell x, y in the map's parent's
// space.
func (l *Layer) cellOrigin(x, y int) geom.Vec2 {
	if l.flat != nil {
		return l.flat.CellOrigin(x, y)
	}
	return l.endless.CellOrigin(x, y)
}

// roomSamples are the points of a cell of size cell a body's centre is
// tried at, nearest its middle first, an eighth of the cell apart.
func roomSamples(cell geom.Vec2) []geom.Vec2 {
	samplesMu.Lock()
	defer samplesMu.Unlock()
	if s, ok := samples[cell]; ok {
		return s
	}
	var out []geom.Vec2
	for y := float32(0); y <= 8; y++ {
		for x := float32(0); x <= 8; x++ {
			out = append(out, geom.V2(cell.X*x/8, cell.Y*y/8))
		}
	}
	mid := cell.Scale(0.5)
	slices.SortStableFunc(out, func(a, b geom.Vec2) int {
		da, db := a.Sub(mid).LenSq(), b.Sub(mid).LenSq()
		switch {
		case da < db:
			return -1
		case da > db:
			return 1
		}
		return 0
	})
	samples[cell] = out
	return out
}

//nolint:gochecknoglobals // a cache every area shares, under its mutex
var (
	samples   = map[geom.Vec2][]geom.Vec2{}
	samplesMu sync.Mutex
)
