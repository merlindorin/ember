package layout //nolint:testpackage // tests the layout tree and its unexported helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/ui/internal/box"
	"github.com/merlindorin/ember/ui/internal/dom"
	"github.com/merlindorin/ember/ui/internal/style"
)

// el is an element with a computed style: Initial, 10px Ahem with a line height of 1, then
// edit.
func el(tag dom.Tag, edit func(*style.Computed), children ...*dom.Node) *dom.Node {
	n := dom.NewElement(tag)
	st := style.Initial()
	st.Font = style.Font{Family: []string{"ahem"}, Size: 10, Weight: 400}
	st.LineHeight = style.LineHeight{Factor: 1}
	if edit != nil {
		edit(&st)
	}
	dom.Data(n).Style = &st
	n.Append(children...)
	return n
}

func text(s string) *dom.Node { return dom.NewText(s) }

func block(st *style.Computed) { st.Display = style.Block }

func width(w float32) func(*style.Computed) {
	return func(st *style.Computed) {
		st.Display = style.Block
		st.Size.Width = style.Px(w)
	}
}

func lay(root *dom.Node, w float32) *box.Tree {
	return Layout(root, geom.V2(w, 600), ahemFonts{})
}

func lines(b *box.Box) []string {
	var out []string
	for _, c := range b.Children {
		for _, f := range c.Lines {
			out = append(out, f.Text)
		}
	}
	return out
}

func TestTextWrapsAtSpaces(t *testing.T) {
	p := el(dom.P, width(50), text("HH HH HHH H"))
	tr := lay(el(dom.Div, block, p), 200)
	b := tr.Of[p]
	assert.Equal(t, geom.Vec2{X: 50, Y: 20}, b.Size)
	assert.Equal(t, []string{"HH HH", "HHH H"}, lines(b), "five glyphs a line")
}

func TestCollapsibleSpacesCollapse(t *testing.T) {
	p := el(dom.P, width(200), text("  HH   \n  HH  "))
	tr := lay(el(dom.Div, block, p), 200)
	require.Len(t, tr.Of[p].Children, 1)
	frags := tr.Of[p].Children[0].Lines
	require.Len(t, frags, 1)
	assert.Equal(t, "HH HH", frags[0].Text)
	assert.Equal(t, float32(50), frags[0].Rect.W)
}

func TestNoWrapKeepsOneLine(t *testing.T) {
	p := el(dom.P, func(st *style.Computed) {
		width(30)(st)
		st.WhiteSpace = style.NoWrapSpace
	}, text("HH HH HH"))
	tr := lay(el(dom.Div, block, p), 200)
	assert.Equal(t, float32(10), tr.Of[p].Size.Y)
	assert.Equal(t, float32(80), tr.Of[p].Content.X, "the overflowing line is the scrollable width")
}

func TestPreBreaksAtNewlinesOnly(t *testing.T) {
	p := el(dom.P, func(st *style.Computed) {
		width(30)(st)
		st.WhiteSpace = style.Pre
	}, text("HH  HH\nH"))
	tr := lay(el(dom.Div, block, p), 200)
	frags := tr.Of[p].Children[0].Lines
	require.Len(t, frags, 2)
	assert.Equal(t, "HH  HH", frags[0].Text)
	assert.Equal(t, "H", frags[1].Text)
}

func TestEllipsisCutsTheOverflowingLine(t *testing.T) {
	p := el(dom.P, func(st *style.Computed) {
		width(45)(st)
		st.WhiteSpace = style.NoWrapSpace
		st.Overflow = style.Size[style.Overflow]{Width: style.Hidden, Height: style.Hidden}
		st.Ellipsis = true
	}, text("HHHHHHHH"))
	tr := lay(el(dom.Div, block, p), 200)
	frags := tr.Of[p].Children[0].Lines
	require.Len(t, frags, 1)
	assert.True(t, frags[0].Ellipsis)
	assert.Equal(t, "HHH…", frags[0].Text, "three glyphs and the ellipsis fit in 45")
	assert.LessOrEqual(t, frags[0].Rect.W, float32(45))
}

