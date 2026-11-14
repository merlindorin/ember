package nav

import (
	"math"

	"github.com/lantervale/ember/geom"
)

// FlowField points every cell of a grid the way to one target, for a
// horde: it is made once from the target and each enemy reads its step
// with Toward, however many there are. It goes round solids the way Find
// does, never cutting a corner, and is made again only when the target
// moves to another cell or the grid changes.
//
//	field := nav.NewFlowField(level)
//	field.Target(hero.Position())
//	if dir, ok := field.Toward(enemy.Position()); ok { … }
type FlowField struct {
	body       geom.Vec2
	grid       Grid
	area       func(at geom.Vec2) Grid
	diagonals  bool
	background bool

	target  geom.Vec2
	goal    int
	version uint64
	made    bool
	// builds counts how many times the field was made.
	builds int

	// shown is the field Toward reads; next is made while shown is read,
	// on another goroutine with Background until done says it is ready.
	shown, next *field
	done        chan struct{}
	// base is the grid's cells as last read, kept while the grid is the
	// same and its version too, so a new goal reads nothing of it.
	base        *field
	baseVersion uint64
}

// field is one made flow field, over the grid as it was then.
type field struct {
	body      geom.Vec2
	grid      Grid
	goal      int
	diagonals bool
	// origin and size are the middle of cell 0, 0 and the step between
	// cells, read from the grid when the field was asked for.
	origin, size geom.Vec2

	// The grid is held padded with a solid border, wide cols+2: cell x, y
	// is at index x+1 + (y+1)*wide, so a neighbour never needs a bounds
	// check.
	cols, rows, wide int
	solid            []bool
	// opaque are the grid's solid cells as they are, which line of sight
	// never crosses; with a body, solid is only those it can't stand in.
	opaque []bool
	// at is where a body stands in a cell its grid's shapes partly fill,
	// for the field's ways; has marks those cells.
	at  []geom.Vec2
	has []bool
	// pass is, for each cell, a bit per step a body can take from where
	// it stands there to where it stands in the next cell.
	pass []uint8
	// seen marks the cells with a clear straight line to the goal: their
	// way is straight at the target.
	seen    []bool
	dist    []int32
	dir     []geom.Vec2
	buckets [15][]int32
	filled  []bool
}

// filler is a grid that reads all its solid cells at once, row by row,
// faster than cell by cell: twod's TileArea.
type filler interface{ FillSolid(dst []bool) }

// Costs are integers, 10 a straight step and 14 a diagonal one, so the
// field is made with a bucket queue rather than a heap.
const (
	straight = 10
	diagonal = 14
	far      = math.MaxInt32
)

// FlowOption sets up a FlowField.
type FlowOption interface{ flowOption(*FlowField) }

type flowOption func(*FlowField)

func (o flowOption) flowOption(f *FlowField) { o(f) }

// Around makes the field cover a window of columns by rows cells of an
// endless grid, cut by area around the target, such as InfiniteMap's
// AreaAround: when the target strays from the window's middle half, a new
// window is cut around it. The grid NewFlowField is given is then unused.
func Around[G Grid](area func(at geom.Vec2, columns, rows int) G, columns, rows int) FlowOption {
	return flowOption(func(f *FlowField) {
		f.area = func(at geom.Vec2) Grid { return area(at, columns, rows) }
	})
}

// Roomy is a grid that knows where in a solid cell a body still fits,
// such as a tile map whose tiles have shapes: Room is the point nearest
// cell x, y's middle where the centre of a body of that size stands
// clear of every shape, false when none does.
//
// Pass is whether such a body walks straight from from to to, its centre
// on the line, clear of every shape.
type Roomy interface {
	Room(x, y int, body geom.Vec2) (geom.Vec2, bool)
	Pass(from, to, body geom.Vec2) bool
}

// Body routes the field for a body w by h points: over a Roomy grid, a
// solid cell its shapes leave room in is passable, its way starting from
// where the body fits (decision 0040). Without it, or over another grid,
// any solid cell blocks the way.
func Body(w, h float32) FlowOption { return flowOption(func(f *FlowField) { f.body = geom.V2(w, h) }) }

