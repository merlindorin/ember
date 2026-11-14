package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/automation"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/world"
)

// heard counts the World's drag and right-click events over the frames
// it reads.
type heard struct {
	contexts []ui.ContextClicked
	drops    []ui.Dropped
	ends     []ui.DragEnded
	backs    int
	cr       world.Reader[ui.ContextClicked]
	dr       world.Reader[ui.Dropped]
	er       world.Reader[ui.DragEnded]
	br       world.Reader[ui.BackPressed]
}

func (h *heard) frames(l *live, n int) {
	for range n {
		l.frames(1)
		for e := range h.cr.Read(l.w) {
			h.contexts = append(h.contexts, e)
		}
		for e := range h.dr.Read(l.w) {
			h.drops = append(h.drops, e)
		}
		for e := range h.er.Read(l.w) {
			h.ends = append(h.ends, e)
		}
		h.backs += h.br.Count(l.w)
	}
}

// press holds button over at for a frame, then the pointer goes through
// each of path a frame apiece.
func (h *heard) press(l *live, button input.MouseButton, at geom.Vec2, path ...geom.Vec2) {
	in := l.win.Input()
	in.SetMouse(at)
	h.frames(l, 1)
	in.SetMouseButton(button, true)
	h.frames(l, 1)
	for _, p := range path {
		in.SetMouse(p)
		h.frames(l, 1)
	}
}

func (h *heard) release(l *live, button input.MouseButton) {
	l.win.Input().SetMouseButton(button, false)
	h.frames(l, 2)
}

func TestARightClickRunsItsElementsOnContextMenuThenItsAncestors(t *testing.T) {
	var order []string
	leaf := ui.Button(ui.Text("Slot 1"), ui.OnContextMenu(func() { order = append(order, "slot") }))
	other := ui.Button(ui.Text("Slot 2"))
	row := ui.Div(ui.OnContextMenu(func(e ui.Event) {
		order = append(order, "row")
		assert.Same(t, leaf, e.Node, "the event is about the element clicked")
	}), leaf, other)
	l := liveDoc(t, row)
	var h heard

	h.press(l, input.MouseRight, leaf.Box().Center())
	h.release(l, input.MouseRight)
	assert.Equal(t, []string{"slot", "row"}, order, "innermost first, as a DOM contextmenu bubbles")
	require.Len(t, h.contexts, 1, "one World event")
	assert.Same(t, leaf, h.contexts[0].Node)
	assert.Empty(t, l.clicks, "a right click isn't a click")

	order = nil
	h.press(l, input.MouseRight, leaf.Box().Center(), other.Box().Center())
	h.release(l, input.MouseRight)
	assert.Empty(t, order, "pressed on one element and released on another, it is no right click")
}

// slots is a row of a draggable slot, a drop target and a bare gap, returned
// in that order after the page and before the drop events they record.
func slots(t *testing.T) (*live, *ui.Node, *ui.Node, *ui.Node, *[]ui.Event) {
	got := &[]ui.Event{}
	source := ui.Button(ui.Text("Bow"), ui.Draggable(), ui.Style("width: 80px; height: 40px"),
		ui.OnDragEnd(func(e ui.Event) { *got = append(*got, e) }))
	target := ui.Div(ui.Style("width: 80px; height: 40px"), ui.Text("Empty"),
		ui.OnDrop(func(e ui.Event) { *got = append(*got, e) }))
	gap := ui.Div(ui.Style("width: 80px; height: 40px"))
	l := liveDoc(t, ui.Div(ui.Style("display: flex; gap: 20px"), source, target, gap))
	return l, source, target, gap, got
}

func TestAPressMovedUnderFourPointsOnADraggableStaysAClick(t *testing.T) {
	l, source, _, _, got := slots(t)
	var h heard
	at := source.Box().Center()
	h.press(l, input.MouseLeft, at, at.Add(geom.V2(3, 0)))
	_, dragging := source.Attr("data-dragging")
	assert.False(t, dragging, "3 points isn't a drag")
	h.release(l, input.MouseLeft)
	assert.Contains(t, l.clicks, source, "it clicks")
	assert.Empty(t, *got)
	assert.Empty(t, h.drops)
	assert.Empty(t, h.ends)
}