func TestEllipsisStartCutsTheLinesStart(t *testing.T) {
	cut := func(st *style.Computed) {
		width(45)(st)
		st.WhiteSpace = style.NoWrapSpace
		st.Overflow = style.Size[style.Overflow]{Width: style.Hidden, Height: style.Hidden}
		st.Ellipsis, st.EllipsisStart = true, true
	}
	p := el(dom.P, cut, text("ABCDEFGH"))
	tr := lay(el(dom.Div, block, p), 200)
	frags := tr.Of[p].Children[0].Lines
	require.Len(t, frags, 1)
	assert.True(t, frags[0].Ellipsis)
	assert.Equal(t, "…FGH", frags[0].Text, "the ellipsis and the last three glyphs fit in 45")
	assert.Equal(t, float32(5), tr.Of[p].Children[0].Pos.X+frags[0].Rect.X,
		"flush with the box's end, as Chrome's right-to-left cut, the room left before it")

	span := el(dom.Span, nil, text("WXYZ"))
	q := el(dom.P, cut, text("ABCD"), span)
	tr = lay(el(dom.Div, block, q), 200)
	assert.Empty(t, tr.Of[q].Children[0].Lines, "a run cut away whole")
	lines := tr.Of[span].Children[0].Lines
	require.Len(t, lines, 1)
	assert.Equal(t, "…XYZ", lines[0].Text)
	assert.Equal(t, float32(5), tr.Of[span].Children[0].Rect().X-tr.Of[q].Rect().X+lines[0].Rect.X)

	fits := el(dom.P, cut, text("ABC"))
	tr = lay(el(dom.Div, block, fits), 200)
	assert.Equal(t, "ABC", tr.Of[fits].Children[0].Lines[0].Text, "nothing cut when it fits")
}

func TestTextAlign(t *testing.T) {
	for _, c := range []struct {
		align style.TextAlign
		x     float32
	}{{style.Start, 0}, {style.Center, 15}, {style.Right, 30}} {
		p := el(dom.P, func(st *style.Computed) {
			width(50)(st)
			st.TextAlign = c.align
		}, text("HH"))
		tr := lay(el(dom.Div, block, p), 200)
		assert.Equal(t, c.x, tr.Of[p].Children[0].Pos.X, "align %v", c.align)
	}
}

func TestSpanCoversItsText(t *testing.T) {
	span := el(dom.Span, nil, text("HHH"))
	p := el(dom.P, width(200), text("HH "), span)
	tr := lay(el(dom.Div, block, p), 200)
	b := tr.Of[span]
	require.NotNil(t, b)
	assert.Same(t, tr.Of[p], b.Parent)
	assert.Equal(t, geom.Vec2{X: 30, Y: 0}, b.Pos)
	assert.Equal(t, geom.Vec2{X: 30, Y: 10}, b.Size)
}

func TestAtomicInlineSitsOnTheLine(t *testing.T) {
	button := el(dom.Button, func(st *style.Computed) {
		st.Display = style.InlineBlock
		st.Padding = style.All(style.Px(4))
		st.Border = style.All[float32](1)
	}, text("HH"))
	p := el(dom.P, width(200), text("H "), button)
	tr := lay(el(dom.Div, block, p), 200)
	b := tr.Of[button]
	assert.Equal(t, geom.Vec2{X: 30, Y: 20}, b.Size, "20 of text and 10 of padding and border")
	assert.Equal(t, float32(20), b.Pos.X, "after H and a space")
	assert.Equal(t, float32(20), tr.Of[p].Size.Y, "the line box grows to the button")
	assert.Equal(t, float32(0), b.Pos.Y, "the button's text sits on the line's baseline")
	assert.Equal(t, box.Edges{Top: 1, Right: 1, Bottom: 1, Left: 1}, b.Border)
	assert.Equal(t, box.Edges{Top: 4, Right: 4, Bottom: 4, Left: 4}, b.Padding)
	require.Len(t, b.Children, 1)
	assert.Equal(t, geom.Vec2{X: 5, Y: 5}, b.Children[0].Pos, "its text inside its border and padding")
}

