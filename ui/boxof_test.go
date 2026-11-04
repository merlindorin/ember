package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/ui"
)

func TestBoxOfIsAnElementsContentBoxAndFollowsAResize(t *testing.T) {
	slot := ui.Div(ui.Style("position: absolute; left: 10px; top: 20px; right: 30px; bottom: 40px; " +
		"padding: 5px; border: 2px solid #000"))
	gone := ui.Div(ui.Style("display: none"))
	l := liveDoc(t, slot, gone)
	r, ok := l.doc.BoxOf(slot)
	assert.True(t, ok)
	assert.Equal(t, geom.Rect{X: 17, Y: 27, W: 640 - 10 - 30 - 14, H: 360 - 20 - 40 - 14}, r,
		"inside its border and padding, in window points")

	l.win.SetSize(geom.V2(800, 400))
	l.frames(2)
	r, _ = l.doc.BoxOf(slot)
	assert.Equal(t, float32(800-10-30-14), r.W, "a resize reaches it")

	_, ok = l.doc.BoxOf(gone)
	assert.False(t, ok, "an element not shown has no box")
	_, ok = l.doc.BoxOf(ui.Div())
	assert.False(t, ok, "nor one outside the document")
}

func TestAClickPassesThroughAPointerEventsNoneOverlay(t *testing.T) {
	under := ui.Button(ui.Style("position: absolute; left: 10px; top: 10px; width: 100px; height: 40px"), ui.Text("Under"))
	glass := ui.Div(ui.Style("position: absolute; inset: 0; pointer-events: none"))
	solid := ui.Div(ui.Style("position: absolute; left: 200px; top: 0; width: 200px; height: 200px"))
	blocked := ui.Button(ui.Style("position: absolute; left: 210px; top: 10px; width: 100px; height: 40px"),
		ui.Text("Blocked"))
	l := liveDoc(t, under, glass, blocked, solid)
	l.clickAt(under)
	l.clickAt(blocked)
	assert.Equal(t, []*ui.Node{under}, l.clicks, "through the glass, never through a box that takes the pointer")
}

func TestPointerEventsSetAtRunTimeLetsTheNextClickThrough(t *testing.T) {
	under := ui.Button(ui.Style("position: absolute; left: 10px; top: 10px; width: 100px; height: 40px"), ui.Text("Under"))
	glass := ui.Div(ui.Style("position: absolute; inset: 0"))
	l := liveDoc(t, under, glass)
	l.clickAt(under)
	assert.Empty(t, l.clicks, "the glass takes the click")
	glass.SetStyle("position: absolute; inset: 0; pointer-events: none")
	l.frames(1)
	l.clickAt(under)
	assert.Equal(t, []*ui.Node{under}, l.clicks, "set to none, it lets the next one through")
}
