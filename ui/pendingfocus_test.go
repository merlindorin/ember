package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/merlindorin/ember/ui"
)

func TestAFocusAskedBeforeAnElementShowsTakesOnceItIsLaidOut(t *testing.T) {
	ok := ui.Button(ui.Text("OK"))
	other := ui.Button(ui.Text("Other"))
	panel := ui.Div(ui.Style("display: none"), ok, other)
	l := liveDoc(t, panel)
	panel.SetStyle("display: block")
	l.doc.Focus(ok)
	assert.False(t, l.doc.Focused(ok), "not laid out yet, it can't take the focus at once")
	l.frames(2)
	assert.True(t, l.doc.Focused(ok), "laid out, it has it")

	panel.SetStyle("display: none")
	l.frames(2)
	panel.SetStyle("display: block")
	l.doc.Focus(ok)
	l.doc.Focus(other)
	l.frames(2)
	assert.True(t, l.doc.Focused(other), "a later Focus replaces the pending one")
}

func TestAPendingFocusIsDroppedWhenItsElementLeaves(t *testing.T) {
	ok := ui.Button(ui.Text("OK"))
	l := liveDoc(t, ui.Div())
	body := l.doc.Body()
	body.Append(ok)
	l.doc.Focus(ok)
	body.Remove(ok)
	l.frames(2)
	body.Append(ok)
	l.frames(2)
	assert.False(t, l.doc.Focused(ok), "gone before it could take the focus, it doesn't take it on coming back")
}
