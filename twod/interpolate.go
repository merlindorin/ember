package twod

import (
	"math"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/node"
	"github.com/merlindorin/ember/world"
)

// pose is a placement's values, compared to see whether something moved
// it since the last fixed step.
type pose struct {
	position geom.Vec2
	rotation float32
	scale    geom.Vec2
}

// steps is where the last fixed step that moved a placement started and
// left it, to draw it between the two, and the World's step clock it
// counts its steps by.
type steps struct {
	from, to pose
	stepped  bool
	// moved is the clock's step that last moved it, toStep the step that
	// left it at to.
	moved, toStep uint64
	clock         *stepClock
	// changed is set by every setter, in a fixed step or not; what keeps
	// placements filed by where they are clears it once it has looked.
	changed bool
	// filed is where a layer's grid filed the placement, nil for none.
	filed *filed
}

// stepClock counts a World's fixed steps, and lists the placements the one
// running has moved, so a step's end looks at those alone.
type stepClock struct {
	step     uint64
	stepping bool
	moved    []*Placement
}

func (p *Placement) pose() pose { return pose{p.position, p.rotation, p.scale} }

// moving marks the placement changed, and notes where it started the
// running fixed step the first time something moves it in that step.
func (p *Placement) moving() {
	p.refile()
	if c := p.clock; c != nil && c.stepping && p.moved != c.step {
		p.from, p.moved = p.pose(), c.step
		c.moved = append(c.moved, p)
	}
}

// Snap draws the entity where it is now rather than blended from where the
// last fixed step started it: after a teleport.
func (p *Placement) Snap() { p.from, p.to = p.pose(), p.pose() }

// drawn is the transform to draw the placement with, alpha of the way
// between the last two fixed steps: blended while the last step moved it
// and it is where that step left it, as it is otherwise.
func (p *Placement) drawn(alpha float32) geom.Transform2D {
	now := p.pose()
	if !p.stepped || p.clock == nil || p.toStep != p.clock.step || now != p.to || p.from == p.to {
		return p.Local()
	}
	a, b := p.from, p.to
	return geom.Transform2DFrom(
		a.position.Lerp(b.position, alpha),
		a.rotation+(b.rotation-a.rotation)*alpha,
		0,
		a.scale.Lerp(b.scale, alpha),
	)
}

type positioned interface{ GetPlacement() *Placement }

// stepStart opens each fixed step on the World's clock, before any game
// system moves a placement; every placement joining the World is given
// that clock.
type stepStart struct {
	world.Group[positioned]
	clock *stepClock
}

func (*stepStart) Phase() world.Phase { return world.FixedUpdate }
func (*stepStart) Priority() int      { return math.MaxInt }

func (s *stepStart) AddByInterface(e world.Identifier) {
	s.Group.AddByInterface(e)
	if p, ok := e.(positioned); ok {
		p.GetPlacement().clock = s.clock
	}
}

func (s *stepStart) Update(world.Frame) {
	s.clock.step++
	s.clock.stepping = true
}

// stepEnd notes where each placement the step moved is as it ends.
type stepEnd struct{ clock *stepClock }

func (*stepEnd) Phase() world.Phase { return world.FixedUpdate }
func (*stepEnd) Priority() int      { return math.MinInt }

func (s *stepEnd) Update(world.Frame) {
	c := s.clock
	for _, p := range c.moved {
		p.to, p.stepped, p.toStep = p.pose(), true, c.step
	}
	c.moved = c.moved[:0]
	c.stepping = false
}

// drawnLocal is e's transform in its parent as drawn at alpha.
func drawnLocal(e Entity, alpha float32) geom.Transform2D {
	if p, ok := e.(positioned); ok {
		return p.GetPlacement().drawn(alpha)
	}
	return e.Local()
}

// drawnGlobal is e's transform in its layer as drawn at alpha.
func drawnGlobal(e node.Noded, alpha float32) geom.Transform2D {
	t := geom.Identity2D()
	if self, ok := e.(Entity); ok {
		t = drawnLocal(self, alpha)
	}
	for p := e.GetSceneNode().ParentNode(); p != nil; p = p.ParentNode() {
		parent, ok := p.Entity().(Entity)
		if !ok {
			break
		}
		t = drawnLocal(parent, alpha).Mul(t)
	}
	return t
}
