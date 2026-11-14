package layout //nolint:testpackage // tests the layout tree and its unexported helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func inlineBlock(st *style.Computed) { st.Display = style.InlineBlock }

// control is a form control showing shown through its shadow text node.
func control(tag dom.Tag, shown string, edit func(*style.Computed), children ...*dom.Node) (*dom.Node, *dom.Node) {
	n := el(tag, func(st *style.Computed) {
		inlineBlock(st)
		if tag != dom.Textarea {
			st.WhiteSpace = style.NoWrapSpace
		} else {
			st.WhiteSpace = style.PreWrap
			st.Overflow = style.Size[style.Overflow]{Width: style.OverflowAuto, Height: style.OverflowAuto}
			st.ScrollbarWidth = 15
		}
		if edit != nil {
			edit(st)
		}
	}, children...)
	shadow := dom.NewText(shown)
	ss := *dom.Data(n).Style
	dom.Data(shadow).Style = &ss
	dom.Data(n).Shadow = shadow
	return n, shadow
}

// Ahem's x is 10 wide at 10px, and the engine's max-char estimate is 1.875 of it.
const ahemInputWidth = 200 + 18.75 - 10

func TestAnInputIsTwentyCharactersWide(t *testing.T) {
	in, _ := control(dom.Input, "HI", nil)
	tr := lay(el(dom.Div, block, in), 400)
	assert.Equal(t, geom.Vec2{X: ahemInputWidth, Y: 10}, tr.Of[in].Size)
}

func TestAValueNeverWidensAnInput(t *testing.T) {
	in, _ := control(dom.Input, "a value far longer than twenty characters", nil)
	tr := lay(el(dom.Div, block, in), 1000)
	assert.Equal(t, float32(ahemInputWidth), tr.Of[in].Size.X)
	assert.Equal(t, float32(10), tr.Of[in].Size.Y, "one line, no wrap")
}

func TestAnEmptyInputIsOneLineHigh(t *testing.T) {
	in, _ := control(dom.Input, "", func(st *style.Computed) {
		st.Padding = style.All(style.Px(2))
	})
	tr := lay(el(dom.Div, block, in), 400)
	assert.Equal(t, float32(14), tr.Of[in].Size.Y)
}

func TestTheShadowIsTheInputsTextRun(t *testing.T) {
	in, shadow := control(dom.Input, "HI", nil)
	tr := lay(el(dom.Div, block, in), 400)
	b := tr.Of[in]
	require.Len(t, b.Children, 1)
	run := b.Children[0]
	assert.Equal(t, box.TextRun, run.Kind)
	assert.Same(t, shadow, run.Node)
	require.Len(t, run.Lines, 1)
	assert.Equal(t, "HI", run.Lines[0].Text)
}

func TestAWidthOverridesTheInputsSize(t *testing.T) {
	in, _ := control(dom.Input, "HI", func(st *style.Computed) { st.Size.Width = style.Px(50) })
	tr := lay(el(dom.Div, block, in), 400)
	assert.Equal(t, float32(50), tr.Of[in].Size.X)
}

func TestASelectIsItsWidestOption(t *testing.T) {
	opt := func(label string) *dom.Node { return el(dom.Option, nil, text(label)) }
	sel, _ := control(dom.Select, "H", nil, opt("H"), opt("HHHH"), opt("HH"))
	tr := lay(el(dom.Div, block, sel), 400)
	assert.Equal(t, geom.Vec2{X: 40, Y: 10}, tr.Of[sel].Size)
	assert.Len(t, tr.Of[sel].Children, 1, "only the shown label, never the options")
}

func TestATextareaIsTwentyColumnsByTwoRows(t *testing.T) {
	ta, _ := control(dom.Textarea, "HH HH", nil)
	tr := lay(el(dom.Div, block, ta), 400)
	assert.Equal(t, geom.Vec2{X: 200 + 15 + 1, Y: 20}, tr.Of[ta].Size)
}

