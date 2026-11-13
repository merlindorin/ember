package collide //nolint:testpackage // white-box: shares the thing, setup and walls helpers

import (
	"testing"

	"github.com/merlindorin/ember/geom"
)

// appending walls answer through AppendSolidIn, and count what is asked.
type appending struct {
	walls
	listed, appended int
}

func (a *appending) SolidIn(area geom.Rect) []geom.Rect {
	a.listed++
	return a.walls.SolidIn(area)
}

func (a *appending) AppendSolidIn(dst []geom.Rect, area geom.Rect) []geom.Rect {
	a.appended++
	for _, r := range a.walls {
		if r.Overlaps(area) {
			dst = append(dst, r)
		}
	}
	return dst
}

func TestAMoveAsksAnAppendingSolidToAppend(t *testing.T) {
	hero := newThing(0, 0, NewBody(Box(10, 10)))
	_, space := setup(hero)
	floor := &appending{walls: walls{{X: -100, Y: 20, W: 200, H: 10}}}
	space.AddSolid(floor)
	_, hits := space.Move(hero, geom.V2(0, 50))
	if len(hits) != 1 || hits[0].Normal != geom.V2(0, -1) {
		t.Fatalf("the appended floor stops the body as a listed one does: %+v", hits)
	}
	if floor.appended == 0 || floor.listed != 0 {
		t.Fatalf("asked %d times to append, %d to list", floor.appended, floor.listed)
	}
	space.Move(hero, geom.V2(0, -20))
	if n := testing.AllocsPerRun(100, func() { space.Move(hero, geom.V2(3, 0)) }); n != 0 {
		t.Fatalf("a move among appending solids allocates %v times", n)
	}
}
