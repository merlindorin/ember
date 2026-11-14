package collide //nolint:testpackage // white-box: shares the unexported thing, setup and walls helpers

import (
	"fmt"
	"slices"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/world"
)

type thing struct {
	*world.BasicEntity
	at geom.Vec2
}

func (t *thing) Position() geom.Vec2     { return t.at }
func (t *thing) SetPosition(p geom.Vec2) { t.at = p }

func newThing(x, y float32, body *Body) *thing {
	t := &thing{BasicEntity: world.NewBasic(), at: geom.V2(x, y)}
	t.AppendChild(body)
	return t
}

type walls []geom.Rect

func (w walls) SolidIn(area geom.Rect) []geom.Rect {
	var out []geom.Rect
	for _, r := range w {
		if r.Overlaps(area) {
			out = append(out, r)
		}
	}
	return out
}

func setup(things ...*thing) (*world.World, *Space) {
	w := world.New()
	space := NewSpace(Grid(32))
	w.AddSystems(NewTracker(space))
	for _, t := range things {
		w.AddEntities(t)
	}
	return w, space
}

func TestMoveStopsAtSolidBodiesAndSlidesAlongThem(t *testing.T) {
	hero := newThing(0, 0, NewBody(Box(10, 10)))
	wallBody := NewBody(Box(10, 100))
	wall := newThing(30, 0, wallBody)
	_, space := setup(hero, wall)
	moved, hits := space.Move(hero, geom.V2(40, 0))
	if len(hits) != 1 || hits[0].Body != wallBody || hits[0].Normal != geom.V2(-1, 0) {
		t.Fatalf("hits %+v", hits)
	}
	if geom.Abs(hero.at.X-20) > 0.01 || geom.Abs(moved.X-20) > 0.01 {
		t.Fatalf("its right edge stops on the wall's face at 25: %v", hero.at)
	}
	space.Move(hero, geom.V2(10, 20))
	if geom.Abs(hero.at.X-20) > 0.01 || geom.Abs(hero.at.Y-20) > 0.01 {
		t.Fatalf("pushing into the wall slides along it: %v", hero.at)
	}
	if wallBody.Owner() != wall {
		t.Fatal("a body knows its owner")
	}
}

func TestSolidsStopBodiesToo(t *testing.T) {
	hero := newThing(0, 0, NewBody(Circle(5), Offset(0, 2)))
	_, space := setup(hero)
	space.AddSolid(walls{{X: -100, Y: 20, W: 200, H: 10}})
	_, hits := space.Move(hero, geom.V2(0, 50))
	if len(hits) != 1 || hits[0].Body != nil || hits[0].Normal != geom.V2(0, -1) {
		t.Fatalf("a solid's rectangle: %+v", hits)
	}
	if geom.Abs(hero.at.Y-13) > 0.01 {
		t.Fatalf("the circle, 2 below its owner, rests on the floor at 20: owner at %v", hero.at)
	}
}

func TestLayersDecideWhatMeets(t *testing.T) {
	ghost := newThing(0, 0, NewBody(Box(10, 10), Group(1), CollidesWith(0)))
	glass := newThing(30, 0, NewBody(Box(10, 10), Group(2)))
	_, space := setup(ghost, glass)
	if _, hits := space.Move(ghost, geom.V2(60, 0)); len(hits) != 0 || ghost.at.X != 60 {
		t.Fatalf("a body only meets the layers in its mask: %v %+v", ghost.at, hits)
	}
}

func TestTriggersAreReportedNotBlocking(t *testing.T) {
	hero := newThing(0, 0, NewBody(Box(10, 10)))
	doorBody := NewBody(Box(20, 20), Trigger())
	door := newThing(40, 0, doorBody)
	w, space := setup(hero, door)
	var entered, left []BodyEntered
	var lefts world.Reader[BodyLeft]
	var enters world.Reader[BodyEntered]
	w.On(world.Update, "read", func(f world.Frame) {
		entered = append(entered, collect(enters.Read(f.World))...)
		for e := range lefts.Read(f.World) {
			left = append(left, BodyEntered(e))
		}
	})
	_, hits := space.Move(hero, geom.V2(40, 0))
	if hero.at.X != 40 || len(hits) != 1 || !hits[0].Trigger || hits[0].Body != doorBody {
		t.Fatalf("walking into a trigger: %v %+v", hero.at, hits)
	}
	w.RunFrames(2)
	if len(entered) != 1 || entered[0].Trigger != doorBody {
		t.Fatalf("entered %+v", entered)
	}
	space.Move(hero, geom.V2(-40, 0))
	w.RunFrames(2)
	if len(left) != 1 {
		t.Fatalf("left %+v", left)
	}
}

func collect(seq func(func(BodyEntered) bool)) []BodyEntered {
	var out []BodyEntered
	seq(func(e BodyEntered) bool { out = append(out, e); return true })
	return out
}

