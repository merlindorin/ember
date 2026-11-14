package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/style"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// split is a span at (10, 10) broken over two lines: 60 points on the
// first, 40 on the second, 20 high each, with a 2-point blue border on
// every side and a red background.
func split() *box.Box {
	b := el(10, 10, 100, 40, func(s *style.Computed) {
		s.Display = style.Inline
		s.Background = red
		s.Border = style.All[float32](2)
		s.BorderStyle = style.All(style.BorderSolid)
		s.BorderColor = style.All(blue)
	})
	b.Fragments = []geom.Rect{{X: 40, Y: 0, W: 60, H: 20}, {X: 0, Y: 20, W: 40, H: 20}}
	return b
}

func TestASplitSpanFillsEachLinesFragment(t *testing.T) {
	r := paintOne(split())
	var fills []geom.Rect
	for _, o := range r.of("rect") {
		if o.Color == red {
			fills = append(fills, o.Rect)
		}
	}
	assert.Equal(t, []geom.Rect{{X: 50, Y: 10, W: 60, H: 20}, {X: 10, Y: 30, W: 40, H: 20}}, fills, "never the union box")
}

func TestASplitSpansSideBordersAreOnItsEnds(t *testing.T) {
	var sides []geom.Rect
	for _, o := range paintOne(split()).of("rect") {
		if o.Color == blue {
			sides = append(sides, o.Rect)
		}
	}
	assert.ElementsMatch(t, []geom.Rect{
		{X: 50, Y: 10, W: 60, H: 2}, {X: 50, Y: 28, W: 60, H: 2}, {X: 50, Y: 10, W: 2, H: 20},
		{X: 10, Y: 30, W: 40, H: 2}, {X: 48, Y: 30, W: 2, H: 20}, {X: 10, Y: 48, W: 40, H: 2},
	}, sides, "top and bottom on every line, left on the first, right on the last")
}

func TestAnUnbrokenSpanPaintsLikeABox(t *testing.T) {
	b := split()
	b.Fragments = []geom.Rect{{W: 100, H: 40}}
	var blues int
	for _, o := range paintOne(b).of("rect") {
		if o.Color == blue {
			blues++
		}
	}
	assert.Equal(t, 4, blues)
}

func TestHitTestingASplitSpanUsesItsFragments(t *testing.T) {
	b := split()
	root := el(0, 0, 400, 300, nil, b)
	tr := tree(root)
	assert.Same(t, b, HitTest(tr, geom.V2(60, 15)))
	assert.Same(t, b, HitTest(tr, geom.V2(20, 40)))
	assert.Same(t, root, HitTest(tr, geom.V2(20, 15)), "the union's empty corner isn't the span")
}

func TestASplitSpansOutlineCirclesEachFragment(t *testing.T) {
	b := split()
	b.Style.OutlineWidth, b.Style.OutlineStyle, b.Style.OutlineColor = 1, style.BorderSolid, green
	outlines := paintOne(b).of("border")
	require.Len(t, outlines, 2)
	assert.Equal(t, geom.Rect{X: 49, Y: 9, W: 62, H: 22}, outlines[0].Rect)
}
