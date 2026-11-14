package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"testing"

	"github.com/lantervale/ember/color"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/internal/textedit"
	"github.com/lantervale/ember/ui/control"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ctl is an element box for a node of tag with attrs (name, value pairs),
// its text and marks in black.
func ctl(tag dom.Tag, w, h float32, attrs ...string) *box.Box {
	n := dom.NewElement(tag)
	for i := 0; i+1 < len(attrs); i += 2 {
		n.SetAttr(attrs[i], attrs[i+1])
	}
	b := el(10, 10, w, h, func(s *style.Computed) { s.Color, s.AccentAuto, s.CaretAuto = black, true, true })
	b.Node = n
	return b
}

func paintOne(b *box.Box, opts ...Option) *recorder {
	r := newRecorder()
	Paint(r, tree(el(0, 0, 400, 300, nil, b)), opts...)
	return r
}

func TestACheckedCheckboxDrawsItsMarkInItsColour(t *testing.T) {
	b := ctl(dom.Input, 16, 16, "type", "checkbox")
	assert.Empty(t, paintOne(b).of("line"), "unchecked: no mark")

	b.Node.SetState(dom.Checked, true)
	lines := paintOne(b).of("line")
	require.Len(t, lines, 2, "a tick is two strokes")
	for _, l := range lines {
		assert.Equal(t, black, l.Color)
		assert.True(t, geom.Rect{X: 10, Y: 10, W: 16, H: 16}.Contains(l.At), "inside the box")
	}
}

func TestAnIndeterminateCheckboxDrawsADashFromItsMachine(t *testing.T) {
	b := ctl(dom.Input, 16, 16, "type", "checkbox")
	m := &control.Checkbox{Indeterminate: true}
	r := paintOne(b, Controls(func(*dom.Node) any { return m }))
	assert.Empty(t, r.of("line"))
	dash := r.of("rect")
	require.Len(t, dash, 1)
	assert.Less(t, dash[0].Rect.H, dash[0].Rect.W, "a dash is flat")

	m.Indeterminate, m.Checked = false, true
	assert.Len(t, paintOne(b, Controls(func(*dom.Node) any { return m })).of("line"), 2,
		"the machine, not the node, says checked")
}

func TestARangeDrawsItsTrackFillAndThumbAtItsValue(t *testing.T) {
	b := ctl(dom.Input, 100, 20, "type", "range", "min", "0", "max", "10", "value", "5")
	r := paintOne(b)
	rounds := r.of("round")
	require.Len(t, rounds, 2, "the track, then its filled part")
	track, fill := rounds[0], rounds[1]
	assert.Equal(t, geom.Rect{X: 10, Y: 17, W: 100, H: 6}, track.Rect, "6 high along the middle")
	assert.Less(t, track.Color.A, float32(1), "without a background-color, the track is the ink, fainter")
	assert.InDelta(t, 50, fill.Rect.W, 1e-4)
	thumb := r.of("circle")
	require.Len(t, thumb, 1)
	assert.Equal(t, geom.V2(60, 20), thumb[0].At, "centred on the value")
	assert.Equal(t, float32(7), thumb[0].Radius, "14 across")

	m := &control.Range{Min: 0, Max: 10, Value: 10}
	thumb = paintOne(b, Controls(func(*dom.Node) any { return m })).of("circle")
	assert.InDelta(t, 110, thumb[0].At.X, 1e-4, "at the end, overhanging as Chrome's")
}

func TestARangesColoursFollowTheConvention(t *testing.T) {
	b := ctl(dom.Input, 100, 20, "type", "range", "value", "50")
	b.Style.Background, b.Style.Color = blue, white
	b.Style.AccentColor, b.Style.AccentAuto = red, false
	r := paintOne(b)
	assert.Empty(t, r.of("rect"), "no box behind it: its background-color is the track")
	rounds := r.of("round")
	require.Len(t, rounds, 2)
	assert.Equal(t, blue, rounds[0].Color, "background-color: the track")
	assert.Equal(t, red, rounds[1].Color, "accent-color: the fill")
	assert.Equal(t, white, r.of("circle")[0].Color, "color: the idle thumb")

	b.Node.SetState(dom.Hover, true)
	assert.Equal(t, red, paintOne(b).of("circle")[0].Color, "accent-color: the hovered thumb")
}

