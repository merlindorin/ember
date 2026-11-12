package layout //nolint:testpackage // tests the layout tree and its unexported helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/merlindorin/ember/ui/internal/dom"
	"github.com/merlindorin/ember/ui/internal/style"
)

// A flex item's negative margins may make its outer size negative: a flex
// container sizing itself to its items adds them as they are, as block
// flow does, rather than flooring the item's outer size at zero.
func TestAFlexContainerSumsAnItemWhoseMarginsOutweighItsSize(t *testing.T) {
	tall := func(h float32, top, bottom float32) *dom.Node {
		return el(dom.Div, func(st *style.Computed) {
			st.Display = style.Block
			st.Size = style.Size[style.Length]{Width: style.Px(40), Height: style.Px(h)}
			st.Margin = style.Sides[style.Length]{Top: style.Px(top), Right: style.Px(0), Bottom: style.Px(bottom),
				Left: style.Px(0)}
		})
	}
	for _, c := range []struct {
		name      string
		direction style.FlexDirection
		display   style.Display
	}{
		{"flex column", style.Column, style.FlexDisplay},
		{"block flow", style.Row, style.Block},
	} {
		plate, row := tall(16, -24, 5), tall(16, 0, 0)
		panel := el(dom.Div, func(st *style.Computed) {
			st.Display = c.display
			st.Direction = c.direction
			st.AlignItems = style.AlignCenter
			st.Padding = style.Sides[style.Length]{Top: style.Px(30), Right: style.Px(8), Bottom: style.Px(1),
				Left: style.Px(8)}
		}, plate, row)
		centring := el(dom.Div, func(st *style.Computed) {
			st.Display = style.FlexDisplay
			st.AlignItems = style.AlignCenter
			st.Size = style.Size[style.Length]{Width: style.Px(300), Height: style.Px(300)}
		}, panel)
		tr := lay(el(dom.Div, block, centring), 400)
		assert.Equal(t, float32(30+(-24+16+5)+16+1), tr.Of[panel].Size.Y, "%s: the panel fits its items", c.name)
		bottom := tr.Of[panel].Size.Y - (tr.Of[row].Pos.Y + tr.Of[row].Size.Y)
		assert.Equal(t, float32(1), bottom, "%s: the last item ends the padding above the bottom", c.name)
	}
}
