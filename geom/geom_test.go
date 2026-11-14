package geom_test

import (
	"testing"

	"github.com/lantervale/ember/geom"
)

const eps = 1e-4

func TestMat4MulIdentity(t *testing.T) {
	m := geom.TRS(geom.V3(1, 2, 3), geom.QuatAxisAngle(geom.Up, 0.7), geom.V3(2, 2, 2))
	if !m.Mul(geom.Ident()).ApproxEq(m, eps) || !geom.Ident().Mul(m).ApproxEq(m, eps) {
		t.Fatal("identity multiplication changed the matrix")
	}
}

func TestTRSAppliesScaleRotateTranslate(t *testing.T) {
	m := geom.TRS(geom.V3(10, 0, 0), geom.QuatAxisAngle(geom.Up, geom.Pi/2), geom.V3(2, 2, 2))
	got := m.MulPoint(geom.V3(1, 0, 0))
	if want := geom.V3(10, 0, -2); !got.ApproxEq(want, eps) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestInverse(t *testing.T) {
	m := geom.TRS(geom.V3(1, -2, 3), geom.QuatEuler(0.3, 1.1, -0.4), geom.V3(1, 2, 3))
	if !m.Mul(m.Inverse()).ApproxEq(geom.Ident(), eps) {
		t.Fatal("m * inverse(m) != identity")
	}
}

func TestLookAtMapsTargetToNegativeZ(t *testing.T) {
	v := geom.Mat4LookAt(geom.V3(0, 10, 10), geom.Zero3, geom.Up)
	p := v.MulPoint(geom.Zero3)
	if geom.Abs(p.X) > eps || geom.Abs(p.Y) > eps || p.Z >= 0 {
		t.Fatalf("target should be straight ahead on -Z, got %v", p)
	}
	if d := p.Len(); geom.Abs(d-geom.Sqrt(200)) > eps {
		t.Fatalf("distance mismatch %v", d)
	}
}

func TestPerspectiveDepthRange(t *testing.T) {
	p := geom.Perspective(geom.Pi/3, 1, 1, 100)
	if z := p.MulPoint(geom.V3(0, 0, -1)).Z; geom.Abs(z+1) > eps {
		t.Fatalf("near plane should map to -1, got %v", z)
	}
	if z := p.MulPoint(geom.V3(0, 0, -100)).Z; geom.Abs(z-1) > eps {
		t.Fatalf("far plane should map to 1, got %v", z)
	}
}

func TestQuatRotateMatchesMatrix(t *testing.T) {
	q := geom.QuatEuler(0.4, -1.2, 0.9)
	v := geom.V3(0.3, -2, 5)
	if a, b := q.Rotate(v), q.Mat4().MulDir(v); !a.ApproxEq(b, eps) {
		t.Fatalf("quat %v vs matrix %v", a, b)
	}
}

func TestQuatYawTurnsForwardTowardMinusX(t *testing.T) {
	if got := geom.QuatYaw(geom.Pi / 2).Rotate(geom.Forward); !got.ApproxEq(geom.Left, eps) {
		t.Fatalf("got %v", got)
	}
}

func TestMat4QuatRoundTrip(t *testing.T) {
	for _, q := range []geom.Quat{
		geom.QuatIdent(),
		geom.QuatEuler(0.1, 2.5, -0.3),
		geom.QuatAxisAngle(geom.Right, geom.Pi*0.99),
		geom.QuatAxisAngle(geom.V3(1, 1, 0), 2),
	} {
		if got := q.Mat4().Quat(); !got.ApproxEq(q, eps) {
			t.Fatalf("round trip %v -> %v", q, got)
		}
	}
}

func TestQuatLookRotation(t *testing.T) {
	dir := geom.V3(1, 0, 1).Normalize()
	if got := geom.QuatLookRotation(dir, geom.Up).Rotate(geom.Forward); !got.ApproxEq(dir, eps) {
		t.Fatalf("got %v want %v", got, dir)
	}
}

func TestSlerpEndpoints(t *testing.T) {
	a, b := geom.QuatIdent(), geom.QuatYaw(1.5)
	if !a.Slerp(b, 0).ApproxEq(a, eps) || !a.Slerp(b, 1).ApproxEq(b, eps) {
		t.Fatal("slerp endpoints wrong")
	}
	if !a.Slerp(b, 0.5).ApproxEq(geom.QuatYaw(0.75), eps) {
		t.Fatal("slerp midpoint wrong")
	}
}

func TestRayGround(t *testing.T) {
	r := geom.Ray{Origin: geom.V3(0, 10, 0), Dir: geom.V3(1, -1, 0).Normalize()}
	p, ok := r.IntersectGround(0)
	if !ok || !p.ApproxEq(geom.V3(10, 0, 0), eps) {
		t.Fatalf("got %v %v", p, ok)
	}
	if _, hits := (geom.Ray{Origin: geom.V3(0, 10, 0), Dir: geom.Up}).IntersectGround(0); hits {
		t.Fatal("ray pointing up must miss the ground")
	}
}

func TestRaySphere(t *testing.T) {
	r := geom.Ray{Origin: geom.V3(0, 0, 10), Dir: geom.Forward}
	d, ok := r.IntersectSphere(geom.Zero3, 1)
	if !ok || geom.Abs(d-9) > eps {
		t.Fatalf("got %v %v", d, ok)
	}
}

func TestAABBTransform(t *testing.T) {
	b := geom.AABB{geom.V3(-1, -1, -1), geom.V3(1, 1, 1)}
	got := b.Transform(geom.TRS(geom.V3(5, 0, 0), geom.QuatYaw(geom.Pi/4), geom.One3))
	h := geom.Sqrt(2)
	if !got.Min.ApproxEq(geom.V3(5-h, -1, -h), eps) || !got.Max.ApproxEq(geom.V3(5+h, 1, h), eps) {
		t.Fatalf("got %v", got)
	}
}

func TestFrustumCulling(t *testing.T) {
	vp := geom.Perspective(geom.Pi/2, 1, 0.1, 100).Mul(geom.Mat4LookAt(geom.Zero3, geom.Forward, geom.Up))
	f := geom.FrustumFromMatrix(vp)
	if !f.IntersectsSphere(geom.V3(0, 0, -10), 1) {
		t.Fatal("sphere in front must be visible")
	}
	if f.IntersectsSphere(geom.V3(0, 0, 10), 1) {
		t.Fatal("sphere behind must be culled")
	}
	if !f.IntersectsAABB(geom.AABBFromCenter(geom.V3(0, 0, -50), geom.One3)) ||
		f.IntersectsAABB(geom.AABBFromCenter(geom.V3(200, 0, -50), geom.One3)) {
		t.Fatal("aabb culling wrong")
	}
}

func TestWrapAndDampAngle(t *testing.T) {
	if a := geom.WrapAngle(3 * geom.Pi); geom.Abs(geom.Abs(a)-geom.Pi) > eps {
		t.Fatalf("wrap 3pi = %v", a)
	}
	got := geom.DampAngle(geom.Pi-0.1, -geom.Pi+0.1, 1000, 1)
	if geom.Abs(geom.WrapAngle(got-(-geom.Pi+0.1))) > eps {
		t.Fatalf("damp should take the short way, got %v", got)
	}
}

func TestRayAABB(t *testing.T) {
	box := geom.AABBFromCenter(geom.V3(0, 0, -10), geom.One3)
	if d, ok := (geom.Ray{Origin: geom.Zero3, Dir: geom.Forward}).IntersectAABB(box); !ok || geom.Abs(d-9) > eps {
		t.Fatalf("hit %v %v", d, ok)
	}
	if _, ok := (geom.Ray{Origin: geom.Zero3, Dir: geom.Up}).IntersectAABB(box); ok {
		t.Fatal("ray going up must miss")
	}
	if d, ok := (geom.Ray{Origin: geom.V3(0, 0, -10), Dir: geom.Right}).IntersectAABB(box); !ok || d != 0 {
		t.Fatal("ray from inside hits at 0")
	}
}

func TestQuatEulerRoundTrip(t *testing.T) {
	for _, e := range [][3]float32{{0, 0, 0}, {0.4, 1.2, -0.3}, {-1.1, -2.5, 0.8}, {0, geom.Pi * 0.9, 0}} {
		q := geom.QuatEuler(e[0], e[1], e[2])
		p, y, r := q.Euler()
		if !geom.QuatEuler(p, y, r).ApproxEq(q, eps) {
			t.Fatalf("euler %v -> (%v %v %v) does not rebuild the rotation", e, p, y, r)
		}
	}
}

func TestAnglesAndSquaredLengths(t *testing.T) {
	if geom.V2(3, 4).LenSq() != 25 || geom.V3(1, 2, 2).LenSq() != 9 {
		t.Fatal("squared lengths")
	}
	if a := geom.V2(0, 2).Angle(); geom.Abs(a-geom.Pi/2) > eps {
		t.Fatalf("+Y is a quarter turn from +X: %v", a)
	}
	if a := geom.V2(-1, 0).Angle(); geom.Abs(a-geom.Pi) > eps {
		t.Fatalf("-X is half a turn: %v", a)
	}
	for _, a := range []float32{0, 1, -2.5, 3} {
		v := geom.FromAngle(a)
		if geom.Abs(v.Len()-1) > eps || geom.Abs(v.Angle()-a) > eps {
			t.Fatalf("FromAngle(%v) is a unit vector at that angle: %v", a, v)
		}
	}
}
