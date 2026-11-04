// Package spatial indexes entity positions on the ground plane for fast
// proximity queries: targeting, aggro, harvesting and network interest
// management all ask "what is near this point?".
package spatial

import (
	"github.com/merlindorin/ember/geom"
)

type cell struct{ x, z int32 }

type entry struct {
	pos  geom.Vec3
	cell cell
}

// Grid is a uniform spatial hash over XZ keyed by entity ID.
type Grid struct {
	size    float32
	cells   map[cell][]uint64
	entries map[uint64]entry
}

func NewGrid(cellSize float32) *Grid {
	if cellSize <= 0 {
		cellSize = 8
	}
	return &Grid{size: cellSize, cells: map[cell][]uint64{}, entries: map[uint64]entry{}}
}

func (g *Grid) cellOf(p geom.Vec3) cell {
	return cell{int32(geom.Floor(p.X / g.size)), int32(geom.Floor(p.Z / g.size))}
}

// Insert adds or moves id to pos.
func (g *Grid) Insert(id uint64, pos geom.Vec3) {
	if _, ok := g.entries[id]; ok {
		g.Move(id, pos)
		return
	}
	c := g.cellOf(pos)
	g.entries[id] = entry{pos, c}
	g.cells[c] = append(g.cells[c], id)
}

// Move updates the position of id, inserting it if unknown.
func (g *Grid) Move(id uint64, pos geom.Vec3) {
	e, ok := g.entries[id]
	if !ok {
		g.Insert(id, pos)
		return
	}
	c := g.cellOf(pos)
	if c != e.cell {
		g.unlink(id, e.cell)
		g.cells[c] = append(g.cells[c], id)
	}
	g.entries[id] = entry{pos, c}
}

func (g *Grid) Remove(id uint64) {
	e, ok := g.entries[id]
	if !ok {
		return
	}
	g.unlink(id, e.cell)
	delete(g.entries, id)
}

func (g *Grid) unlink(id uint64, c cell) {
	ids := g.cells[c]
	for i, x := range ids {
		if x == id {
			ids[i] = ids[len(ids)-1]
			ids = ids[:len(ids)-1]
			break
		}
	}
	if len(ids) == 0 {
		delete(g.cells, c)
	} else {
		g.cells[c] = ids
	}
}

// Position returns the indexed position of id.
func (g *Grid) Position(id uint64) (geom.Vec3, bool) {
	e, ok := g.entries[id]
	return e.pos, ok
}

func (g *Grid) Len() int { return len(g.entries) }

// Query calls fn for every entry within radius of center on the XZ plane.
// Returning false from fn stops the query.
func (g *Grid) Query(center geom.Vec3, radius float32, fn func(id uint64, pos geom.Vec3) bool) {
	lo := g.cellOf(center.Sub(geom.V3(radius, 0, radius)))
	hi := g.cellOf(center.Add(geom.V3(radius, 0, radius)))
	r2 := radius * radius
	for x := lo.x; x <= hi.x; x++ {
		for z := lo.z; z <= hi.z; z++ {
			for _, id := range g.cells[cell{x, z}] {
				p := g.entries[id].pos
				dx, dz := p.X-center.X, p.Z-center.Z
				if dx*dx+dz*dz <= r2 && !fn(id, p) {
					return
				}
			}
		}
	}
}

// Nearest returns the closest entry within radius accepted by filter
// (nil accepts all).
func (g *Grid) Nearest(center geom.Vec3, radius float32, filter func(id uint64) bool) (uint64, bool) {
	var best uint64
	bestD := radius*radius + 1
	found := false
	g.Query(center, radius, func(id uint64, p geom.Vec3) bool {
		if filter != nil && !filter(id) {
			return true
		}
		dx, dz := p.X-center.X, p.Z-center.Z
		if d := dx*dx + dz*dz; d < bestD {
			best, bestD, found = id, d, true
		}
		return true
	})
	return best, found
}

// Collect returns the IDs within radius, appended to dst.
func (g *Grid) Collect(dst []uint64, center geom.Vec3, radius float32) []uint64 {
	g.Query(center, radius, func(id uint64, _ geom.Vec3) bool {
		dst = append(dst, id)
		return true
	})
	return dst
}
