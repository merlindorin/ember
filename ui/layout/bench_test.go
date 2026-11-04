package layout //nolint:testpackage // tests the layout tree and its unexported helpers

import (
	"fmt"
	"testing"

	"github.com/merlindorin/ember/ui/internal/dom"
	"github.com/merlindorin/ember/ui/internal/style"
)

// inspectorTree is shaped like the tileset inspector: a title, a max-content 1fr grid of
// names and values, and rows of buttons.
func inspectorTree() *dom.Node {
	grid := el(dom.Div, func(st *style.Computed) {
		st.Display = style.GridDisplay
		st.Columns = []style.TrackRepeat{
			{Count: 1, Tracks: []style.Track{{
				Min: style.Length{Kind: style.MaxContent},
				Max: style.Length{Kind: style.MaxContent},
			}}},
			{Count: 1, Tracks: []style.Track{{
				Min: style.Length{Kind: style.Fr, Px: 1},
				Max: style.Length{Kind: style.Fr, Px: 1},
			}}},
		}
		st.Gap = style.Size[style.Length]{Width: style.Px(12), Height: style.Px(6)}
	})
	for i := range 24 {
		grid.Append(
			el(dom.Span, nil, text(fmt.Sprintf("property_%d", i))),
			el(dom.Div, func(st *style.Computed) {
				st.Display = style.Block
				st.MinSize.Width = style.Px(0)
				st.WhiteSpace = style.NoWrapSpace
				st.Overflow = style.Size[style.Overflow]{Width: style.Hidden, Height: style.Hidden}
				st.Ellipsis = true
			}, text("a value that may be long enough to cut")),
		)
	}
	row := func() *dom.Node {
		r := el(dom.Div, func(st *style.Computed) {
			st.Display = style.FlexDisplay
			st.Gap.Width = style.Px(8)
		})
		for range 3 {
			r.Append(el(dom.Button, func(st *style.Computed) {
				st.Display = style.InlineBlock
				st.Padding = style.All(style.Px(4))
			}, text("Action")))
		}
		return r
	}
	return el(dom.Div, block,
		el(dom.H2, block, text("tile 12")),
		grid,
		row(), row(), row(), row(),
	)
}

func BenchmarkInspectorLayout(b *testing.B) {
	root := inspectorTree()
	lay(root, 360)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		lay(root, 360)
	}
}
