// Package control is the behaviour of ui's form controls, apart from any
// tree: what keys, pad buttons and the pointer do to a button, a checkbox,
// a range, a text field, a select, details or a dialog, and which events
// that makes. ui wires each element to one and emits the events it
// returns.
package control

import "github.com/merlindorin/ember/input"

// Intent is what a key or a pad button asks of the focused control.
type Intent int

const (
	None Intent = iota
	// Activate is Enter, Space or the pad's south button.
	Activate
	// Back is Escape or the pad's east button.
	Back
	Up
	Down
	Left
	Right
	Home
	End
	PageUp
	PageDown
)

// KeyIntent is what key asks, as the old ember/ui reads it.
func KeyIntent(key input.Key) Intent {
	switch key {
	case input.KeyEnter, input.KeyPadEnter, input.KeySpace:
		return Activate
	case input.KeyEscape:
		return Back
	case input.KeyUp:
		return Up
	case input.KeyDown:
		return Down
	case input.KeyLeft:
		return Left
	case input.KeyRight:
		return Right
	case input.KeyHome:
		return Home
	case input.KeyEnd:
		return End
	case input.KeyPageUp:
		return PageUp
	case input.KeyPageDown:
		return PageDown
	case input.KeyUnknown, input.KeyApostrophe, input.KeyComma, input.KeyMinus, input.KeyPeriod, input.KeySlash,
		input.Key0, input.Key1, input.Key2, input.Key3, input.Key4, input.Key5, input.Key6, input.Key7, input.Key8,
		input.Key9, input.KeySemicolon, input.KeyEqual, input.KeyA, input.KeyB, input.KeyC, input.KeyD, input.KeyE,
		input.KeyF, input.KeyG, input.KeyH, input.KeyI, input.KeyJ, input.KeyK, input.KeyL, input.KeyM, input.KeyN,
		input.KeyO, input.KeyP, input.KeyQ, input.KeyR, input.KeyS, input.KeyT, input.KeyU, input.KeyV, input.KeyW,
		input.KeyX, input.KeyY, input.KeyZ, input.KeyLeftBracket, input.KeyBackslash, input.KeyRightBracket,
		input.KeyGraveAccent, input.KeyTab, input.KeyBackspace, input.KeyInsert, input.KeyDelete, input.KeyCapsLock,
		input.KeyScrollLock, input.KeyNumLock, input.KeyPrintScreen, input.KeyPause, input.KeyF1, input.KeyF2, input.KeyF3,
		input.KeyF4, input.KeyF5, input.KeyF6, input.KeyF7, input.KeyF8, input.KeyF9, input.KeyF10, input.KeyF11,
		input.KeyF12, input.KeyF13, input.KeyF14, input.KeyF15, input.KeyF16, input.KeyF17, input.KeyF18, input.KeyF19,
		input.KeyF20, input.KeyF21, input.KeyF22, input.KeyF23, input.KeyF24, input.KeyF25, input.KeyPad0, input.KeyPad1,
		input.KeyPad2, input.KeyPad3, input.KeyPad4, input.KeyPad5, input.KeyPad6, input.KeyPad7, input.KeyPad8,
		input.KeyPad9, input.KeyPadDecimal, input.KeyPadDivide, input.KeyPadMultiply, input.KeyPadSubtract, input.KeyPadAdd,
		input.KeyPadEqual, input.KeyLeftShift, input.KeyLeftControl, input.KeyLeftAlt, input.KeyLeftSuper,
		input.KeyRightShift, input.KeyRightControl, input.KeyRightAlt, input.KeyRightSuper, input.KeyMenu, input.KeyShift,
		input.KeyControl, input.KeyAlt, input.KeySuper: // keys that ask nothing
	}
	return None
}

// PadIntent is what a pad button asks: the cross and the left stick steer.
func PadIntent(b input.GamepadButton) Intent {
	switch b {
	case input.GamepadSouth:
		return Activate
	case input.GamepadEast:
		return Back
	case input.GamepadUp, input.GamepadLeftStickUp:
		return Up
	case input.GamepadDown, input.GamepadLeftStickDown:
		return Down
	case input.GamepadLeft, input.GamepadLeftStickLeft:
		return Left
	case input.GamepadRight, input.GamepadLeftStickRight:
		return Right
	case input.GamepadWest, input.GamepadNorth, input.GamepadLeftShoulder, input.GamepadRightShoulder, input.GamepadBack,
		input.GamepadStart, input.GamepadGuide, input.GamepadLeftStickPress, input.GamepadRightStickPress,
		input.GamepadTouchpad, input.GamepadLeftTrigger, input.GamepadRightTrigger, input.GamepadRightStickLeft,
		input.GamepadRightStickRight, input.GamepadRightStickUp, input.GamepadRightStickDown: // buttons that ask nothing
	}
	return None
}

// Events is the set of events a control makes from one intent or pointer
// action, emitted by ui as its World events.
type Events uint8

const (
	Clicked Events = 1 << iota
	Input
	Changed
	Submitted
	Toggled
	Closed
)

func (e Events) Has(x Events) bool { return e&x != 0 }

func when(cond bool, e Events) Events {
	if cond {
		return e
	}
	return 0
}

// Button is a button element: Activate or a click is Clicked.
type Button struct{ Disabled bool }

func (b *Button) Handles(i Intent) bool { return i == Activate && !b.Disabled }

func (b *Button) Do(i Intent) Events { return when(b.Handles(i), Clicked) }

func (b *Button) Click() Events { return when(!b.Disabled, Clicked) }

// Checkbox is an input of type checkbox: Activate or a click toggles it.
type Checkbox struct {
	Checked, Indeterminate, Disabled bool
}

func (c *Checkbox) Handles(i Intent) bool { return i == Activate && !c.Disabled }

func (c *Checkbox) Do(i Intent) Events {
	if !c.Handles(i) {
		return 0
	}
	return c.Click()
}

func (c *Checkbox) Click() Events {
	if c.Disabled {
		return 0
	}
	c.Checked, c.Indeterminate = !c.Checked, false
	return Input | Changed
}

// Details is a details element, opened and closed from its summary.
type Details struct{ Open bool }

func (d *Details) Handles(i Intent) bool { return i == Activate }

func (d *Details) Do(i Intent) Events {
	if !d.Handles(i) {
		return 0
	}
	return d.SetOpen(!d.Open)
}

func (d *Details) Click() Events { return d.SetOpen(!d.Open) }

func (d *Details) SetOpen(open bool) Events {
	changed := d.Open != open
	d.Open = open
	return when(changed, Toggled)
}