// Body is the size the field routes for, zero for whole cells.
func (f *FlowField) Body() geom.Vec2 { return f.body }

// Background makes the field again on another goroutine: SetTarget reads
// the grid, and the next SetTarget waits for the field and shows it, so
// Toward leads by the field made for the call before until then. Only the
// first field is made at once. A game that calls SetTarget once a frame
// keeps the making off its frames and gets the same field every run.
func Background() FlowOption { return flowOption(func(f *FlowField) { f.background = true }) }

// NewFlowField returns a field over grid, without a target yet.
func NewFlowField(grid Grid, opts ...FlowOption) *FlowField {
	f := &FlowField{grid: grid, diagonals: true, goal: -1}
	for _, o := range opts {
		o.flowOption(f)
	}
	return f
}

// Grid is the grid the field covers now.
func (f *FlowField) Grid() Grid { return f.grid }

// Background reports whether the field is made on another goroutine.
func (f *FlowField) Background() bool { return f.background }

// Diagonals reports whether the field steps diagonally.
func (f *FlowField) Diagonals() bool { return f.diagonals }

// Target is where the field leads.
func (f *FlowField) Target() geom.Vec2 { return f.target }

// SetTarget makes the field lead to at, making it again if at is in
// another cell than before, the grid changed, or, with Around, at strayed
// from the window's middle.
func (f *FlowField) SetTarget(at geom.Vec2) {
	f.target = at
	f.wait()
	if f.area != nil && (f.grid == nil || f.stray(at)) {
		f.grid, f.made = f.area(at), false
	}
	if f.grid == nil {
		return
	}
	x, y, ok := f.grid.Cell(at)
	goal := -1
	if ok {
		goal = y*f.grid.Columns() + x
	}
	var version uint64
	if v, isVersioned := f.grid.(versioned); isVersioned {
		version = v.Version()
	}
	if f.made && goal == f.goal && version == f.version {
		return
	}
	f.goal, f.version, f.made = goal, version, true
	f.build()
}

// wait shows the field being made on another goroutine, once it is.
func (f *FlowField) wait() {
	if f.done == nil {
		return
	}
	<-f.done
	f.done = nil
	f.shown, f.next = f.next, f.shown
}

// stray reports whether at is outside the middle half of the window.
func (f *FlowField) stray(at geom.Vec2) bool {
	x, y, ok := f.grid.Cell(at)
	cols, rows := f.grid.Columns(), f.grid.Rows()
	return !ok || x < cols/4 || x >= cols-cols/4 || y < rows/4 || y >= rows-rows/4
}

// build makes the field for the goal: it reads the grid at once, and works
// the field out now, or on another goroutine with Background once one was
// shown.
func (f *FlowField) build() {
	f.builds++
	if f.next == nil {
		f.next = &field{}
	}
	n := f.next
	n.grid, n.goal, n.diagonals, n.body = f.grid, f.goal, f.diagonals, f.body
	n.origin = f.grid.Center(0, 0)
	n.size = f.grid.Center(1, 1).Sub(n.origin)
	if _, kept := f.grid.(versioned); kept && f.base != nil && f.base.grid == f.grid && f.baseVersion == f.version {
		n.adopt(f.base)
	} else {
		n.solids()
		if kept {
			if f.base == nil {
				f.base = &field{}
			}
			f.base.adopt(n)
			f.baseVersion = f.version
		}
	}
	if !f.background || f.shown == nil {
		n.make()
		f.shown, f.next = n, f.shown
		return
	}
	f.done = make(chan struct{})
	go func(done chan struct{}) {
		n.make()
		close(done)
	}(f.done)
}

// make fills the distances from the goal and each cell's way: a Dijkstra
// over the grid with the moves Find allows, its queue a ring of buckets
// by distance.
func (f *field) make() {
	if f.goal < 0 {
		return
	}
	goal := f.index(f.goal%f.cols, f.goal/f.cols)
	if f.solid[goal] {
		return
	}
	moves := f.moves()
	f.spread(goal, moves)
	f.sight(f.goal%f.cols, f.goal/f.cols)
	f.point(goal, moves)
}

