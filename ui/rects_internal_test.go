package ui

import (
	"fmt"
	"testing"
	"testing/fstest"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/ui/internal/dom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func placedRight(t *testing.T, d *Document) {
	t.Helper()
	for n, b := range d.tree.Of {
		require.Equal(t, b.Rect(), n.Box(), "%s", n.Tag())
	}
}

func scrollingList(rows int) (*Document, *Node, []*Node, *Node) {
	sheet := fstest.MapFS{"a.css": {Data: []byte(`
		#list { overflow: auto; height: 100px; width: 200px; }
		.row { height: 20px; }
		.sticky { position: sticky; top: 0; height: 10px; }
		.abs { position: absolute; left: 3px; top: 4px; width: 5px; height: 5px; }`)}}
	list := Div(ID("list"), Div(Class("sticky")))
	var items []*Node
	for i := range rows {
		r := Div(Class("row"), Text(fmt.Sprint("row ", i)), Span(Class("abs")))
		items = append(items, r)
		list.Append(r)
	}
	outside := Div(Text("outside"), Div(Style("display: none"), Span(Text("hidden"))),
		Div(Style("display: contents"), Span(Text("contents"))))
	d := NewDocument(Stylesheet(sheet, "a.css"))
	d.Body().Append(outside, Div(Style("position: relative"), list))
	return d, list, items, outside
}

func TestLayoutPlacesEveryNodeAtItsBox(t *testing.T) {
	d, _, _, _ := scrollingList(20)
	d.Layout(geom.V2(300, 200))
	placedRight(t, d)
}

func TestAScrollMovesTheBoxesAndTouchesNoNode(t *testing.T) {
	d, list, items, _ := scrollingList(20)
	d.Layout(geom.V2(300, 200))
	top, placed := items[0].Box().Y, d.placed

	dom.Data(list).Scroll = geom.V2(0, 60)
	d.Layout(geom.V2(300, 200))

	assert.Equal(t, placed, d.placed)
	assert.InDelta(t, top-60, items[0].Box().Y, 0.01)
	placedRight(t, d)
}

func TestAQuietFrameTouchesNoNode(t *testing.T) {
	d, _, _, _ := scrollingList(20)
	d.Layout(geom.V2(300, 200))
	placed := d.placed
	d.Layout(geom.V2(300, 200))
	assert.Equal(t, placed, d.placed)
}

func TestAHiddenNodeLosesItsRect(t *testing.T) {
	d, _, _, outside := scrollingList(3)
	shown := Span(Text("shown"))
	outside.Append(shown)
	d.Layout(geom.V2(300, 200))
	require.Positive(t, shown.Box().W)

	shown.SetStyle("display: none")
	d.Layout(geom.V2(300, 200))
	assert.Equal(t, geom.Rect{}, shown.Box())
	placedRight(t, d)
}

func TestARemovedNodeLosesItsBox(t *testing.T) {
	d, list, items, outside := scrollingList(3)
	d.Layout(geom.V2(300, 200))
	require.Positive(t, items[1].Box().W)

	list.Remove(items[1])
	outside.SetText("replaced")
	d.Layout(geom.V2(300, 200))

	assert.Equal(t, geom.Rect{}, items[1].Box())
	placedRight(t, d)
}

func TestWhatAHiddenElementHoldsLosesItsBox(t *testing.T) {
	inner := Span(Text("inner"))
	inline := Span(Text("inline"))
	wrap := Div(inner)
	d := NewDocument()
	d.Body().Append(wrap, P(Text("a "), inline))
	d.Layout(geom.V2(300, 200))
	require.Positive(t, inner.Box().W)
	require.Positive(t, inline.Box().W)

	wrap.SetStyle("display: none")
	inline.SetStyle("display: none")
	d.Layout(geom.V2(300, 200))
	assert.Equal(t, geom.Rect{}, inner.Box(), "under a hidden block")
	assert.Equal(t, geom.Rect{}, inline.Box(), "a hidden inline element")
}
