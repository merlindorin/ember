package layout //nolint:testpackage // tests the layout tree and its unexported helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/ui/internal/dom"
	"github.com/merlindorin/ember/ui/internal/style"
)

func TestAFitContentBoxIsAsTallAsItsContentAtItsClampedWidth(t *testing.T) {
	icon := blockOf(40, nil)
	text := el(dom.P, block, text("HHHH HHHH HHHH HHHH"))
	fixed := el(dom.Div, func(st *style.Computed) {
		st.Display = style.FlexDisplay
		st.Direction = style.Column
		st.Position = style.FixedPosition
		st.Size = style.Size[style.Length]{Width: style.Length{Kind: style.FitContent},
			Height: style.Length{Kind: style.FitContent}}
		st.MaxSize.Width = style.Px(100)
		st.Padding.Bottom = style.Px(16)
	}, icon, text)
	tr := lay(el(dom.Div, block, fixed), 400)
	assert.Equal(t, float32(100), tr.Of[fixed].Size.X, "fit-content clamped by max-width")
	assert.Equal(t, float32(40), tr.Of[icon].Size.Y, "nothing shrinks")
	assert.Equal(t, float32(40+2*10+16), tr.Of[fixed].Size.Y, "two lines at 100 wide, then the padding")
}

func TestAnAutoSizedBoxIsAsTallAsItsContentAtItsClampedWidth(t *testing.T) {
	icon := blockOf(40, nil)
	text := el(dom.P, block, text("HHHH HHHH HHHH HHHH"))
	fixed := el(dom.Div, func(st *style.Computed) {
		st.Display = style.FlexDisplay
		st.Direction = style.Column
		st.Position = style.FixedPosition
		st.MaxSize.Width = style.Px(100)
	}, icon, text)
	tr := lay(el(dom.Div, block, fixed), 400)
	assert.Equal(t, float32(100), tr.Of[fixed].Size.X)
	assert.Equal(t, float32(40+2*10), tr.Of[fixed].Size.Y)
}

func TestAFitContentHeightMeasuresAtTheClampedStretchWidth(t *testing.T) {
	icon := blockOf(40, nil)
	text := el(dom.P, block, text("HHHH HHHH HHHH HHHH"))
	fixed := el(dom.Div, func(st *style.Computed) {
		st.Display = style.FlexDisplay
		st.Direction = style.Column
		st.Position = style.FixedPosition
		st.Inset = style.Sides[style.Length]{Top: style.Px(0), Right: style.Px(0), Bottom: style.Px(0), Left: style.Px(0)}
		st.Size.Height = style.Length{Kind: style.FitContent}
		st.MaxSize.Width = style.Px(100)
	}, icon, text)
	tr := lay(el(dom.Div, block, fixed), 400)
	assert.Equal(t, float32(100), tr.Of[fixed].Size.X)
	assert.Equal(t, float32(40+2*10), tr.Of[fixed].Size.Y)
}

func TestAnAbsoluteBoxInAHiddenSubtreeIsZero(t *testing.T) {
	abs := el(dom.Span, func(st *style.Computed) {
		st.Display = style.Block
		st.Position = style.Absolute
		st.Size = style.Size[style.Length]{Width: style.Px(5), Height: style.Px(5)}
	})
	glyph := el(dom.Div, func(st *style.Computed) { st.Display = style.Block; st.Position = style.Relative }, abs)
	hidden := el(dom.Div, func(st *style.Computed) { st.Display = style.DisplayNone }, el(dom.Div, block, glyph))
	parent := el(dom.Div, block, hidden)
	tr := lay(el(dom.Div, block, blockOf(30, nil), parent), 400)
	assert.Equal(t, geom.Rect{}, tr.Of[abs].Rect(), "nothing shows under display: none")
}

func TestATextNodesBoxIsItsTextsOnTheLines(t *testing.T) {
	words := text("HH HHH")
	p := el(dom.P, func(st *style.Computed) { st.Display = style.Block; st.TextAlign = style.Center }, words)
	tr := lay(el(dom.Div, block, p), 100)
	b := tr.Of[words]
	if assert.NotNil(t, b) {
		assert.Equal(t, geom.Rect{X: 20, Y: 0, W: 60, H: 10}, b.Rect(), "centred in 100")
	}
}