func TestATextareaWraps(t *testing.T) {
	ta, _ := control(dom.Textarea, "HHHHHHHHHH HHHHHHHHHH HHHHHHHHHH", func(st *style.Computed) {
		st.Size.Width = style.Px(150)
		st.Overflow = style.Size[style.Overflow]{Width: style.Hidden, Height: style.Hidden}
	})
	tr := lay(el(dom.Div, block, ta), 400)
	require.Len(t, tr.Of[ta].Children, 1)
	assert.Len(t, tr.Of[ta].Children[0].Lines, 3)
	assert.Equal(t, float32(20), tr.Of[ta].Size.Y, "the box stays two rows; the rest overflows")
}

func tall(st *style.Computed) {
	st.Size.Height = style.Px(32)
	st.Padding = style.All(style.Px(3))
}

func TestControlsCentreTheirLine(t *testing.T) {
	in, _ := control(dom.Input, "HI", tall)
	sel, _ := control(dom.Select, "H", tall, el(dom.Option, nil, text("H")))
	btn := el(dom.Button, func(st *style.Computed) { inlineBlock(st); tall(st) }, text("OK"))
	tr := lay(el(dom.Div, block, in, sel, btn), 600)
	for _, n := range []*dom.Node{in, sel, btn} {
		require.Len(t, tr.Of[n].Children, 1, "%v", n.Tag())
		assert.Equal(t, float32(11), tr.Of[n].Children[0].Pos.Y, "%v: 3 of padding and (26-10)/2", n.Tag())
	}
}

func TestAButtonCentresItsBlocks(t *testing.T) {
	inner := el(dom.Div, func(st *style.Computed) { st.Display = style.Block; st.Size.Height = style.Px(10) })
	btn := el(dom.Button, func(st *style.Computed) { inlineBlock(st); tall(st) }, inner)
	tr := lay(el(dom.Div, block, btn), 600)
	assert.Equal(t, float32(11), tr.Of[inner].Pos.Y)
}

func TestATextareaKeepsItsTextAtTheTop(t *testing.T) {
	ta, _ := control(dom.Textarea, "HH", tall)
	tr := lay(el(dom.Div, block, ta), 600)
	assert.Equal(t, float32(3), tr.Of[ta].Children[0].Pos.Y)
}

func sizedArea(st *style.Computed) {
	st.Display = style.Block
	st.Size = style.Size[style.Length]{Width: style.Px(100), Height: style.Px(20)}
}

func TestATextareasContentIsItsTextAtItsWidth(t *testing.T) {
	empty, _ := control(dom.Textarea, "", sizedArea)
	full, _ := control(dom.Textarea, "HHHH HHHH HHHH HHHH HHHH HHHH", sizedArea)
	tr := lay(el(dom.Div, block, empty, full), 400)
	assert.Equal(t, geom.Vec2{X: 100, Y: 20}, tr.Of[empty].Content, "never less than its box, never the twenty columns")
	assert.Equal(t, geom.Vec2{X: 85, Y: 60}, tr.Of[full].Content, "three lines overflow two rows, so the gutter takes 15")
	require.Len(t, tr.Of[full].Children, 1)
	assert.Len(t, tr.Of[full].Children[0].Lines, 6, "HHHH HHHH fits 100, not 85")
}

func TestATextareaThatFitsKeepsItsWholeWidth(t *testing.T) {
	ta, _ := control(dom.Textarea, "HHHHH HHHH", sizedArea)
	tr := lay(el(dom.Div, block, ta), 400)
	assert.Equal(t, geom.Vec2{X: 100, Y: 20}, tr.Of[ta].Content)
	require.Len(t, tr.Of[ta].Children, 1)
	assert.Len(t, tr.Of[ta].Children[0].Lines, 1)
}

func TestATallControlsBaselineIsItsCentredTexts(t *testing.T) {
	label := el(dom.Span, nil, text("HH"))
	in, _ := control(dom.Input, "HH", func(st *style.Computed) { st.Size.Height = style.Px(30) })
	row := el(dom.Div,
		func(st *style.Computed) { st.Display = style.FlexDisplay; st.AlignItems = style.AlignBaseline }, label, in)
	tr := lay(el(dom.Div, block, row), 400)
	assert.Equal(t, float32(10), tr.Of[label].Pos.Y, "the label's baseline on the input's text, 10 down in its 30")
}
