package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"testing"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/ui/internal/box"
	"github.com/merlindorin/ember/ui/internal/style"
	"github.com/stretchr/testify/assert"
)

// rows is a 100×100 scroller of n 20-point rows, each a red box holding
// a text run, scrolled to y.
func rows(n int, y float32) *box.Box {
	sc := list(y)
	sc.Content = geom.V2(100, float32(n*20))
	for i := range n {
		r := el(0, float32(i*20), 100, 20, bg(red))
		r.Append(text(0, 0, "row", r.Style))
		sc.Append(r)
	}
	return sc
}

func drawn(r *recorder, kind string, c color.Color) int {
	n := 0
	for _, o := range r.of(kind) {
		if c == (color.Color{}) || o.Color == c {
			n++
		}
	}
	return n
}

func TestRowsScrolledOutOfTheirScrollerAreNotDrawn(t *testing.T) {
	r := paintOne(rows(1000, 0))
	assert.Equal(t, 5, drawn(r, "rect", red), "the five rows in view")
	assert.InDelta(t, 5, drawn(r, "text", color.Color{}), 1, "and their text, glyphs allowed an em past their lines")

	r = paintOne(rows(1000, 210))
	assert.Equal(t, 6, drawn(r, "rect", red), "a row cut at each edge still draws")
}

func TestBoxesBeyondTheViewportAreNotDrawn(t *testing.T) {
	root := el(0, 0, 400, 300, nil)
	for i := range 100 {
		root.Append(el(0, float32(i*20), 100, 20, bg(red)))
	}
	r := newRecorder()
	Paint(r, tree(root))
	assert.Equal(t, 15, drawn(r, "rect", red), "300 points of 20-point rows")
}

func TestAShadowReachingIntoViewStillDraws(t *testing.T) {
	root := el(0, 0, 400, 300, nil, el(0, 310, 100, 20, func(s *style.Computed) {
		s.Background = red
		s.Shadows = []style.Shadow{{Y: -20, Color: blue}}
	}))
	r := newRecorder()
	Paint(r, tree(root))
	assert.Equal(t, 1, drawn(r, "rect", blue), "its shadow shows above the edge")
}

func TestATransformedBoxIsNeverCulledByWhereItWas(t *testing.T) {
	moved := el(0, 1000, 100, 20, func(s *style.Computed) {
		s.Background = red
		s.Transform = []style.TransformOp{{Kind: style.Translate, X: style.Px(0), Y: style.Px(-950)}}
	})
	r := newRecorder()
	Paint(r, tree(el(0, 0, 400, 300, nil, moved)))
	assert.Equal(t, 1, drawn(r, "rect", red))
}
