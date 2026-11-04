package geom

import "math"

// Circle is a disc in the plane.
type Circle struct {
	Center Vec2    `json:"Center"`
	Radius float32 `json:"Radius"`
}

// Contains reports whether p is inside the circle.
func (c Circle) Contains(p Vec2) bool {
	return p.Sub(c.Center).Dot(p.Sub(c.Center)) < c.Radius*c.Radius
}

// Overlaps reports whether the circles share some area; touching is not
// overlapping.
func (c Circle) Overlaps(o Circle) bool {
	r := c.Radius + o.Radius
	return c.Center.Sub(o.Center).Dot(c.Center.Sub(o.Center)) < r*r
}

// Overlaps reports whether the rectangles share some area; touching edges
// do not overlap, so a box can slide along a wall.
func (r Rect) Overlaps(o Rect) bool {
	return r.X < o.X+o.W && o.X < r.X+r.W && r.Y < o.Y+o.H && o.Y < r.Y+r.H
}

// OverlapsCircle reports whether the rectangle and the circle share some
// area.
func (r Rect) OverlapsCircle(c Circle) bool {
	nearest := V2(Clamp(c.Center.X, r.X, r.X+r.W), Clamp(c.Center.Y, r.Y, r.Y+r.H))
	return c.Contains(nearest)
}

// Sweep moves the rectangle moving by delta towards still and finds the
// first contact: the fraction of delta travelled, the normal pointing out
// of still's face that was hit, and whether it hits at all. A rectangle
// already overlapping still, or moving away along its face, does not hit
// it, so stuck bodies can part.
func Sweep(moving Rect, delta Vec2, still Rect) (float32, Vec2, bool) {
	half := V2(moving.W/2, moving.H/2)
	grown := Rect{X: still.X - half.X, Y: still.Y - half.Y, W: still.W + moving.W, H: still.H + moving.H}
	return ray(moving.Center(), delta, grown)
}

// SweepCircle is Sweep for a circle, against the rectangle with its
// corners rounded by the circle's radius, so it rolls past corners it only
// grazes.
func SweepCircle(c Circle, delta Vec2, still Rect) (float32, Vec2, bool) {
	r := c.Radius
	grown := Rect{X: still.X - r, Y: still.Y - r, W: still.W + 2*r, H: still.H + 2*r}
	t, normal, hit := ray(c.Center, delta, grown)
	if !hit {
		return 0, Vec2{}, false
	}
	at := c.Center.Add(delta.Scale(t))
	cx, cy := Clamp(at.X, still.X, still.X+still.W), Clamp(at.Y, still.Y, still.Y+still.H)
	if (at.X > still.X && at.X < still.X+still.W) || (at.Y > still.Y && at.Y < still.Y+still.H) {
		return t, normal, true
	}
	corner := V2(cx, cy)
	ct, ok := rayCircle(c.Center, delta, Circle{Center: corner, Radius: r})
	if !ok {
		return 0, Vec2{}, false
	}
	return ct, c.Center.Add(delta.Scale(ct)).Sub(corner).Normalize(), true
}

// ray finds where the point from, moving by delta, enters r.
func ray(from, delta Vec2, r Rect) (float32, Vec2, bool) {
	entry, exit := float32(math.Inf(-1)), float32(math.Inf(1))
	var normal Vec2
	axis := func(p, d, lo, hi float32, n Vec2) bool {
		if d == 0 {
			return p > lo && p < hi
		}
		t1, t2 := (lo-p)/d, (hi-p)/d
		if t1 > t2 {
			t1, t2 = t2, t1
		}
		if t1 > entry {
			entry, normal = t1, n
		}
		exit = min(exit, t2)
		return true
	}
	if !axis(from.X, delta.X, r.X, r.X+r.W, V2(-sign(delta.X), 0)) ||
		!axis(from.Y, delta.Y, r.Y, r.Y+r.H, V2(0, -sign(delta.Y))) {
		return 0, Vec2{}, false
	}
	if entry >= exit || entry < 0 || entry > 1 {
		return 0, Vec2{}, false
	}
	return entry, normal, true
}

// rayCircle finds where the point from, moving by delta, enters c.
func rayCircle(from, delta Vec2, c Circle) (float32, bool) {
	f := from.Sub(c.Center)
	a := delta.Dot(delta)
	b := 2 * f.Dot(delta)
	k := f.Dot(f) - c.Radius*c.Radius
	disc := b*b - 4*a*k
	if a == 0 || disc < 0 {
		return 0, false
	}
	t := (-b - float32(math.Sqrt(float64(disc)))) / (2 * a)
	if t < 0 || t > 1 {
		return 0, false
	}
	return t, true
}

func sign(v float32) float32 {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	}
	return 0
}
