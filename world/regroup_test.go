package world_test

import (
	"strings"
	"testing"

	"github.com/merlindorin/ember/world"
)

type patient struct {
	*world.BasicEntity
	poisoned bool
}

type poisoned struct{ world.Group[*patient] }

func (g *poisoned) Matches(o world.Identifier) bool {
	p, ok := o.(*patient)
	return ok && p.poisoned
}

func (g *poisoned) AddByInterface(o world.Identifier) {
	if g.Matches(o) {
		g.Group.AddByInterface(o)
	}
}

// healer cures every poisoned patient in its own loop, regrouping each
// as it goes, and notes how many the group held at each step.
type healer struct {
	sick  poisoned
	sizes []int
}

func (h *healer) Update(world.Frame) {
	for p := range h.sick.All() {
		h.sizes = append(h.sizes, h.sick.Len())
		p.poisoned = false
		world.Regroup(p)
	}
}

func TestRegroupOffersAnEntityToTheGroupsItNowMatches(t *testing.T) {
	w := world.New()
	h := &healer{}
	w.AddSystems(h)
	p := &patient{BasicEntity: world.NewBasic()}
	w.AddEntities(p)
	p.poisoned = true
	if h.sick.Len() != 0 {
		t.Fatal("a group takes an entity as it is added, not as it changes")
	}
	if got := w.Explain(p); !strings.Contains(got, "matches since it was added; world.Regroup(e) offers it again") {
		t.Fatalf("Explain says why:\n%s", got)
	}
	world.Regroup(p)
	if h.sick.Len() != 1 {
		t.Fatal("regrouped, it is in")
	}
	p.poisoned = false
	if got := w.Explain(p); !strings.Contains(got, "no longer matches; world.Regroup(e) lets it go") {
		t.Fatalf("and the other way:\n%s", got)
	}
	world.Regroup(p)
	if h.sick.Len() != 0 {
		t.Fatal("regrouped again, it is out")
	}
	world.Regroup(&patient{BasicEntity: world.NewBasic(), poisoned: true})
	if h.sick.Len() != 0 {
		t.Fatal("an entity in no World is left alone")
	}
}

func TestRegroupDuringAnUpdateWaitsForTheSystemToReturn(t *testing.T) {
	w := world.New()
	h := &healer{}
	w.AddSystems(h)
	for range 3 {
		w.AddEntities(&patient{BasicEntity: world.NewBasic(), poisoned: true})
	}
	w.RunFrames(1)
	if len(h.sizes) != 3 || h.sizes[0] != 3 || h.sizes[2] != 3 {
		t.Fatalf("the group holds all three through the loop that cures them: %v", h.sizes)
	}
	if h.sick.Len() != 0 {
		t.Fatalf("once the system returns, the cured leave it: %d", h.sick.Len())
	}
}