func TestAProgressBarsTrackIsItsBackgroundAndItsFillTheAccent(t *testing.T) {
	b := ctl(dom.Progress, 200, 10, "value", "1", "max", "2")
	b.Style.Background = blue
	b.Style.AccentColor, b.Style.AccentAuto = red, false
	fills := paintOne(b).of("rect")
	require.Len(t, fills, 2, "the track once, then the fill")
	assert.Equal(t, blue, fills[0].Color)
	assert.Equal(t, red, fills[1].Color)
}

func TestProgressFillsItsShareOfTheContentBox(t *testing.T) {
	b := ctl(dom.Progress, 200, 10, "value", "30", "max", "120")
	fills := paintOne(b).of("rect")
	require.Len(t, fills, 2, "the empty bar, then its value")
	assert.Equal(t, float32(200), fills[0].Rect.W)
	assert.InDelta(t, 50, fills[1].Rect.W, 1e-4)
	assert.Equal(t, black, fills[1].Color)

	none := paintOne(ctl(dom.Progress, 200, 10)).of("rect")
	assert.Len(t, none, 1, "indeterminate: only the bar")
}

func TestAMeterIsColouredByItsRegion(t *testing.T) {
	good := ctl(dom.Meter, 100, 10, "value", "0.9", "low", "0.25", "high", "0.75", "optimum", "1")
	bad := ctl(dom.Meter, 100, 10, "value", "0.1", "low", "0.25", "high", "0.75", "optimum", "1")
	g, b := paintOne(good).of("rect"), paintOne(bad).of("rect")
	require.Len(t, g, 2)
	require.Len(t, b, 2)
	assert.Equal(t, meterColors[control.Optimum], g[1].Color)
	assert.Equal(t, meterColors[control.EvenLessGood], b[1].Color)
	assert.InDelta(t, 90, g[1].Rect.W, 1e-4)
}

func summary(ls style.ListStyle) *box.Box {
	sum := dom.NewElement(dom.Summary)
	b := el(10, 10, 200, 20, func(s *style.Computed) { s.Color, s.ListStyle = black, ls })
	b.Node = sum
	m := &box.Box{Kind: box.Marker, Node: sum, Style: b.Style, Size: geom.V2(20, 20),
		Lines: []box.Fragment{{Text: "▸", Rect: geom.Rect{H: 20}, Baseline: 15}}}
	b.Append(m, text(20, 0, "tile 0", b.Style))
	return b
}

func TestASummarysTriangleIsDrawnInItsMarkerBox(t *testing.T) {
	b := summary(style.ListDisclosureClosed)
	b.Style.Font.Size = 16
	r := paintOne(b)
	tri := r.of("triangles")
	require.Len(t, tri, 1)
	assert.Equal(t, black, tri[0].Color)
	assert.InDelta(t, 10, tri[0].Rect.X, 1e-4, "from the marker's start")
	assert.InDelta(t, 0.86*0.657*16, tri[0].Rect.W, 1e-3)
	assert.InDelta(t, 0.657*16, tri[0].Rect.H, 1e-3, "closed: points right")
	assert.InDelta(t, 10+15-0.670*16, tri[0].Rect.Y, 1e-3, "its top 0.670em above the baseline")
}

func TestAnOpenSummarysTrianglePointsDown(t *testing.T) {
	b := summary(style.ListDisclosureOpen)
	tri := paintOne(b).of("triangles")
	require.Len(t, tri, 1)
	assert.Greater(t, tri[0].Rect.W, tri[0].Rect.H)
}

func TestASummaryWithoutAMarkerDrawsNoTriangle(t *testing.T) {
	b := summary(style.ListNone)
	b.Children = b.Children[1:]
	assert.Empty(t, paintOne(b).of("triangles"), "list-style: none lays out no marker")
}

