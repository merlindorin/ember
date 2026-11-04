package layout //nolint:testpackage // tests the layout tree and its unexported helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/merlindorin/ember/ui/internal/dom"
	"github.com/merlindorin/ember/ui/internal/style"
)

func absolute(st *style.Computed) {
	st.Display = style.Block
	st.Position = style.Absolute
	st.Inset.Top = style.Px(0)
	st.Size.Width, st.Size.Height = style.Px(10), style.Px(10)
}

func TestABoxPlacedFromAboveMarksTheBoxesItEscapes(t *testing.T) {
	badge := el(dom.Div, absolute)
	cell := blockOf(20, nil, badge)
	row := blockOf(20, nil, cell)
	plain := blockOf(20, nil)
	cb := blockOf(200, func(st *style.Computed) { st.Position = style.Relative }, plain, row)
	tr := lay(el(dom.Div, block, cb), 400)
	assert.True(t, tr.Of[badge].Escapes, "the hoisted box")
	assert.True(t, tr.Of[cell].Escapes)
	assert.True(t, tr.Of[row].Escapes, "and each box up to its containing block")
	assert.False(t, tr.Of[cb].Escapes, "not the containing block, which lays it out")
	assert.False(t, tr.Of[plain].Escapes)
}

func TestAKeptLayoutForgetsAnEscapeThatWent(t *testing.T) {
	badge := el(dom.Div, absolute)
	row := blockOf(20, nil, badge)
	cb := blockOf(200, func(st *style.Computed) { st.Position = style.Relative }, row)
	root := el(dom.Div, block, cb)
	var r Retained
	tr := layKept(&r, root, 400)
	assert.True(t, tr.Of[row].Escapes)
	row.Remove(badge)
	tr = layKept(&r, root, 400)
	assert.False(t, tr.Of[row].Escapes, "cleared once the absolute child is gone")
}
