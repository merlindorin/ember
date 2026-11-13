package collide

import (
	"math"
	"slices"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/world"
)

// Solid is anything with solid rectangles, such as a tile map's walls.
type Solid interface {
	SolidIn(area geom.Rect) []geom.Rect
}

// SolidAppender is a Solid that appends its rectangles to a slice it is
// given: the space asks it so, and a move allocates nothing for it.
type SolidAppender interface {
	Solid
	AppendSolidIn(dst []geom.Rect, area geom.Rect) []geom.Rect
}

// Hit is something a move or a ray met: a body, or a solid's rectangle.
type Hit struct {
	// Body is what was met, nil for a solid's rectangle.
	Body *Body
	// Rect is the solid's rectangle met, or the body's bounds.
	Rect geom.Rect
	// Normal points out of what was met, towards the mover.
	Normal geom.Vec2
	// Time is the fraction of the move done when it was met.
	Time float32
	// Trigger is set for a trigger the mover overlaps after its move.
	Trigger bool
}

// Space holds the bodies and solids things move among. The game makes
// one, keeps it, and adds NewTracker(space) to its World.
type Space struct {
	cell float32
	// hard files the bodies that aren't soft, move by move, and crowd the
	// soft ones, once a step: a soft body's move looks only at hard ones.
	hard     hash
	crowd    crowd
	owners   map[uint64]*Body
	solids   []Solid
	bodies   []*Body
	triggers []*Body
	// step counts the fixed steps the Tracker closed, so it knows the
	// bodies a move has filed since.
	step uint64
	// found is near's result, obstacles gather's and rects a solid's,
	// reused.
	found     []*Body
	obstacles []obstacle
	rects     []geom.Rect
}

// cellRange is the grid cells a rectangle covers, from x0, y0 to x1, y1.
type cellRange struct {
	x0, y0, x1, y1 int32
	ok             bool
}

// SpaceOption sets up a Space.
type SpaceOption func(*Space)

// Grid is the side of the cells bodies are filed in (default 64): about
// the size of the bodies.
func Grid(size float32) SpaceOption {
	return func(s *Space) {
		if size > 0 {
			s.cell = size
		}
	}
}

// Grid is the side of the cells bodies are filed in.
func (s *Space) Grid() float32 { return s.cell }

