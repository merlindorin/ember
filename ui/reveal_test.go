package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/ui"
)

func TestScrollIntoViewShowsAnElementLaidOutLater(t *testing.T) {
	sc := ui.Div(ui.Style("height: 100px; overflow: auto"))
	for range 10 {
		sc.Append(ui.Div(ui.Style("height: 40px")))
	}
	l := liveDoc(t, sc)
	last := ui.Div(ui.Style("height: 40px"), ui.Text("added"))
	sc.Append(last)
	l.doc.ScrollIntoView(last)
	l.frames(3)
	view, box := sc.Box(), last.Box()
	assert.GreaterOrEqual(t, box.Y, view.Y-0.5, "its top is in the scroller's view")
	assert.LessOrEqual(t, box.Y+box.H, view.Y+view.H+0.5, "and its bottom")
}

func TestScrollIntoViewOfARemovedElementDoesNothing(t *testing.T) {
	sc := ui.Div(ui.Style("height: 100px; overflow: auto"))
	for range 10 {
		sc.Append(ui.Div(ui.Style("height: 40px")))
	}
	l := liveDoc(t, sc)
	first := sc.Children()[0].Box()
	l.doc.ScrollIntoView(ui.Div())
	l.frames(2)
	assert.Equal(t, first, sc.Children()[0].Box())
}
