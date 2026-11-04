package slot_test

import (
	"testing"

	"github.com/merlindorin/ember/internal/slot"
	"github.com/merlindorin/ember/world"
)

type thing struct{ n int }

func TestEachWorldHoldsItsOwnValuePerType(t *testing.T) {
	a, b := world.New(), world.New()
	if slot.Get[thing](a) != nil {
		t.Fatal("nothing set yet")
	}
	x := &thing{1}
	slot.Set(a, x)
	if slot.Get[thing](a) != x || slot.Get[thing](b) != nil || slot.Get[int](a) != nil {
		t.Fatal("a holds x, b and other types hold nothing")
	}
	y := &thing{2}
	slot.Set(a, y)
	if slot.Get[thing](a) != y {
		t.Fatal("setting again replaces")
	}
	if _, ok := world.TryResource[thing](a); ok {
		t.Fatal("the value itself is no resource")
	}
}
