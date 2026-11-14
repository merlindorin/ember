package layout //nolint:testpackage // tests the layout tree and its unexported helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func blockOf(h float32, edit func(*style.Computed), children ...*dom.Node) *dom.Node {
	return el(dom.Div, func(st *style.Computed) {
		st.Display = style.Block
		st.Size.Height = style.Px(h)
		if edit != nil {
			edit(st)
		}
	}, children...)
}

func scrollBox(scroll float32, children ...*dom.Node) *dom.Node {
	b := blockOf(100, func(st *style.Computed) {
		st.Size.Width = style.Px(100)
		st.Overflow = style.Size[style.Overflow]{Width: style.Hidden, Height: style.Hidden}
	}, children...)
	dom.Data(b).Scroll = geom.V2(0, scroll)
	return b
}

func stickyTop(top float32) func(*style.Computed) {
	return func(st *style.Computed) {
		st.Position = style.Sticky
		st.Inset.Top = style.Px(top)
	}
}

func TestStickyStaysInFlowUntilReached(t *testing.T) {
	header := blockOf(10, stickyTop(0))
	sc := scrollBox(10, blockOf(20, nil), header, blockOf(500, nil))
	tr := lay(el(dom.Div, block, sc), 400)
	assert.Equal(t, float32(20), tr.Of[header].Pos.Y)
}

func TestStickySticksToTheScrollportTop(t *testing.T) {
	header := blockOf(10, stickyTop(5))
	sc := scrollBox(50, blockOf(20, nil), header, blockOf(500, nil))
	tr := lay(el(dom.Div, block, sc), 400)
	assert.Equal(t, float32(55), tr.Of[header].Pos.Y, "the scroll offset plus top")
	assert.Equal(t, tr.Of[sc].Rect().Y+5, tr.Of[header].Rect().Y)
}

func TestStickyStaysInItsContainingBlock(t *testing.T) {
	header := blockOf(10, stickyTop(0))
	section := blockOf(60, nil, header)
	sc := scrollBox(100, section, blockOf(500, nil))
	tr := lay(el(dom.Div, block, sc), 400)
	assert.Equal(t, float32(50), tr.Of[header].Pos.Y, "its bottom meets the section's bottom")
}

func TestStickyBottomSticksToTheScrollportBottom(t *testing.T) {
	footer := blockOf(20, func(st *style.Computed) {
		st.Position = style.Sticky
		st.Inset.Bottom = style.Px(0)
	})
	sc := scrollBox(0, blockOf(150, nil), footer)
	tr := lay(el(dom.Div, block, sc), 400)
	assert.Equal(t, float32(80), tr.Of[footer].Pos.Y)
}

func TestAStickyTableHeaderSticksWithinTheTable(t *testing.T) {
	th := cell("Name", stickyTop(0))
	var rows []*dom.Node
	for range 20 {
		rows = append(rows, row(cell("tile")))
	}
	tb := table(el(dom.Thead, nil, row(th)), el(dom.Tbody, nil, rows...))
	sc := scrollBox(100, tb)
	tr := lay(el(dom.Div, block, sc), 400)
	assert.InDelta(t, tr.Of[sc].Rect().Y, tr.Of[th].Rect().Y, 0.01,
		"a header cell sticks to the scrollport, not held in its row")
}
