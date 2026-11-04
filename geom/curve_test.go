package geom_test

import (
	"testing"

	"github.com/merlindorin/ember/geom"
)

func onCurve(a, b geom.Vec2) bool { return a.Dist(b) < 1e-4 }

func TestBeziersRunFromTheirFirstToTheirLastPoint(t *testing.T) {
	a, b, c, d := geom.V2(0, 0), geom.V2(0, 10), geom.V2(10, 10), geom.V2(10, 0)
	if !onCurve(geom.QuadBezier(a, b, d, 0), a) ||
		!onCurve(geom.QuadBezier(a, b, d, 1), d) ||
		!onCurve(geom.QuadBezier(a, b, d, 0.5), geom.V2(2.5, 5)) {
		t.Fatal("quadratic: ends, and halfway pulled up by the control point")
	}
	if !onCurve(geom.CubicBezier(a, b, c, d, 0), a) ||
		!onCurve(geom.CubicBezier(a, b, c, d, 1), d) ||
		!onCurve(geom.CubicBezier(a, b, c, d, 0.5), geom.V2(5, 7.5)) {
		t.Fatal("cubic: ends, and halfway")
	}
	got := geom.CubicBezier(geom.V3(0, 0, 0), geom.V3(0, 0, 3), geom.V3(0, 0, 6), geom.V3(0, 0, 9), 0.5)
	if got != geom.V3(0, 0, 4.5) {
		t.Fatalf("on Vec3 too: %v", got)
	}
}

func TestCatmullRomPassesThroughEveryPoint(t *testing.T) {
	path := []geom.Vec2{geom.V2(0, 0), geom.V2(10, 0), geom.V2(10, 10), geom.V2(20, 10)}
	for i, p := range path {
		if got := geom.CatmullRom(path, float32(i)/3); !onCurve(got, p) {
			t.Fatalf("point %d at t %v: %v", i, float32(i)/3, got)
		}
	}
	if geom.CatmullRom(path, -1) != path[0] || geom.CatmullRom(path, 2) != path[3] {
		t.Fatal("clamped outside 0..1")
	}
	if mid := geom.CatmullRom(path, 0.5); mid.X != 10 || mid.Y <= 0 || mid.Y >= 10 {
		t.Fatalf("between two points it curves smoothly: %v", mid)
	}
	if geom.CatmullRom([]geom.Vec2{geom.V2(3, 4)}, 0.5) != geom.V2(3, 4) ||
		geom.CatmullRom[geom.Vec2](nil, 0.5) != (geom.Vec2{}) {
		t.Fatal("one point or none")
	}
}

func TestAClosedCatmullRomLoopsBackSmoothly(t *testing.T) {
	square := []geom.Vec2{geom.V2(0, 0), geom.V2(10, 0), geom.V2(10, 10), geom.V2(0, 10)}
	for i, p := range square {
		if got := geom.ClosedCatmullRom(square, float32(i)/4); !onCurve(got, p) {
			t.Fatalf("point %d: %v", i, got)
		}
	}
	if !onCurve(geom.ClosedCatmullRom(square, 1), square[0]) || !onCurve(geom.ClosedCatmullRom(square, 1.25), square[1]) {
		t.Fatal("t wraps")
	}
	before, after := geom.ClosedCatmullRom(square, 0.999), geom.ClosedCatmullRom(square, 0.001)
	if d := before.Sub(geom.V2(0, 0)).Normalize().Dot(geom.V2(0, 0).Sub(after).Normalize()); d < 0.9 {
		t.Fatalf("no corner at the join: %v", d)
	}
}