func TestAtomicInlinesWrap(t *testing.T) {
	var buttons []*dom.Node
	for range 3 {
		buttons = append(buttons, el(dom.Button, func(st *style.Computed) {
			st.Display = style.InlineBlock
			st.Size.Width = style.Px(40)
		}, text("H")))
	}
	p := el(dom.P, width(100), buttons[0], buttons[1], buttons[2])
	tr := lay(el(dom.Div, block, p), 200)
	assert.Equal(t, geom.Vec2{X: 40, Y: 0}, tr.Of[buttons[1]].Pos)
	assert.Equal(t, geom.Vec2{X: 0, Y: 10}, tr.Of[buttons[2]].Pos)
}

func TestTextBesideBlocksGetsAnAnonymousBlock(t *testing.T) {
	inner := el(dom.Div, block, text("HH"))
	root := el(dom.Div, block, text("H"), inner)
	tr := lay(root, 200)
	b := tr.Of[root]
	require.Len(t, b.Children, 2)
	assert.Equal(t, box.AnonymousBlock, b.Children[0].Kind)
	assert.Equal(t, box.TextRun, b.Children[0].Children[0].Kind)
	assert.Equal(t, float32(10), tr.Of[inner].Pos.Y)
}

func TestFlexTextItem(t *testing.T) {
	label := el(dom.Span, nil, text("HHHH"))
	row := el(dom.Div, func(st *style.Computed) { st.Display = style.FlexDisplay }, label)
	tr := lay(row, 200)
	assert.Equal(t, geom.Vec2{X: 40, Y: 10}, tr.Of[label].Size, "an inline child of a flex container is blockified")
}

func TestTextBreaksAfterHyphens(t *testing.T) {
	p := el(dom.P, width(40), text("aa-bb-cc"))
	tr := lay(el(dom.Div, block, p), 200)
	assert.Equal(t, []string{"aa-", "bb-", "cc"}, lines(tr.Of[p]))
}

func TestBreaksAcrossRuns(t *testing.T) {
	s := el(dom.Span, nil, text("-bb"))
	p := el(dom.P, width(30), text("aa"), s)
	tr := lay(el(dom.Div, block, p), 200)
	assert.Equal(t, []string{"aa"}, lines(tr.Of[p]))
	assert.Equal(t, float32(20), tr.Of[p].Size.Y, "aa-, then bb: the break after the span's hyphen")
}

func TestNoWrapKeepsHyphenatedWordsWhole(t *testing.T) {
	p := el(dom.P, func(st *style.Computed) { width(40)(st); st.WhiteSpace = style.NoWrapSpace }, text("aa-bb-cc"))
	tr := lay(el(dom.Div, block, p), 200)
	assert.Equal(t, []string{"aa-bb-cc"}, lines(tr.Of[p]))
}

func TestSpanEdgesTakeRoom(t *testing.T) {
	s := el(dom.Span, func(st *style.Computed) {
		st.Padding = style.Sides[style.Length]{Top: style.Px(2), Right: style.Px(5), Bottom: style.Px(2), Left: style.Px(5)}
		st.Margin.Left = style.Px(3)
		st.Border = style.Sides[float32]{Left: 1, Right: 1}
		st.BorderStyle = style.All(style.BorderSolid)
	}, text("HH"))
	p := el(dom.P, width(200), text("H"), s, text("H"))
	tr := lay(el(dom.Div, block, p), 200)
	sb := tr.Of[s].Rect()
	assert.Equal(t, geom.Rect{X: 13, Y: -2, W: 32, H: 14}, sb,
		"border box: after H and the margin, 1+5+20+5+1 wide, content area plus vertical padding")
	last := tr.Of[p].Children[2]
	assert.Equal(t, float32(45), last.Rect().X, "10 + 3 + 32: no right margin")
}

func TestNestedSpansAreInTheTree(t *testing.T) {
	inner := el(dom.Span, nil, text("b"))
	outer := el(dom.Span, nil, text("a"), inner)
	empty := el(dom.Span, nil)
	p := el(dom.P, width(200), outer, empty, text("c"))
	tr := lay(el(dom.Div, block, p), 200)
	require.NotNil(t, tr.Of[inner])
	assert.Same(t, tr.Of[outer], tr.Of[inner].Parent)
	assert.Same(t, tr.Of[p], tr.Of[outer].Parent)
	assert.Same(t, tr.Of[p], tr.Of[empty].Parent, "an empty span is in the tree too")
}