// solids sizes the field to its grid, a solid border round it, and reads
// which cells are solid, everything else unreached.
func (f *field) solids() {
	g := f.grid
	f.cols, f.rows = g.Columns(), g.Rows()
	f.wide = f.cols + 2
	n := f.wide * (f.rows + 2)
	if cap(f.solid) < n {
		f.solid, f.seen, f.dist, f.dir = make([]bool, n), make([]bool, n), make([]int32, n), make([]geom.Vec2, n)
		f.opaque, f.at, f.has, f.pass = make([]bool, n), make([]geom.Vec2, n), make([]bool, n), make([]uint8, n)
	}
	f.solid, f.seen, f.dist, f.dir = f.solid[:n], f.seen[:n], f.dist[:n], f.dir[:n]
	f.opaque, f.at, f.has, f.pass = f.opaque[:n], f.at[:n], f.has[:n], f.pass[:n]
	defer f.rooms()
	for i := range n {
		f.solid[i], f.seen[i], f.dist[i], f.dir[i] = true, false, far, geom.Vec2{}
	}
	if fl, ok := g.(filler); ok {
		if cap(f.filled) < f.cols*f.rows {
			f.filled = make([]bool, f.cols*f.rows)
		}
		f.filled = f.filled[:f.cols*f.rows]
		fl.FillSolid(f.filled)
		for y := range f.rows {
			copy(f.solid[f.index(0, y):], f.filled[y*f.cols:(y+1)*f.cols])
		}
		return
	}
	for y := range f.rows {
		for x := range f.cols {
			f.solid[f.index(x, y)] = g.Solid(x, y)
		}
	}
}

// adopt takes src's cells as read from its grid, with nothing of a goal.
func (f *field) adopt(src *field) {
	f.grid, f.body = src.grid, src.body
	f.cols, f.rows, f.wide = src.cols, src.rows, src.wide
	n := len(src.solid)
	if cap(f.solid) < n {
		f.solid, f.seen, f.dist, f.dir = make([]bool, n), make([]bool, n), make([]int32, n), make([]geom.Vec2, n)
		f.opaque, f.at, f.has, f.pass = make([]bool, n), make([]geom.Vec2, n), make([]bool, n), make([]uint8, n)
	}
	f.solid, f.seen, f.dist, f.dir = f.solid[:n], f.seen[:n], f.dist[:n], f.dir[:n]
	f.opaque, f.at, f.has, f.pass = f.opaque[:n], f.at[:n], f.has[:n], f.pass[:n]
	copy(f.solid, src.solid)
	copy(f.opaque, src.opaque)
	copy(f.at, src.at)
	copy(f.has, src.has)
	copy(f.pass, src.pass)
	clear(f.seen)
	clear(f.dir)
	for i := range f.dist {
		f.dist[i] = far
	}
}

// rooms keeps the grid's solid cells as opaque and, for a body over a
// Roomy grid, opens those of them, and the cells beside them, that the
// body still fits in, from where it fits.
func (f *field) rooms() {
	copy(f.opaque, f.solid)
	clear(f.has)
	for i := range f.pass {
		f.pass[i] = 0xff
	}
	r, ok := f.grid.(Roomy)
	if !ok || f.body == (geom.Vec2{}) {
		return
	}
	defer f.passes(r)
	for y := range f.rows {
		for x := range f.cols {
			i := f.index(x, y)
			if !f.opaque[i] && !f.opaque[i-1] && !f.opaque[i+1] && !f.opaque[i-f.wide] && !f.opaque[i+f.wide] &&
				!f.opaque[i-f.wide-1] && !f.opaque[i-f.wide+1] && !f.opaque[i+f.wide-1] && !f.opaque[i+f.wide+1] {
				continue
			}
			at, room := r.Room(x, y, f.body)
			f.solid[i] = !room
			f.at[i], f.has[i] = at, room && (f.opaque[i] || at != f.center(x, y))
		}
	}
}