func TestTheTriangleScalesWithTheFont(t *testing.T) {
	small := summary(style.ListDisclosureClosed)
	big := summary(style.ListDisclosureClosed)
	big.Style.Font.Size = 32
	assert.Greater(t, paintOne(big).of("triangles")[0].Rect.H, paintOne(small).of("triangles")[0].Rect.H)
}

// line is a one-line text layout where each rune is 8 points wide.
type line struct{ n int }

func (l line) Lines() int              { return 1 }
func (l line) Line(int) int            { return 0 }
func (l line) Span(int) (int, int)     { return 0, l.n }
func (l line) X(i int) float32         { return float32(i) * 8 }
func (l line) At(_ int, x float32) int { return int(x / 8) }
func (l line) Page() int               { return 1 }

func TestAFocusedTextInputDrawsItsSelectionUnderTheTextAndItsCaret(t *testing.T) {
	b := ctl(dom.Input, 100, 20, "type", "text")
	b.Padding = box.Edges{Left: 4}
	b.Append(text(4, 0, "hello", b.Style))
	m := control.NewText("hello")
	m.Layout = line{5}
	m.Editor.Select(1, 3)
	opts := Controls(func(*dom.Node) any { return m })

	assert.Empty(t, paintOne(b, opts).of("rect"), "unfocused: neither")
	b.Node.SetState(dom.Focus, true)
	r := paintOne(b, opts)
	rects := r.of("rect")
	require.Len(t, rects, 2, "the selection, then the caret")
	sel, caret := rects[0], rects[1]
	assert.Equal(t, geom.Rect{X: 10 + 4 + 8, Y: 10, W: 16, H: 20}, sel.Rect)
	assert.Equal(t, float32(10+4+24), caret.Rect.X, "the caret is at the selection's moving end")
	assert.Equal(t, float32(1), caret.Rect.W)
	var selAt, textAt int
	for i, o := range r.ops {
		switch {
		case o.Kind == "rect" && o.Rect == sel.Rect:
			selAt = i
		case o.Kind == "text":
			textAt = i
		}
	}
	assert.Less(t, selAt, textAt, "the selection is behind the text")

	m.Editor.Place(5, false)
	rects = paintOne(b, opts).of("rect")
	require.Len(t, rects, 1, "nothing selected: the caret alone")
}

func TestCaretColorAndSelectionColorPaintTheField(t *testing.T) {
	b := ctl(dom.Input, 100, 20, "type", "text")
	b.Style.Color, b.Style.CaretColor, b.Style.CaretAuto = black, red, false
	b.Style.SelectionColor, b.Style.HasSelection = blue, true
	b.Append(text(0, 0, "hello", b.Style))
	b.Node.SetState(dom.Focus, true)
	m := control.NewText("hello")
	m.Layout = line{5}
	m.Editor.Select(1, 3)
	rects := paintOne(b, Controls(func(*dom.Node) any { return m })).of("rect")
	require.Len(t, rects, 2)
	assert.Equal(t, blue, rects[0].Color, "::selection's background-color")
	assert.Equal(t, red, rects[1].Color, "caret-color")

	b.Style.CaretAuto, b.Style.HasSelection = true, false
	rects = paintOne(b, Controls(func(*dom.Node) any { return m })).of("rect")
	assert.Equal(t, selectionColor, rects[0].Color, "the default selection without a ::selection rule")
	assert.Equal(t, black, rects[1].Color, "auto: the text's colour")
}

func TestAReadOnlyInputDrawsNoCaret(t *testing.T) {
	b := ctl(dom.Input, 100, 20, "type", "text")
	b.Node.SetState(dom.Focus, true)
	m := control.NewText("x")
	m.Layout = line{1}
	m.SetReadOnly(true)
	assert.Empty(t, paintOne(b, Controls(func(*dom.Node) any { return m })).of("rect"))
}