func TestOverflowWrapBreaksLongWords(t *testing.T) {
	p := el(dom.P, func(st *style.Computed) { width(30)(st); st.BreakWord = true }, text("HH HHHHHHHH H"))
	tr := lay(el(dom.Div, block, p), 200)
	assert.Equal(t, []string{"HH", "HHH", "HHH", "HH", "H"}, lines(tr.Of[p]), "HH H would be 40 wide")
}

func TestOverflowWrapAndMinContent(t *testing.T) {
	word := func(anywhere bool) *dom.Node {
		return el(dom.Div, func(st *style.Computed) {
			st.Display = style.InlineBlock
			st.BreakWord = true
			st.BreakAnywhere = anywhere
		}, text("HHHHHH"))
	}
	bw, ba := word(false), word(true)
	tr := lay(el(dom.Div, width(5), bw, ba), 200)
	assert.Equal(t, float32(60), tr.Of[bw].Size.X, "break-word leaves min-content alone")
	assert.Equal(t, float32(10), tr.Of[ba].Size.X, "anywhere: one grapheme")
}

func TestJustify(t *testing.T) {
	p := el(dom.P, func(st *style.Computed) { width(70)(st); st.TextAlign = style.Justify }, text("H H HH HH H"))
	tr := lay(el(dom.Div, block, p), 200)
	run := tr.Of[p].Children[0]
	var firstLine []float32
	for _, f := range run.Lines {
		if f.Rect.Y == 0 {
			firstLine = append(firstLine, f.Rect.X)
		}
	}
	assert.Equal(t, []float32{0, 25, 50}, firstLine, "H H HH is 60 wide in 70: 5 more a space, words placed one by one")
	last := run.Lines[len(run.Lines)-1]
	assert.Equal(t, "HH H", last.Text, "the last line isn't justified")
}

func TestAtomsInsideSpansAreTheirChildren(t *testing.T) {
	atom := el(dom.Div, func(st *style.Computed) {
		st.Display = style.InlineBlock
		st.Size = style.Size[style.Length]{Width: style.Px(20), Height: style.Px(10)}
	})
	s := el(dom.Span, func(st *style.Computed) { st.Padding.Left = style.Px(5) }, text("H"), atom)
	p := el(dom.P, width(200), text("HH"), s)
	tr := lay(el(dom.Div, block, p), 200)
	assert.Same(t, tr.Of[s], tr.Of[atom].Parent)
	assert.Equal(t, float32(20+5+10), tr.Of[atom].Rect().X, "after HH, the span's padding and its H")
}

func TestAtomSizingKeywords(t *testing.T) {
	atom := func(w style.Length) *dom.Node {
		return el(dom.Div, func(st *style.Computed) {
			st.Display = style.InlineBlock
			st.Size.Width = w
		}, text("HH HHH H"))
	}
	minC := atom(style.Length{Kind: style.MinContent})
	maxC := atom(style.Length{Kind: style.MaxContent})
	fit := atom(style.Length{Kind: style.FitContent, Px: 45})
	tr := lay(el(dom.Div, width(50), minC, maxC, fit), 200)
	assert.Equal(t, float32(30), tr.Of[minC].Size.X, "min-content: the widest word")
	assert.Equal(t, float32(80), tr.Of[maxC].Size.X, "max-content: one line, wider than the 50 it sits in")
	assert.Equal(t, float32(45), tr.Of[fit].Size.X, "fit-content(45px): min(80, max(30, 45))")
}

func TestSpanEdgesAndFragments(t *testing.T) {
	s := el(dom.Span, func(st *style.Computed) {
		st.Padding = style.Sides[style.Length]{Top: style.Px(1), Right: style.Px(2), Bottom: style.Px(1), Left: style.Px(2)}
		st.Border = style.Sides[float32]{Top: 1, Right: 1, Bottom: 1, Left: 1}
		st.BorderStyle = style.All(style.BorderSolid)
	}, text("HHH HHH"))
	p := el(dom.P, width(60), text("HH "), s)
	tr := lay(el(dom.Div, block, p), 200)
	b := tr.Of[s]
	assert.Equal(t, box.Edges{Top: 1, Right: 1, Bottom: 1, Left: 1}, b.Border)
	assert.Equal(t, box.Edges{Top: 1, Right: 2, Bottom: 1, Left: 2}, b.Padding)
	require.Len(t, b.Fragments, 2, "HH |HHH, then HHH| on the next line")
	first, last := b.Fragments[0], b.Fragments[1]
	assert.Equal(t, float32(3+30), first.W, "left edge and HHH")
	assert.Equal(t, float32(30+3), last.W, "HHH and right edge")
	assert.Equal(t, float32(0), last.X, "the second line starts at the line's start")
	assert.Equal(t, float32(10), last.Y-first.Y, "one line apart")
}