// passes marks the steps a body can't take between cells it stands in,
// from or into one its grid's shapes partly fill: those whose straight
// way from where it stands to where it stands next meets a shape.
func (f *field) passes(r Roomy) {
	for y := range f.rows {
		for x := range f.cols {
			i := f.index(x, y)
			if f.solid[i] {
				continue
			}
			for b, d := range steps {
				nx, ny := x+d.x, y+d.y
				if !f.free(nx, ny) {
					continue
				}
				if j := f.index(nx, ny); (f.has[i] || f.has[j]) && !r.Pass(f.from(x, y), f.from(nx, ny), f.body) {
					f.pass[i] &^= 1 << b
				}
			}
		}
	}
}

// from is where the field's ways start in cell x, y: where a body fits in
// a cell partly filled, else its middle.
func (f *field) from(x, y int) geom.Vec2 {
	if i := f.index(x, y); f.has[i] {
		return f.at[i]
	}
	return f.center(x, y)
}

// clear is whether cell x, y lets a line of sight through: inside the
// grid and no solid of it.
func (f *field) clear(x, y int) bool {
	return x >= 0 && y >= 0 && x < f.cols && y < f.rows && !f.opaque[f.index(x, y)]
}

// open is whether move m from cell i lands on open ground, past no
// wall's corner.
func (f *field) open(i int, m move) bool {
	return !f.solid[i+m.off] && (m.cost != diagonal || !f.solid[i+m.side1] && !f.solid[i+m.side2]) && f.pass[i]&m.bit != 0
}

// spread works out every cell's walking distance to goal, by buckets of
// distance.
func (f *field) spread(goal int, moves []move) {
	f.dist[goal] = 0
	for i := range f.buckets {
		f.buckets[i] = f.buckets[i][:0]
	}
	f.buckets[0] = append(f.buckets[0], int32(goal)) //nolint:gosec // a cell index: the grids are far smaller
	queued := 1
	for d := int32(0); queued > 0; d++ {
		b := &f.buckets[d%int32(len(f.buckets))]
		for len(*b) > 0 {
			cur := int((*b)[len(*b)-1])
			*b = (*b)[:len(*b)-1]
			queued--
			if f.dist[cur] != d {
				continue
			}
			for _, m := range moves {
				next := cur + m.off
				if !f.open(cur, m) {
					continue
				}
				if nd := d + m.cost; nd < f.dist[next] {
					f.dist[next] = nd
					q := &f.buckets[nd%int32(len(f.buckets))]
					*q = append(*q, int32(next)) //nolint:gosec // a cell index: the grids are far smaller
					queued++
				}
			}
		}
	}
}

// point turns each reached cell but goal toward its nearest neighbour.
func (f *field) point(goal int, moves []move) {
	for y := range f.rows {
		for x := range f.cols {
			i := f.index(x, y)
			if f.solid[i] || f.dist[i] == far || i == goal {
				continue
			}
			best, bestDist := -1, f.dist[i]
			for _, m := range moves {
				if next := i + m.off; f.open(i, m) && f.dist[next] < bestDist {
					best, bestDist = next, f.dist[next]
				}
			}
			if best >= 0 {
				bx, by := best%f.wide-1, best/f.wide-1
				f.dir[i] = f.from(bx, by).Sub(f.from(x, y)).Normalize()
			}
		}
	}
}

// sight marks the cells that see the goal, gx, gy, along a straight line
// that crosses no solid cell nor slips between two diagonal ones. It goes
// out ring by ring: a cell sees the goal when it is free and the cells its
// line enters next towards the goal see it too, the one along its larger
// axis and the diagonal one. At obstacle edges it errs towards unseen, and
// those cells keep the field's way.
func (f *field) sight(gx, gy int) {
	f.seen[f.index(gx, gy)] = true
	sees := func(x, y int) bool { return x == gx && y == gy || f.clear(x, y) && f.seen[f.index(x, y)] }
	for r := 1; r < max(f.cols, f.rows); r++ {
		for y := max(gy-r, 0); y <= min(gy+r, f.rows-1); y++ {
			step := 1
			if y != gy-r && y != gy+r {
				step = 2 * r
			}
			for x := gx - r; x <= gx+r; x += step {
				if x < 0 || x >= f.cols || !f.clear(x, y) {
					continue
				}
				dx, dy := x-gx, y-gy
				sx, sy := intSign(dx), intSign(dy)
				ax, ay := abs(dx), abs(dy)
				var ok bool
				switch {
				case ax > ay:
					ok = sees(x-sx, y) && (ay == 0 || sees(x-sx, y-sy))
				case ay > ax:
					ok = sees(x, y-sy) && (ax == 0 || sees(x-sx, y-sy))
				default:
					ok = sees(x-sx, y-sy) && f.clear(x-sx, y) && f.clear(x, y-sy)
				}
				f.seen[f.index(x, y)] = ok
			}
		}
	}
}

