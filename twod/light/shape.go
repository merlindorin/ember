package light

import (
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/internal/draw"
)

// Shape is what a shape light is full inside, in the light's own space.
type Shape struct {
	kind   draw.ShapeKind
	size   geom.Vec2
	points []geom.Vec2
}

// Rect is a w×h rectangle centred on the light.
func Rect(w, h float32) Shape { return Shape{kind: draw.RectShape, size: geom.V2(w, h)} }

// Circle is a circle of radius r round the light.
func Circle(r float32) Shape {
	return Shape{kind: draw.CircleShape, size: geom.V2(r, r), points: []geom.Vec2{{}}}
}

// Capsule is every point within r of the line from a to b.
func Capsule(a, b geom.Vec2, r float32) Shape {
	return Shape{kind: draw.CapsuleShape, size: geom.V2(r, r), points: []geom.Vec2{a, b}}
}

// Line is the line from a to b, a light's thinnest shape: a neon tube.
func Line(a, b geom.Vec2) Shape {
	return Shape{kind: draw.LineShape, points: []geom.Vec2{a, b}}
}

// Polygon is the polygon of points, in order, up to MaxPolygonPoints of
// them: a window's light on the floor.
func Polygon(points ...geom.Vec2) Shape {
	return Shape{
		kind:   draw.PolygonShape,
		points: append([]geom.Vec2(nil), points[:min(len(points), MaxPolygonPoints)]...),
	}
}

// MaxPolygonPoints is how many corners a polygon light has at most.
const MaxPolygonPoints = 32

// outline is points along the shape's edge, n of them at most, in its
// space: where a shape light casts its shadows from.
func (s Shape) outline(n int) []geom.Vec2 {
	var ring []geom.Vec2
	switch s.kind {
	case draw.RectShape:
		hw, hh := s.size.X/2, s.size.Y/2
		ring = []geom.Vec2{{X: -hw, Y: -hh}, {X: hw, Y: -hh}, {X: hw, Y: hh}, {X: -hw, Y: hh}}
	case draw.CircleShape, draw.CapsuleShape, draw.LineShape:
		ring = s.points
	case draw.PolygonShape:
		ring = s.points
	}
	if len(ring) == 0 {
		return []geom.Vec2{{}}
	}
	if len(ring) == 1 || s.kind == draw.LineShape || s.kind == draw.CapsuleShape {
		return spread(ring, n, false)
	}
	return spread(ring, n, true)
}

// spread is n points evenly along the path through ring, closed or not.
func spread(ring []geom.Vec2, n int, closed bool) []geom.Vec2 {
	if len(ring) == 1 || n <= 1 {
		return []geom.Vec2{ring[0]}
	}
	edges := len(ring) - 1
	if closed {
		edges = len(ring)
	}
	total := float32(0)
	for i := range edges {
		total += ring[(i+1)%len(ring)].Sub(ring[i]).Len()
	}
	out := make([]geom.Vec2, 0, n)
	steps := n
	if !closed {
		steps = n - 1
	}
	for k := range n {
		want := total * float32(k) / float32(max(steps, 1))
		for i := range edges {
			a, b := ring[i], ring[(i+1)%len(ring)]
			d := b.Sub(a).Len()
			if want <= d || i == edges-1 {
				t := float32(0)
				if d > 0 {
					t = min(want/d, 1)
				}
				out = append(out, a.Add(b.Sub(a).Scale(t)))
				break
			}
			want -= d
		}
	}
	return out
}
