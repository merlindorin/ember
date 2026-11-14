package ui

import (
	"slices"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/world"
)

// Handler is what a callback may be: a plain func(), one taking the
// Event, or one taking the World too (decision 0057).
type Handler interface {
	func() | func(Event) | func(*world.World, Event)
}

// EventType is the kind of an Event.
type EventType int

const (
	EventClick EventType = iota + 1
	EventInput
	EventChange
	EventSubmit
	EventToggle
	EventClose
	EventSort
	EventKey
	EventBack
	EventPointerEnter
	EventPointerLeave
	EventMove
	EventResize
	EventFold
	EventContext
	EventDrop
	EventDragEnd
)

// Event is what a callback is told: the event's type and element, and
// everything the typed World event for it carries.
type Event struct {
	Type EventType
	Node *Node
	// Clicks is 2 or 3 for a double or triple click, else 1 (EventClick).
	Clicks int
	// Value is a dialog's return value (Close).
	Value string
	// Column and Descending are a sort's (Sort).
	Column     int
	Descending bool
	// Key, Text and Command are a key's (Key); Pad is a pad button's,
	// Key then KeyUnknown.
	Key     input.Key
	Text    string
	Command bool
	Pad     input.GamepadButton
	// At is a window's top-left once moved (Move), Size its size once
	// resized (Resize), and Folded whether it folded (Fold).
	At     geom.Vec2
	Size   geom.Vec2
	Folded bool
	// Dragged is what was dropped on the element (Drop), and Dropped
	// whether a target took the element's drag (DragEnd).
	Dragged *Node
	Dropped bool

	// stopped is where StopPropagation tells the dispatch to go no higher.
	stopped *bool
}

// StopPropagation keeps a bubbling Back or key from reaching the handlers
// of the element's ancestors further up (decision 0060).
func (e Event) StopPropagation() {
	if e.stopped != nil {
		*e.stopped = true
	}
}

// bubbles reports whether t runs on the element's ancestors too, innermost
// first, as a DOM keydown or contextmenu does: Back and keys, which land on
// the focused element (decision 0060), and a right click (decision 0062).
// Every other event stays on its element.
func bubbles(t EventType) bool { return t == EventBack || t == EventKey || t == EventContext }

// OnClick runs fn after each Clicked on the element.
func OnClick[F Handler](fn F) Arg { return on(EventClick, fn) }

// OnInput runs fn after each Edited on the element.
func OnInput[F Handler](fn F) Arg { return on(EventInput, fn) }

// OnChange runs fn after each Changed on the element.
func OnChange[F Handler](fn F) Arg { return on(EventChange, fn) }

// OnSubmit runs fn after each Submitted on the element.
func OnSubmit[F Handler](fn F) Arg { return on(EventSubmit, fn) }

// OnToggle runs fn after each Toggled on the element.
func OnToggle[F Handler](fn F) Arg { return on(EventToggle, fn) }

// OnClose runs fn after each Closed on the element.
func OnClose[F Handler](fn F) Arg { return on(EventClose, fn) }

// OnSort runs fn after each Sorted on the element.
func OnSort[F Handler](fn F) Arg { return on(EventSort, fn) }

// OnKey runs fn after each KeyPressed on the element or one inside it, the
// focused element's handlers first, then each ancestor's (decision 0060).
func OnKey[F Handler](fn F) Arg { return on(EventKey, fn) }

// OnBack runs fn after each BackPressed on the element or one inside it,
// innermost first, so one OnBack on a screen's root hears its Escape and
// pad East; on the body or html it also hears a Back with nothing focused,
// which emits no World event (decision 0060). e.StopPropagation keeps it
// from going higher.
func OnBack[F Handler](fn F) Arg { return on(EventBack, fn) }

// OnContextMenu runs fn after each right click on the element or one
// inside it, innermost first, as the DOM's contextmenu, which opens no menu
// by itself (decision 0062).
func OnContextMenu[F Handler](fn F) Arg { return on(EventContext, fn) }

// OnDrop makes the element a drop target: fn runs when a Draggable element
// is let go over it, its Event's Dragged that element (decision 0062).
func OnDrop[F Handler](fn F) Arg { return on(EventDrop, fn) }

// OnDragEnd runs fn when the element's drag ends, dropped or cancelled
// (Escape, or a release over no target), its Event's Dropped telling which.
func OnDragEnd[F Handler](fn F) Arg { return on(EventDragEnd, fn) }

// OnPointerEnter runs fn when the pointer comes over the element or into
// it, once, as mouseenter: not again for its children.
func OnPointerEnter[F Handler](fn F) Arg { return on(EventPointerEnter, fn) }

// OnPointerLeave runs fn when the pointer leaves the element and all of
// its children.
func OnPointerLeave[F Handler](fn F) Arg { return on(EventPointerLeave, fn) }

// OnMove runs fn after each WindowMoved of the window.
func OnMove[F Handler](fn F) Arg { return on(EventMove, fn) }

// OnResize runs fn after each WindowResized of the window.
func OnResize[F Handler](fn F) Arg { return on(EventResize, fn) }

