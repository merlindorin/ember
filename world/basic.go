package world

import (
	"sync/atomic"
)

var (
	idInc uint64 //nolint:gochecknoglobals // entity IDs are unique across every World of the process
)

// A BasicEntity is simply a set of components with a unique ID attached to it,
// nothing more. It belongs to any amount of Systems, and has a number of
// Components.
type BasicEntity struct {
	// Entity ID.
	id       uint64
	parent   *BasicEntity
	children []*BasicEntity
	// self is the whole entity, when it was appended as one, so adding its
	// parent to a World adds it too.
	self BasicFace
	// world is the World the entity is in, so a child appended to it later
	// joins that World at once.
	world *World
}

// Identifier is an interface for anything that implements the basic ID() uint64,
// as the BasicEntity does.  It is useful as more specific interface for an
// entity registry than just the interface{} interface.
type Identifier interface {
	ID() uint64
}

// IdentifierSlice implements the sort.Interface, so you can use the
// store entites in slices, and use the P=n*log n lookup for them.
type IdentifierSlice []Identifier

// NewBasic creates a new BasicEntity with a new unique identifier.
// It is safe for concurrent use.
func NewBasic() *BasicEntity {
	return &BasicEntity{id: atomic.AddUint64(&idInc, 1)}
}

// NewBasics creates a slice of new BasicEntity instances with unique identifiers.
// It is safe for concurrent use and performs better than calling NewBasic
// multiple times for large numbers of entities.
func NewBasics(amount int) []BasicEntity {
	entities := make([]BasicEntity, amount)

	n := uint64(len(entities))
	first := atomic.AddUint64(&idInc, n) - n
	for i := range entities {
		entities[i].id = first + uint64(i) + 1
	}

	return entities
}

// ID returns the unique identifier of the BasicEntity.
func (e *BasicEntity) ID() uint64 {
	return e.id
}

// GetBasicEntity returns a pointer to the BasicEntity itself.
// This method enables all entities containing a BasicEntity to automatically
// implement the BasicFace interface, allowing systems to accept entities
// through a unified interface.
//
// Example:
//
//	s.AddByInterface(a interface{
//	  GetBasicEntity() *BasicEntity
//	  GetSpaceComponent() *SpaceComponent
//	}) {
//	  s.Add(a.GetBasicEntity(), a.GetSpaceComponent())
//	}
func (e *BasicEntity) GetBasicEntity() *BasicEntity {
	return e
}

// AppendChild makes child a child of this entity. Given the whole entity
// (not only its *BasicEntity), adding this entity to a World adds the child
// too.
func (e *BasicEntity) AppendChild(entity BasicFace) {
	child := entity.GetBasicEntity()
	if entity != BasicFace(child) {
		child.self = entity
	}
	if child.parent != nil {
		child.parent.RemoveChild(child)
	}
	child.parent = e
	e.children = append(e.children, child)
	if e.world != nil && entity != BasicFace(child) {
		e.world.AddEntity(entity)
	}
}

// RemoveChild removes the specified child entity from this BasicEntity's children.
// If the child is not found, this method does nothing; otherwise the child
// no longer has a parent.
//
// It looks from the last child back: appending a child again, as adding
// a node does, removes the one just appended.
func (e *BasicEntity) RemoveChild(child *BasicEntity) {
	at := -1
	for i := len(e.children) - 1; i >= 0; i-- {
		if e.children[i].ID() == child.ID() {
			at = i
			break
		}
	}
	if at >= 0 {
		e.children = append(e.children[:at], e.children[at+1:]...)
		child.parent = nil
	}
}

// Of is the World e is in, nil when it is in none: for a parent
// letting go of a child for good, which RemoveChild alone leaves in the
// World.
func Of(e BasicFace) *World { return e.GetBasicEntity().world }

