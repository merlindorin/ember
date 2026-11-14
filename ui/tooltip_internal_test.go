package ui

import (
	"testing"
	"testing/fstest"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const tipSheet = `
body { margin: 0; }
.anchor { position: absolute; width: 40px; height: 20px; }
.tip { position: absolute; left: 100%; top: 0; margin-left: 8px; display: none; width: 120px; height: 30px; }
.tip.below { left: 0; top: 100%; margin-left: 0; margin-top: 8px; }
.wide { width: 380px; }
.anchor:hover > .tip { display: block; }
.tip[data-side="left"] { left: auto; right: 100%; margin-left: 0; margin-right: 8px; }
.tip[data-side="above"] { top: auto; bottom: 100%; margin-top: 0; margin-bottom: 8px; }
.tip[data-clamp] { left: var(--tip-x); top: var(--tip-y); right: auto; bottom: auto; margin: 0; }
`

func tipView() geom.Vec2 { return geom.V2(400, 300) }

func tipDoc(at string, tipClasses ...string) (*Document, *Node, *Node) {
	tip := Div(Class(append([]string{"tip"}, tipClasses...)...), Attr("role", "tooltip"), Text("a tip"))
	anchor := Div(Class("anchor"), Style(at), tip)
	d := NewDocument(Stylesheet(fstest.MapFS{"t.css": {Data: []byte(tipSheet)}}, "t.css"))
	d.Body().Append(anchor)
	d.Layout(tipView())
	return d, anchor, tip
}

func hoverTip(d *Document, anchor *Node, on bool) {
	anchor.SetState(dom.Hover, on)
	d.Layout(tipView())
}

func inWindow(t *testing.T, tip *Node) {
	t.Helper()
	r := tip.Box()
	require.Positive(t, r.W)
	inside := r.X >= tipMargin-0.5 && r.Y >= tipMargin-0.5 &&
		r.X+r.W <= tipView().X-tipMargin+0.5 && r.Y+r.H <= tipView().Y-tipMargin+0.5
	assert.True(t, inside, "%v inside the window", r)
}

func TestATipThatFitsIsLeftAlone(t *testing.T) {
	d, anchor, tip := tipDoc("left: 20px; top: 20px")
	hoverTip(d, anchor, true)
	_, side := tip.Attr("data-side")
	_, clamp := tip.Attr("data-clamp")
	assert.False(t, side || clamp)
	assert.InDelta(t, 68, tip.Box().X, 0.01)
}

func TestATipAtTheRightEdgeShowsOnTheLeft(t *testing.T) {
	d, anchor, tip := tipDoc("left: 300px; top: 20px")
	hoverTip(d, anchor, true)
	v, _ := tip.Attr("data-side")
	assert.Equal(t, "left", v)
	assert.InDelta(t, 300-8-120, tip.Box().X, 0.01)
	inWindow(t, tip)
}

func TestATipBelowAtTheBottomShowsAbove(t *testing.T) {
	d, anchor, tip := tipDoc("left: 20px; top: 270px", "below")
	hoverTip(d, anchor, true)
	v, _ := tip.Attr("data-side")
	assert.Equal(t, "above", v)
	inWindow(t, tip)
}

func TestATipTooWideForEitherSideIsHeldInsideTheWindow(t *testing.T) {
	d, anchor, tip := tipDoc("left: 200px; top: 20px", "wide")
	hoverTip(d, anchor, true)
	_, clamp := tip.Attr("data-clamp")
	assert.True(t, clamp)
	inWindow(t, tip)
}

func TestAPlacedTipKeepsItsPlaceUntilItHides(t *testing.T) {
	d, anchor, tip := tipDoc("left: 300px; top: 20px")
	hoverTip(d, anchor, true)
	placed := d.placed
	d.Layout(tipView())
	d.Layout(tipView())
	assert.Equal(t, placed, d.placed, "no layout while nothing changes")
	v, _ := tip.Attr("data-side")
	assert.Equal(t, "left", v, "it stays where it was placed")

	hoverTip(d, anchor, false)
	_, side := tip.Attr("data-side")
	assert.False(t, side, "hidden, it is placed afresh next time")
}

func TestPlacingATipLaysOutOnceMore(t *testing.T) {
	d, anchor, _ := tipDoc("left: 300px; top: 20px")
	before := d.placed
	hoverTip(d, anchor, true)
	assert.Equal(t, 2*len(d.tree.Of), d.placed-before, "the layout that showed it and one after placing it")
}

func TestATipAgainstTheWindowsEdgeButInsideStays(t *testing.T) {
	d, anchor, tip := tipDoc("left: 20px; top: 0")
	hoverTip(d, anchor, true)
	_, clamp := tip.Attr("data-clamp")
	assert.False(t, clamp)
	assert.Equal(t, float32(0), tip.Box().Y)
}