func TestSummaryMarker(t *testing.T) {
	sum := el(dom.Summary,
		func(st *style.Computed) { block(st); st.ListItem = true; st.ListStyle = style.ListDisclosureClosed }, text("HH"))
	plain := el(dom.Summary,
		func(st *style.Computed) { block(st); st.ListItem = true; st.ListStyle = style.ListNone }, text("HH"))
	asBlock := el(dom.Summary,
		func(st *style.Computed) { block(st); st.ListStyle = style.ListDisclosureClosed }, text("HH"))
	tr := lay(el(dom.Div, block, el(dom.Details, block, sum),
		el(dom.Details, block, plain), el(dom.Details, block, asBlock)), 200)
	b := tr.Of[sum]
	require.NotEmpty(t, b.Children)
	m := b.Children[0]
	assert.Equal(t, box.Marker, m.Kind)
	assert.Same(t, sum, m.Node)
	w := float32(2 * 10)
	assert.InDelta(t, w, m.Size.X, 0.01, "▸ and a space, as Chrome's ::marker text")
	assert.InDelta(t, w, b.Children[1].Pos.X, 0.01, "the text after the marker")
	require.Len(t, m.Lines, 1)
	assert.Equal(t, "▸", m.Lines[0].Text)
	assert.Equal(t, b.Children[1].Pos.Y+b.Children[1].Lines[0].Baseline,
		m.Pos.Y+m.Lines[0].Baseline, "on the text's baseline")
	for _, c := range tr.Of[plain].Children {
		assert.NotEqual(t, box.Marker, c.Kind, "list-style: none has no marker")
	}
	for _, c := range tr.Of[asBlock].Children {
		assert.NotEqual(t, box.Marker, c.Kind, "a display: block summary has no marker, as in Chrome")
	}
}

func TestAnOpenSummarysMarkerIsTheDownTriangle(t *testing.T) {
	sum := el(dom.Summary,
		func(st *style.Computed) { block(st); st.ListItem = true; st.ListStyle = style.ListDisclosureOpen }, text("HH"))
	m := lay(el(dom.Div, block, el(dom.Details, block, sum)), 200).Of[sum].Children[0]
	require.Len(t, m.Lines, 1)
	assert.Equal(t, "▾", m.Lines[0].Text)
}

// laidNodes lays root out as Layout does and returns its nodes by their element.
func laidNodes(root *dom.Node, w float32) map[*dom.Node]*node {
	t := &tree{fonts: ahemFonts{}, out: &box.Tree{Of: map[*dom.Node]*box.Box{}}}
	r := t.build(root, nil)
	t.root(r, spaces{def(w), def(600)})
	keepLayout(r, point{})
	out := map[*dom.Node]*node{}
	var walk func(n *node)
	walk = func(n *node) {
		if n.box != nil && n.box.Node != nil {
			out[n.box.Node] = n
		}
		for _, k := range n.kids {
			walk(k)
		}
	}
	walk(r)
	return out
}

