package textedit

import (
	"runtime"

	"github.com/lantervale/ember/input"
)

// Platform is whose key conventions an editor follows.
type Platform int

const (
	// Others is Windows and Linux: Control for shortcuts, Control+arrows
	// by word.
	Others Platform = iota
	// Mac is macOS: Command for shortcuts and line ends, Option by word.
	Mac
)

// Current is the platform the program runs on.
func Current() Platform {
	if runtime.GOOS == "darwin" {
		return Mac
	}
	return Others
}

// Action is what a key asks of an editor.
type Action int

const (
	None Action = iota
	Left
	Right
	WordLeft
	WordRight
	LineStart
	LineEnd
	Start
	End
	Up
	Down
	PageUp
	PageDown
	SelectAll
	Backspace
	Delete
	DeleteWord
	DeleteToLineStart
	Copy
	Cut
	Paste
	Undo
	Redo
	Tab
	Newline
	Submit
)

// moves reports whether a is a move, which Shift extends the selection
// with.
func moves(a Action) bool {
	switch a {
	case Left, Right, WordLeft, WordRight, LineStart, LineEnd, Start, End, Up, Down, PageUp, PageDown:
		return true
	case None, SelectAll, Backspace, Delete, DeleteWord, DeleteToLineStart, Copy, Cut, Paste, Undo, Redo, Tab,
		Newline, Submit: // edits and the rest, which Shift doesn't extend
	}
	return false
}

// Mods are the modifiers held with a key.
type Mods struct{ Shift, Control, Alt, Super bool }

// Resolve is what key with mods asks on platform p, and whether Shift
// extends the selection with it.
func Resolve(p Platform, key input.Key, m Mods) (Action, bool) {
	cmd, word := m.Control, m.Control
	if p == Mac {
		cmd, word = m.Super, m.Alt
	}
	a := None
	switch key {
	case input.KeyLeft:
		a = pick(p == Mac && m.Super, LineStart, pick(word, WordLeft, Left))
	case input.KeyRight:
		a = pick(p == Mac && m.Super, LineEnd, pick(word, WordRight, Right))
	case input.KeyUp:
		a = pick(p == Mac && m.Super, Start, Up)
	case input.KeyDown:
		a = pick(p == Mac && m.Super, End, Down)
	case input.KeyHome:
		a = pick(m.Control, Start, LineStart)
	case input.KeyEnd:
		a = pick(m.Control, End, LineEnd)
	case input.KeyPageUp:
		a = PageUp
	case input.KeyPageDown:
		a = PageDown
	case input.KeyBackspace:
		a = pick(p == Mac && m.Super, DeleteToLineStart, pick(word, DeleteWord, Backspace))
	case input.KeyDelete:
		a = Delete
	case input.KeyTab:
		a = pick(!cmd && !m.Alt, Tab, None)
	case input.KeyEnter, input.KeyPadEnter:
		a = pick(m.Super || m.Control, Submit, Newline)
	case input.KeyA:
		a = pick(cmd, SelectAll, None)
	case input.KeyC:
		a = pick(cmd, Copy, None)
	case input.KeyX:
		a = pick(cmd, Cut, None)
	case input.KeyV:
		a = pick(cmd, Paste, None)
	case input.KeyZ:
		a = pick(cmd, pick(m.Shift, Redo, Undo), None)
	case input.KeyY:
		a = pick(cmd && p == Others, Redo, None)
	case input.KeyUnknown, input.KeySpace, input.KeyApostrophe, input.KeyComma, input.KeyMinus, input.KeyPeriod,
		input.KeySlash, input.Key0, input.Key1, input.Key2, input.Key3, input.Key4, input.Key5, input.Key6, input.Key7,
		input.Key8, input.Key9, input.KeySemicolon, input.KeyEqual, input.KeyB, input.KeyD, input.KeyE, input.KeyF,
		input.KeyG, input.KeyH, input.KeyI, input.KeyJ, input.KeyK, input.KeyL, input.KeyM, input.KeyN, input.KeyO,
		input.KeyP, input.KeyQ, input.KeyR, input.KeyS, input.KeyT, input.KeyU, input.KeyW, input.KeyLeftBracket,
		input.KeyBackslash, input.KeyRightBracket, input.KeyGraveAccent, input.KeyEscape, input.KeyInsert, input.KeyCapsLock,
		input.KeyScrollLock, input.KeyNumLock, input.KeyPrintScreen, input.KeyPause, input.KeyF1, input.KeyF2, input.KeyF3,
		input.KeyF4, input.KeyF5, input.KeyF6, input.KeyF7, input.KeyF8, input.KeyF9, input.KeyF10, input.KeyF11,
		input.KeyF12, input.KeyF13, input.KeyF14, input.KeyF15, input.KeyF16, input.KeyF17, input.KeyF18, input.KeyF19,
		input.KeyF20, input.KeyF21, input.KeyF22, input.KeyF23, input.KeyF24, input.KeyF25, input.KeyPad0, input.KeyPad1,
		input.KeyPad2, input.KeyPad3, input.KeyPad4, input.KeyPad5, input.KeyPad6, input.KeyPad7, input.KeyPad8,
		input.KeyPad9, input.KeyPadDecimal, input.KeyPadDivide, input.KeyPadMultiply, input.KeyPadSubtract, input.KeyPadAdd,
		input.KeyPadEqual, input.KeyLeftShift, input.KeyLeftControl, input.KeyLeftAlt, input.KeyLeftSuper,
		input.KeyRightShift, input.KeyRightControl, input.KeyRightAlt, input.KeyRightSuper, input.KeyMenu, input.KeyShift,
		input.KeyControl, input.KeyAlt, input.KeySuper: // keys an editor ignores
	}
	return a, m.Shift && moves(a)
}

func pick(cond bool, yes, no Action) Action {
	if cond {
		return yes
	}
	return no
}

// Keys are the keys an editor listens to, for a caller to read.
var Keys = []input.Key{ //nolint:gochecknoglobals // a fixed table callers read; a function would copy it each frame
	input.KeyLeft, input.KeyRight, input.KeyUp, input.KeyDown, input.KeyHome, input.KeyEnd,
	input.KeyPageUp, input.KeyPageDown, input.KeyBackspace, input.KeyDelete, input.KeyTab,
	input.KeyEnter, input.KeyPadEnter, input.KeyA, input.KeyC, input.KeyX, input.KeyV, input.KeyZ, input.KeyY,
}
