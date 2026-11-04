package collide

import (
	"cmp"
	"slices"

	"github.com/merlindorin/ember/world"
)

// BodyEntered is emitted when a body starts overlapping a trigger.
type BodyEntered struct{ Trigger, Body *Body }

// BodyLeft is emitted when a body stops overlapping a trigger.
type BodyLeft struct{ Trigger, Body *Body }

type bodied interface {
	world.BasicFace
	GetCollider() *Body
}

// Tracker keeps a Space up to date with the World's bodies, spreads soft
// bodies out of crowded spots, and reports triggers entered and left, late
// in each fixed step, after the moves: in a stable order, so a fixed-step
// simulation stays deterministic.
type Tracker struct {
	world.Group[bodied]
	space  *Space
	world  *world.World
	inside map[[2]*Body]bool
	now    map[[2]*Body]bool
	// unowned are bodies still looking for their owner.
	unowned []*Body
}

// NewTracker returns the system keeping space up to date.
func NewTracker(space *Space) *Tracker {
	return &Tracker{space: space, inside: map[[2]*Body]bool{}, now: map[[2]*Body]bool{}}
}

func (t *Tracker) New(w *world.World) { t.world = w }
func (t *Tracker) Phase() world.Phase { return world.FixedUpdate }
func (t *Tracker) Priority() int      { return -1000 }

func (t *Tracker) Remove(e world.Identifier) {
	if b, ok := e.(bodied); ok {
		t.unowned = slices.DeleteFunc(t.unowned, func(u *Body) bool { return u == b.GetCollider() })
		t.space.forget(b.GetCollider())
		for pair := range t.inside {
			if pair[0] == b.GetCollider() || pair[1] == b.GetCollider() {
				delete(t.inside, pair)
			}
		}
	}
	t.Group.Remove(e)
}

// AddByInterface files a body as soon as it joins the World, so its owner
// collides from its first move.
func (t *Tracker) AddByInterface(e world.Identifier) {
	t.Group.AddByInterface(e)
	if b, ok := e.(bodied); ok {
		t.track(b.GetCollider())
		if b.GetCollider().owner == nil {
			t.unowned = append(t.unowned, b.GetCollider())
		}
	}
}

func (t *Tracker) track(b *Body) {
	if b.owner == nil && t.world != nil {
		if p := b.Parent(); p != nil {
			if o, ok := t.world.Entity(p.ID()); ok {
				b.owner, _ = o.(Owner)
			}
		}
	}
	t.space.file(b)
}

// Update files the bodies moved since the last step otherwise than by
// Move, which files its own, spreads the crowd, and reports triggers.
func (t *Tracker) Update(world.Frame) {
	t.unowned = slices.DeleteFunc(t.unowned, func(b *Body) bool {
		t.track(b)
		return b.owner != nil
	})
	for _, b := range t.space.bodies {
		if b.stepped != t.space.step || b.stepped == 0 {
			t.space.file(b)
		}
	}
	t.space.step++
	t.space.spread()
	now := t.now
	clear(now)
	for _, trigger := range t.space.triggers {
		if trigger.owner == nil {
			continue
		}
		for _, b := range t.space.near(trigger.Bounds(), inAll) {
			if b != trigger && !b.trigger && b.collides(trigger) && b.overlaps(trigger) {
				now[[2]*Body{trigger, b}] = true
			}
		}
	}
	for _, pair := range sorted(now) {
		if !t.inside[pair] {
			world.Emit(t.world, BodyEntered{Trigger: pair[0], Body: pair[1]})
		}
	}
	for _, pair := range sorted(t.inside) {
		if !now[pair] {
			world.Emit(t.world, BodyLeft{Trigger: pair[0], Body: pair[1]})
		}
	}
	t.inside, t.now = now, t.inside
}

// sorted lists pairs by their trigger's then their body's ID.
func sorted(pairs map[[2]*Body]bool) [][2]*Body {
	out := make([][2]*Body, 0, len(pairs))
	for pair := range pairs {
		out = append(out, pair)
	}
	slices.SortFunc(out, func(a, b [2]*Body) int {
		return cmp.Or(cmp.Compare(a[0].ID(), b[0].ID()), cmp.Compare(a[1].ID(), b[1].ID()))
	})
	return out
}
