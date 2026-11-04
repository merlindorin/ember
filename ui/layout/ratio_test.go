package layout //nolint:testpackage // tests the layout tree and its unexported helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/ui/internal/dom"
	"github.com/merlindorin/ember/ui/internal/style"
)

func TestAnAspectRatioItemsCrossSizeFollowsItsFlexedMainSize(t *testing.T) {
	for _, c := range []struct {
		name      string
		item, row func(*style.Computed)
		want      geom.Vec2
	}{
		{"shrunk in a row",
			func(st *style.Computed) { st.Size.Width = style.Px(40); st.MinSize.Width = style.Px(32); st.Shrink = 1 },
			func(st *style.Computed) { st.Size.Width = style.Px(32) }, geom.V2(32, 32)},
		{"grown in a row", func(st *style.Computed) { st.Size.Width = style.Px(20); st.Grow = 1 },
			func(st *style.Computed) { st.Size.Width = style.Px(120) }, geom.V2(120, 120)},
		{"shrunk in a column", func(st *style.Computed) {
			st.Size.Height = style.Px(40)
			st.MinSize.Height = style.Px(32)
			st.Shrink = 1
		},
			func(st *style.Computed) { st.Direction = style.Column; st.Size.Height = style.Px(32) }, geom.V2(32, 32)},
	} {
		t.Run(c.name, func(t *testing.T) {
			item := el(dom.Div, func(st *style.Computed) { st.Display = style.Block; st.AspectRatio = 1; c.item(st) })
			row := el(dom.Div, func(st *style.Computed) {
				st.Display = style.FlexDisplay
				st.AlignItems = style.AlignFlexStart
				c.row(st)
			}, item)
			tr := lay(el(dom.Div, block, row), 400)
			assert.Equal(t, c.want, tr.Of[item].Size)
		})
	}
}
