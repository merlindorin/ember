package collide

import (
	"math"

	"github.com/merlindorin/ember/geom"
)

// capacity is how much of a crowd cell soft bodies may cover: a soft body
// moves into a cell only while it has room, and an over-full cell spreads.
const capacity = 1

// maxNudge is the most a spread pushes a body in one step, as a share of
// its smaller side, so a dense pile opens over a few steps instead of
// bursting.
const maxNudge = 0.25

// margin is how many cells the crowd's grid reaches past its bodies, for
// the moves until the next rebuild.
const margin = 4

// crowd files the soft bodies, rebuilt once a step rather than kept up to
// date move by move: each body once, in the cell of its centre, in one
// array sorted by cell, row after row. Each cell also counts how much of
// it soft bodies cover, kept up to date as they move. Bodies made soft
// since the rebuild wait in loose, and slack is how far any filed body has
// moved since, so a query widened by it misses none.
type crowd struct {
	cell       float32
	x0, y0     int32
	cols, rows int32
	half       float32
	starts     []int32
	entries    []*Body
	boxes      []geom.Rect
	density    []float32
	over       []float32
	slopes     []geom.Vec2
	centres    []geom.Vec2
	loose      []*Body
	slack      float32
}

func (c *crowd) cellOf(p geom.Vec2) (int32, int32) {
	return int32(math.Floor(float64(p.X / c.cell))), int32(math.Floor(float64(p.Y / c.cell)))
}

// slotOf is the cell p is in, -1 off the grid.
func (c *crowd) slotOf(p geom.Vec2) int32 {
	cx, cy := c.cellOf(p)
	cx, cy = cx-c.x0, cy-c.y0
	if cx < 0 || cy < 0 || cx >= c.cols || cy >= c.rows {
		return -1
	}
	return cy*c.cols + cx
}

// rebuild files every soft body of s where it is now, in the order s
// lists them. Cells are twice the largest soft body's side, larger when
// the bodies are spread too far apart for a grid that size.
func (c *crowd) rebuild(s *Space) {
	n := 0
	var lo, hi geom.Vec2
	c.half = 0
	for _, b := range s.bodies {
		b.indexed = b.soft() && b.owner != nil
		if !b.indexed {
			b.slot = -1
			continue
		}
		b.box = b.Bounds()
		at := b.box.Center()
		if n == 0 {
			lo, hi = at, at
		}
		lo, hi = geom.V2(min(lo.X, at.X), min(lo.Y, at.Y)), geom.V2(max(hi.X, at.X), max(hi.Y, at.Y))
		c.half = max(c.half, b.box.W/2, b.box.H/2)
		n++
	}
	c.cell = max(4*c.half, 1)
	for {
		x0, y0 := c.cellOf(lo)
		x1, y1 := c.cellOf(hi)
		c.x0, c.y0 = x0-margin, y0-margin
		c.cols, c.rows = x1-x0+1+2*margin, y1-y0+1+2*margin
		if n == 0 || int(c.cols)*int(c.rows) <= 4*n+4096 {
			break
		}
		c.cell *= 2
	}
	cells := 0
	if n > 0 {
		cells = int(c.cols) * int(c.rows)
	}
	c.starts = resized(c.starts, cells+1)
	c.density = resized(c.density, cells)
	c.over = resized(c.over, cells)
	c.slopes = resized(c.slopes, cells)
	c.centres = resized(c.centres, cells)
	c.entries = resized(c.entries, n)
	c.boxes = resized(c.boxes, n)
	c.file(s)
	for _, b := range c.loose {
		b.waiting = false
	}
	clear(c.loose)
	c.loose, c.slack = c.loose[:0], 0
}

// file files the indexed bodies of s in the cells of their centres, in
// entries sorted by cell, and counts each cell's density and centre.
func (c *crowd) file(s *Space) {
	area := c.cell * c.cell
	for _, b := range s.bodies {
		if b.indexed {
			b.slot = c.slotOf(b.box.Center())
			b.share = b.area() / area
			c.starts[b.slot+1]++
			c.density[b.slot] += b.share
			c.centres[b.slot] = c.centres[b.slot].Add(b.box.Center())
		}
	}
	for i := range c.centres {
		if n := c.starts[i+1]; n > 0 {
			c.centres[i] = c.centres[i].Scale(1 / float32(n))
		}
	}
	for i := 1; i < len(c.starts); i++ {
		c.starts[i] += c.starts[i-1]
	}
	for _, b := range s.bodies {
		if b.indexed {
			i := c.starts[b.slot]
			c.entries[i], c.boxes[i] = b, b.box
			c.starts[b.slot]++
		}
	}
	for i := len(c.starts) - 1; i > 0; i-- {
		c.starts[i] = c.starts[i-1]
	}
	if len(c.starts) > 0 {
		c.starts[0] = 0
	}
}

// moved notes that b, filed, now covers box: it widens the queries' slack
// and moves b's share to the cell it is in now.
func (c *crowd) moved(b *Body, box geom.Rect) {
	c.slack = max(c.slack, geom.Abs(box.X-b.box.X), geom.Abs(box.Y-b.box.Y))
	if box.W != b.box.W || box.H != b.box.H {
		c.slack = max(c.slack, geom.Abs(box.X+box.W-b.box.X-b.box.W), geom.Abs(box.Y+box.H-b.box.Y-b.box.H))
	}
	if slot := c.slotOf(box.Center()); slot != b.slot {
		c.leave(b)
		if b.slot = slot; slot >= 0 {
			c.density[slot] += b.share
		}
	}
}

