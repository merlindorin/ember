package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/style"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// roundPanel is a 100×80 box at 10, 10 with a 2-point border and 10-point
// corners, clipping overflow on the axes given, holding a child that
// covers it whole.
func roundPanel(x, y style.Overflow) (*box.Box, *box.Box) {
	child := el(0, 0, 100, 80, nil)
	panel := el(10, 10, 100, 80, func(s *style.Computed) {
		rounded(10)(s)
		s.Overflow = style.Size[style.Overflow]{Width: x, Height: y}
	}, child)
	panel.Border = box.Edges{Top: 2, Right: 2, Bottom: 2, Left: 2}
	return panel, child
}

func TestARoundedOverflowClipCutsToThePaddingBoxsInnerCurve(t *testing.T) {
	panel, _ := roundPanel(style.Hidden, style.Hidden)
	r := newRecorder()
	Paint(r, tree(el(0, 0, 400, 300, nil, panel)))
	pushes := r.of("push-round", "push")
	require.GreaterOrEqual(t, len(pushes), 2)
	assert.Equal(t, "push-round", pushes[0].Kind, "the curve first, the scissor rect inside it")
	assert.Equal(t, geom.Rect{X: 12, Y: 12, W: 96, H: 76}, pushes[0].Rect, "the padding box")
	assert.Equal(t, [4]float32{8, 8, 8, 8}, pushes[0].Radii, "the radius less the border")
	assert.Equal(t, geom.Rect{X: 12, Y: 12, W: 96, H: 76}, pushes[1].Rect)
	assert.Equal(t, len(r.of("push", "push-round")), len(r.of("pop")), "every push popped")
}

func TestAClipWithAVisibleAxisIsSquare(t *testing.T) {
	panel, _ := roundPanel(style.Visible, style.Hidden)
	r := newRecorder()
	Paint(r, tree(el(0, 0, 400, 300, nil, panel)))
	assert.Empty(t, r.of("push-round"), "with one axis visible, the corners cut nothing")
	assert.NotEmpty(t, r.of("push"))
}

func TestAPointerInARoundedClipsCutCornerMissesTheChild(t *testing.T) {
	panel, child := roundPanel(style.Hidden, style.Hidden)
	tr := tree(el(0, 0, 400, 300, nil, panel))
	assert.Same(t, child, HitTest(tr, geom.V2(60, 50)), "inside the curve, the child")
	assert.Same(t, child, HitTest(tr, geom.V2(20, 14)), "on a straight edge, still the child")
	assert.Same(t, panel, HitTest(tr, geom.V2(13, 13)), "in the cut corner, the panel under it")
	assert.Same(t, panel, HitTest(tr, geom.V2(107, 87)), "and in the opposite one")
}

func TestARotatedRoundedClipIsHitAsItIsDrawnSquare(t *testing.T) {
	var s probe
	s.xf = geom.Transform2DFrom(geom.Vec2{}, 0.3, 0, geom.V2(1, 1))
	s.PushClipRounded(geom.Rect{W: 100, H: 80}, [4]float32{20, 20, 20, 20})
	assert.True(t, s.holds(s.xf.Xform(geom.V2(2, 2)), geom.Rect{W: 100, H: 80}),
		"turned, the corner is kept, as the draw list keeps it square")
	var flat probe
	flat.xf = geom.Identity2D()
	flat.PushClipRounded(geom.Rect{W: 100, H: 80}, [4]float32{20, 20, 20, 20})
	assert.False(t, flat.holds(geom.V2(2, 2), geom.Rect{W: 100, H: 80}), "upright, the corner is cut")
}