func TestAModalDialogDimsTheWholeViewportBehindIt(t *testing.T) {
	b := ctl(dom.Dialog, 100, 60)
	b.Style.Background, b.Style.Position, b.Style.ZAuto, b.Style.Backdrop = white, style.FixedPosition, true, backdrop
	m := &control.Dialog{}
	m.ShowModal()
	r := paintOne(b, Controls(func(*dom.Node) any { return m }))
	fills := r.of("rect")
	require.Len(t, fills, 2)
	assert.Equal(t, geom.Rect{W: 400, H: 300}, fills[0].Rect, "the backdrop over the root")
	assert.Equal(t, backdrop, fills[0].Color, "in its ::backdrop colour")
	assert.Equal(t, white, fills[1].Color, "then the dialog")

	m.Show()
	assert.Len(t, paintOne(b, Controls(func(*dom.Node) any { return m })).of("rect"), 1, "a plain dialog: no backdrop")
}

var _ textedit.Layout = line{}

func TestAModalDialogsBackdropTakesThePagesClicks(t *testing.T) {
	button := el(0, 0, 50, 20, nil)
	dlg := ctl(dom.Dialog, 100, 60)
	dlg.Style.Position, dlg.Style.ZAuto = style.FixedPosition, true
	dlg.Pos = geom.V2(150, 100)
	m := &control.Dialog{}
	find := Controls(func(n *dom.Node) any {
		if n == dlg.Node {
			return m
		}
		return nil
	})
	tr := tree(el(0, 0, 400, 300, nil, button, dlg))
	assert.Same(t, button, HitTest(tr, geom.V2(5, 5), find), "closed, the page takes it")
	m.ShowModal()
	assert.Same(t, dlg, HitTest(tr, geom.V2(5, 5), find), "modal, the backdrop is the dialog's")
}

func TestTheCaretBlinksOnHalfSecondsLikeUIs(t *testing.T) {
	b := ctl(dom.Input, 100, 20, "type", "text")
	b.Node.SetState(dom.Focus, true)
	m := control.NewText("x")
	m.Layout = line{1}
	find := Controls(func(*dom.Node) any { return m })
	assert.Len(t, paintOne(b, find).of("rect"), 1, "without a time, always lit")
	assert.Len(t, paintOne(b, find, Time(0.2)).of("rect"), 1)
	assert.Empty(t, paintOne(b, find, Time(0.7)).of("rect"))
	assert.Len(t, paintOne(b, find, Time(1.1)).of("rect"), 1)
}

func TestAnIndeterminateProgressSlidesASegmentAcrossItsBar(t *testing.T) {
	b := ctl(dom.Progress, 300, 10)
	at := func(s float32) op {
		r := paintOne(b, Time(s))
		fills := r.of("rect")
		require.Len(t, fills, 2, "the bar, then the segment")
		return fills[1]
	}
	a, z := at(0.3), at(0.9)
	assert.Equal(t, float32(100), a.Rect.W, "a third of the bar")
	assert.Greater(t, z.Rect.X, a.Rect.X, "it moves right")
	assert.Equal(t, geom.Rect{X: 10, Y: 10, W: 300, H: 10}, a.Clip, "clipped to the bar")
	assert.InDelta(t, at(0.3).Rect.X, at(0.3+progressPeriod).Rect.X, 1e-3, "and comes round again")
}

func TestAPlaceholderDrawsInItsColourAt54Percent(t *testing.T) {
	b := ctl(dom.Input, 100, 20, "type", "text", "placeholder", "Name")
	shadow := dom.NewText("Name")
	run := text(2, 0, "Name", b.Style)
	run.Node = shadow
	b.Append(run)
	dom.Data(b.Node).Shadow = shadow

	assert.Equal(t, black, paintOne(b).of("text")[0].Color, "a value draws in the colour")
	dom.Data(b.Node).Placeholder = true
	c := paintOne(b).of("text")[0].Color
	assert.Equal(t, float32(0), c.R)
	assert.InDelta(t, 0.54, c.A, 1e-6, "Chrome's ::placeholder")

	other := dom.NewText("x")
	run.Node = other
	assert.Equal(t, black, paintOne(b).of("text")[0].Color, "only the shadow run is the placeholder")
}