func TestASlotDraggedOntoATargetIsDroppedThere(t *testing.T) {
	l, source, target, _, got := slots(t)
	var h heard
	from, to := source.Box().Center(), target.Box().Center()
	h.press(l, input.MouseLeft, from, from.Add(geom.V2(6, 0)), to)
	_, dragging := source.Attr("data-dragging")
	_, over := target.Attr("data-drag-over")
	assert.True(t, dragging, "the source carries data-dragging while dragged")
	assert.True(t, over, "the target under the pointer carries data-drag-over")

	h.release(l, input.MouseLeft)
	require.Len(t, *got, 2, "the target's OnDrop, then the source's OnDragEnd")
	assert.Equal(t, ui.EventDrop, (*got)[0].Type)
	assert.Same(t, target, (*got)[0].Node)
	assert.Same(t, source, (*got)[0].Dragged)
	assert.Equal(t, ui.EventDragEnd, (*got)[1].Type)
	assert.Same(t, source, (*got)[1].Node)
	assert.True(t, (*got)[1].Dropped)
	assert.Equal(t, []ui.Dropped{{Target: target, Dragged: source}}, h.drops, "one Dropped")
	assert.Equal(t, []ui.DragEnded{{Source: source, Dropped: true}}, h.ends, "one DragEnded")
	assert.NotContains(t, l.clicks, source, "a drag is no click")
	_, dragging = source.Attr("data-dragging")
	_, over = target.Attr("data-drag-over")
	assert.False(t, dragging || over, "both attributes go once it's over")
}

func TestEscapeCancelsADragAndIsNoBack(t *testing.T) {
	l, source, target, _, got := slots(t)
	var h heard
	from, to := source.Box().Center(), target.Box().Center()
	h.press(l, input.MouseLeft, from, from.Add(geom.V2(6, 0)), to)
	in := l.win.Input()
	in.SetKey(input.KeyEscape, true)
	h.frames(l, 1)
	in.SetKey(input.KeyEscape, false)
	h.frames(l, 1)
	h.release(l, input.MouseLeft)
	assert.Empty(t, h.drops, "nothing is dropped")
	assert.Equal(t, []ui.DragEnded{{Source: source}}, h.ends, "the drag ends, not dropped")
	require.Len(t, *got, 1)
	assert.False(t, (*got)[0].Dropped)
	assert.Zero(t, h.backs, "Escape went to the drag")
	_, over := target.Attr("data-drag-over")
	assert.False(t, over)
}

func TestADragReleasedOverNoTargetIsCancelled(t *testing.T) {
	l, source, _, gap, _ := slots(t)
	var h heard
	from := source.Box().Center()
	h.press(l, input.MouseLeft, from, from.Add(geom.V2(6, 0)), gap.Box().Center())
	h.release(l, input.MouseLeft)
	assert.Empty(t, h.drops)
	assert.Equal(t, []ui.DragEnded{{Source: source}}, h.ends)
	_, dragging := source.Attr("data-dragging")
	assert.False(t, dragging)
}

func TestAutomationDragsAndRightClicks(t *testing.T) {
	l, source, target, _, _ := slots(t)
	source.SetAttr("data-name", "bow")
	target.SetAttr("data-name", "empty")
	target.SetAttr("tabindex", "0")
	right := 0
	ui.On(target, ui.EventContext, func() { right++ })
	var h heard
	h.frames(l, 2)
	path := func(name string) string {
		_, all := l.auto.Snapshot("", "")
		for _, e := range all {
			if e.ID == name || len(e.ID) > len(name) && e.ID[len(e.ID)-len(name)-1:] == "/"+name {
				return e.ID
			}
		}
		require.Failf(t, "not recorded", "%s", name)
		return ""
	}
	_, all := l.auto.Snapshot("", "")
	for _, e := range all {
		switch e.ID {
		case path("bow"):
			assert.Contains(t, e.Acts, automation.ActDrag, "a draggable records drag")
		case path("empty"):
			assert.Contains(t, e.Acts, automation.ActContext, "a right-clickable records context")
		}
	}
	l.auto.Queue(path("bow"), automation.ActDrag, path("empty"))
	h.frames(l, 3)
	assert.Equal(t, []ui.Dropped{{Target: target, Dragged: source}}, h.drops, "drag drops on the path given")
	l.auto.Queue(path("empty"), automation.ActContext, nil)
	h.frames(l, 3)
	assert.Equal(t, 1, right, "context right-clicks the element")
}
