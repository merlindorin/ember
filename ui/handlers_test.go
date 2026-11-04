package ui_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/merlindorin/ember/automation"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/world"
)

func TestAClickRunsItsCallbackOnceBesideTheEvent(t *testing.T) {
	plain, withEvent, withWorld := 0, 0, 0
	var got ui.Event
	b := ui.Button(ui.Text("Save"),
		ui.OnClick(func() { plain++ }),
		ui.OnClick(func(e ui.Event) { withEvent++; got = e }),
		ui.OnClick(func(w *world.World, _ ui.Event) {
			if w != nil {
				withWorld++
			}
		}))
	l := liveDoc(t, b)
	l.clickAt(b)
	assert.Equal(t, 1, plain)
	assert.Equal(t, 1, withEvent)
	assert.Equal(t, 1, withWorld, "the World is passed")
	assert.Equal(t, []*ui.Node{b}, l.clicks, "the event is still emitted")
	assert.Equal(t, ui.EventClick, got.Type)
	assert.Same(t, b, got.Node)
	assert.Equal(t, 1, got.Clicks)
}

func TestACallbackChangesTheTreeForTheNextFrame(t *testing.T) {
	label := ui.Span(ui.Text("0"))
	n := 0
	b := ui.Button(ui.Text("Add"), ui.OnClick(func() { n++; label.SetText("1") }))
	l := liveDoc(t, b, label)
	l.clickAt(b)
	assert.Equal(t, "1", label.Text())
	assert.Greater(t, label.Box().W, float32(0))
}

func TestARemovedNodesCallbacksNeverRun(t *testing.T) {
	ran := 0
	b := ui.Button(ui.Text("Save"))
	remove := ui.On(b, ui.EventClick, func() { ran++ })
	l := liveDoc(t, b)
	l.clickAt(b)
	require.Equal(t, 1, ran)

	remove()
	l.clickAt(b)
	assert.Equal(t, 1, ran, "a removed callback")

	other := ui.Button(ui.Text("Other"), ui.OnClick(func() { ran++ }))
	l.doc.Body().Append(other)
	l.frames(2)
	at := other.Box().Center()
	l.doc.Body().Remove(other)
	in := l.win.Input()
	in.SetMouse(at)
	l.frames(1)
	in.SetMouseButton(input.MouseLeft, true)
	l.frames(1)
	in.SetMouseButton(input.MouseLeft, false)
	l.frames(2)
	assert.Equal(t, 1, ran, "a node out of the Document")
}

func TestAutomationsPressRunsOnClick(t *testing.T) {
	ran := 0
	b := ui.Button(ui.ID("save"), ui.Text("Save"), ui.OnClick(func() { ran++ }))
	l := liveDoc(t, b)
	l.act(t, "save", automation.ActPress)
	l.frames(2)
	assert.Equal(t, 1, ran)
}

func TestADoubleClickCountsItsClicks(t *testing.T) {
	var clicks []int
	b := ui.Button(ui.Text("Open"), ui.OnClick(func(e ui.Event) { clicks = append(clicks, e.Clicks) }))
	l := liveDoc(t, b)
	l.clickAt(b)
	l.clickAt(b)
	assert.Equal(t, []int{1, 2}, clicks)
}

func TestPointerEnterAndLeaveRunOnTheElementsTheyCross(t *testing.T) {
	var log []string
	inner := ui.Button(ui.Text("In"), ui.OnPointerEnter(func() { log = append(log, "enter inner") }),
		ui.OnPointerLeave(func() { log = append(log, "leave inner") }))
	outer := ui.Div(ui.Style("padding: 20px"), inner,
		ui.OnPointerEnter(func() { log = append(log, "enter outer") }), ui.OnPointerLeave(func() {
			log = append(log,
				"leave outer")
		}))
	l := liveDoc(t, outer)
	in := l.win.Input()
	in.SetMouse(geom.V2(outer.Box().X+5, outer.Box().Y+5))
	l.frames(2)
	var r world.Reader[ui.PointerEntered]
	r.Read(l.w)
	in.SetMouse(inner.Box().Center())
	l.frames(1)
	entered := slices.Collect(r.Read(l.w))
	l.frames(1)
	in.SetMouse(geom.V2(630, 350))
	l.frames(2)
	assert.Equal(t, []string{"enter outer", "enter inner", "leave inner", "leave outer"}, log,
		"no bubbling: entering the child doesn't enter the parent again")
	require.Len(t, entered, 1, "the World event is emitted as the others")
	assert.Same(t, inner, entered[0].Node)
}

func TestAnEventACallbackRaisesReachesItsOwnCallbacks(t *testing.T) {
	closed := 0
	box := ui.Dialog(ui.Text("Settings"), ui.OnClose(func() { closed++ }))
	var l *live
	shut := ui.Button(ui.Text("Shut"), ui.OnClick(func() { l.doc.Close(box, "") }))
	box.Append(shut)
	l = liveDoc(t, box)
	l.doc.Show(box)
	l.frames(1)
	l.clickAt(shut)
	assert.False(t, l.doc.Open(box))
	assert.Equal(t, 1, closed, "a dialog a click's callback closes runs its own close callback")
}

func TestACallbacksEventChainsThroughTheCallbacksItRaises(t *testing.T) {
	var order []string
	b := ui.Dialog(ui.Text("B"), ui.Style("top: 150px"), ui.OnClose(func() { order = append(order, "B closed") }))
	var l *live
	a := ui.Dialog(ui.Text("A"), ui.OnClose(func() {
		order = append(order, "A closed")
		l.doc.Close(b, "")
	}))
	shut := ui.Button(ui.Text("Shut"), ui.OnClick(func() { l.doc.Close(a, "") }))
	a.Append(shut)
	l = liveDoc(t, a, b)
	l.doc.Show(a)
	l.doc.Show(b)
	l.frames(1)
	l.clickAt(shut)
	assert.False(t, l.doc.Open(a))
	assert.False(t, l.doc.Open(b))
	assert.Equal(t, []string{"A closed", "B closed"}, order,
		"a click closes A, whose callback closes B, whose own callback runs")
}
