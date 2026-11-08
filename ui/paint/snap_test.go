package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"testing"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/ui/internal/box"
	"github.com/merlindorin/ember/ui/internal/style"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fractional is a box at a fractional place and size, red, with a 1pt
// blue border.
func fractional() *box.Box {
	return el(10.3, 5.2, 50.4, 20.6, func(s *style.Computed) {
		s.Background = red
		s.Border = style.All[float32](1)
		s.BorderStyle, s.BorderColor = style.All(style.BorderSolid), style.All(blue)
	})
}

func TestAScaleSnapsABoxsEdgesToDevicePixels(t *testing.T) {
	r := newRecorder()
	Paint(r, tree(el(0, 0, 400, 300, nil, fractional())), Scale(2))
	var fill geom.Rect
	for _, o := range r.of("rect") {
		if o.Color == red {
			fill = o.Rect
		}
	}
	assert.Equal(t, geom.Rect{X: 10.5, Y: 5, W: 50, H: 21}, fill,
		"10.3→10.5, 60.7→60.5, 5.2→5, 25.8→26 at 2 pixels a point")
}

func TestWithoutAScaleNothingIsSnapped(t *testing.T) {
	r := newRecorder()
	Paint(r, tree(el(0, 0, 400, 300, nil, fractional())))
	for _, o := range r.of("rect") {
		if o.Color == red {
			assert.Equal(t, geom.Rect{X: 10.3, Y: 5.2, W: 50.4, H: 20.6}, o.Rect)
		}
	}
}

func TestTextIsNeverSnapped(t *testing.T) {
	b := fractional()
	b.Append(text(1.3, 0.4, "hi", b.Style))
	r := newRecorder()
	Paint(r, tree(el(0, 0, 400, 300, nil, b)), Scale(2))
	ts := r.of("text")
	require.Len(t, ts, 1)
	assert.InDelta(t, 10.3+1.3, ts[0].At.X, 1e-4)
}

func TestATransformedBoxIsNotSnapped(t *testing.T) {
	b := fractional()
	b.Style.Transform = []style.TransformOp{{Kind: style.Rotate, A: [6]float32{0.3}}}
	r := newRecorder()
	Paint(r, tree(el(0, 0, 400, 300, nil, b)), Scale(2))
	for _, o := range r.of("rect") {
		if o.Color == red {
			assert.Equal(t, float32(10.3), o.Rect.X)
		}
	}
}

func TestAZoomedPaintSnapsToTheWindowsPixels(t *testing.T) {
	// A Document at zoom 3 paints under a 3× transform at 3 pixels a
	// point: a box centred at y 26.5 points lands on 79.5 window pixels,
	// and is drawn from 80, a whole pixel, as Chrome draws it.
	r := newRecorder()
	r.SetTransform(geom.Transform2D{X: geom.V2(3, 0), Y: geom.V2(0, 3)})
	b := el(10.5, 26.5, 48, 48.25, func(s *style.Computed) { s.Background = red })
	Paint(r, tree(el(0, 0, 200, 101, nil, b)), Scale(3))
	var fill geom.Rect
	for _, o := range r.of("rect") {
		if o.Color == red {
			fill = o.Rect
		}
	}
	for _, v := range []float32{fill.X, fill.Y, fill.X + fill.W, fill.Y + fill.H} {
		px := v * 3
		assert.InDelta(t, float64(px), float64(int(px+0.5)), 1e-3, "every edge on a whole window pixel: %v", fill)
	}
	assert.InDelta(t, 80.0/3, fill.Y, 1e-4, "26.5 points, 79.5 pixels, drawn from 80")
}

func TestATransformPastTheZoomIsNotSnapped(t *testing.T) {
	r := newRecorder()
	r.SetTransform(geom.Transform2D{X: geom.V2(3, 0), Y: geom.V2(0, 3)})
	b := el(10.5, 26.5, 48, 48, func(s *style.Computed) {
		s.Background = red
		s.Transform = []style.TransformOp{{Kind: style.Rotate, A: [6]float32{0.1}}}
	})
	Paint(r, tree(el(0, 0, 200, 101, nil, b)), Scale(3))
	for _, o := range r.of("rect") {
		if o.Color == red {
			require.Equal(t, float32(48), o.Rect.W, "a turned box keeps its size: pixels don't lie along it")
		}
	}
}