// NewSpace returns an empty space.
func NewSpace(opts ...SpaceOption) *Space {
	s := &Space{cell: 64, hard: newHash(), owners: map[uint64]*Body{}}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// AddSolid makes solid's rectangles stop moving bodies.
func (s *Space) AddSolid(solid Solid) { s.solids = append(s.solids, solid) }

// SolidSources are the solids added with AddSolid, such as tile maps, for
// what reads more of them than their rectangles: the light their shadows
// cast.
func (s *Space) SolidSources() []Solid { return s.solids }

// BodyOf is the body of owner, nil without one.
func (s *Space) BodyOf(owner world.Identifier) *Body { return s.owners[owner.ID()] }

func (s *Space) cells(r geom.Rect) cellRange {
	cell := func(v float32) int32 { return int32(math.Floor(float64(v / s.cell))) }
	return cellRange{cell(r.X), cell(r.Y), cell(r.X + r.W), cell(r.Y + r.H), true}
}

// file puts b in the cells its bounds cover, unless it is in them already.
func (s *Space) file(b *Body) {
	if b.owner == nil {
		s.unfile(b)
		s.leaveCrowd(b)
		return
	}
	if !b.listed {
		s.owners[b.owner.ID()] = b
		b.listed = true
		s.bodies = append(s.bodies, b)
		if b.trigger {
			s.triggers = append(s.triggers, b)
		}
	}
	box := b.Bounds()
	if b.soft() {
		s.unfile(b)
		switch {
		case b.indexed:
			s.crowd.moved(b, box)
		case !b.waiting:
			b.waiting = true
			s.crowd.loose = append(s.crowd.loose, b)
		}
		return
	}
	s.leaveCrowd(b)
	if box == b.box && b.filed.ok {
		return
	}
	r := s.cells(box)
	if r == b.filed {
		s.hard.update(b, box, r)
		b.box = box
		return
	}
	s.unfile(b)
	s.hard.add(b, box, r)
	b.filed, b.box = r, box
	if s.hard.filed > 2*len(s.hard.buckets) {
		s.grow()
	}
}

func (s *Space) unfile(b *Body) {
	if b.filed.ok {
		s.hard.remove(b, b.filed)
	}
	b.filed, b.box = cellRange{}, geom.Rect{}
}

// leaveCrowd takes b out of the crowd until it is soft again.
func (s *Space) leaveCrowd(b *Body) {
	if b.indexed {
		s.crowd.leave(b)
		b.indexed = false
	}
}

// grow doubles the hash's buckets and files its bodies again.
func (s *Space) grow() {
	s.hard = hash{buckets: make([][]entry, 2*len(s.hard.buckets))}
	for _, b := range s.bodies {
		if b.filed.ok {
			s.hard.add(b, b.box, b.filed)
		}
	}
}

func (s *Space) forget(b *Body) {
	s.unfile(b)
	s.leaveCrowd(b)
	if b.owner != nil && s.owners[b.owner.ID()] == b {
		delete(s.owners, b.owner.ID())
	}
	if b.listed {
		b.listed = false
		if i := slices.Index(s.bodies, b); i >= 0 {
			s.bodies = slices.Delete(s.bodies, i, i+1)
		}
		if b.trigger {
			s.triggers = slices.DeleteFunc(s.triggers, func(t *Body) bool { return t == b })
		}
	}
}

// What near looks in: the bodies that aren't soft, the soft ones, or both.
const (
	inHard = 1 << iota
	inSoft
	inAll = inHard | inSoft
)

// near lists the bodies of in whose bounds, as last filed, touch area,
// each once. The list is reused by the next call.
func (s *Space) near(area geom.Rect, in int) []*Body {
	out := s.found[:0]
	if in&inHard != 0 && s.hard.filed > 0 {
		out = s.hard.collect(out, area, s.cells(area))
	}
	if in&inSoft != 0 {
		out = s.crowd.collect(out, area)
	}
	s.found = out
	return out
}

// touches reports whether a and b overlap or meet edge to edge.
func touches(a, b geom.Rect) bool {
	return a.X <= b.X+b.W && b.X <= a.X+a.W && a.Y <= b.Y+b.H && b.Y <= a.Y+a.H
}

// Move moves owner's body by delta, stopping at solid bodies and solids
// and sliding along them, and moves owner to where it ends. It returns the
// distance travelled, what stopped it, and the triggers it overlaps at the
// end. An owner without a body moves freely.
func (s *Space) Move(owner Owner, delta geom.Vec2) (geom.Vec2, []Hit) {
	b := s.BodyOf(owner)
	start := owner.Position()
	if b == nil {
		owner.SetPosition(start.Add(delta))
		return delta, nil
	}
	b.owner = owner
	if b.indexed {
		delta = s.crowd.room(b, delta, capacity)
	}
	at, hits := s.slide(b, start, delta, true)
	owner.SetPosition(at)
	s.file(b)
	b.stepped = s.step
	if len(s.triggers) == 0 {
		return at.Sub(start), hits
	}
	for _, o := range s.near(b.Bounds(), inHard) {
		if o != b && o.trigger && b.collides(o) && b.overlaps(o) {
			hits = append(hits, Hit{Body: o, Rect: o.Bounds(), Trigger: true})
		}
	}
	return at.Sub(start), hits
}

// slide moves b from at by delta, stopping at what is solid to it and
// sliding along it, and returns where it ends, with what it met when keep
// is set.
func (s *Space) slide(b *Body, at, delta geom.Vec2, keep bool) (geom.Vec2, []Hit) {
	reach := delta.Len() + 0.01
	box := b.boundsAt(at)
	s.gather(b, geom.Rect{X: box.X - reach, Y: box.Y - reach, W: box.W + 2*reach, H: box.H + 2*reach})
	delta = s.outward(box, delta)
	var hits []Hit
	for range 3 {
		if delta.Dot(delta) < 1e-12 {
			break
		}
		first, ok := s.first(b, at, delta)
		if !ok {
			at = at.Add(delta)
			break
		}
		if keep {
			hits = append(hits, first)
		}
		at = at.Add(delta.Scale(first.Time)).Add(first.Normal.Scale(1e-3))
		rest := delta.Scale(1 - first.Time)
		delta = rest.Sub(first.Normal.Scale(rest.Dot(first.Normal)))
	}
	return at, hits
}

// obstacle is a rectangle that stops a mover: a body's, or a solid's
// with no body.
type obstacle struct {
	rect geom.Rect
	body *Body
}

// gather lists in s.obstacles what can stop b within area.
func (s *Space) gather(b *Body, area geom.Rect) {
	s.obstacles = s.obstacles[:0]
	in := inAll
	if b.soft() {
		in = inHard
	}
	for _, o := range s.near(area, in) {
		if o != b && !o.trigger && b.collides(o) {
			s.obstacles = append(s.obstacles, obstacle{o.Bounds(), o})
		}
	}
	for _, solid := range s.solids {
		for _, r := range s.solidIn(solid, area) {
			s.obstacles = append(s.obstacles, obstacle{r, nil})
		}
	}
}

// solidIn is what solid stops bodies with in area, into s.rects when it
// appends.
func (s *Space) solidIn(solid Solid, area geom.Rect) []geom.Rect {
	if a, ok := solid.(SolidAppender); ok {
		s.rects = a.AppendSolidIn(s.rects[:0], area)
		return s.rects
	}
	return solid.SolidIn(area)
}

// first is the earliest of s.obstacles b meets moving by delta from at.
func (s *Space) first(b *Body, at, delta geom.Vec2) (Hit, bool) {
	from := b.boundsAt(at)
	swept := union(from, b.boundsAt(at.Add(delta)))
	best, found := Hit{Time: 2}, false
	c, round := b.circle()
	for _, o := range s.obstacles {
		if !touches(swept, o.rect) {
			continue
		}
		var t float32
		var n geom.Vec2
		var hit bool
		if round {
			c.Center = at.Add(b.offset)
			t, n, hit = geom.SweepCircle(c, delta, o.rect)
		} else {
			t, n, hit = geom.Sweep(from, delta, o.rect)
		}
		if hit && t < best.Time {
			best, found = Hit{Body: o.body, Rect: o.rect, Normal: n, Time: t}, true
		}
	}
	return best, found
}

// At lists the bodies covering p.
func (s *Space) At(p geom.Vec2) []*Body {
	var out []*Body
	for _, b := range s.near(geom.Rect{X: p.X, Y: p.Y}, inAll) {
		if c, ok := b.circle(); ok && c.Contains(p) || !ok && b.Bounds().Contains(p) {
			out = append(out, b)
		}
	}
	return out
}

// Overlapping lists the bodies a shape centred on at would overlap.
func (s *Space) Overlapping(shape Shape, at geom.Vec2) []*Body {
	probe := &Body{shape: shape, meets: ^uint32(0), group: ^uint32(0), owner: point(at)}
	var out []*Body
	for _, b := range s.near(probe.Bounds(), inAll) {
		if b.overlaps(probe) {
			out = append(out, b)
		}
	}
	return out
}

// Cast follows a ray from from to to and returns the first solid body or
// solid it meets, for line of sight.
func (s *Space) Cast(from, to geom.Vec2) (Hit, bool) {
	probe := &Body{shape: Box(0, 0), meets: ^uint32(0), group: ^uint32(0), owner: point(from)}
	s.gather(probe, union(probe.boundsAt(from), probe.boundsAt(to)))
	return s.first(probe, from, to.Sub(from))
}

// Solids are the rectangles the space's solids stop bodies with within
// area, such as a tile map's tile shapes, for drawing what blocks.
func (s *Space) Solids(area geom.Rect) []geom.Rect {
	var out []geom.Rect
	for _, solid := range s.solids {
		out = append(out, solid.SolidIn(area)...)
	}
	return out
}

// Bodies are the bodies, hard, soft and triggers, whose bounds meet area.
func (s *Space) Bodies(area geom.Rect) []*Body {
	var out []*Body
	for _, b := range s.near(area, inAll) {
		if touches(b.Bounds(), area) {
			out = append(out, b)
		}
	}
	return out
}

// outward keeps a body standing in box, already inside solids, from
// going deeper into any of them: of delta, it keeps what overlaps none of
// them more, whole, else across, else along, else nothing; so one placed
// overlapping a wall can slide along it or step out but never walk on in.
// The solids it isn't inside are the sweep's.
func (s *Space) outward(box geom.Rect, delta geom.Vec2) geom.Vec2 {
	inside := false
	for _, o := range s.obstacles {
		inside = inside || o.body == nil && overlap(box, o.rect) > 0
	}
	if !inside {
		return delta
	}
	for _, d := range [3]geom.Vec2{delta, {X: delta.X}, {Y: delta.Y}} {
		moved := geom.Rect{X: box.X + d.X, Y: box.Y + d.Y, W: box.W, H: box.H}
		deeper := false
		for _, o := range s.obstacles {
			if before := overlap(box, o.rect); o.body == nil && before > 0 && overlap(moved, o.rect) > before+1e-3 {
				deeper = true
				break
			}
		}
		if !deeper {
			return d
		}
	}
	return geom.Vec2{}
}

// overlap is how much of a and b they share.
func overlap(a, b geom.Rect) float32 {
	ox := min(a.X+a.W, b.X+b.W) - max(a.X, b.X)
	oy := min(a.Y+a.H, b.Y+b.H) - max(a.Y, b.Y)
	if ox <= 0 || oy <= 0 {
		return 0
	}
	return ox * oy
}

// Clear reports whether no solid lies on the straight line from from to
// to; bodies don't count. Two things touching across a thin wall aren't
// in reach of each other.
func (s *Space) Clear(from, to geom.Vec2) bool {
	probe := &Body{shape: Box(0, 0), owner: point(from)}
	s.gather(probe, union(probe.boundsAt(from), probe.boundsAt(to)))
	_, hit := s.first(probe, from, to.Sub(from))
	return !hit
}

func union(a, b geom.Rect) geom.Rect {
	x0, y0 := min(a.X, b.X), min(a.Y, b.Y)
	x1, y1 := max(a.X+a.W, b.X+b.W), max(a.Y+a.H, b.Y+b.H)
	return geom.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}

// point is a still owner for queries.
type point geom.Vec2

func (point) ID() uint64            { return 0 }
func (p point) Position() geom.Vec2 { return geom.Vec2(p) }
func (point) SetPosition(geom.Vec2) {}
