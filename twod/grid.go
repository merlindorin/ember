package twod

import (
	"slices"

	"github.com/merlindorin/ember/internal/twodgrid"

	"github.com/merlindorin/ember/geom"
)

// A layer with at least gridFrom children files its still sprites in
// square cells gridCell points wide, so collecting it visits only the
// cells a view shows. Its other children, and its sprites with children
// of their own, are visited every frame as before.
const (
	gridFrom = 128
	gridCell = 256
)

// A grid whose views take in more than half its children for gridWide
// frames in a row costs more than it saves: its layer is visited whole
// for gridRetry frames before it files them again.
const (
	gridWide  = 30
	gridRetry = 120
)

// gridSlack is how far, in points, a filed sprite may be drawn from where
// it was filed: between its last two fixed steps, where it is drawn, and
// where it is now.
const gridSlack = 64

// grid is a layer's children by where they are: the sprites in cells, the
// rest always visited, each with its place among the children so they
// are visited in their order.
type grid struct {
	kin     uint64
	stamp   uint64
	byChild map[Entity]*filed
	cells   map[[2]int32][]*filed
	always  []*filed
	// reach is how far the largest sprite filed draws from the middle it
	// was filed by, so a view looks that much further.
	reach float32
	// dirty are the sprites moved or reshaped since they were filed.
	dirty []*filed
	// seen is the frame's visit, its memory reused.
	seen []*filed
	// wide counts the frames in a row its visit took in most children.
	wide int
}

// filed is a child of a gridded layer: its place among the children, and
// for a sprite, the cell it is filed in and its index there.
type filed struct {
	e     Entity
	order int
	cell  [2]int32
	at    int
	g     *grid
	cells bool
	stamp uint64
}

// refile marks a filed placement moved or reshaped, once, for its grid to
// look at it again before the next collect.
func (p *Placement) refile() {
	if p.changed {
		return
	}
	p.changed = true
	if f := p.filed; f != nil {
		f.g.dirty = append(f.g.dirty, f)
	}
}

// gridded is l's grid, kept in step with its children and their moves,
// or nil when l is visited whole: fixed, or with few children.
func (l *Layer) gridded() *grid {
	n := l.GetSceneNode()
	if l.fixed || twodgrid.Off {
		l.dropGrid()
		return nil
	}
	if l.gridIdle > 0 {
		l.gridIdle--
		return nil
	}
	if g := l.grid; g != nil && g.wide > gridWide {
		l.dropGrid()
		l.gridIdle = gridRetry
		return nil
	}
	if g := l.grid; g != nil && g.kin == n.ChildrenChanged() {
		g.settle()
		return g
	}
	count := 0
	for range n.EachChild() {
		count++
	}
	if count < gridFrom {
		l.dropGrid()
		return nil
	}
	if l.grid == nil {
		l.grid = &grid{cells: map[[2]int32][]*filed{}, byChild: map[Entity]*filed{}}
	}
	l.grid.reconcile(l)
	return l.grid
}

// reconcile brings g in step with l's children: their order, those new
// filed, those gone dropped.
func (g *grid) reconcile(l *Layer) {
	g.settle()
	g.kin = l.GetSceneNode().ChildrenChanged()
	g.stamp++
	g.always = g.always[:0]
	i := 0
	for k := range l.GetSceneNode().EachChild() {
		child, ok := k.Entity().(Entity)
		if !ok {
			continue
		}
		f := g.byChild[child]
		if f == nil {
			f = &filed{e: child, g: g}
			g.byChild[child] = f
			if s, isSprite := child.(*Sprite); isSprite && !hasChildren(s) && g.file(f) {
				s.filed, s.changed = f, false
			}
		}
		f.order, f.stamp = i, g.stamp
		i++
		if !f.cells {
			g.always = append(g.always, f)
		}
	}
	for child, f := range g.byChild {
		if f.stamp != g.stamp {
			g.forget(f)
			delete(g.byChild, child)
		}
	}
}

// forget unfiles f for good.
func (g *grid) forget(f *filed) {
	g.unfile(f)
	if p, ok := f.e.(positioned); ok && p.GetPlacement().filed == f {
		p.GetPlacement().filed = nil
	}
}

// keepVisiting moves f's sprite out of the cells, to be visited every
// frame: it has children of its own now.
func (f *filed) keepVisiting() {
	if !f.cells {
		return
	}
	f.g.unfile(f)
	if s, ok := f.e.(*Sprite); ok {
		s.filed = nil
	}
	f.g.always = append(f.g.always, f)
}

// dropGrid unfiles l's sprites.
func (l *Layer) dropGrid() {
	g := l.grid
	if g == nil {
		return
	}
	for _, f := range g.byChild {
		g.forget(f)
	}
	l.grid = nil
}

