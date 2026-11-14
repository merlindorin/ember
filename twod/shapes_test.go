package twod //nolint:testpackage // white-box: shares twod's test helpers, reaches unexported state

import (
	"math"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/stretchr/testify/assert"
)

func area(pts []geom.Vec2, idx []int) float32 {
	var sum float32
	for i := 0; i+2 < len(idx); i += 3 {
		a, b, c := pts[idx[i]], pts[idx[i+1]], pts[idx[i+2]]
		sum += float32(math.Abs(float64((b.X-a.X)*(c.Y-a.Y)-(c.X-a.X)*(b.Y-a.Y)))) / 2
	}
	return sum
}

func TestPolygonsTriangulateConcaveShapes(t *testing.T) {
	l := []geom.Vec2{geom.V2(0, 0), geom.V2(20, 0), geom.V2(20, 10), geom.V2(10, 10), geom.V2(10, 20), geom.V2(0, 20)}
	idx := triangulate(l)
	assert.Len(t, idx, 4*3)
	assert.InDelta(t, 300, area(l, idx), 1e-3)
	reversed := []geom.Vec2{l[5], l[4], l[3], l[2], l[1], l[0]}
	assert.InDelta(t, 300, area(reversed, triangulate(reversed)), 1e-3, "either winding")
	assert.Empty(t, triangulate(l[:2]))
}

func extent(pts []geom.Vec2) (geom.Vec2, geom.Vec2) {
	var lo, hi geom.Vec2
	lo, hi = geom.V2(math.MaxFloat32, math.MaxFloat32), geom.V2(-math.MaxFloat32, -math.MaxFloat32)
	for _, p := range pts {
		lo, hi = geom.V2(min(lo.X, p.X), min(lo.Y, p.Y)), geom.V2(max(hi.X, p.X), max(hi.Y, p.Y))
	}
	return lo, hi
}

func has(pts []geom.Vec2, p geom.Vec2) bool {
	for _, q := range pts {
		if q.Sub(p).Len() < 1e-3 {
			return true
		}
	}
	return false
}

func TestLineGeometry(t *testing.T) {
	line := NewLine([]geom.Vec2{geom.V2(0, 0), geom.V2(100, 0)}, Width(10))
	pts, idx := line.mesh()
	lo, hi := extent(pts)
	assert.Equal(t, geom.V2(0, -5), lo, "width 10, flat caps")
	assert.Equal(t, geom.V2(100, 5), hi)
	assert.InDelta(t, 1000, area(pts, idx), 1e-2)

	Caps(Square, RoundCap).line(line)
	pts, _ = line.mesh()
	lo, hi = extent(pts)
	assert.InDelta(t, -5, lo.X, 1e-4, "a square cap")
	assert.InDelta(t, 105, hi.X, 1e-4, "a round cap")

	corner := func(j Joint) []geom.Vec2 {
		joined, _ := NewLine([]geom.Vec2{geom.V2(0, 0), geom.V2(100, 0), geom.V2(100, 100)}, Width(10), Joints(j)).mesh()
		return joined
	}
	assert.True(t, has(corner(Miter), geom.V2(105, -5)), "a miter meets at the tip")
	assert.False(t, has(corner(Bevel), geom.V2(105, -5)), "a bevel cuts it")
	assert.True(t, has(corner(Bevel), geom.V2(105, 0)))
	assert.True(
		t,
		has(corner(RoundJoint), geom.V2(100+5*float32(math.Cos(-math.Pi/4)), 5*float32(math.Sin(-math.Pi/4)))),
		"a round joint",
	)

	open := NewLine([]geom.Vec2{geom.V2(0, 0), geom.V2(100, 0), geom.V2(100, 100)})
	closed := NewLine(open.Points(), Closed())
	_, a := open.mesh()
	_, b := closed.mesh()
	assert.Greater(t, len(b), len(a), "a closed line has one more segment")
	assert.Equal(t, float32(2), open.Width(), "2 points wide by default")
}
