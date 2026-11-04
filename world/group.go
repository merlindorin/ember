package world

import "iter"

// Group is a ready-made entity collection keyed by an interface T. Embed it
// in a system and the World will feed it every entity implementing T:
//
//	type Movable interface {
//		world.BasicFace
//		GetVelocity() *Velocity
//	}
//
//	type MoveSystem struct{ world.Group[Movable] }
//
//	func (s *MoveSystem) Update(dt float32) {
//		for m := range s.All() { ... }
//	}
//
//	world.AddSystem(&MoveSystem{})
//
// A system collects several kinds through Group fields, named or
// embedded, exported or not; the World fills them all:
//
//	type HuntSystem struct {
//		world.Group[Hunter]
//		prey world.Group[Prey]
//	}
//
// A Group on its own (not in a system) is a View, added with AddView: kept up to date
// by the World but never updated, which is how renderers observe the world.
type Group[T any] struct {
	items []T
	ids   []uint64
	index map[uint64]int
}

// Matches reports whether o implements T.
func (g *Group[T]) Matches(o Identifier) bool {
	_, ok := o.(T)
	return ok
}

func (g *Group[T]) AddByInterface(o Identifier) {
	if v, ok := o.(T); ok {
		g.Add(o.ID(), v)
	}
}

// Add inserts v under id, replacing any previous entry.
func (g *Group[T]) Add(id uint64, v T) {
	if g.index == nil {
		g.index = map[uint64]int{}
	}
	if i, ok := g.index[id]; ok {
		g.items[i] = v
		return
	}
	g.index[id] = len(g.items)
	g.items = append(g.items, v)
	g.ids = append(g.ids, id)
}

func (g *Group[T]) Remove(e Identifier) {
	i, ok := g.index[e.ID()]
	if !ok {
		return
	}
	last := len(g.items) - 1
	g.items[i], g.ids[i] = g.items[last], g.ids[last]
	g.index[g.ids[i]] = i
	var zero T
	g.items[last] = zero
	g.items, g.ids = g.items[:last], g.ids[:last]
	delete(g.index, e.ID())
}

// All iterates the members. Removing entities through the World while
// iterating is safe: the World defers removals until the system returns.
func (g *Group[T]) All() iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, v := range g.items {
			if !yield(v) {
				return
			}
		}
	}
}

// Get returns the member with the given entity ID.
func (g *Group[T]) Get(id uint64) (T, bool) {
	if i, ok := g.index[id]; ok {
		return g.items[i], true
	}
	var zero T
	return zero, false
}

func (g *Group[T]) Len() int { return len(g.items) }

// Slice exposes the members for index-based loops and sorting. Do not
// retain it across frames.
func (g *Group[T]) Slice() []T { return g.items }
