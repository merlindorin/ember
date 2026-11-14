package collide //nolint:testpackage // white-box: shares the unexported thing, setup and walls helpers

import (
	"testing"

	"github.com/lantervale/ember/geom"
)

// chase moves e towards hero for steps fixed steps of one point, aiming
// straight at him each step as the demo's Chase does.
func chase(space *Space, e *thing, hero geom.Vec2, steps int) {
	for range steps {
		space.Move(e, hero.Sub(e.at).Normalize())
	}
}

// Aiming straight at the hero through a solid, an enemy slides along the
// solid until it faces the hero head on, and stops there: the slide keeps
// only the motion along the solid, and head on there is none. Going round
// is steering's job, not collision's.
func TestAnEnemyAimingThroughASolidSlidesToFaceItAndStops(t *testing.T) {
	column := walls{{X: 40, Y: -16, W: 32, H: 32}}
	for _, heroY := range []float32{0, 2, 12} {
		e := newThing(0, 0, NewBody(Box(16, 10)))
		_, space := setup(e)
		space.AddSolid(column)
		chase(space, e, geom.V2(120, heroY), 600)
		if want := geom.V2(40-8-1e-3, heroY); e.at.Sub(want).Len() > 0.05 {
			t.Errorf("hero at y %v: the enemy ends at %v, facing him against the column at %v", heroY, e.at, want)
		}
	}
	e := newThing(0, 0, NewBody(Box(16, 10)))
	_, space := setup(e)
	space.AddSolid(column)
	chase(space, e, geom.V2(120, 30), 600)
	if e.at.X < 72 {
		t.Errorf("a hero past the column's end draws the enemy round it: %v", e.at)
	}
}

func TestEnemiesInACorridorStopAtWhatIsAheadOfThem(t *testing.T) {
	corridor := walls{{X: -100, Y: -20, W: 400, H: 8}, {X: -100, Y: 12, W: 400, H: 8}, {X: 100, Y: -12, W: 16, H: 24}}
	front := newThing(40, 0, NewBody(Box(16, 10)))
	back := newThing(0, 0, NewBody(Box(16, 10)))
	_, space := setup(front, back)
	space.AddSolid(corridor)
	for range 300 {
		for _, e := range []*thing{front, back} {
			space.Move(e, geom.V2(200, 0).Sub(e.at).Normalize())
		}
	}
	if front.at.X < 91.9 || back.at.X < 75.9 {
		t.Errorf("the front one against the block, the back one against it: %v %v", front.at, back.at)
	}
}

func TestACrowdAimingThroughASolidPilesAgainstIt(t *testing.T) {
	column := walls{{X: 80, Y: -40, W: 32, H: 80}}
	var crowd []*thing
	for i := range 9 {
		crowd = append(crowd, newThing(0, float32(i-4)*12, NewBody(Box(16, 10))))
	}
	_, space := setup(crowd...)
	space.AddSolid(column)
	for range 900 {
		for _, e := range crowd {
			space.Move(e, geom.V2(200, 0).Sub(e.at).Normalize())
		}
	}
	for _, e := range crowd {
		if e.at.X > 72 {
			t.Errorf("every one stays against the column's face, none goes round: %v", e.at)
		}
	}
}
