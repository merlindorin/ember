package ui_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/world"
)

func TestBackAndKeysBubbleFromTheFocusedElementUp(t *testing.T) {
	var got []string
	note := func(name string) func(ui.Event) {
		return func(e ui.Event) { got = append(got, name+":"+e.Node.ID()) }
	}
	b := ui.Button(ui.ID("b"), ui.Text("B"), ui.OnBack(note("button")), ui.OnKey(note("button")))
	inner := ui.Div(ui.ID("inner"), ui.OnBack(note("inner")), ui.OnKey(note("inner")), b)
	outer := ui.Div(ui.ID("outer"), ui.OnBack(note("outer")), inner)
	l := liveDoc(t, outer)
	ui.On(l.doc.Body(), ui.EventBack, note("body"))
	l.doc.Focus(b)
	l.frames(1)

	press(l, input.KeyEscape)
	l.frames(1)
	assert.Equal(t, []string{"button:b", "inner:b", "outer:b", "body:b"}, got, "innermost first, the target as Node")

	got = nil
	press(l, input.KeyF)
	l.frames(1)
	assert.Equal(t, []string{"button:b", "inner:b"}, got, "a key bubbles too")
}

func TestAStoppedBackGoesNoHigher(t *testing.T) {
	var got []string
	b := ui.Button(ui.ID("b"), ui.Text("B"))
	inner := ui.Div(ui.OnBack(func(e ui.Event) { got = append(got, "inner"); e.StopPropagation() }), b)
	outer := ui.Div(ui.OnBack(func() { got = append(got, "outer") }), inner)
	l := liveDoc(t, outer)
	l.doc.Focus(b)
	l.frames(1)
	var r world.Reader[ui.BackPressed]
	press(l, input.KeyEscape)
	backs := slices.Collect(r.Read(l.w))
	l.frames(1)
	backs = append(backs, slices.Collect(r.Read(l.w))...)
	assert.Equal(t, []string{"inner"}, got)
	assert.Len(t, backs, 1, "the World event is emitted once, stopped or not")
}

func TestClicksDontBubble(t *testing.T) {
	clicks := 0
	b := ui.Button(ui.Text("B"))
	l := liveDoc(t, ui.Div(ui.OnClick(func() { clicks++ }), b))
	l.clickAt(b)
	assert.Zero(t, clicks)
}

func TestBackWithNothingFocusedReachesTheBodysHandlers(t *testing.T) {
	backs := 0
	l := liveDoc(t, ui.Button(ui.Text("B")))
	ui.On(l.doc.Body(), ui.EventBack, func() { backs++ })
	var r world.Reader[ui.BackPressed]
	press(l, input.KeyEscape)
	events := slices.Collect(r.Read(l.w))
	l.frames(1)
	events = append(events, slices.Collect(r.Read(l.w))...)
	assert.Equal(t, 1, backs)
	assert.Empty(t, events, "no World event without a focused element, as before")
}

func TestAnAncestorRemovedOnTheWayIsSkipped(t *testing.T) {
	var got []string
	b := ui.Button(ui.Text("B"))
	inner := ui.Div(b)
	outer := ui.Div(ui.OnBack(func() { got = append(got, "outer") }), inner)
	l := liveDoc(t, outer)
	ui.On(inner, ui.EventBack, func() { got = append(got, "inner"); l.doc.Body().Remove(outer) })
	ui.On(l.doc.Body(), ui.EventBack, func() { got = append(got, "body") })
	l.doc.Focus(b)
	l.frames(1)
	press(l, input.KeyEscape)
	l.frames(1)
	assert.Equal(t, []string{"inner"}, got, "outer left the Document, and the body is no longer its ancestor")
}

func TestAPadButtonNavigationLeavesReachesOnKey(t *testing.T) {
	var got []ui.Event
	b := ui.Button(ui.ID("b"), ui.Text("B"), ui.OnKey(func(e ui.Event) { got = append(got, e) }))
	l := liveDoc(t, b)
	l.doc.Focus(b)
	l.frames(1)
	in := l.win.Input()
	in.ConnectGamepad(0, "pad")
	for _, pad := range []input.GamepadButton{input.GamepadWest, input.GamepadNorth, input.GamepadSouth} {
		in.SetGamepadButton(0, pad, true)
		l.frames(1)
		in.SetGamepadButton(0, pad, false)
		l.frames(1)
	}
	if assert.Len(t, got, 2, "West and North; South activates the button") {
		assert.Equal(t, input.KeyUnknown, got[0].Key)
		assert.Equal(t, []input.GamepadButton{input.GamepadWest, input.GamepadNorth},
			[]input.GamepadButton{got[0].Pad, got[1].Pad})
	}
}
