package layout //nolint:testpackage // tests the layout tree and its unexported helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/ui/internal/box"
	"github.com/merlindorin/ember/ui/internal/dom"
	"github.com/merlindorin/ember/ui/internal/style"
)

func onPage(b *box.Box) geom.Vec2 {
	var p geom.Vec2
	for ; b != nil; b = b.Parent {
		p = p.Add(b.Pos)
	}
	return p
}

func absBox(edit func(*style.Computed)) *dom.Node {
	return el(dom.Div, func(st *style.Computed) {
		st.Display = style.Block
		st.Position = style.Absolute
		st.Size = style.Size[style.Length]{Width: style.Px(5), Height: style.Px(5)}
		if edit != nil {
			edit(st)
		}
	})
}

func TestAnOutOfFlowBoxInASpanStartsWhereItStandsInTheLine(t *testing.T) {
	abs := absBox(nil)
	p := el(dom.P, width(200), text("HHH "), el(dom.Span, nil, text("HH"), abs, text("H")))
	tr := Layout(el(dom.Div, block, p), geom.V2(400, 300), ahemFonts{})
	assert.Equal(t, onPage(tr.Of[p]).Add(geom.V2(60, 0)), onPage(tr.Of[abs]), "after HHH, the space and HH")
}

func TestAPositionedSpanIsTheContainingBlockOfWhatItHolds(t *testing.T) {
	abs := absBox(func(st *style.Computed) {
		st.Inset.Right = style.Px(0)
		st.Inset.Bottom = style.Px(0)
	})
	s := el(dom.Span, func(st *style.Computed) {
		st.Position = style.Relative
		st.Padding.Left = style.Px(2)
	}, text("HH"), abs, text("H"))
	outer := el(dom.Div, func(st *style.Computed) { st.Display = style.Block; st.Position = style.Relative },
		el(dom.P, width(200), text("HHH "), s))
	tr := Layout(el(dom.Div, block, outer), geom.V2(400, 300), ahemFonts{})
	p := onPage(tr.Of[outer.Children()[0]])
	assert.Equal(t, p.Add(geom.V2(40+2+30-5, 10-5)), onPage(tr.Of[abs]), "the span's padding box, not the outer div")
}
