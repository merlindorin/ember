package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"testing"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/ui/control"
	"github.com/merlindorin/ember/ui/internal/box"
	"github.com/merlindorin/ember/ui/internal/dom"
	"github.com/merlindorin/ember/ui/internal/style"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// selectPage is a 400×300 root holding a select at (x, y), 120×24, inside a
// clipping panel, with its machine.
func selectPage(x, y float32, options ...control.Option) (*box.Tree, *box.Box, *control.Select, Option) {
	sel := ctl(dom.Select, 120, 24)
	sel.Pos = geom.V2(x, y)
	sel.Style.Background, sel.Style.Color = white, black
	sel.Style.Border = style.All[float32](1)
	sel.Border = box.Edges{Top: 1, Right: 1, Bottom: 1, Left: 1}
	sel.Style.BorderStyle, sel.Style.BorderColor = style.All(style.BorderSolid), style.All(black)
	panel := el(0, 0, 400, 300, func(s *style.Computed) {
		s.Overflow = style.Size[style.Overflow]{Width: style.Hidden, Height: style.Hidden}
	}, sel)
	m := control.NewSelect(options...)
	find := Controls(func(n *dom.Node) any {
		if n == sel.Node {
			return m
		}
		return nil
	})
	return tree(el(0, 0, 400, 300, nil, panel)), sel, m, find
}

func three() []control.Option {
	return []control.Option{{Label: "Apple"}, {Label: "Banana", Disabled: true}, {Label: "Cherry"}}
}

// list is the pop-up's background: the last fill as wide as the select.
func listBg(r *recorder) op {
	var out op
	for _, o := range r.ops {
		if (o.Kind == "rect" || o.Kind == "round") && o.Color == white && o.Rect.Y != 50 {
			out = o
		}
	}
	return out
}

func TestAClosedSelectShowsNoList(t *testing.T) {
	tr, _, _, find := selectPage(20, 50, three()...)
	r := newRecorder()
	Paint(r, tr, find)
	assert.Empty(t, r.of("text"))
}

func TestAnOpenSelectListsItsOptionsUnderItOnTopOfEverything(t *testing.T) {
	tr, _, m, find := selectPage(20, 50, three()...)
	m.Do(control.Activate)
	r := newRecorder()
	Paint(r, tr, find, Fonts(ahem{}))
	bg := listBg(r)
	assert.Equal(t, geom.Rect{X: 20, Y: 74, W: 120, H: 3*rowHeight(ahem{}, style.Font{Size: 16}) + 2}, bg.Rect,
		"a row per option, inside its border")
	assert.Equal(t, geom.Rect{}, bg.Clip, "not clipped by the panel")
	texts := r.of("text")
	require.Len(t, texts, 3)
	assert.Equal(t, []string{"Apple", "Banana", "Cherry"}, []string{texts[0].Text, texts[1].Text, texts[2].Text})
	assert.Less(t, texts[1].Color.A, texts[0].Color.A, "a disabled option is dimmed")
	last := 0
	for i, o := range r.ops {
		if o.Kind == "text" {
			last = i
		}
	}
	for _, o := range r.ops[last+1:] {
		assert.Contains(t, []string{"pop"}, o.Kind, "the list is the last thing drawn")
	}
}

func TestTheHighlightedOptionIsMarked(t *testing.T) {
	tr, _, m, find := selectPage(20, 50, three()...)
	m.Do(control.Activate)
	m.Do(control.Down)
	r := newRecorder()
	Paint(r, tr, find, Fonts(ahem{}))
	var mark op
	for _, o := range r.ops {
		if o.Color == highlight {
			mark = o
		}
	}
	h := rowHeight(ahem{}, style.Font{Size: 16})
	assert.Equal(t, geom.Rect{X: 21, Y: 75 + 2*h, W: 118, H: h}, mark.Rect, "Cherry's row, inside the list's border")
	assert.Equal(t, highlightText, r.of("text")[2].Color)
}

func TestAnAccentColourHighlightsTheOptionInTheSelectsColour(t *testing.T) {
	tr, sel, m, find := selectPage(20, 50, three()...)
	sel.Style.AccentColor, sel.Style.AccentAuto = red, false
	m.Do(control.Activate)
	m.Do(control.Down)
	r := newRecorder()
	Paint(r, tr, find, Fonts(ahem{}))
	var marked bool
	for _, o := range r.ops {
		marked = marked || o.Kind == "rect" && o.Color == red
	}
	assert.True(t, marked, "accent-color: the highlighted row")
	assert.Equal(t, black, r.of("text")[2].Color, "its text in the select's colour")
}

func TestAListThatDoesntFitBelowOpensAbove(t *testing.T) {
	tr, _, m, find := selectPage(20, 260, three()...)
	m.Do(control.Activate)
	r := newRecorder()
	Paint(r, tr, find, Fonts(ahem{}))
	bg := listBg(r)
	assert.Equal(t, float32(260), bg.Rect.Y+bg.Rect.H, "its bottom on the select's top")
}