func intSign(v int) int {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	}
	return 0
}

// move is one of the steps a field may take in the padded grid: the
// offset to the cell it reaches, its cost, and for a diagonal the offsets
// of the two cells beside it, which must both be free.
type move struct {
	off, side1, side2 int
	cost              int32
	// bit is the move's in a cell's pass mask.
	bit uint8
}

func (f *field) moves() []move {
	var out []move
	for b, d := range steps {
		if d.diagonal && !f.diagonals {
			continue
		}
		m := move{off: d.x + d.y*f.wide, cost: straight, bit: 1 << b}
		if d.diagonal {
			m.cost, m.side1, m.side2 = diagonal, d.x, d.y*f.wide
		}
		out = append(out, m)
	}
	return out
}

// center is the middle of cell x, y.
func (f *field) center(x, y int) geom.Vec2 {
	return geom.V2(f.origin.X+float32(x)*f.size.X, f.origin.Y+float32(y)*f.size.Y)
}

// index is cell x, y in the padded grid.
func (f *field) index(x, y int) int { return x + 1 + (y+1)*f.wide }

func (f *field) free(x, y int) bool {
	return x >= 0 && y >= 0 && x < f.cols && y < f.rows && !f.solid[f.index(x, y)]
}

// Point is where the field's ways start in the cell under at: where a
// body fits in a cell its grid's shapes partly fill, else its middle; at
// itself outside the field.
func (f *FlowField) Point(at geom.Vec2) geom.Vec2 {
	if f.shown == nil {
		return at
	}
	x, y, ok := f.shown.grid.Cell(at)
	if !ok {
		return at
	}
	return f.shown.from(x, y)
}

// Toward is the way to step from at towards the target, a unit vector
// blended from the ways of the cells around at so a walker turns smoothly;
// in the target's cell, straight at it. ok is false outside the field, on
// a solid cell or where the target can't be reached, and the game then
// goes its own way, often straight at the target.
func (f *FlowField) Toward(at geom.Vec2) (geom.Vec2, bool) {
	if f.shown == nil {
		return geom.Vec2{}, false
	}
	return f.shown.toward(at, f.target)
}

func (f *field) toward(at, target geom.Vec2) (geom.Vec2, bool) {
	x, y, ok := f.grid.Cell(at)
	if !ok {
		return geom.Vec2{}, false
	}
	i := f.index(x, y)
	if f.solid[i] || f.dist[i] == far {
		return geom.Vec2{}, false
	}
	if y*f.cols+x == f.goal || f.seen[i] {
		return target.Sub(at).Normalize(), true
	}
	o, size := f.origin, f.size
	u, v := (at.X-o.X)/size.X, (at.Y-o.Y)/size.Y
	x0, y0 := int(geom.Floor(u)), int(geom.Floor(v))
	fu, fv := u-float32(x0), v-float32(y0)
	var sum geom.Vec2
	for _, c := range [4]struct {
		dx, dy int
		w      float32
	}{{0, 0, (1 - fu) * (1 - fv)}, {1, 0, fu * (1 - fv)}, {0, 1, (1 - fu) * fv}, {1, 1, fu * fv}} {
		cx, cy := x0+c.dx, y0+c.dy
		if !f.free(cx, cy) {
			continue
		}
		j := f.index(cx, cy)
		if cy*f.cols+cx == f.goal || f.seen[j] {
			sum = sum.Add(target.Sub(at).Normalize().Scale(c.w))
		} else {
			sum = sum.Add(f.dir[j].Scale(c.w))
		}
	}
	if sum.LenSq() < 1e-6 {
		return f.dir[i], true
	}
	return sum.Normalize(), true
}