func TestASwitchDrawsItsThumbAtTheEndItIsSetTo(t *testing.T) {
	b := ctl(dom.Input, 32, 18, "type", "checkbox", "switch", "")
	off := paintOne(b)
	assert.Empty(t, off.of("line"), "no tick on a switch")
	thumb := off.of("circle")
	require.Len(t, thumb, 1)
	assert.Equal(t, geom.V2(10+9, 10+9), thumb[0].At, "off: at the start")
	assert.Equal(t, float32(7), thumb[0].Radius, "inset 2 from the track")
	assert.Equal(t, black, thumb[0].Color, "in the element's colour")

	b.Node.SetState(dom.Checked, true)
	on := paintOne(b).of("circle")
	assert.Equal(t, geom.V2(10+32-9, 10+9), on[0].At, "on: at the end")
	assert.Empty(t, paintOne(b).of("line"))

	m := &control.Checkbox{}
	assert.Equal(t, float32(19), paintOne(b, Controls(func(*dom.Node) any { return m })).of("circle")[0].At.X,
		"the machine says off")
}

func TestALiveSetKeepsWhatTimeMoves(t *testing.T) {
	quiet := ctl(dom.Input, 100, 20, "type", "text")
	m := control.NewText("x")
	m.Layout = line{1}
	find := Controls(func(*dom.Node) any { return m })
	var live LiveSet
	paintOne(quiet, find, Live(&live, false))
	assert.False(t, live.Any(), "an unfocused field")
	quiet.Node.SetState(dom.Focus, true)
	paintOne(quiet, find, Time(0.7), Live(&live, false))
	assert.True(t, live.Any(), "a focused field's caret blinks, even while it is off")

	paintOne(ctl(dom.Progress, 200, 10), Time(0.3), Live(&live, false))
	assert.True(t, live.Any(), "an indeterminate progress slides")
	paintOne(ctl(dom.Progress, 200, 10, "value", "1"), Time(0.3), Live(&live, false))
	assert.False(t, live.Any(), "a known one doesn't")
}

// focusedField is a focused text input with its machine, at (10, 10).
func focusedField() (*box.Box, Option) {
	b := ctl(dom.Input, 100, 20, "type", "text")
	b.Append(text(0, 0, "hello", b.Style))
	m := control.NewText("hello")
	m.Layout = line{5}
	b.Node.SetState(dom.Focus, true)
	return b, Controls(func(n *dom.Node) any {
		if n == b.Node {
			return m
		}
		return nil
	})
}

func TestApartLivePartsDrawOverTheRestAsTheyWouldInPlace(t *testing.T) {
	field, find := focusedField()
	whole := paintOne(field, find, Time(0))

	var live LiveSet
	r := newRecorder()
	Paint(r, tree(el(0, 0, 400, 300, nil, field)), find, Time(0), Live(&live, true))
	assert.Len(t, r.of("rect"), len(whole.of("rect"))-1, "the caret left out")
	require.True(t, live.Any())
	assert.False(t, live.Covered())
	PaintLive(r, &live, 0, find)
	assert.Equal(t, whole.of("rect"), r.of("rect"), "then drawn on top: the same caret, clipped the same")
}

func TestALiveSetSaysWhenSomethingPaintedLaterCoversAPart(t *testing.T) {
	field, find := focusedField()
	field.Style.OutlineWidth, field.Style.OutlineStyle, field.Style.OutlineColor = 2, style.BorderSolid, red
	var live LiveSet
	Paint(newRecorder(), tree(el(0, 0, 400, 300, nil, field)), find, Live(&live, true))
	assert.False(t, live.Covered(), "the field's own outline rings it")

	over := el(50, 15, 80, 40, func(s *style.Computed) { s.Background, s.Position, s.ZAuto = blue, style.Absolute, true })
	Paint(newRecorder(), tree(el(0, 0, 400, 300, nil, field, over)), find, Live(&live, true))
	assert.True(t, live.Covered(), "a popover over it")

	beside := el(80, 12, 25, 16, func(s *style.Computed) {
		s.Background, s.Position, s.ZAuto = blue, style.Absolute, true
	})
	Paint(newRecorder(), tree(el(0, 0, 400, 300, nil, field, beside)), find, Live(&live, true))
	assert.False(t, live.Covered(), "a button inside the field, right of the caret at 50: not over its ink")

	away := el(200, 200, 80, 40, func(s *style.Computed) {
		s.Background, s.Position, s.ZAuto = blue, style.Absolute, true
	})
	Paint(newRecorder(), tree(el(0, 0, 400, 300, nil, field, away)), find, Live(&live, true))
	assert.False(t, live.Covered(), "one elsewhere")
}