// OnFold runs fn after each WindowFolded of the window.
func OnFold[F Handler](fn F) Arg { return on(EventFold, fn) }

// On adds fn to n's callbacks for t, after those it has, and returns what
// removes it again.
func On[F Handler](n *Node, t EventType, fn F) func() {
	hs := handlersOf(n, true)
	hs.next++
	id := hs.next
	hs.list = append(hs.list, handler{t, id, wrap(fn)})
	return func() {
		hs.list = slices.DeleteFunc(hs.list, func(h handler) bool { return h.id == id })
	}
}

func on[F Handler](t EventType, fn F) Arg {
	return option(func(n *Node) { On(n, t, fn) })
}

type handler struct {
	t  EventType
	id int
	fn func(*world.World, Event)
}

type handlers struct {
	list []handler
	next int
}

func handlersOf(n *Node, create bool) *handlers {
	hs, _ := dom.Data(n).Handlers.(*handlers)
	if hs == nil && create {
		hs = &handlers{}
		dom.Data(n).Handlers = hs
	}
	return hs
}

func wrap[F Handler](fn F) func(*world.World, Event) {
	switch f := any(fn).(type) {
	case func():
		return func(*world.World, Event) { f() }
	case func(Event):
		return func(_ *world.World, e Event) { f(e) }
	case func(*world.World, Event):
		return f
	}
	return nil
}

// dispatch runs the callbacks of the element an emitted event is about,
// in the order they were added, when it is still in the Document.
func (d *Document) dispatch(w *world.World, raw any) {
	e, ok := eventOf(raw)
	if _, body := raw.(bodyBack); body {
		e, ok = Event{Type: EventBack, Node: d.body}, true
	}
	if !ok || e.Node == nil || !d.holds(e.Node) {
		return
	}
	stopped := false
	e.stopped = &stopped
	for n := e.Node; n != nil && !stopped; n = n.Parent() {
		if n != e.Node && !d.holds(n) {
			continue
		}
		if hs := handlersOf(n, false); hs != nil {
			for _, h := range slices.Clone(hs.list) {
				if h.t == e.Type {
					h.fn(w, e)
				}
			}
		}
		if !bubbles(e.Type) {
			return
		}
	}
}

// bodyBack is a Back pressed with nothing focused: no World event, as
// ever, but the body's handlers and the html's run (decision 0060).
type bodyBack struct{}

// handles reports whether n has a handler for t.
func handles(n *Node, t EventType) bool {
	if hs := handlersOf(n, false); hs != nil {
		for _, h := range hs.list {
			if h.t == t {
				return true
			}
		}
	}
	return false
}

// hasBack reports whether n or an ancestor has a Back handler.
func hasBack(n *Node) bool {
	for ; n != nil; n = n.Parent() {
		if hs := handlersOf(n, false); hs != nil {
			for _, h := range hs.list {
				if h.t == EventBack {
					return true
				}
			}
		}
	}
	return false
}

// holds reports whether n is in d's tree.
func (d *Document) holds(n *Node) bool {
	for p := n; p != nil; p = p.Parent() {
		if p == d.html {
			return true
		}
	}
	return false
}

func eventOf(raw any) (Event, bool) {
	switch e := raw.(type) {
	case Clicked:
		return Event{Type: EventClick, Node: e.Node, Clicks: max(e.Clicks, 1)}, true
	case Edited:
		return Event{Type: EventInput, Node: e.Node}, true
	case Changed:
		return Event{Type: EventChange, Node: e.Node}, true
	case Submitted:
		return Event{Type: EventSubmit, Node: e.Node}, true
	case Toggled:
		return Event{Type: EventToggle, Node: e.Node}, true
	case Closed:
		return Event{Type: EventClose, Node: e.Node, Value: e.Value}, true
	case Sorted:
		return Event{Type: EventSort, Node: e.Node, Column: e.Column, Descending: e.Descending}, true
	case KeyPressed:
		return Event{Type: EventKey, Node: e.Node, Key: e.Key, Text: e.Text, Command: e.Command, Pad: e.Pad}, true
	case BackPressed:
		return Event{Type: EventBack, Node: e.Node}, true
	case PointerEntered:
		return Event{Type: EventPointerEnter, Node: e.Node}, true
	case PointerLeft:
		return Event{Type: EventPointerLeave, Node: e.Node}, true
	case WindowMoved:
		return Event{Type: EventMove, Node: e.Node, At: e.At}, true
	case WindowResized:
		return Event{Type: EventResize, Node: e.Node, Size: e.Size}, true
	case WindowFolded:
		return Event{Type: EventFold, Node: e.Node, Folded: e.Folded}, true
	case ContextClicked:
		return Event{Type: EventContext, Node: e.Node}, true
	case Dropped:
		return Event{Type: EventDrop, Node: e.Target, Dragged: e.Dragged}, true
	case DragEnded:
		return Event{Type: EventDragEnd, Node: e.Source, Dropped: e.Dropped}, true
	}
	return Event{}, false
}
