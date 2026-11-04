package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/ui"
)

func TestAZoomedDocumentLaysOutInAViewportThatMuchSmaller(t *testing.T) {
	full := ui.Div(ui.Style("height: 10px"))
	l := liveDocWith(t, []ui.DocumentOption{ui.Zoom(2)}, full)
	assert.Equal(t, float32(2), l.doc.Zoom())
	assert.Equal(t, float32(304), full.Box().W, "a 640pt window holds 320pt of page at zoom 2, less body's margins")
	r, ok := l.doc.BoxOf(full)
	require.True(t, ok)
	assert.Equal(t, float32(608), r.W, "BoxOf is in window points")

	l.doc.SetZoom(1)
	l.frames(2)
	assert.Equal(t, float32(624), full.Box().W)
}

func TestAZoomedDocumentTakesClicksWhereItDrawsThem(t *testing.T) {
	b := ui.Button(ui.Style("position: absolute; left: 100px; top: 50px; width: 60px; height: 20px"), ui.Text("Play"))
	l := liveDocWith(t, []ui.DocumentOption{ui.Zoom(1.5)}, b)
	in := l.win.Input()
	in.SetMouse(geom.V2(130*1.5, 60*1.5))
	l.frames(1)
	in.SetMouseButton(input.MouseLeft, true)
	l.frames(1)
	in.SetMouseButton(input.MouseLeft, false)
	l.frames(2)
	assert.Equal(t, []*ui.Node{b}, l.clicks)

	_, all := l.auto.Snapshot("", "")
	for _, e := range all {
		if e.ID == "play" {
			assert.Equal(t, geom.Rect{X: 150, Y: 75, W: 90, H: 30}, e.Bounds, "automation's bounds are window points")
		}
	}
}