func TestOutOfFlowInsideASpanIsAnchoredInItsLine(t *testing.T) {
	abs := el(dom.Div, func(st *style.Computed) {
		st.Display = style.Block
		st.Position = style.Absolute
		st.Size = style.Size[style.Length]{Width: style.Px(5), Height: style.Px(5)}
	})
	s := el(dom.Span, func(st *style.Computed) {
		st.Position = style.Relative
		st.Padding.Left = style.Px(2)
	}, text("HH"), abs, text("H"))
	plain := el(dom.Div, func(st *style.Computed) {
		st.Display = style.Block
		st.Position = style.Absolute
	})
	still := el(dom.Span, nil, text("H"), plain)
	p := el(dom.P, width(200), text("HHH "), s, still)
	nodes := laidNodes(el(dom.Div, block, p), 200)
	c := nodes[p].inline
	o := c.oofOf(nodes[abs])
	if !assert.NotNil(t, o, "the element is the owner's kid, anchored in the text") {
		return
	}
	assert.Equal(t, point{40 + 2 + 20, 0}, o.static, "after HHH, the space, the span's padding and HH")
	assert.True(t, o.positioned, "its span is positioned")
	assert.Equal(t, point{40, 0}, o.areaAt, "the span's padding box")
	assert.Equal(t, size{2 + 30, 10}, o.areaSize)
	q := c.oofOf(nodes[plain])
	if assert.NotNil(t, q) {
		assert.False(t, q.positioned, "a static span leaves the containing block to the owner's")
		assert.Equal(t, point{40 + 2 + 30 + 10, 0}, q.static)
	}
}

func TestAnInlineTextareaKeepsItsOverflow(t *testing.T) {
	content := make(map[style.Display]geom.Vec2, 2)
	for _, d := range []style.Display{style.Block, style.InlineBlock} {
		ta, _ := control(dom.Textarea, "HHHHHHHHHH HHHHHHHHHH HHHHHHHHHH", func(st *style.Computed) {
			st.Display = d
			st.Size.Width = style.Px(150)
			st.Overflow = style.Size[style.Overflow]{Width: style.Hidden, Height: style.Hidden}
		})
		content[d] = lay(el(dom.Div, block, ta), 400).Of[ta].Content
	}
	assert.NotZero(t, content[style.InlineBlock])
	assert.Equal(t, content[style.Block], content[style.InlineBlock], "inline, its scrollable overflow is a block's")
}

func TestTextUnderDisplayNoneHasNoBoxes(t *testing.T) {
	span := el(dom.Span, nil, text("HH"))
	nested := el(dom.Span, nil, text("HH"))
	hidden := el(dom.Div, func(st *style.Computed) { st.Display = style.DisplayNone }, span, el(dom.P, block, nested))
	tr := lay(el(dom.Div, block, hidden), 200)
	for _, n := range []*dom.Node{span, nested} {
		assert.Zero(t, tr.Of[n].Size, "a span under display: none")
		assert.Empty(t, tr.Of[n].Fragments)
	}
}

func transform(tt style.TextTransform) func(*style.Computed) {
	return func(st *style.Computed) { block(st); st.TextTransform = tt }
}

func TestTextTransformShowsCasedText(t *testing.T) {
	up := el(dom.P, transform(style.Uppercase), text("straße"))
	tr := lay(el(dom.Div, block, up), 400)
	frag := tr.Of[up].Children[0].Lines[0]
	assert.Equal(t, "STRASSE", frag.Text, "ß becomes SS")
	assert.Equal(t, float32(70), frag.Rect.W, "measured as shown")
	assert.Equal(t, "straße", up.Children()[0].Text(), "the node keeps its text")

	low := el(dom.P, transform(style.Lowercase), text("ÉCOLE"))
	tr = lay(el(dom.Div, block, low), 400)
	assert.Equal(t, "école", tr.Of[low].Children[0].Lines[0].Text)

	span := el(dom.Span, func(st *style.Computed) { st.TextTransform = style.Capitalize }, text("lo wörld"))
	capd := el(dom.P, transform(style.Capitalize), text("(hel"), span, text(" o'neil 3rd-rate"))
	tr = lay(el(dom.Div, block, capd), 400)
	var got string
	for _, b := range []*box.Box{tr.Of[capd].Children[0], tr.Of[span].Children[0], tr.Of[capd].Children[2]} {
		got += b.Lines[0].Text
	}
	assert.Equal(t, "(Hello Wörld O'neil 3rd-Rate", got,
		"each word's first letter as Chrome splits words: past punctuation and hyphens, not apostrophes or digits")
}

func TestAnEditableFieldShowsItsTextAsTyped(t *testing.T) {
	in, _ := control(dom.Input, "straße", func(st *style.Computed) { st.TextTransform = style.Uppercase })
	tr := lay(el(dom.Div, block, in), 400)
	assert.Equal(t, "straße", tr.Of[in].Children[0].Lines[0].Text, "edit fields stay untransformed for now")
}
