package world_test

import (
	"strings"
	"testing"

	"github.com/merlindorin/ember/world"
)

type stepper interface {
	Move(dx float32)
	Speed() float32
}

type strider struct{ x float32 }

func (w *strider) Move(dx float32) { w.x += dx }
func (w *strider) Speed() float32  { return 1 }

type wish struct{ Move float32 }

type explained struct {
	*world.BasicEntity
	*strider
	wish
}

type striders struct{ world.Group[stepper] }

func (*striders) Update(world.Frame) {}

func TestExplainSaysWhyAnEntityIsLeftOutOfAGroup(t *testing.T) {
	w := world.New()
	ws := &striders{}
	w.AddSystems(ws)
	h := &explained{world.NewBasic(), &strider{}, wish{}}
	w.AddEntities(h)
	if ws.Len() != 0 {
		t.Fatal("the field Move hides the promoted method, so the entity is left out")
	}
	got := w.Explain(h)
	for _, want := range []string{
		"out *world_test.striders: wants world_test.stepper",
		"Move is missing: a field named Move hides the method",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("%q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Speed") {
		t.Fatalf("Speed is fine:\n%s", got)
	}

	ok := &struct {
		*world.BasicEntity
		*strider
	}{world.NewBasic(), &strider{}}
	w.AddEntities(ok)
	if !strings.Contains(w.Explain(ok), "in  *world_test.striders") {
		t.Fatalf("a matching entity is in:\n%s", w.Explain(ok))
	}
}
