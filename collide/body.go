// Package collide moves things in the plane and stops them at what is
// solid: no physics and no forces, just "move this much, stop at walls,
// and say what was touched". It is pure Go, for games and servers alike.
//
// A Body is an entity appended to what it belongs to, any entity with a
// position in the plane, such as a twod sprite. The game holds a Space,
// adds a Tracker that keeps it up to date, and moves its owners through it:
//
//	space := collide.NewSpace()
//	w.AddSystems(collide.NewTracker(space))
//	hero.AppendChild(collide.NewBody(collide.Box(12, 8)))
//	moved, hits := space.Move(hero, geom.V2(dx, dy))
package collide

import (
	"math/bits"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/world"
)

// Shape is a body's outline, centred on its position plus its offset.
type Shape struct {
	size   geom.Vec2
	radius float32
}

// Box is a w×h rectangle.
func Box(w, h float32) Shape { return Shape{size: geom.V2(w, h)} }

// Circle is a disc of radius r.
func Circle(r float32) Shape { return Shape{radius: r, size: geom.V2(2*r, 2*r)} }

// Owner is what a body belongs to: an entity placed in the plane.
type Owner interface {
	world.Identifier
	Position() geom.Vec2
	SetPosition(geom.Vec2)
}

// Body gives its parent entity a shape that solid bodies, tiles and
// triggers meet.
type Body struct {
	*world.BasicEntity
	shape   Shape
	offset  geom.Vec2
	trigger bool
	push    float32
	group   uint32
	meets   uint32
	owner   Owner
	// filed is the cells the body is filed in, with bounds box.
	filed cellRange
	box   geom.Rect
	// listed is set while the space lists the body. A soft body is
	// indexed once the crowd has filed it, with bounds box, in cell slot
	// (-1 off the crowd's grid), covering share of it; it is waiting while
	// it is in the crowd's loose list.
	listed  bool
	indexed bool
	waiting bool
	slot    int32
	share   float32
	// stepped is the space's step its move last filed it in.
	stepped uint64
}

// BodyOption sets up a Body.
type BodyOption func(*Body)

// NewBody returns a solid body of shape, in group 0, colliding with every
// group.
func NewBody(shape Shape, opts ...BodyOption) *Body {
	b := &Body{BasicEntity: world.NewBasic(), shape: shape, group: 1, meets: ^uint32(0)}
	for _, opt := range opts {
		opt(b)
	}
	return b
}

// Offset moves the shape from its owner's position, such as down to its
// feet.
func Offset(x, y float32) BodyOption { return func(b *Body) { b.offset = geom.V2(x, y) } }

// Trigger makes a body that nothing bumps into: it only tells what enters
// and leaves it.
func Trigger() BodyOption { return func(b *Body) { b.trigger = true } }

// Soft makes a body of a crowd: soft bodies pass through each other as
// they move and spread out of crowded spots instead, so a horde neither
// piles on one point nor jams. push, from 0 to 1, is how fast an
// over-full spot spreads (0 is not soft). They move into a spot only while
// it has room, and still stop exactly at solids and at bodies that aren't
// soft, such as the hero, which they never push.
func Soft(push float32) BodyOption { return func(b *Body) { b.SetSoft(push) } }

// Group puts the body in group n, from 0 to 31: the player, enemies,
// pickups.
func Group(n int) BodyOption { return func(b *Body) { b.group = 1 << uint(n&31) } }

// CollidesWith lists the groups whose bodies stop this one, or trigger
// it; every group by default.
func CollidesWith(groups ...int) BodyOption {
	return func(b *Body) {
		b.meets = 0
		for _, n := range groups {
			b.meets |= 1 << uint(n&31)
		}
	}
}

func (b *Body) GetCollider() *Body { return b }

func (b *Body) Shape() Shape      { return b.shape }
func (b *Body) Offset() geom.Vec2 { return b.offset }
func (b *Body) Trigger() bool     { return b.trigger }

// Soft is how fast the body spreads out of a crowded spot, 0 for a body
// that isn't soft.
func (b *Body) Soft() float32 { return b.push }

// SetSoft makes the body soft with push, or not soft with 0.
func (b *Body) SetSoft(push float32) { b.push = geom.Clamp01(push) }

func (b *Body) soft() bool { return b.push > 0 && !b.trigger }

// Group is the body's group.
func (b *Body) Group() int { return bits.TrailingZeros32(b.group) }

// CollidesWith lists the groups the body collides with.
func (b *Body) CollidesWith() []int {
	var out []int
	for n := range 32 {
		if b.meets&(1<<uint(n)) != 0 {
			out = append(out, n)
		}
	}
	return out
}

// Owner is the entity the body belongs to, once the tracker has seen it.
func (b *Body) Owner() Owner { return b.owner }

// Bounds is the rectangle the body covers now.
func (b *Body) Bounds() geom.Rect {
	if b.owner == nil {
		return geom.Rect{}
	}
	return b.boundsAt(b.owner.Position())
}

func (b *Body) boundsAt(at geom.Vec2) geom.Rect {
	c := at.Add(b.offset)
	return geom.Rect{X: c.X - b.shape.size.X/2, Y: c.Y - b.shape.size.Y/2, W: b.shape.size.X, H: b.shape.size.Y}
}

// area is how much of the plane the body covers.
func (b *Body) area() float32 {
	if b.shape.radius > 0 {
		return geom.Pi * b.shape.radius * b.shape.radius
	}
	return b.shape.size.X * b.shape.size.Y
}

func (b *Body) circle() (geom.Circle, bool) {
	if b.shape.radius <= 0 || b.owner == nil {
		return geom.Circle{}, false
	}
	return geom.Circle{Center: b.owner.Position().Add(b.offset), Radius: b.shape.radius}, true
}

// collides reports whether b and o collide or trigger by their groups.
func (b *Body) collides(o *Body) bool { return b.meets&o.group != 0 && o.meets&b.group != 0 }

// overlaps reports whether the two bodies share area now.
func (b *Body) overlaps(o *Body) bool {
	bc, bok := b.circle()
	oc, ook := o.circle()
	switch {
	case bok && ook:
		return bc.Overlaps(oc)
	case bok:
		return o.Bounds().OverlapsCircle(bc)
	case ook:
		return b.Bounds().OverlapsCircle(oc)
	}
	return b.Bounds().Overlaps(o.Bounds())
}