// hasChildren is whether e has 2D children, which collect would visit.
func hasChildren(e Entity) bool {
	for k := range e.GetSceneNode().EachChild() {
		if _, ok := k.Entity().(Entity); ok {
			return true
		}
	}
	return false
}

// gridLargest is the widest or tallest sprite filed in cells, in points;
// larger ones are visited every frame.
const gridLargest = 2 * gridCell

// file puts the sprite of f in the cell of its middle, and reports whether
// it fits one.
func (g *grid) file(f *filed) bool {
	s, _ := f.e.(*Sprite) // a grid files only sprites
	b := s.Rect()
	box := transformedBounds(f.e.Local(), b)
	if box.W > gridLargest || box.H > gridLargest {
		return false
	}
	g.reach = max(g.reach, box.W/2, box.H/2)
	mid := geom.V2(box.X+box.W/2, box.Y+box.H/2)
	f.cell = cellOf(mid)
	f.cells = true
	f.at = len(g.cells[f.cell])
	g.cells[f.cell] = append(g.cells[f.cell], f)
	return true
}

// unfile takes f out of its cell.
func (g *grid) unfile(f *filed) {
	if !f.cells {
		return
	}
	list := g.cells[f.cell]
	last := len(list) - 1
	list[f.at] = list[last]
	list[f.at].at = f.at
	list[last] = nil
	if last == 0 {
		delete(g.cells, f.cell)
	} else {
		g.cells[f.cell] = list[:last]
	}
	f.cells = false
}

// settle files again the sprites moved or reshaped since.
func (g *grid) settle() {
	for _, f := range g.dirty {
		s, _ := f.e.(*Sprite) // a grid files only sprites
		s.changed = false
		if s.filed != f {
			continue
		}
		box := transformedBounds(s.Local(), s.Rect())
		if box.W <= gridLargest && box.H <= gridLargest && cellOf(geom.V2(box.X+box.W/2, box.Y+box.H/2)) == f.cell {
			g.reach = max(g.reach, box.W/2, box.H/2)
			continue
		}
		g.unfile(f)
		if !g.file(f) {
			s.filed = nil
			g.always = append(g.always, f)
		}
	}
	clear(g.dirty)
	g.dirty = g.dirty[:0]
}

func cellOf(p geom.Vec2) [2]int32 {
	return [2]int32{int32(geom.Floor(p.X / gridCell)), int32(geom.Floor(p.Y / gridCell))}
}

// transformedBounds is the box round r placed by t.
func transformedBounds(t geom.Transform2D, r geom.Rect) geom.Rect {
	lo, hi := geom.V2(1e30, 1e30), geom.V2(-1e30, -1e30)
	for _, p := range [4]geom.Vec2{
		{X: r.X, Y: r.Y},
		{X: r.X + r.W, Y: r.Y},
		{X: r.X, Y: r.Y + r.H},
		{X: r.X + r.W, Y: r.Y + r.H},
	} {
		q := t.Xform(p)
		lo, hi = geom.V2(min(lo.X, q.X), min(lo.Y, q.Y)), geom.V2(max(hi.X, q.X), max(hi.Y, q.Y))
	}
	return geom.Rect{X: lo.X, Y: lo.Y, W: hi.X - lo.X, H: hi.Y - lo.Y}
}

// visit is the children of g's layer that views showing areas of it, in
// its space, may draw, in their order: the sprites filed in cells round
// the areas and those always visited.
func (g *grid) visit(areas []geom.Rect) []*filed {
	out := g.seen[:0]
	out = append(out, g.always...)
	grow := g.reach + gridSlack
	for i, a := range areas {
		lo := cellOf(geom.V2(a.X-grow, a.Y-grow))
		hi := cellOf(geom.V2(a.X+a.W+grow, a.Y+a.H+grow))
		for y := lo[1]; y <= hi[1]; y++ {
			for x := lo[0]; x <= hi[0]; x++ {
				if covered(areas[:i], x, y, grow) {
					continue
				}
				out = append(out, g.cells[[2]int32{x, y}]...)
			}
		}
	}
	slices.SortFunc(out, func(a, b *filed) int { return a.order - b.order })
	g.seen = out
	if 2*len(out) > len(g.byChild) {
		g.wide++
	} else {
		g.wide = 0
	}
	return out
}

// covered is whether an earlier area already visited cell x, y.
func covered(areas []geom.Rect, x, y int32, grow float32) bool {
	for _, a := range areas {
		lo := cellOf(geom.V2(a.X-grow, a.Y-grow))
		hi := cellOf(geom.V2(a.X+a.W+grow, a.Y+a.H+grow))
		if x >= lo[0] && x <= hi[0] && y >= lo[1] && y <= hi[1] {
			return true
		}
	}
	return false
}
