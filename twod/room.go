package twod

import (
	"slices"
	"sync"

	"github.com/merlindorin/ember/geom"
)

// cellShapes are the rects tile id stops bodies with in a cell of size
// cell, from its top-left corner, read once per tile and cell size, and
// whether one fills the cell and whether any reaches past it.
func (t *Tileset) cellShapes(id int, cell geom.Vec2) shaped {
	t.roomMu.Lock()
	defer t.roomMu.Unlock()
	key := roomKey{id, cell}
	if s, ok := t.roomShapes[key]; ok {
		return s
	}
	if t.roomShapes == nil {
		t.roomShapes = map[roomKey]shaped{}
	}
	s := shaped{rects: t.solids(nil, id, geom.Vec2{}, cell, 0)}
	for _, r := range s.rects {
		s.full = s.full || r.X <= 0 && r.Y <= 0 && r.X+r.W >= cell.X && r.Y+r.H >= cell.Y
		s.spills = s.spills || r.X < 0 || r.Y < 0 || r.X+r.W > cell.X || r.Y+r.H > cell.Y
	}
	t.roomShapes[key] = s
	return s
}

// shaped is a tile's shapes in its cell.
type shaped struct {
	rects        []geom.Rect
	full, spills bool
}

type roomKey struct {
	id   int
	cell geom.Vec2
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

//nolint:gochecknoglobals // a cache every room shares, under its mutex
var (
	samples   = map[geom.Vec2][]geom.Vec2{}
	samplesMu sync.Mutex
)

// Room is the point of the area's cell x, y nearest its middle where a
// body of size body, centred there, overlaps none of the shapes of the
// cell's tile and its neighbours', false where there is none: what
// routing over the map asks of a cell its tiles partly fill (nav.Roomy).
func (a *TileArea) Room(x, y int, body geom.Vec2) (geom.Vec2, bool) {
	m := a.m
	mx, my := a.x+x, a.y+y
	if m.tileset == nil {
		return m.Center(mx, my), true
	}
	cell := m.tileSize
	if own := a.idAt(mx, my); own != 0 && m.tileset.cellShapes(own, cell).full {
		return m.Center(mx, my), false
	}
	m.roomScratch = m.roomScratch[:0]
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			id := a.idAt(mx+dx, my+dy)
			if id == 0 {
				continue
			}
			s := m.tileset.cellShapes(id, cell)
			if (dx != 0 || dy != 0) && !s.spills {
				continue
			}
			for _, r := range s.rects {
				r.X, r.Y = r.X+float32(dx)*cell.X, r.Y+float32(dy)*cell.Y
				m.roomScratch = append(m.roomScratch, r)
			}
		}
	}
	corner := m.position.Add(m.origin(mx, my))
	if len(m.roomScratch) == 0 {
		return m.Center(mx, my), true
	}
	for _, p := range roomSamples(cell) {
		b := geom.Rect{X: p.X - body.X/2, Y: p.Y - body.Y/2, W: body.X, H: body.Y}.Inset(0.01)
		free := true
		for _, r := range m.roomScratch {
			if b.Overlaps(r) {
				free = false
				break
			}
		}
		if free {
			return corner.Add(p), true
		}
	}
	return m.Center(mx, my), false
}

// Pass reports whether a body of size body walks straight from from to
// to, its centre on the line, overlapping no shape of the tiles round
// either end: what routing asks of a step between cells partly filled.
func (a *TileArea) Pass(from, to, body geom.Vec2) bool {
	m := a.m
	if m.tileset == nil {
		return true
	}
	cell := m.tileSize
	m.roomScratch = m.roomScratch[:0]
	ends := [2]geom.Vec2{from, to}
	var seen [18][2]int
	n := 0
	for _, p := range ends {
		cx, cy := m.Cell(p)
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				k := [2]int{cx + dx, cy + dy}
				dup := false
				for _, s := range seen[:n] {
					dup = dup || s == k
				}
				if dup {
					continue
				}
				seen[n] = k
				n++
				id := a.idAt(k[0], k[1])
				if id == 0 {
					continue
				}
				corner := m.position.Add(m.origin(k[0], k[1]))
				for _, r := range m.tileset.cellShapes(id, cell).rects {
					r.X, r.Y = r.X+corner.X, r.Y+corner.Y
					m.roomScratch = append(m.roomScratch, r)
				}
			}
		}
	}
	if len(m.roomScratch) == 0 {
		return true
	}
	steps := max(int(to.Sub(from).Len()/(min(body.X, body.Y)/2)), 1)
	for i := 0; i <= steps; i++ {
		c := from.Lerp(to, float32(i)/float32(steps))
		b := geom.Rect{X: c.X - body.X/2, Y: c.Y - body.Y/2, W: body.X, H: body.Y}.Inset(0.01)
		for _, r := range m.roomScratch {
			if b.Overlaps(r) {
				return false
			}
		}
	}
	return true
}

// idAt is the tile at map cell x, y, read from the area's copy of its
// tiles and a border of two cells round it, kept until the map changes.
func (a *TileArea) idAt(x, y int) int {
	m := a.m
	w, h := a.columns+2*idBorder, a.rows+2*idBorder
	if !a.idsFilled || a.idsVersion != m.Version() {
		if cap(a.ids) < w*h {
			a.ids = make([]int, w*h)
		}
		a.ids = a.ids[:w*h]
		for ry := range h {
			for rx := 0; rx < w; {
				k, lx, ly := m.split(a.x-idBorder+rx, a.y-idBorder+ry)
				cells := m.chunk(k).cells[ly]
				for ; lx < m.chunkSize && rx < w; lx, rx = lx+1, rx+1 {
					a.ids[ry*w+rx] = cells[lx]
				}
			}
		}
		a.idsFilled, a.idsVersion = true, m.Version()
	}
	lx, ly := x-a.x+idBorder, y-a.y+idBorder
	if lx < 0 || ly < 0 || lx >= w || ly >= h {
		return m.At(x, y)
	}
	return a.ids[ly*w+lx]
}

// idBorder is how many cells round an area idAt keeps.
const idBorder = 2
