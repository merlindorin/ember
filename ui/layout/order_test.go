package layout //nolint:testpackage // tests the layout tree and its unexported helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func sized(w float32, order int) *dom.Node {
	return el(dom.Div, func(st *style.Computed) {
		st.Display = style.Block
		st.Size = style.Size[style.Length]{Width: style.Px(w), Height: style.Px(10)}
		st.Order = order
	})
}

func TestFlexItemsFollowOrder(t *testing.T) {
	a, b, c := sized(10, 2), sized(20, 0), sized(30, 1)
	row := el(dom.Div, func(st *style.Computed) { st.Display = style.FlexDisplay }, a, b, c)
	tr := lay(el(dom.Div, block, row), 400)
	assert.Equal(t, float32(0), tr.Of[b].Pos.X)
	assert.Equal(t, float32(20), tr.Of[c].Pos.X)
	assert.Equal(t, float32(50), tr.Of[a].Pos.X)
}

func TestEqualOrdersKeepSourceOrder(t *testing.T) {
	a, b, c := sized(10, 1), sized(20, 1), sized(30, -1)
	row := el(dom.Div, func(st *style.Computed) { st.Display = style.FlexDisplay }, a, b, c)
	tr := lay(el(dom.Div, block, row), 400)
	assert.Equal(t, float32(0), tr.Of[c].Pos.X)
	assert.Equal(t, float32(30), tr.Of[a].Pos.X)
	assert.Equal(t, float32(40), tr.Of[b].Pos.X)
}

func TestGridAutoPlacementFollowsOrder(t *testing.T) {
	a, b := sized(10, 1), sized(10, 0)
	grid := el(dom.Div, func(st *style.Computed) {
		st.Display = style.GridDisplay
		st.Columns = []style.TrackRepeat{{Count: 2, Tracks: []style.Track{{Min: style.Px(50), Max: style.Px(50)}}}}
	}, a, b)
	tr := lay(el(dom.Div, block, grid), 400)
	assert.Equal(t, float32(0), tr.Of[b].Pos.X)
	assert.Equal(t, float32(50), tr.Of[a].Pos.X)
}
