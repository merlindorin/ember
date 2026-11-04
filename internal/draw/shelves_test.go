package draw_test

import (
	"testing"

	"github.com/merlindorin/ember/internal/draw"

	"github.com/merlindorin/ember/geom"
)

func TestShelvesPackAlongShelvesThenOntoANewPage(t *testing.T) {
	got := draw.Shelves(
		[]geom.Vec2{{X: 60, Y: 40}, {X: 50, Y: 30}, {X: 30, Y: 50}, {X: 80, Y: 80}, {X: 100, Y: 20}, {X: 200, Y: 10}},
		100,
		nil,
	)
	want := []draw.Shelf{
		{0, geom.Rect{X: 0, Y: 0, W: 60, H: 40}},
		{0, geom.Rect{X: 0, Y: 40, W: 50, H: 30}},
		{0, geom.Rect{X: 50, Y: 40, W: 30, H: 50}},
		{1, geom.Rect{X: 0, Y: 0, W: 80, H: 80}},
		{1, geom.Rect{X: 0, Y: 80, W: 100, H: 20}},
		{2, geom.Rect{X: 0, Y: 0, W: 100, H: 10}},
	}
	if len(got) != len(want) {
		t.Fatalf("one shelf a size: %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("shelf %d: %v, want %v", i, got[i], want[i])
		}
	}
	overlap := func(a, b geom.Rect) bool { return a.X < b.X+b.W && b.X < a.X+a.W && a.Y < b.Y+b.H && b.Y < a.Y+a.H }
	for i, a := range got {
		for _, b := range got[i+1:] {
			if a.Page == b.Page && overlap(a.At, b.At) {
				t.Fatalf("%v and %v overlap", a, b)
			}
		}
	}
}
