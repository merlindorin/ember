package collide //nolint:testpackage // white-box: shares the unexported thing, setup and walls helpers

import (
	"testing"

	"github.com/lantervale/ember/geom"
)

// barMap is a bar 4 points wide, x 16 to 20, all the way down three rows
// of 16-point cells, one rect a row, as a tile map's thin bar tiles stop
// bodies.
func barMap() walls {
	return walls{{X: 16, W: 4, H: 16}, {X: 16, Y: 16, W: 4, H: 16}, {X: 16, Y: 32, W: 4, H: 16}}
}

func TestABodyStopsAtABarNotItsCell(t *testing.T) {
	for _, shape := range []Shape{Box(6, 6), Circle(3)} {
		left, right := newThing(4, 24, NewBody(shape)), newThing(28, 24, NewBody(shape))
		_, space := setup(left, right)
		space.AddSolid(barMap())
		space.Move(left, geom.V2(40, 0))
		if geom.Abs(left.at.X-13) > 0.01 {
			t.Errorf("%T: stops with its edge on the bar's face at 16: %v", shape, left.at)
		}
		space.Move(right, geom.V2(-40, 0))
		if geom.Abs(right.at.X-23) > 0.01 {
			t.Errorf("%T: walks into the bar's cell up to its far face at 20: %v", shape, right.at)
		}
	}
}

func TestABodyNeverGoesThroughABar(t *testing.T) {
	hero := newThing(4, 24, NewBody(Box(6, 6)))
	_, space := setup(hero)
	space.AddSolid(barMap())
	for range 50 {
		space.Move(hero, geom.V2(7, 0))
		if hero.at.X > 13.01 {
			t.Fatalf("through the bar: %v", hero.at)
		}
	}
}

func TestABodySlidesAlongABar(t *testing.T) {
	hero := newThing(4, 16, NewBody(Box(6, 6)))
	_, space := setup(hero)
	space.AddSolid(barMap())
	space.Move(hero, geom.V2(40, 12))
	if geom.Abs(hero.at.X-13) > 0.01 || geom.Abs(hero.at.Y-28) > 0.01 {
		t.Fatalf("stopped across, slid along: %v", hero.at)
	}
	space.Move(hero, geom.V2(0, -20))
	if geom.Abs(hero.at.Y-8) > 0.01 {
		t.Fatalf("free along its face: %v", hero.at)
	}
}

func TestClearSeesPastBodiesButNotThroughASolid(t *testing.T) {
	hero := newThing(28, 24, NewBody(Box(6, 6)))
	_, space := setup(hero)
	space.AddSolid(barMap())
	if space.Clear(geom.V2(4, 24), geom.V2(40, 24)) {
		t.Fatal("across the bar")
	}
	if !space.Clear(geom.V2(22, 4), geom.V2(40, 44)) {
		t.Fatal("beside the bar, past a body")
	}
}

func TestSolidsAndBodiesListWhatBlocksAnArea(t *testing.T) {
	hero := newThing(4, 24, NewBody(Box(6, 6)))
	far := newThing(400, 400, NewBody(Box(6, 6)))
	_, space := setup(hero, far)
	space.AddSolid(barMap())
	got := space.Solids(geom.Rect{X: 0, Y: 0, W: 40, H: 20})
	if len(got) == 0 || got[0].W != 4 {
		t.Fatalf("the bar's shapes in the area, 4 wide: %v", got)
	}
	bodies := space.Bodies(geom.Rect{X: 0, Y: 0, W: 40, H: 40})
	if len(bodies) != 1 || bodies[0].Owner() != hero {
		t.Fatalf("the hero's body alone: %v", bodies)
	}
}

func TestABodyPlacedInAShapeNeverWalksDeeper(t *testing.T) {
	b := newThing(15, 24, NewBody(Box(6, 6)))
	_, space := setup(b)
	space.AddSolid(barMap())
	space.Move(b, geom.V2(3, 0))
	if b.at.X > 15 {
		t.Fatalf("overlapping the bar, it went on in: %v", b.at)
	}
	space.Move(b, geom.V2(0, 5))
	if geom.Abs(b.at.Y-29) > 0.01 {
		t.Fatalf("it still slides along: %v", b.at)
	}
	space.Move(b, geom.V2(-3, 0))
	if geom.Abs(b.at.X-12) > 0.01 {
		t.Fatalf("and steps out: %v", b.at)
	}
}
