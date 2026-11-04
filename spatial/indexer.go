package spatial

import (
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/scene"
	"github.com/merlindorin/ember/world"
)

// Spatial opts an entity into the world index. Radius is informational
// (pick ranges, broad-phase sizes).
type Spatial struct {
	Radius float32
}

func (s *Spatial) GetSpatial() *Spatial { return s }

type Indexed interface {
	scene.Transformable
	GetSpatial() *Spatial
}

// Index is a world-wide Grid of the Indexed entities, which an Indexer
// keeps in sync. The game makes it and keeps it to query.
type Index struct {
	*Grid
}

// NewIndex returns an empty index of cells cell wide.
func NewIndex(cell float32) *Index { return &Index{NewGrid(cell)} }

// Indexer keeps an Index in sync with the Indexed entities, right after
// transforms are resolved in PostUpdate.
type Indexer struct {
	world.Group[Indexed]
	index *Index
	fixed bool
}

// IndexerOption sets up an Indexer.
type IndexerOption func(*Indexer)

// Fixed indexes local positions at the start of every FixedUpdate step
// instead, for authoritative simulations whose entities are roots.
func Fixed() IndexerOption { return func(s *Indexer) { s.fixed = true } }

// NewIndexer returns the system keeping idx in sync.
func NewIndexer(idx *Index, opts ...IndexerOption) *Indexer {
	s := &Indexer{index: idx}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *Indexer) Phase() world.Phase {
	if s.fixed {
		return world.FixedUpdate
	}
	return world.PostUpdate
}

func (s *Indexer) Priority() int { return 900 }

func (s *Indexer) Update(world.Frame) {
	for _, e := range s.Slice() {
		s.index.Move(e.ID(), s.position(e))
	}
}

func (s *Indexer) position(e Indexed) geom.Vec3 {
	if s.fixed {
		return e.GetTransform().Position
	}
	return e.GetTransform().WorldPosition()
}

func (s *Indexer) AddByInterface(o world.Identifier) {
	s.Group.AddByInterface(o)
	if e, ok := o.(Indexed); ok && s.index != nil {
		s.index.Insert(e.ID(), s.position(e))
	}
}

func (s *Indexer) Remove(e world.Identifier) {
	s.Group.Remove(e)
	if s.index != nil {
		s.index.Remove(e.ID())
	}
}
