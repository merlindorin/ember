package geom

// Vector is a vector curves work on: Vec2 or Vec3.
type Vector[V any] interface {
	Add(V) V
	Sub(V) V
	Scale(float32) V
}

// QuadBezier is the point at t, from 0 to 1, on the quadratic Bezier curve
// from a to c pulled towards b.
func QuadBezier[V Vector[V]](a, b, c V, t float32) V {
	u := 1 - t
	return a.Scale(u * u).Add(b.Scale(2 * u * t)).Add(c.Scale(t * t))
}

// CubicBezier is the point at t, from 0 to 1, on the cubic Bezier curve
// from a to d pulled towards b, then c.
func CubicBezier[V Vector[V]](a, b, c, d V, t float32) V {
	u := 1 - t
	return a.Scale(u * u * u).Add(b.Scale(3 * u * u * t)).Add(c.Scale(3 * u * t * t)).Add(d.Scale(t * t * t))
}

// CatmullRom is the point at t, from 0 to 1, on the smooth path through
// every one of points, each segment between two of them taking an equal
// share of t. It is the first point for t ≤ 0 and the last for t ≥ 1.
func CatmullRom[V Vector[V]](points []V, t float32) V {
	var zero V
	switch n := len(points); {
	case n == 0:
		return zero
	case n == 1 || t <= 0:
		return points[0]
	case t >= 1:
		return points[n-1]
	}
	n := len(points)
	at := t * float32(n-1)
	i := min(int(at), n-2)
	return catmullRom(points[max(i-1, 0)], points[i], points[i+1], points[min(i+2, n-1)], at-float32(i))
}

// ClosedCatmullRom is the point at t on the smooth loop through every one
// of points and back to the first: t wraps, so 0 and 1 are the first
// point.
func ClosedCatmullRom[V Vector[V]](points []V, t float32) V {
	var zero V
	n := len(points)
	if n == 0 {
		return zero
	}
	t -= Floor(t)
	at := t * float32(n)
	i := min(int(at), n-1)
	p := func(k int) V { return points[((k%n)+n)%n] }
	return catmullRom(p(i-1), p(i), p(i+1), p(i+2), at-float32(i))
}

// catmullRom is the point at s, from 0 to 1, between p1 and p2.
func catmullRom[V Vector[V]](p0, p1, p2, p3 V, s float32) V {
	a := p1.Scale(2)
	b := p2.Sub(p0)
	c := p0.Scale(2).Sub(p1.Scale(5)).Add(p2.Scale(4)).Sub(p3)
	d := p1.Scale(3).Sub(p0).Sub(p2.Scale(3)).Add(p3)
	return a.Add(b.Scale(s)).Add(c.Scale(s * s)).Add(d.Scale(s * s * s)).Scale(0.5)
}