// Children are the entity's children, each the whole entity it was
// appended as, so a game type-asserts them to its own types:
//
//	for _, c := range hero.Children() {
//		if fx, ok := c.(*Effect); ok { … }
//	}
//
// The slice is new; changing it changes nothing.
func (e *BasicEntity) Children() []BasicFace {
	out := make([]BasicFace, 0, len(e.children))
	for _, c := range e.children {
		out = append(out, c.whole())
	}
	return out
}

// whole is the entity e was appended as, else e itself.
func (e *BasicEntity) whole() BasicFace {
	if e.self != nil {
		return e.self
	}
	return e
}

// Descendents are all the entity's descendants, children, grandchildren
// and so on, depth first, each the whole entity, without the entity
// itself.
func (e *BasicEntity) Descendents() []BasicFace {
	return descendents(nil, e, e)
}

// HasDescendents returns true if the entity has any descendant entities
// (children, grandchildren, etc.).
func (e *BasicEntity) HasDescendents() bool {
	return len(e.children) > 0
}

func descendents(in []BasicFace, this, top *BasicEntity) []BasicFace {
	for _, child := range this.children {
		in = descendents(in, child, top)
	}
	if this == top {
		return in
	}
	return append(in, this.whole())
}

// Parent returns the parent entity of this BasicEntity, or nil if it has no parent.
func (e *BasicEntity) Parent() *BasicEntity {
	return e.parent
}

// HasParent returns true if the BasicEntity has a parent entity.
func (e *BasicEntity) HasParent() bool {
	return e.parent != nil
}

// HasAncestor returns true if the BasicEntity has an entity as ancestor.
func (e *BasicEntity) HasAncestor(ancestor *BasicEntity) bool {
	if e.parent == nil {
		return false
	}

	if e.parent.ID() == ancestor.ID() {
		return true
	}

	return e.parent.HasAncestor(ancestor)
}

// Siblings returns all sibling entities (other children of the same parent).
// Returns nil if the entity has no parent.
func (e *BasicEntity) Siblings() []*BasicEntity {
	return e.parent.children
}

// Next returns the next sibling in the parent's children list,
// or nil if this is the last child or the entity has no parent.
func (e *BasicEntity) Next() *BasicEntity {
	var found *BasicEntity
	for _, entity := range e.Siblings() {
		if found != nil {
			return entity
		}

		if entity.ID() == e.ID() {
			found = entity
		}
	}

	return nil
}

// Prev returns the previous sibling in the parent's children list,
// or nil if this is the first child or the entity has no parent.
func (e *BasicEntity) Prev() *BasicEntity {
	var prev *BasicEntity
	for _, entity := range e.Siblings() {
		if entity.ID() == e.ID() {
			return prev
		}
		prev = entity
	}

	return nil
}

func (e *BasicEntity) Left() []*BasicEntity {
	siblings := e.Siblings()

	for id, entity := range siblings {
		if entity.ID() == e.ID() {
			return siblings[:id]
		}
	}

	return []*BasicEntity{}
}

func (e *BasicEntity) Right() []*BasicEntity {
	siblings := e.Siblings()

	for id, entity := range siblings {
		if entity.ID() == e.ID() {
			return siblings[id:]
		}
	}

	return []*BasicEntity{}
}

// Len returns the number of elements in the slice.
// This method is part of the sort.Interface implementation.
func (is IdentifierSlice) Len() int {
	return len(is)
}

// Less reports whether the element with index i should sort before
// the element with index j based on their IDs.
// This method is part of the sort.Interface implementation.
func (is IdentifierSlice) Less(i, j int) bool {
	return is[i].ID() < is[j].ID()
}

// Swap exchanges the elements at indices i and j.
// This method is part of the sort.Interface implementation.
func (is IdentifierSlice) Swap(i, j int) {
	is[i], is[j] = is[j], is[i]
}

// BasicFace is an interface that BasicEntity and entities containing
// a BasicEntity implement.
type BasicFace interface {
	Identifier
	GetBasicEntity() *BasicEntity
}
