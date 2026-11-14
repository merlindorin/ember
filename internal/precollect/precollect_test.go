package precollect_test

import (
	"testing"

	"github.com/lantervale/ember/internal/precollect"
	"github.com/lantervale/ember/world"
)

func TestHooksRunInOrderPerWorld(t *testing.T) {
	a, b := world.New(), world.New()
	var got []string
	precollect.Add(a, func(*world.World) { got = append(got, "first") })
	precollect.Add(a, func(*world.World) { got = append(got, "second") })
	precollect.Run(a)
	precollect.Run(b)
	if len(got) != 2 || got[0] != "first" || got[1] != "second" {
		t.Fatalf("a's hooks, in order, and none of b's: %v", got)
	}
}
