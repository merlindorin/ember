package render_test

import (
	"testing"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/render"
	"github.com/merlindorin/ember/scene"
)

func testView() render.View {
	t := scene.At(0, 20, 20).LookingAt(geom.Zero3)
	return render.Resolve(&render.Camera{}, t.Local(), geom.V2(800, 600), 800.0/600)
}

func TestScreenCenterRayHitsFocus(t *testing.T) {
	v := testView()
	hit, ok := v.ScreenRay(geom.V2(400, 300)).IntersectGround(0)
	if !ok || !hit.ApproxEq(geom.Zero3, 1e-2) {
		t.Fatalf("hit %v %v", hit, ok)
	}
	if !v.Focus.ApproxEq(geom.Zero3, 1e-3) {
		t.Fatalf("focus %v", v.Focus)
	}
}

func TestWorldToScreenRoundTrip(t *testing.T) {
	v := testView()
	p := geom.V3(3, 0, -2)
	s, ok := v.WorldToScreen(p)
	if !ok {
		t.Fatal("point should be visible")
	}
	hit, _ := v.ScreenRay(s).IntersectGround(0)
	if !hit.ApproxEq(p, 1e-2) {
		t.Fatalf("round trip %v -> %v -> %v", p, s, hit)
	}
	if _, seen := v.WorldToScreen(geom.V3(0, 40, 60)); seen {
		t.Fatal("point behind camera must be rejected")
	}
}

func TestOrthoProjection(t *testing.T) {
	tr := scene.At(0, 10, 0).LookingAt(geom.V3(0, 0, -0.001))
	v := render.Resolve(&render.Camera{OrthoHeight: 10}, tr.Local(), geom.V2(100, 100), 1)
	s, _ := v.WorldToScreen(geom.V3(5, 0, 0))
	if geom.Abs(s.X-100) > 0.5 {
		t.Fatalf("ortho edge at %v", s)
	}
}
