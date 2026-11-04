package layout //nolint:testpackage // tests the layout tree and its unexported helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/merlindorin/ember/ui/internal/dom"
	"github.com/merlindorin/ember/ui/internal/style"
)

func TestALeafMeasuredAtAKnownWidthWrapsInsideItsPadding(t *testing.T) {
	p := el(dom.P, func(st *style.Computed) {
		st.Display = style.Block
		st.Padding.Left, st.Padding.Right = style.Px(5), style.Px(5)
	}, text("HH HHH"))
	col := el(dom.Div, func(st *style.Computed) {
		st.Display = style.FlexDisplay
		st.Direction = style.Column
		st.Size.Width = style.Px(60)
	}, p)
	tr := lay(el(dom.Div, block, col), 400)
	assert.Equal(t, float32(20), tr.Of[p].Size.Y, "its 60 points less 10 of padding hold one word a line")
	assert.Len(t, tr.Of[p].Children[0].Lines, 2)
}
