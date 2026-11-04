package layout //nolint:testpackage // tests the layout tree and its unexported helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/ui/internal/dom"
	"github.com/merlindorin/ember/ui/internal/style"
)

func scroller(scroll geom.Vec2) (*dom.Node, *dom.Node) {
	content := el(dom.Div, func(st *style.Computed) {
		st.Display = style.Block
		st.Size = style.Size[style.Length]{Width: style.Px(300), Height: style.Px(500)}
	})
	box := el(dom.Div, func(st *style.Computed) {
		st.Display = style.Block
		st.Size = style.Size[style.Length]{Width: style.Px(100), Height: style.Px(100)}
		st.Overflow = style.Size[style.Overflow]{Width: style.OverflowAuto, Height: style.OverflowAuto}
	}, content)
	dom.Data(box).Scroll = scroll
	return box, content
}

func TestScrollIsClampedToTheOverflow(t *testing.T) {
	for _, c := range []struct{ in, want geom.Vec2 }{
		{geom.V2(50, 60), geom.V2(50, 60)},
		{geom.V2(1000, 1000), geom.V2(200, 400)},
		{geom.V2(-5, -5), geom.V2(0, 0)},
	} {
		b, content := scroller(c.in)
		tr := lay(el(dom.Div, block, b), 400)
		assert.Equal(t, c.want, dom.Data(b).Scroll, "kept offset for %v", c.in)
		assert.Equal(t, c.want, tr.Of[b].Scroll, "box offset for %v", c.in)
		assert.Equal(t, tr.Of[b].Rect().Y-c.want.Y, tr.Of[content].Rect().Y)
	}
}

func TestScrollOfANonScrollerIsZero(t *testing.T) {
	b := el(dom.Div, block)
	dom.Data(b).Scroll = geom.V2(10, 10)
	tr := lay(el(dom.Div, block, b), 400)
	assert.Equal(t, geom.Vec2{}, dom.Data(b).Scroll)
	assert.Equal(t, geom.Vec2{}, tr.Of[b].Scroll)
}

func TestZeroPercentBasisIsContentSizedWhenIndefinite(t *testing.T) {
	item := el(dom.Div, func(st *style.Computed) {
		st.Display = style.Block
		st.Basis = style.Length{Kind: style.Fixed, IsPercent: true}
	}, text("HHHH"))
	col := el(dom.Div, func(st *style.Computed) {
		st.Display = style.FlexDisplay
		st.Direction = style.Column
	}, item)
	tr := lay(el(dom.Div, block, col), 400)
	assert.Equal(t, float32(10), tr.Of[item].Size.Y, "0% of an indefinite height is content-sized, not 0")
}

func autoBox(w, h float32) func(*style.Computed) {
	return func(st *style.Computed) {
		st.Display = style.Block
		st.Size = style.Size[style.Length]{Width: style.Px(w), Height: style.Px(h)}
		st.Overflow = style.Size[style.Overflow]{Width: style.OverflowAuto, Height: style.OverflowAuto}
		st.ScrollbarWidth = 15
	}
}

func TestAnAutoScrollerThatOverflowsMakesRoomForItsBar(t *testing.T) {
	p := el(dom.P, block, text("HHHH HHHH HHHH HHHH HHHH HHHH"))
	sc := el(dom.Div, autoBox(100, 20), p)
	tr := lay(el(dom.Div, block, sc), 400)
	assert.Equal(t, geom.Vec2{X: 15}, tr.Of[sc].Scrollbar, "a vertical bar, no horizontal one")
	assert.Equal(t, float32(85), tr.Of[p].Size.X, "the paragraph beside the bar")
	assert.Len(t, lines(tr.Of[p]), 6, "HHHH HHHH fits 100, not 85")
	assert.Equal(t, geom.Vec2{X: 85, Y: 60}, tr.Of[sc].Content)
}

func TestAnAutoScrollerThatFitsKeepsItsWidth(t *testing.T) {
	p := el(dom.P, block, text("HHHH HHHH"))
	sc := el(dom.Div, autoBox(100, 20), p)
	tr := lay(el(dom.Div, block, sc), 400)
	assert.Zero(t, tr.Of[sc].Scrollbar)
	assert.Equal(t, float32(100), tr.Of[p].Size.X)
}

func TestAVerticalBarCanMakeItOverflowSideways(t *testing.T) {
	wide := el(dom.Div, func(st *style.Computed) {
		st.Display = style.Block
		st.Size = style.Size[style.Length]{Width: style.Px(90), Height: style.Px(40)}
	})
	sc := el(dom.Div, autoBox(100, 20), wide)
	tr := lay(el(dom.Div, block, sc), 400)
	assert.Equal(t, geom.Vec2{X: 15, Y: 15}, tr.Of[sc].Scrollbar, "90 fits 100, not the 85 beside the bar")
}

func TestAScrollerWithoutBarsReservesNothing(t *testing.T) {
	p := el(dom.P, block, text("HHHH HHHH HHHH HHHH HHHH HHHH"))
	sc := el(dom.Div, func(st *style.Computed) {
		autoBox(100, 20)(st)
		st.ScrollbarWidth = 0
	}, p)
	tr := lay(el(dom.Div, block, sc), 400)
	assert.Zero(t, tr.Of[sc].Scrollbar)
	assert.Equal(t, float32(100), tr.Of[p].Size.X)
}
