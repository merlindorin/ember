package twod //nolint:testpackage // white-box: shares twod's test helpers, reaches unexported state

import (
	"testing"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/internal/draw"
	"github.com/merlindorin/ember/world"
)

// drawnX is the middle of what the window drew, along X.
func drawnX(list *draw.List) float32 {
	lo, hi := corners(list.Unindexed())
	return (lo.X + hi.X) / 2
}

func TestWhatFixedStepsMoveIsDrawnBetweenThem(t *testing.T) {
	w, win, _ := app()
	hero := NewSprite(atlas(8, 8), At(100, 100))
	eye := NewCamera(Follow(hero))
	win.Add(hero, eye)
	teleport := false
	w.On(world.FixedUpdate, "walk", func(world.Frame) {
		hero.Translate(geom.V2(6, 0))
		if teleport {
			hero.SetPosition(geom.V2(500, 100))
			hero.Snap()
			teleport = false
		}
	})
	w.Tick(1.0 / 60)
	w.Frame(0)

	for _, c := range []struct {
		dt   float32
		want float32
	}{
		{1.0 / 144, 106 - 6 + 6*(1.0/144)/(1.0/60)},
		{1.0 / 144, 106 - 6 + 6*(2.0/144)/(1.0/60)},
		{1.0 / 144, 112 - 6 + 6*(3.0/144-1.0/60)/(1.0/60)},
	} {
		w.Frame(c.dt)
		at := hero.Position().X
		drawn := drawnGlobal(hero, w.Time().Alpha).Origin.X
		if geom.Abs(drawn-c.want) > 1e-3 {
			t.Fatalf("at %v, alpha %v: drawn at %v, want %v", at, w.Time().Alpha, drawn, c.want)
		}
		if geom.Abs(eye.Center().X-drawn) > 1e-3 {
			t.Fatalf("the camera follows where the hero is drawn: %v, want %v", eye.Center().X, drawn)
		}
	}

	teleport = true
	for hero.Position().X < 500 {
		w.Frame(1.0 / 144)
	}
	if x := drawnGlobal(hero, w.Time().Alpha).Origin.X; x != 500 {
		t.Fatalf("a snapped teleport draws where it lands, not on the way: %v", x)
	}
}

func TestWhatUpdateMovesIsDrawnWhereItIs(t *testing.T) {
	w, win, list := app()
	cursor := NewSprite(atlas(8, 8), At(100, 100))
	win.Add(cursor)
	w.On(world.Update, "follow the mouse", func(world.Frame) { cursor.Translate(geom.V2(10, 0)) })
	for range 5 {
		w.Frame(1.0 / 144)
	}
	if x := drawnX(list); geom.Abs(x-cursor.Position().X) > 1e-3 {
		t.Fatalf("drawn at %v, where it is: %v", x, cursor.Position().X)
	}
}

func TestTheDrawListShowsTheBlend(t *testing.T) {
	w, win, list := app()
	box := NewSprite(atlas(8, 8), At(100, 100))
	win.Add(box)
	w.On(world.FixedUpdate, "slide", func(world.Frame) { box.Translate(geom.V2(12, 0)) })
	w.Frame(1.0 / 60)
	w.Frame(1.0 / 120)
	if x := drawnX(list); geom.Abs(x-106) > 1e-3 {
		t.Fatalf("half way between 100 and the 112 the step left it at: %v", x)
	}
}

func TestOneThatStopsMovingIsDrawnWhereItStopped(t *testing.T) {
	w, win, _ := app()
	box := NewSprite(atlas(8, 8), At(100, 100))
	win.Add(box)
	steps := 0
	w.On(world.FixedUpdate, "slide", func(world.Frame) {
		if steps++; steps <= 2 {
			box.Translate(geom.V2(12, 0))
		}
	})
	for range 10 {
		w.Frame(1.0 / 144)
	}
	if x := drawnGlobal(box, w.Time().Alpha).Origin.X; x != 124 {
		t.Fatalf("stopped at 124, drawn at %v", x)
	}
}

func TestAFixedStepVisitsOnlyWhatItMoved(t *testing.T) {
	w, win, _ := app()
	var still []*Sprite
	for i := range 50 {
		s := NewSprite(atlas(8, 8), At(float32(i)*10, 0))
		still = append(still, s)
		win.Add(s)
	}
	mover := NewSprite(atlas(8, 8), At(0, 50))
	win.Add(mover)
	var clock *stepClock
	for _, s := range w.Systems() {
		if e, ok := s.(*stepEnd); ok {
			clock = e.clock
		}
	}
	moved := 0
	w.On(world.FixedUpdate, "slide", func(world.Frame) {
		mover.Translate(geom.V2(3, 0))
		moved = len(clock.moved)
	})
	w.Frame(1.0 / 60)
	if moved != 1 {
		t.Fatalf("the step lists %d moved placements, want the one", moved)
	}
	for _, s := range still {
		if s.stepped {
			t.Fatal("a placement nothing moved is never snapshotted")
		}
		if got, want := drawnGlobal(s, w.Time().Alpha), s.Local(); got != want {
			t.Fatalf("a still one is drawn where it is: %v, want %v", got, want)
		}
	}
}