func TestQueries(t *testing.T) {
	aBody, bBody := NewBody(Box(10, 10)), NewBody(Circle(5))
	_, space := setup(newThing(0, 0, aBody), newThing(50, 0, bBody))
	space.AddSolid(walls{{X: 100, Y: -50, W: 10, H: 100}})
	if at := space.At(geom.V2(3, 3)); len(at) != 1 || at[0] != aBody {
		t.Fatalf("At: %v", at)
	}
	if over := space.Overlapping(Box(60, 4), geom.V2(25, 0)); len(over) != 2 {
		t.Fatalf("a 60-wide box between them overlaps both: %v", over)
	}
	hit, ok := space.Cast(geom.V2(20, 0), geom.V2(200, 0))
	if !ok || hit.Body != bBody {
		t.Fatalf("the ray meets the circle's bounds first: %+v", hit)
	}
	hit, ok = space.Cast(geom.V2(80, 0), geom.V2(200, 0))
	if !ok || hit.Body != nil || geom.Abs(hit.Time-20.0/120) > 1e-4 {
		t.Fatalf("then the solid: %+v", hit)
	}
}

func TestRemovingAnOwnerTakesItsBodyOutOfTheSpace(t *testing.T) {
	hero := newThing(0, 0, NewBody(Box(10, 10)))
	wall := newThing(30, 0, NewBody(Box(10, 100)))
	w, space := setup(hero, wall)
	w.RemoveEntity(wall)
	if _, hits := space.Move(hero, geom.V2(60, 0)); len(hits) != 0 || hero.at.X != 60 {
		t.Fatalf("the wall is gone: %+v", hits)
	}
	free := &thing{BasicEntity: world.NewBasic()}
	if moved, hits := space.Move(free, geom.V2(5, 5)); moved != geom.V2(5, 5) || hits != nil {
		t.Fatal("an owner without a body moves freely")
	}
}

func TestOptionsReadBack(t *testing.T) {
	b := NewBody(Circle(4), Offset(1, 2), Trigger(), Group(3), CollidesWith(0, 5))
	if b.Offset() != geom.V2(1, 2) || !b.Trigger() || b.Group() != 3 || len(b.CollidesWith()) != 2 ||
		b.CollidesWith()[1] != 5 {
		t.Fatalf("offset %v trigger %v group %d collides with %v", b.Offset(), b.Trigger(), b.Group(), b.CollidesWith())
	}
	if d := NewBody(Box(1, 1)); d.Group() != 0 || len(d.CollidesWith()) != 32 {
		t.Fatal("group 0, colliding with every group, by default")
	}
	if NewSpace(Grid(32)).Grid() != 32 || NewSpace().Grid() != 64 {
		t.Fatal("the grid's cell")
	}
}

func TestTriggersFireInTheFixedStepThatCrossesThem(t *testing.T) {
	hero := newThing(0, 0, NewBody(Box(10, 10)))
	plate := newThing(40, 0, NewBody(Box(20, 20), Trigger()))
	w, space := setup(hero, plate)
	var steps []string
	var enters world.Reader[BodyEntered]
	var lefts world.Reader[BodyLeft]
	step := 0
	w.On(world.FixedUpdate, "walk", func(world.Frame) {
		step++
		switch step {
		case 1:
			space.Move(hero, geom.V2(40, 0))
		case 2:
			space.Move(hero, geom.V2(-40, 0))
		}
	})
	w.AddSystems(&after{func(f world.Frame) {
		for range enters.Read(f.World) {
			steps = append(steps, fmt.Sprintf("entered in step %d", step))
		}
		for range lefts.Read(f.World) {
			steps = append(steps, fmt.Sprintf("left in step %d", step))
		}
	}})
	var seen int
	var late world.Reader[BodyEntered]
	w.On(world.Update, "show", func(f world.Frame) {
		for range late.Read(f.World) {
			seen++
		}
	})
	fixed := w.Time().FixedDelta
	w.Frame(fixed / 4)
	if step != 0 || len(steps) != 0 {
		t.Fatalf("a frame with no fixed step moves and reports nothing: %d %v", step, steps)
	}
	w.Frame(fixed*2 - fixed/4)
	w.Frame(fixed / 4)
	if step != 2 {
		t.Fatalf("two steps in one frame: %d", step)
	}
	want := []string{"entered in step 1", "left in step 2"}
	if !slices.Equal(steps, want) {
		t.Fatalf("on and off the plate within a frame, each step reported: %v, want %v", steps, want)
	}
	if seen != 1 {
		t.Fatalf("Update still sees what the steps reported: %d", seen)
	}
}

// after reads what the tracker reported, after it in each fixed step.
type after struct{ fn func(world.Frame) }

func (a *after) Phase() world.Phase   { return world.FixedUpdate }
func (a *after) Priority() int        { return -2000 }
func (a *after) Update(f world.Frame) { a.fn(f) }
