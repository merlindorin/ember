package geom_test

import (
	"testing"

	"github.com/lantervale/ember/geom"
)

func near(a, b geom.Vec2) bool { return a.Sub(b).Len() < 1e-4 }

func TestTransform2DComposesAxesAndOrigin(t *testing.T) {
	m := geom.Transform2DFrom(geom.V2(10, 5), geom.Pi/2, 0, geom.V2(2, 3))
	if p := m.Xform(geom.V2(1, 0)); !near(p, geom.V2(10, 7)) {
		t.Fatalf("x axis rotated a quarter turn and scaled by 2: %v", p)
	}
	if p := m.Xform(geom.V2(0, 1)); !near(p, geom.V2(7, 5)) {
		t.Fatalf("y axis: %v", p)
	}
	if !near(m.Inverse().Xform(m.Xform(geom.V2(3, -4))), geom.V2(3, -4)) {
		t.Fatal("the inverse undoes the transform")
	}
	parent := geom.Transform2DFrom(geom.V2(100, 0), 0, 0, geom.V2(2, 2))
	child := geom.Transform2DFrom(geom.V2(5, 5), 0, 0, geom.V2(1, 1))
	if p := parent.Mul(child).Xform(geom.V2(1, 0)); !near(p, geom.V2(112, 10)) {
		t.Fatalf("a parent applies after its child: %v", p)
	}
	skewed := geom.Transform2DFrom(geom.Vec2{}, 0, geom.Pi/4, geom.V2(1, 1))
	if p := skewed.Xform(geom.V2(0, 1)); !near(p, geom.V2(-0.70710677, 0.70710677)) {
		t.Fatalf("skew turns the y axis only: %v", p)
	}
	if r, s := m.Rotation(), m.Scale(); geom.Abs(r-geom.Pi/2) > 1e-5 || !near(s, geom.V2(2, 3)) {
		t.Fatalf("rotation %v scale %v", r, s)
	}
	if geom.Identity2D().Xform(geom.V2(3, 4)) != geom.V2(3, 4) {
		t.Fatal("identity")
	}
}
