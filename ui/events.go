package ui

import (
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
)

// Clicked is a button, a summary or any element with a tabindex pressed,
// by the pointer, a key, a pad or automation. Clicks is 2 or 3 for the
// second and third pointer click on it within 400ms, else 1.
type Clicked struct {
	Node   *Node
	Clicks int
}

// Edited is a control's value changed while being edited: every keystroke
// in a text field, every step of a range, a checkbox toggled.
type Edited struct{ Node *Node }

// Changed is a control's value committed: a checkbox toggled, an option
// chosen, a text field left or submitted after an edit.
type Changed struct{ Node *Node }

// Submitted is Enter in a one-line text field, or a textarea's submit
// shortcut.
type Submitted struct{ Node *Node }

// Toggled is a details element opened or closed.
type Toggled struct{ Node *Node }

// KeyPressed is a key pressed that no control, menu, grid, listbox or focus move took:
// on Node, the focused element, else the open modal dialog. Text is what it typed, ""
// for none, and Command reports the platform's shortcut modifier held (Command on macOS,
// Control elsewhere). With nothing focused and no modal dialog the keys are the game's,
// and none is reported. A pad button focus navigation leaves (West, North, the
// shoulders and triggers, Back, Start) comes the same way, as Pad with Key
// KeyUnknown.
type KeyPressed struct {
	Node    *Node
	Key     input.Key
	Text    string
	Command bool
	Pad     input.GamepadButton
}

// BackPressed is Escape or the pad's East on the focused element when no
// menu, modal dialog or control took it: a screen's way back.
type BackPressed struct{ Node *Node }

// ContextClicked is the right button pressed and released on Node, the
// innermost element under the pointer (decision 0062).
type ContextClicked struct{ Node *Node }

// Dropped is Dragged, a draggable element, let go over Target, the drop
// target under the pointer (decision 0062).
type Dropped struct{ Target, Dragged *Node }

// DragEnded is a drag of Source over, by a drop or a cancel: Dropped
// reports whether a target took it.
type DragEnded struct {
	Source  *Node
	Dropped bool
}

// PointerEntered is the pointer coming over Node or into it, once, as
// mouseenter: not again for its children.
type PointerEntered struct{ Node *Node }

// PointerLeft is the pointer leaving Node and all of its children.
type PointerLeft struct{ Node *Node }

// WindowMoved is a window dropped somewhere else by its bar: At is its
// top-left now, in the Document's points.
type WindowMoved struct {
	Node *Node
	At   geom.Vec2
}

// WindowResized is a window resized by its edges, once let go: Size is its
// border box now.
type WindowResized struct {
	Node *Node
	Size geom.Vec2
}

// WindowFolded is a window folded to its bar, or unfolded, by its minimise
// light, or unfolded by its zoom light.
type WindowFolded struct {
	Node   *Node
	Folded bool
}
