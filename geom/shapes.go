package geom

type Ray struct {
	Origin Vec3 `json:"Origin"`
	Dir    Vec3 `json:"Dir"`
}

func (r Ray) At(t float32) Vec3 { return r.Origin.Add(r.Dir.Scale(t)) }

// IntersectPlane returns the distance along the ray to the plane defined by
// a point and a normal. ok is false when the ray is parallel or points away.
func (r Ray) IntersectPlane(point, normal Vec3) (float32, bool) {
	denom := normal.Dot(r.Dir)
	if Abs(denom) < Epsilon {
		return 0, false
	}
	t := point.Sub(r.Origin).Dot(normal) / denom
	return t, t >= 0
}

// IntersectGround intersects the horizontal plane y = height.
func (r Ray) IntersectGround(height float32) (Vec3, bool) {
	t, ok := r.IntersectPlane(Vec3{0, height, 0}, Up)
	if !ok {
		return Vec3{}, false
	}
	return r.At(t), true
}

// IntersectSphere returns the nearest non-negative hit distance.
func (r Ray) IntersectSphere(center Vec3, radius float32) (float32, bool) {
	oc := r.Origin.Sub(center)
	b := oc.Dot(r.Dir)
	c := oc.Dot(oc) - radius*radius
	disc := b*b - c
	if disc < 0 {
		return 0, false
	}
	sq := Sqrt(disc)
	if t := -b - sq; t >= 0 {
		return t, true
	}
	if t := -b + sq; t >= 0 {
		return t, true
	}
	return 0, false
}

// IntersectAABB returns the distance to the nearest hit of the box, or 0
// when the ray starts inside it.
func (r Ray) IntersectAABB(b AABB) (float32, bool) {
	tmin, tmax := float32(0), float32(1e30)
	o, d := [3]float32{r.Origin.X, r.Origin.Y, r.Origin.Z}, [3]float32{r.Dir.X, r.Dir.Y, r.Dir.Z}
	lo, hi := [3]float32{b.Min.X, b.Min.Y, b.Min.Z}, [3]float32{b.Max.X, b.Max.Y, b.Max.Z}
	for i := 0; i < 3; i++ {
		if Abs(d[i]) < Epsilon {
			if o[i] < lo[i] || o[i] > hi[i] {
				return 0, false
			}
			continue
		}
		t1, t2 := (lo[i]-o[i])/d[i], (hi[i]-o[i])/d[i]
		if t1 > t2 {
			t1, t2 = t2, t1
		}
		tmin, tmax = max(tmin, t1), min(tmax, t2)
		if tmin > tmax {
			return 0, false
		}
	}
	return tmin, true
}

// AABB is an axis-aligned bounding box.
type AABB struct {
	Min Vec3 `json:"Min"`
	Max Vec3 `json:"Max"`
}

func AABBFromCenter(center, halfExtents Vec3) AABB {
	return AABB{center.Sub(halfExtents), center.Add(halfExtents)}
}

func (b AABB) Center() Vec3      { return b.Min.Add(b.Max).Scale(0.5) }
func (b AABB) Size() Vec3        { return b.Max.Sub(b.Min) }
func (b AABB) Union(o AABB) AABB { return AABB{b.Min.Min(o.Min), b.Max.Max(o.Max)} }

func (b AABB) Contains(p Vec3) bool {
	return p.X >= b.Min.X && p.X <= b.Max.X && p.Y >= b.Min.Y && p.Y <= b.Max.Y && p.Z >= b.Min.Z && p.Z <= b.Max.Z
}

func (b AABB) Intersects(o AABB) bool {
	return b.Min.X <= o.Max.X && b.Max.X >= o.Min.X &&
		b.Min.Y <= o.Max.Y && b.Max.Y >= o.Min.Y &&
		b.Min.Z <= o.Max.Z && b.Max.Z >= o.Min.Z
}

// Transform returns the bounds of the box after applying m.
func (b AABB) Transform(m Mat4) AABB {
	c := m.MulPoint(b.Center())
	e := b.Size().Scale(0.5)
	var h Vec3
	h.X = Abs(m[0])*e.X + Abs(m[4])*e.Y + Abs(m[8])*e.Z
	h.Y = Abs(m[1])*e.X + Abs(m[5])*e.Y + Abs(m[9])*e.Z
	h.Z = Abs(m[2])*e.X + Abs(m[6])*e.Y + Abs(m[10])*e.Z
	return AABB{c.Sub(h), c.Add(h)}
}

// Rect is a 2D rectangle in screen space (origin top-left).
type Rect struct {
	X float32 `json:"X"`
	Y float32 `json:"Y"`
	W float32 `json:"W"`
	H float32 `json:"H"`
}

func (r Rect) Contains(p Vec2) bool {
	return p.X >= r.X && p.X <= r.X+r.W && p.Y >= r.Y && p.Y <= r.Y+r.H
}

func (r Rect) Center() Vec2 { return Vec2{r.X + r.W/2, r.Y + r.H/2} }

// Inset shrinks the rectangle by d on every side.
func (r Rect) Inset(d float32) Rect { return Rect{r.X + d, r.Y + d, r.W - 2*d, r.H - 2*d} }

// Frustum holds six normalized planes (a,b,c,d) with normals pointing inward.
type Frustum [6]Vec4

// FrustumFromMatrix extracts planes from a view-projection matrix.
func FrustumFromMatrix(m Mat4) Frustum {
	row := func(i int) Vec4 { return Vec4{m[i], m[4+i], m[8+i], m[12+i]} }
	r0, r1, r2, r3 := row(0), row(1), row(2), row(3)
	planes := Frustum{
		r3.Add(r0), r3.Add(r0.Scale(-1)),
		r3.Add(r1), r3.Add(r1.Scale(-1)),
		r3.Add(r2), r3.Add(r2.Scale(-1)),
	}
	for i, p := range planes {
		l := Sqrt(p.X*p.X + p.Y*p.Y + p.Z*p.Z)
		planes[i] = p.Scale(1 / l)
	}
	return planes
}

// IntersectsAABB reports whether the box is at least partially inside.
func (f Frustum) IntersectsAABB(b AABB) bool {
	for _, p := range f {
		v := b.Min
		if p.X >= 0 {
			v.X = b.Max.X
		}
		if p.Y >= 0 {
			v.Y = b.Max.Y
		}
		if p.Z >= 0 {
			v.Z = b.Max.Z
		}
		if p.X*v.X+p.Y*v.Y+p.Z*v.Z+p.W < 0 {
			return false
		}
	}
	return true
}

// IntersectsSphere reports whether the sphere is at least partially inside.
func (f Frustum) IntersectsSphere(c Vec3, r float32) bool {
	for _, p := range f {
		if p.X*c.X+p.Y*c.Y+p.Z*c.Z+p.W < -r {
			return false
		}
	}
	return true
}