// leave takes b's share out of its cell.
func (c *crowd) leave(b *Body) {
	if b.slot >= 0 {
		c.density[b.slot] -= b.share
	}
	b.slot = -1
}

// room is the part of delta that keeps b, filed, out of cells that would
// then cover more than limit: a move along an axis into such a cell is
// dropped.
func (c *crowd) room(b *Body, delta geom.Vec2, limit float32) geom.Vec2 {
	at := b.Bounds().Center()
	full := func(d geom.Vec2) bool {
		slot := c.slotOf(at.Add(d))
		return slot >= 0 && slot != b.slot && c.density[slot]+b.share > limit
	}
	if delta.X != 0 && full(geom.V2(delta.X, 0)) {
		delta.X = 0
	}
	if delta.Y != 0 && full(geom.V2(0, delta.Y)) {
		delta.Y = 0
	}
	if delta != (geom.Vec2{}) && full(delta) {
		delta = geom.Vec2{}
	}
	return delta
}

// collect appends to out the soft bodies whose bounds touch area.
func (c *crowd) collect(out []*Body, area geom.Rect) []*Body {
	if len(c.entries) > 0 {
		pad := c.half + c.slack
		wide := geom.Rect{X: area.X - c.slack, Y: area.Y - c.slack, W: area.W + 2*c.slack, H: area.H + 2*c.slack}
		cx0, cy0 := c.cellOf(geom.V2(area.X-pad, area.Y-pad))
		cx1, cy1 := c.cellOf(geom.V2(area.X+area.W+pad, area.Y+area.H+pad))
		cx0, cy0 = max(cx0-c.x0, 0), max(cy0-c.y0, 0)
		cx1, cy1 = min(cx1-c.x0, c.cols-1), min(cy1-c.y0, c.rows-1)
		for cy := cy0; cy <= cy1 && cx0 <= cx1; cy++ {
			row := cy * c.cols
			from, to := c.starts[row+cx0], c.starts[row+cx1+1]
			for i := from; i < to; i++ {
				b := c.entries[i]
				if touches(wide, c.boxes[i]) && b.indexed && (c.slack == 0 || touches(area, b.Bounds())) {
					out = append(out, b)
				}
			}
		}
	}
	for _, b := range c.loose {
		if b.waiting && !b.indexed && b.soft() && touches(area, b.Bounds()) {
			out = append(out, b)
		}
	}
	return out
}

// spread files the crowd again and pushes each soft body down the slope
// of its cells' overflow, where they cover more than capacity, and out
// from the middle of an over-full cell, sliding each push against solids
// and bodies that aren't soft. A push enters a cell with room, or one less
// full than the cell it leaves, so overflow trapped among full cells
// still drains to the crowd's edge. Bodies are pushed
// in the order they are filed, from the overflow as the step found it, so
// the same steps always give the same positions.
func (s *Space) spread() {
	c := &s.crowd
	c.rebuild(s)
	spilling := false
	for i, d := range c.density {
		c.over[i] = max(d-capacity, 0)
		spilling = spilling || d > capacity
	}
	if !spilling {
		return
	}
	at := func(x, y int32) float32 {
		if x < 0 || y < 0 || x >= c.cols || y >= c.rows {
			return 0
		}
		return c.over[y*c.cols+x]
	}
	for y := range c.rows {
		for x := range c.cols {
			c.slopes[y*c.cols+x] = geom.V2(at(x+1, y)-at(x-1, y), at(x, y+1)-at(x, y-1)).Scale(0.5)
		}
	}
	for _, b := range c.entries {
		if !b.indexed || b.owner == nil {
			continue
		}
		at := b.Bounds().Center()
		dir := c.slope(at).Scale(-1)
		if b.slot >= 0 && c.over[b.slot] > 0 {
			over := c.over[b.slot]
			out := at.Sub(c.centres[b.slot])
			if out.LenSq() < 1e-6 {
				out = geom.FromAngle(float32(b.ID()) * 2.39996)
			}
			dir = dir.Add(out.Normalize().Scale(over / 2))
		}
		push := dir.Scale(c.cell * b.push)
		if push == (geom.Vec2{}) {
			continue
		}
		if most := maxNudge * min(b.shape.size.X, b.shape.size.Y); push.LenSq() > most*most {
			push = push.Normalize().Scale(most)
		}
		limit := float32(capacity)
		if b.slot >= 0 {
			limit = max(limit, c.density[b.slot]-b.share)
		}
		push = c.room(b, push, limit)
		start := b.owner.Position()
		to, _ := s.slide(b, start, push, false)
		b.owner.SetPosition(to)
		s.file(b)
	}
}

// slope is how the overflow rises at p, per cell, interpolated between the
// slopes at the centres of the cells around it.
func (c *crowd) slope(p geom.Vec2) geom.Vec2 {
	u, v := p.X/c.cell-0.5-float32(c.x0), p.Y/c.cell-0.5-float32(c.y0)
	i, j := int32(math.Floor(float64(u))), int32(math.Floor(float64(v)))
	fx, fy := u-float32(i), v-float32(j)
	at := func(x, y int32) geom.Vec2 {
		if x < 0 || y < 0 || x >= c.cols || y >= c.rows {
			return geom.Vec2{}
		}
		return c.slopes[y*c.cols+x]
	}
	return at(i, j).Scale((1 - fx) * (1 - fy)).Add(at(i+1, j).Scale(fx * (1 - fy))).
		Add(at(i, j+1).Scale((1 - fx) * fy)).Add(at(i+1, j+1).Scale(fx * fy))
}

func resized[T any](s []T, n int) []T {
	if cap(s) < n {
		return make([]T, n)
	}
	s = s[:n]
	clear(s)
	return s
}
