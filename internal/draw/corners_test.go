package draw //nolint:testpackage // white-box: reaches draw's unexported state

import (
	"testing"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// piece is one quad drawn: its rectangle, UVs at its top-left and
// bottom-right, its shape and its extra.
type piece struct {
	rect  geom.Rect
	uv0   geom.Vec2
	uv1   geom.Vec2
	shape geom.Vec4
	extra float32
}

func pieces(c *List) []piece {
	var out []piece
	for i := 0; i+3 < len(c.Vertices); i += 4 {
		v := c.Vertices[i : i+4]
		out = append(out, piece{
			rect:  geom.Rect{X: v[0].Pos.X, Y: v[0].Pos.Y, W: v[2].Pos.X - v[0].Pos.X, H: v[2].Pos.Y - v[0].Pos.Y},
			uv0:   v[0].UV,
			uv1:   v[2].UV,
			shape: v[0].Shape,
			extra: v[0].Extra,
		})
	}
	return out
}

// area is the summed area of ps's rectangles inside box, so pieces that
// tile a box add up to it whatever their outer pads.
func area(ps []piece, box geom.Rect) float32 {
	var a float32
	for _, p := range ps {
		r := p.rect
		x0, y0 := max(r.X, box.X), max(r.Y, box.Y)
		x1, y1 := min(r.X+r.W, box.X+box.W), min(r.Y+r.H, box.Y+box.H)
		a += max(x1-x0, 0) * max(y1-y0, 0)
	}
	return a
}

func TestEqualCornersDrawOneQuadAsRoundRectDoes(t *testing.T) {
	var a, b List
	a.RoundRectCorners(geom.Rect{X: 10, Y: 10, W: 40, H: 20}, [4]float32{5, 5, 5, 5}, color.White)
	b.RoundRect(geom.Rect{X: 10, Y: 10, W: 40, H: 20}, 5, color.White)
	assert.Equal(t, b.Vertices, a.Vertices)
}

func TestRadiiScaleDownTogetherWhenTheyWouldOverlap(t *testing.T) {
	assert.Equal(
		t,
		[4]float32{40, 40, 8, 8},
		scaleRadii([4]float32{50, 50, 10, 10}, 80, 60),
		"the top's 100 into its 80",
	)
	assert.Equal(
		t,
		[4]float32{40, 0, 0, 0},
		scaleRadii([4]float32{40, 0, 0, 0}, 60, 60),
		"a corner past half the box is fine alone",
	)
	assert.Equal(t, [4]float32{0, 3, 0, 0}, scaleRadii([4]float32{-2, 3, 0, 0}, 60, 60), "never negative")
}

func TestACornerPastHalfTheBoxKeepsItsWholeRadius(t *testing.T) {
	var c List
	box := geom.Rect{X: 10, Y: 10, W: 60, H: 60}
	c.RoundRectCorners(box, [4]float32{40, 0, 0, 0}, color.White)
	ps := pieces(&c)
	require.NotEmpty(t, ps)
	corner := ps[0]
	assert.Equal(t, geom.Rect{X: 9, Y: 9, W: 41, H: 41}, corner.rect, "40 square at its corner, padded outside")
	assert.Equal(
		t,
		geom.Vec4{X: KindRoundRect, Y: 60, Z: 60, W: 40},
		corner.shape,
		"a box twice the size, so the radius is never clamped",
	)
	assert.Equal(t, geom.V2(-61, -61), corner.uv0, "from the virtual box's centre, the real box's far corner")
	assert.Equal(t, geom.V2(-20, -20), corner.uv1)
	assert.InDelta(t, 60*60, area(ps, box), 1e-3, "corner and strips tile the box once")
}

func TestStripsAreSquareShapesOfTheRealBox(t *testing.T) {
	var c List
	box := geom.Rect{X: 0, Y: 0, W: 80, H: 60}
	c.RoundRectCorners(box, [4]float32{50, 50, 10, 10}, color.White)
	ps := pieces(&c)
	assert.LessOrEqual(t, len(ps), 4+5, "four corners, at most five strips")
	assert.InDelta(t, 80*60, area(ps, box), 1e-3)
	for _, p := range ps[4:] {
		assert.Equal(t, geom.Vec4{X: KindRoundRect, Y: 40, Z: 30, W: 0}, p.shape, "square, against the whole box")
	}
}

func TestBorderCornersOwnAtLeastTheirThickness(t *testing.T) {
	var c List
	box := geom.Rect{X: 0, Y: 0, W: 80, H: 60}
	c.BorderCorners(box, [4]float32{2, 20, 2, 20}, 6, color.White)
	ps := pieces(&c)
	require.GreaterOrEqual(t, len(ps), 4+4)
	assert.Equal(
		t,
		geom.Rect{X: -1, Y: -1, W: 7, H: 7},
		ps[0].rect,
		"a 2 radius under a 6 border: the corner is 6 square",
	)
	assert.Equal(t, geom.Vec4{X: KindBorder, Y: 80, Z: 60, W: 2}, ps[0].shape)
	for _, p := range ps {
		assert.Equal(t, float32(6), p.extra)
	}
	var strips float32
	for _, p := range ps[4:] {
		strips += area([]piece{p}, box)
		assert.Equal(t, geom.Vec4{X: KindBorder, Y: 40, Z: 30, W: 0}, p.shape)
	}
	assert.InDelta(
		t,
		(80-6-20)*6+(60-20-6)*6+(80-20-6)*6+(60-6-20)*6,
		strips,
		1e-3,
		"four side strips, each its thickness wide",
	)
}
