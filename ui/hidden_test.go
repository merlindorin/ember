package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/ui"
)

func TestTextKeptUnderANewlyHiddenElementIsNeitherRecordedHitNorPlaced(t *testing.T) {
	press := ui.Button(ui.ID("press"), ui.Text("Secret"))
	inner := ui.Div(press)
	panel := ui.Div(ui.ID("panel"), inner)
	l := liveDoc(t, panel)
	text := press.Children()[0]
	at := press.Box().Center()
	require.NotZero(t, text.Box().W, "shown, the text is placed")

	panel.SetAttr("style", "display: none")
	l.frames(2)

	_, all := l.auto.Snapshot("", "")
	for _, e := range all {
		assert.NotEqual(t, "press", e.ID, "a hidden button isn't recorded")
		assert.NotEqual(t, "Secret", e.Label, "nor is its text")
	}
	in := l.win.Input()
	in.SetMouse(at)
	l.frames(1)
	in.SetMouseButton(input.MouseLeft, true)
	l.frames(1)
	in.SetMouseButton(input.MouseLeft, false)
	l.frames(2)
	assert.Empty(t, l.clicks, "a click where it stood presses nothing")
	_, ok := l.doc.BoxOf(text)
	assert.False(t, ok, "BoxOf reports no box for the text")
	assert.Equal(t, geom.Rect{}, text.Box(), "nor does its Box")
}