func TestALivePartKeepsItsClipsThroughOtherPaints(t *testing.T) {
	field, find := focusedField()
	scroller := func() *box.Box {
		return el(0, 0, 300, 200, func(s *style.Computed) {
			s.Overflow = style.Size[style.Overflow]{Width: style.Hidden, Height: style.Hidden}
		}, field)
	}
	var live LiveSet
	Paint(newRecorder(), tree(el(0, 0, 400, 300, nil, scroller())), find, Time(0), Live(&live, true))
	require.True(t, live.Any())
	clipped := func(children ...*box.Box) *box.Box {
		return el(1, 1, 30, 20, func(s *style.Computed) {
			s.Overflow = style.Size[style.Overflow]{Width: style.Hidden, Height: style.Hidden}
		}, children...)
	}
	other := clipped(clipped(clipped(clipped(el(0, 0, 10, 10, bg(red))))))
	for range 4 {
		Paint(newRecorder(), tree(el(0, 0, 400, 300, nil, other)))
	}
	on := newRecorder()
	PaintLive(on, &live, 0, find)
	var pushed []geom.Rect
	for _, o := range on.of("push") {
		pushed = append(pushed, o.Rect)
	}
	assert.Contains(t, pushed, geom.Rect{W: 300, H: 200},
		"the caret clipped by its own scroller, not by what was painted since")
}

func TestEachTakesTheFrameAsItStandsAtEachPaint(t *testing.T) {
	field, find := focusedField()
	page := tree(el(0, 0, 400, 300, nil, field))
	whole := len(paintOne(field, find, Time(0)).of("rect"))
	var live LiveSet
	f := Frame{Live: &live, Apart: true}
	each := Each(&f)
	r := newRecorder()
	Paint(r, page, find, each)
	assert.Len(t, r.of("rect"), whole-1, "apart: the caret left out")
	f.Apart = false
	r = newRecorder()
	Paint(r, page, find, each)
	assert.Len(t, r.of("rect"), whole, "then in place, by the same option")
	assert.True(t, live.Any())
}

func TestAModalDialogWithoutABackdropColourDimsNothingButTakesThePointer(t *testing.T) {
	b := ctl(dom.Dialog, 100, 60)
	b.Style.Background, b.Style.Position, b.Style.ZAuto = white, style.FixedPosition, true
	m := &control.Dialog{}
	m.ShowModal()
	find := Controls(func(*dom.Node) any { return m })
	assert.Len(t, paintOne(b, find).of("rect"), 1, "the dialog alone")
	assert.Same(t, b, HitTest(tree(el(0, 0, 400, 300, nil, b)), geom.V2(300, 250), find),
		"its backdrop still the dialog's")
}

// backdrop is a ::backdrop colour for the tests' dialogs.
//
//nolint:gochecknoglobals // the tests' fixed values; Go has no constant colours or pictures
var backdrop = color.RGBA(0, 0, 0, 0.4)

func TestACheckedRadioDrawsItsDotInItsColour(t *testing.T) {
	b := ctl(dom.Input, 16, 16, "type", "radio")
	assert.Empty(t, paintOne(b).of("circle"), "unchecked: no dot")

	b.Node.SetState(dom.Checked, true)
	dots := paintOne(b).of("circle")
	require.Len(t, dots, 1)
	assert.Equal(t, black, dots[0].Color)
	assert.Equal(t, geom.V2(18, 18), dots[0].At, "at the content box's centre")
	assert.InDelta(t, 6.5, dots[0].Radius, 1e-6, "Chrome's gap of 1.5 inside the ring")

	m := &control.Radio{}
	assert.Empty(t, paintOne(b, Controls(func(*dom.Node) any { return m })).of("circle"),
		"its machine unchecked wins over a stale state")
}
