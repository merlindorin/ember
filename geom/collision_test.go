package geom_test

import (
	"testing"

	"github.com/lantervale/ember/geom"
)

func TestRectsAndCirclesOverlapButTouchingDoesNot(t *testing.T) {
	a := geom.Rect{W: 10, H: 10}
	if !a.Overlaps(geom.Rect{X: 5, Y: 5, W: 10, H: 10}) || a.Overlaps(geom.Rect{X: 10, W: 10, H: 10}) {
		t.Fatal("rects share area; touching edges do not")
	}
	c := geom.Circle{Center: geom.V2(0, 0), Radius: 5}
	if !c.Overlaps(geom.Circle{Center: geom.V2(9, 0), Radius: 5}) ||
		c.Overlaps(geom.Circle{Center: geom.V2(10, 0), Radius: 5}) {
		t.Fatal("circles")
	}
	if !c.Contains(geom.V2(3, 3)) || c.Contains(geom.V2(4, 4)) {
		t.Fatal("a point in a circle")
	}
	if !a.OverlapsCircle(geom.Circle{Center: geom.V2(12, 5), Radius: 3}) ||
		a.OverlapsCircle(geom.Circle{Center: geom.V2(13, 13), Radius: 4}) {
		t.Fatal("a rect and a circle: by the side yes, past the corner no")
	}
}

func TestSweepFindsTheFirstContactAndItsFace(t *testing.T) {
	wall := geom.Rect{X: 20, Y: -50, W: 10, H: 100}
	box := geom.Rect{W: 10, H: 10}
	at, n, hit := geom.Sweep(box, geom.V2(20, 0), wall)
	if !hit || at != 0.5 || n != geom.V2(-1, 0) {
		t.Fatalf("halfway along, on the wall's left face: %v %v %v", at, n, hit)
	}
	if _, _, ok := geom.Sweep(box, geom.V2(5, 0), wall); ok {
		t.Fatal("stopping short is no hit")
	}
	if _, _, ok := geom.Sweep(box, geom.V2(0, 30), geom.Rect{X: 10, Y: 0, W: 10, H: 100}); ok {
		t.Fatal("sliding along a face it touches is no hit")
	}
	if _, _, ok := geom.Sweep(geom.Rect{X: 22, W: 4, H: 4}, geom.V2(-10, 0), wall); ok {
		t.Fatal("a box already inside moves out freely")
	}
	at, n, hit = geom.Sweep(box, geom.V2(0, 40), geom.Rect{X: -5, Y: 30, W: 20, H: 5})
	if !hit || at != 0.5 || n != geom.V2(0, -1) {
		t.Fatalf("falling onto a floor: %v %v %v", at, n, hit)
	}
}

func TestSweepCircleRollsPastCornersItGrazes(t *testing.T) {
	block := geom.Rect{X: 10, Y: 10, W: 10, H: 10}
	at, n, hit := geom.SweepCircle(geom.Circle{Center: geom.V2(0, 15), Radius: 2}, geom.V2(20, 0), block)
	if !hit || geom.Abs(at-0.4) > 1e-5 || n != geom.V2(-1, 0) {
		t.Fatalf("face on: %v %v %v", at, n, hit)
	}
	if _, _, ok := geom.SweepCircle(geom.Circle{Center: geom.V2(0, 16.46), Radius: 2}, geom.V2(12, -12), block); ok {
		t.Fatal("a circle passing 2.5 from the corner crosses the bounds' square corner but misses the block")
	}
	at, n, hit = geom.SweepCircle(geom.Circle{Center: geom.V2(0, 0), Radius: 2}, geom.V2(20, 20), block)
	if !hit || n.X >= 0 || n.Y >= 0 || geom.Abs(n.Len()-1) > 1e-4 {
		t.Fatalf("diagonally into the corner, the normal points out of it: %v %v %v", at, n, hit)
	}
}