func TestAListIsClampedIntoTheRoot(t *testing.T) {
	tr, _, m, find := selectPage(330, 50, three()...)
	m.Do(control.Activate)
	r := newRecorder()
	Paint(r, tr, find, Fonts(ahem{}))
	bg := listBg(r)
	assert.Equal(t, float32(400), bg.Rect.X+bg.Rect.W, "pushed back inside the right edge")
}

func TestPickingSaysWhichOptionIsUnderThePointer(t *testing.T) {
	tr, sel, m, find := selectPage(20, 50, three()...)
	closed := PickOption(tr, geom.V2(30, 80), find)
	assert.Nil(t, closed.Select, "closed: no list")
	assert.Equal(t, Pick{Index: -1}, Pick{Index: closed.Index, List: closed.List, Self: closed.Self})

	m.Do(control.Activate)
	h := rowHeight(ahem{}, style.Font{Size: 16})
	pick := PickOption(tr, geom.V2(30, 75+2*h+1), find, Fonts(ahem{}))
	assert.Same(t, sel.Node, pick.Select)
	assert.Equal(t, 2, pick.Index)
	assert.True(t, pick.List)

	pick = PickOption(tr, geom.V2(30, 60), find, Fonts(ahem{}))
	assert.Equal(t, -1, pick.Index)
	assert.True(t, pick.Self, "on the select itself: its own click closes it")
	assert.False(t, pick.List)

	pick = PickOption(tr, geom.V2(300, 280), find, Fonts(ahem{}))
	assert.Same(t, sel.Node, pick.Select)
	assert.False(t, pick.List || pick.Self, "outside: ClickOutside")
}

func TestOverAnOpenListHitTestingFindsTheSelect(t *testing.T) {
	under := el(20, 80, 120, 40, nil)
	tr, sel, m, find := selectPage(20, 50, three()...)
	tr.Root.Append(under)
	assert.Same(t, under, HitTest(tr, geom.V2(30, 90), find, Fonts(ahem{})))
	m.Do(control.Activate)
	assert.Same(t, sel, HitTest(tr, geom.V2(30, 90), find, Fonts(ahem{})), "the list covers what is under it")
}

func TestALongListScrollsToKeepItsHighlightInView(t *testing.T) {
	var opts []control.Option
	for range 40 {
		opts = append(opts, control.Option{Label: "x"})
	}
	tr, _, m, find := selectPage(20, 50, opts...)
	m.Do(control.Activate)
	m.Do(control.End)
	r := newRecorder()
	Paint(r, tr, find, Fonts(ahem{}))
	bg := listBg(r)
	assert.LessOrEqual(t, bg.Rect.Y+bg.Rect.H, float32(300), "within the root")
	var mark op
	for _, o := range r.ops {
		if o.Color == highlight {
			mark = o
		}
	}
	assert.True(t, mark.Rect.Y >= bg.Rect.Y && mark.Rect.Y+mark.Rect.H <= bg.Rect.Y+bg.Rect.H, "the last option is shown")
	h := rowHeight(ahem{}, style.Font{Size: 16})
	pick := PickOption(tr, geom.V2(30, mark.Rect.Y+h/2), find, Fonts(ahem{}))
	assert.Equal(t, 39, pick.Index, "and picked where it is drawn")
}

var _ = color.White

func TestAnOpenListIsLive(t *testing.T) {
	tr, _, m, find := selectPage(20, 50, three()...)
	var live LiveSet
	Paint(newRecorder(), tr, find, Fonts(ahem{}), Live(&live, false))
	assert.False(t, live.Any())
	m.Do(control.Activate)
	whole := newRecorder()
	Paint(whole, tr, find, Fonts(ahem{}), Live(&live, false))
	assert.True(t, live.Any(), "its highlight follows keys and the pointer")

	apart := newRecorder()
	Paint(apart, tr, find, Fonts(ahem{}), Live(&live, true))
	assert.Less(t, len(apart.ops), len(whole.ops), "left out apart")
	PaintLive(apart, &live, 0, find, Fonts(ahem{}))
	assert.Equal(t, len(whole.ops), len(apart.ops), "and drawn on top")
}

func TestAnOpenListShowsTheOptionsAsThePlayerReadsThem(t *testing.T) {
	tr, _, m, find := selectPage(20, 50, control.Option{Label: "Red", Shown: "Rouge"}, control.Option{Label: "Blue"})
	m.Do(control.Activate)
	r := newRecorder()
	Paint(r, tr, find, Fonts(ahem{}))
	texts := r.of("text")
	require.Len(t, texts, 2)
	assert.Equal(t, []string{"Rouge", "Blue"}, []string{texts[0].Text, texts[1].Text})
}
