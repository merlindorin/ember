package input

import "fmt"

// keyName is k's name when it isn't a letter, a digit, an F key or a pad
// digit, which String works out; false for a key with no name.
func keyName(k Key) (string, bool) {
	for _, n := range [...]struct {
		key  Key
		name string
	}{
		{KeySpace, "Space"},
		{KeyApostrophe, "'"},
		{KeyComma, ","},
		{KeyMinus, "-"},
		{KeyPeriod, "."},
		{KeySlash, "/"},
		{KeySemicolon, ";"},
		{KeyEqual, "="},
		{KeyLeftBracket, "["},
		{KeyBackslash, "\\"},
		{KeyRightBracket, "]"},
		{KeyGraveAccent, "` (² on AZERTY)"},
		{KeyEscape, "Escape"},
		{KeyEnter, "Enter"},
		{KeyTab, "Tab"},
		{KeyBackspace, "Backspace"},
		{KeyInsert, "Insert"},
		{KeyDelete, "Delete"},
		{KeyRight, "Right"},
		{KeyLeft, "Left"},
		{KeyDown, "Down"},
		{KeyUp, "Up"},
		{KeyPageUp, "PageUp"},
		{KeyPageDown, "PageDown"},
		{KeyHome, "Home"},
		{KeyEnd, "End"},
		{KeyCapsLock, "CapsLock"},
		{KeyLeftShift, "LeftShift"},
		{KeyLeftControl, "LeftControl"},
		{KeyLeftAlt, "LeftAlt"},
		{KeyLeftSuper, "LeftSuper"},
		{KeyRightShift, "RightShift"},
		{KeyRightControl, "RightControl"},
		{KeyRightAlt, "RightAlt"},
		{KeyRightSuper, "RightSuper"},
		{KeyMenu, "Menu"},
		{KeyPadEnter, "PadEnter"},
		{KeyPadDecimal, "PadDecimal"},
		{KeyPadDivide, "PadDivide"},
		{KeyPadMultiply, "PadMultiply"},
		{KeyPadSubtract, "PadSubtract"},
		{KeyPadAdd, "PadAdd"},
		{KeyPadEqual, "PadEqual"},
		{KeyShift, "Shift"},
		{KeyControl, "Control"},
		{KeyAlt, "Alt"},
		{KeySuper, "Super"},
	} {
		if n.key == k {
			return n.name, true
		}
	}
	return "", false
}

// String names a key as printed on a US keyboard: "A", "F1", "Escape".
func (k Key) String() string {
	switch {
	case k >= KeyA && k <= KeyZ, k >= Key0 && k <= Key9:
		return string(rune(k)) //nolint:gosec // k is a letter or a digit here, its ASCII code
	case k >= KeyF1 && k <= KeyF25:
		return fmt.Sprintf("F%d", k-KeyF1+1)
	case k >= KeyPad0 && k <= KeyPad9:
		return fmt.Sprintf("Pad%d", k-KeyPad0)
	}
	if n, ok := keyName(k); ok {
		return n
	}
	return fmt.Sprintf("Key(%d)", int(k))
}
