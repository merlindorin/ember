package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/style"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// topOnly is a w×h box at (10, 10) with only a top border of width bw in
// style s.
func topOnly(w, h, bw float32, s style.BorderStyle) *box.Box {
	b := el(10, 10, w, h, func(st *style.Computed) {
		st.Border = style.Sides[float32]{Top: bw}
		st.BorderStyle = style.Sides[style.BorderStyle]{Top: s}
		st.BorderColor = style.All(black)
	})
	return b
}

func TestADashedSideHasDashesFlushAtBothEndsAsChromes(t *testing.T) {
	for _, c := range []struct {
		w, bw, dash float32
		n           int
	}{
		{40, 1, 3, 8}, {73, 1, 3, 15}, {120, 1, 3, 24},
		{40, 2, 4, 7}, {40, 3, 6, 5}, {40, 4, 8, 4}, {73, 4, 8, 6},
	} {
		dashes := paintOne(topOnly(c.w, 24, c.bw, style.BorderDashed)).of("rect")
		require.Len(t, dashes, c.n, "%v px wide, %v long", c.bw, c.w)
		assert.Equal(t, float32(10), dashes[0].Rect.X)
		last := dashes[c.n-1].Rect
		assert.InDelta(t, 10+c.w, last.X+last.W, 1e-3, "ends flush")
		for _, d := range dashes {
			assert.InDelta(t, c.dash, d.Rect.W, 1e-3)
			assert.Equal(t, c.bw, d.Rect.H)
			assert.Equal(t, float32(10), d.Rect.Y)
		}
	}
}

func TestADottedSideHasRoundDotsAtBothEnds(t *testing.T) {
	for _, c := range []struct {
		w, bw float32
		n     int
	}{{40, 2, 11}, {40, 3, 7}, {40, 4, 6}, {73, 4, 10}} {
		dots := paintOne(topOnly(c.w, 24, c.bw, style.BorderDotted)).of("circle")
		require.Len(t, dots, c.n, "%v px wide, %v long", c.bw, c.w)
		assert.InDelta(t, 10+c.bw/2, dots[0].At.X, 1e-3)
		assert.InDelta(t, 10+c.w-c.bw/2, dots[c.n-1].At.X, 1e-3)
		assert.InDelta(t, c.bw/2, dots[0].Radius, 1e-3)
		assert.InDelta(t, 10+c.bw/2, dots[0].At.Y, 1e-3)
	}
}

func TestOnePixelDotsAreSquares(t *testing.T) {
	r := paintOne(topOnly(40, 24, 1, style.BorderDotted))
	assert.Empty(t, r.of("circle"))
	assert.Len(t, r.of("rect"), 21)
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 1, H: 1}, r.of("rect")[0].Rect)
}

func TestSidesRunCornerToCornerSoCornersJoin(t *testing.T) {
	b := el(10, 10, 40, 24, func(st *style.Computed) {
		st.Border = style.All[float32](2)
		st.BorderStyle = style.All(style.BorderDashed)
		st.BorderColor = style.All(black)
	})
	var left []geom.Rect
	for _, o := range paintOne(b).of("rect") {
		if o.Rect.X == 10 && o.Rect.W == 2 {
			left = append(left, o.Rect)
		}
	}
	require.NotEmpty(t, left)
	assert.Equal(t, float32(10), left[0].Y, "the left side's first dash starts in the corner")
	assert.InDelta(t, 34, left[len(left)-1].Y+left[len(left)-1].H, 1e-3)
}

func TestARoundedDashedBorderStaysSolidForNow(t *testing.T) {
	b := el(10, 10, 40, 24, func(st *style.Computed) {
		st.Border = style.All[float32](2)
		st.BorderStyle = style.All(style.BorderDashed)
		st.BorderColor = style.All(black)
		c := style.Size[style.Length]{Width: style.Px(6), Height: style.Px(6)}
		st.Radius = style.Corners[style.Size[style.Length]]{TopLeft: c, TopRight: c, BottomRight: c, BottomLeft: c}
	})
	assert.Len(t, paintOne(b).of("border"), 1, "a known gap: dashes along arcs")
}
