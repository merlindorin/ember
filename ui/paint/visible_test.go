package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"testing"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/style"
	"github.com/stretchr/testify/assert"
)

// scrolledRows is a 200-point scroller of 1,000 20-point rows, each with a
// background, scrolled down 4,010 points.
func scrolledRows() (*box.Box, []*box.Box) {
	var rows []*box.Box
	for i := range 1000 {
		r := el(0, float32(i)*20, 300, 20, bg(color.Hex("#123456")))
		r.Content = r.Size
		rows = append(rows, r)
	}
	sc := el(0, 0, 300, 200, func(s *style.Computed) {
		s.Overflow = style.Size[style.Overflow]{Width: style.Hidden, Height: style.Hidden}
	}, rows...)
	sc.Scroll = geom.V2(0, 4010)
	sc.Content = geom.V2(300, 20000)
	return sc, rows
}

func TestAScrollerPaintsOnlyTheRowsItShows(t *testing.T) {
	sc, rows := scrolledRows()
	r := newRecorder()
	Paint(r, tree(el(0, 0, 400, 300, nil, sc)))
	assert.Len(t, r.of("rect"), 11, "rows 200 to 210, the first and last cut by the scrollport")

	assert.Same(t, rows[205], HitTest(tree(el(0, 0, 400, 300, nil, sc)), geom.V2(10, 100)),
		"the hit test finds the row under the point")
}

func TestARowDrawingIntoTheScrollportFromFarAboveStillShows(t *testing.T) {
	sc, rows := scrolledRows()
	spill := el(0, 4090, 300, 20, bg(color.Hex("#ff0000")))
	rows[3].Append(spill)
	rows[3].Content = geom.V2(300, 4110)
	r := newRecorder()
	Paint(r, tree(el(0, 0, 400, 300, nil, sc)))
	assert.Len(t, r.of("rect"), 12, "the eleven rows shown, and row 3's child drawn among them")
}

func TestABoxPlacedFromAboveInASkippedRowStillDrawsAndTakesThePointer(t *testing.T) {
	sc, rows := scrolledRows()
	popover := el(10, 4040-60, 50, 30, func(s *style.Computed) {
		s.Background, s.Position = color.Hex("#00ff00"), style.Absolute
	})
	popover.Escapes = true
	rows[3].Append(popover)
	rows[3].Escapes = true
	r := newRecorder()
	root := el(0, 0, 400, 300, nil, sc)
	Paint(r, tree(root))
	assert.Len(t, r.of("rect"), 12, "the eleven rows shown and the popover row 3 holds")
	assert.Same(t, popover, HitTest(tree(root), geom.V2(20, 40)), "and it takes the pointer")
}

func TestInkFromARowJustAboveTheScrollportStillDraws(t *testing.T) {
	sc, rows := scrolledRows()
	sc.Scroll = geom.V2(0, 4001)
	rows[199].Style.Shadows = []style.Shadow{{Blur: 6, Color: color.Hex("#000000")}}
	r := newRecorder()
	Paint(r, tree(el(0, 0, 400, 300, nil, sc)))
	assert.NotEmpty(t, r.of("gradient"), "row 199 ends a point above the view; its shadow reaches in")
}

func TestAFocusedChildsOutlineShowsThoughItsRowIsOutOfView(t *testing.T) {
	sc, rows := scrolledRows()
	field := el(0, 0, 300, 20, func(s *style.Computed) {
		s.OutlineWidth, s.OutlineStyle, s.OutlineColor, s.OutlineOffset = 2, style.BorderSolid, color.Hex("#ffce76"), 40
	})
	rows[198].Append(field)
	r := newRecorder()
	Paint(r, tree(el(0, 0, 400, 300, nil, sc)), Keep(field))
	assert.NotEmpty(t, r.of("border"), "the outline reaching 40 points out of row 198 shows")
}

func TestKeepingNothingIsKeepingNothing(t *testing.T) {
	sc, _ := scrolledRows()
	with, without := newRecorder(), newRecorder()
	Paint(with, tree(el(0, 0, 400, 300, nil, sc)), Keep(nil))
	Paint(without, tree(el(0, 0, 400, 300, nil, sc)))
	assert.Equal(t, without.ops, with.ops, "nothing focused, nothing kept")
}

func TestAChildsGlowReachingOutOfARowOutOfViewStillDraws(t *testing.T) {
	sc, rows := scrolledRows()
	card := el(0, 0, 300, 20, func(s *style.Computed) {
		s.Shadows = []style.Shadow{{Blur: 24,
			Color: color.Hex("#000000")}}
	})
	rows[198].Append(card)
	r := newRecorder()
	Paint(r, tree(el(0, 0, 400, 300, nil, sc)))
	assert.NotEmpty(t, r.of("gradient"),
		"row 198 ends 30 points above the view, past its own reach; its child's glow reaches in")
}

// The limit of inkMargin, a gap in the README: ink reaching more than 48
// points out of a row out of view is cut.
func TestInkFromFurtherThanTheMarginIsCut(t *testing.T) {
	sc, rows := scrolledRows()
	card := el(0, 0, 300, 20, func(s *style.Computed) {
		s.Shadows = []style.Shadow{{Blur: 64,
			Color: color.Hex("#000000")}}
	})
	rows[195].Append(card)
	r := newRecorder()
	Paint(r, tree(el(0, 0, 400, 300, nil, sc)))
	assert.Empty(t, r.of("gradient"),
		"row 195 ends 90 points above the view, past the margin; its child's 96-point reach is not drawn")
}
