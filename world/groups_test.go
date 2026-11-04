package world_test

import (
	"strings"
	"testing"

	"github.com/merlindorin/ember/world"
)

type walker interface {
	world.BasicFace
	Walk()
}

type flier interface {
	world.BasicFace
	Fly()
}

type hero struct{ *world.BasicEntity }

func (*hero) Walk() {}

type bat struct{ *world.BasicEntity }

func (*bat) Fly() {}

type hunt struct {
	world.Group[walker]
	bats    world.Group[flier]
	Walkers *world.Group[walker]
	missing *world.Group[flier]
}

func (*hunt) Update(world.Frame) {}

type counted struct {
	world.Group[walker]
	added int
}

func (c *counted) AddByInterface(e world.Identifier) { c.added++; c.Group.AddByInterface(e) }
func (*counted) Update(world.Frame)                  {}

func TestEveryGroupFieldOfASystemIsFilled(t *testing.T) {
	w := world.New()
	h, b := &hero{world.NewBasic()}, &bat{world.NewBasic()}
	w.AddEntities(h)
	s := &hunt{Walkers: &world.Group[walker]{}}
	w.AddSystems(s)
	w.AddEntities(b)
	if s.Len() != 1 || s.bats.Len() != 1 || s.Walkers.Len() != 1 {
		t.Fatalf("embedded, unexported and pointer fields, before and after the system: %d %d %d",
			s.Len(), s.bats.Len(), s.Walkers.Len())
	}
	if s.missing != nil {
		t.Fatal("a nil pointer field stays nil")
	}
	got := w.Explain(b)
	for _, want := range []string{
		"in  hunt.bats",
		"out hunt.Walkers: wants world_test.walker",
		"out *world_test.hunt: wants world_test.walker",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("%q in:\n%s", want, got)
		}
	}
	w.RemoveEntity(b)
	w.RemoveEntity(h)
	if s.Len() != 0 || s.bats.Len() != 0 || s.Walkers.Len() != 0 {
		t.Fatal("removed from every group")
	}
}

func TestAnEmbeddedGroupFilledByTheSystemIsFilledOnce(t *testing.T) {
	w := world.New()
	c := &counted{}
	w.AddSystems(c)
	w.AddEntities(&hero{world.NewBasic()})
	if c.added != 1 || c.Len() != 1 {
		t.Fatalf("through the system's own AddByInterface, once: %d %d", c.added, c.Len())
	}
}
