package input

import "fmt"

// Key is a key's position on the keyboard, named after the US layout:
// KeyW is the key AZERTY prints Z on, which is what movement wants. What a
// key writes in the player's layout comes from State.Text. Key values
// match GLFW, so backends convert without tables.
type Key int

const (
	KeyUnknown Key = -1

	KeySpace        Key = 32
	KeyApostrophe   Key = 39
	KeyComma        Key = 44
	KeyMinus        Key = 45
	KeyPeriod       Key = 46
	KeySlash        Key = 47
	Key0            Key = 48
	Key1            Key = 49
	Key2            Key = 50
	Key3            Key = 51
	Key4            Key = 52
	Key5            Key = 53
	Key6            Key = 54
	Key7            Key = 55
	Key8            Key = 56
	Key9            Key = 57
	KeySemicolon    Key = 59
	KeyEqual        Key = 61
	KeyA            Key = 65
	KeyB            Key = 66
	KeyC            Key = 67
	KeyD            Key = 68
	KeyE            Key = 69
	KeyF            Key = 70
	KeyG            Key = 71
	KeyH            Key = 72
	KeyI            Key = 73
	KeyJ            Key = 74
	KeyK            Key = 75
	KeyL            Key = 76
	KeyM            Key = 77
	KeyN            Key = 78
	KeyO            Key = 79
	KeyP            Key = 80
	KeyQ            Key = 81
	KeyR            Key = 82
	KeyS            Key = 83
	KeyT            Key = 84
	KeyU            Key = 85
	KeyV            Key = 86
	KeyW            Key = 87
	KeyX            Key = 88
	KeyY            Key = 89
	KeyZ            Key = 90
	KeyLeftBracket  Key = 91
	KeyBackslash    Key = 92
	KeyRightBracket Key = 93
	KeyGraveAccent  Key = 96
	keyWorld1       Key = 161
	keyWorld2       Key = 162

	KeyEscape       Key = 256
	KeyEnter        Key = 257
	KeyTab          Key = 258
	KeyBackspace    Key = 259
	KeyInsert       Key = 260
	KeyDelete       Key = 261
	KeyRight        Key = 262
	KeyLeft         Key = 263
	KeyDown         Key = 264
	KeyUp           Key = 265
	KeyPageUp       Key = 266
	KeyPageDown     Key = 267
	KeyHome         Key = 268
	KeyEnd          Key = 269
	KeyCapsLock     Key = 280
	KeyScrollLock   Key = 281
	KeyNumLock      Key = 282
	KeyPrintScreen  Key = 283
	KeyPause        Key = 284
	KeyF1           Key = 290
	KeyF2           Key = 291
	KeyF3           Key = 292
	KeyF4           Key = 293
	KeyF5           Key = 294
	KeyF6           Key = 295
	KeyF7           Key = 296
	KeyF8           Key = 297
	KeyF9           Key = 298
	KeyF10          Key = 299
	KeyF11          Key = 300
	KeyF12          Key = 301
	KeyF13          Key = 302
	KeyF14          Key = 303
	KeyF15          Key = 304
	KeyF16          Key = 305
	KeyF17          Key = 306
	KeyF18          Key = 307
	KeyF19          Key = 308
	KeyF20          Key = 309
	KeyF21          Key = 310
	KeyF22          Key = 311
	KeyF23          Key = 312
	KeyF24          Key = 313
	KeyF25          Key = 314
	KeyPad0         Key = 320
	KeyPad1         Key = 321
	KeyPad2         Key = 322
	KeyPad3         Key = 323
	KeyPad4         Key = 324
	KeyPad5         Key = 325
	KeyPad6         Key = 326
	KeyPad7         Key = 327
	KeyPad8         Key = 328
	KeyPad9         Key = 329
	KeyPadDecimal   Key = 330
	KeyPadDivide    Key = 331
	KeyPadMultiply  Key = 332
	KeyPadSubtract  Key = 333
	KeyPadAdd       Key = 334
	KeyPadEnter     Key = 335
	KeyPadEqual     Key = 336
	KeyLeftShift    Key = 340
	KeyLeftControl  Key = 341
	KeyLeftAlt      Key = 342
	KeyLeftSuper    Key = 343
	KeyRightShift   Key = 344
	KeyRightControl Key = 345
	KeyRightAlt     Key = 346
	KeyRightSuper   Key = 347
	KeyMenu         Key = 348
	keyLast         Key = KeyMenu
)

// Either side's modifier: KeyShift is down while the left or the right
// shift key is.
const (
	KeyShift Key = 400 + iota
	KeyControl
	KeyAlt
	KeySuper
)

// sides are a modifier's left and right keys; false for any other key,
// since only the four modifiers have two sides.
func sides(k Key) ([2]Key, bool) {
	if k < KeyShift || k > KeySuper {
		return [2]Key{}, false
	}
	return [...][2]Key{
		{KeyLeftShift, KeyRightShift},
		{KeyLeftControl, KeyRightControl},
		{KeyLeftAlt, KeyRightAlt},
		{KeyLeftSuper, KeyRightSuper},
	}[k-KeyShift], true
}

// raw reads k, now or the frame before, whatever holds the keyboard.
func (k Key) raw(s *State, before bool) bool {
	if pair, ok := sides(k); ok {
		return pair[0].raw(s, before) || pair[1].raw(s, before)
	}
	if !valid(k) {
		return false
	}
	if before {
		return s.prevKeys[k]
	}
	return s.keys[k]
}

func (k Key) level(r Reader, before bool) float32 { return r.keyLevel(k, before) }

type MouseButton int

const (
	MouseLeft        MouseButton = 0
	MouseRight       MouseButton = 1
	MouseMiddle      MouseButton = 2
	MouseBack        MouseButton = 3
	MouseForward     MouseButton = 4
	mouseButtonCount             = 8
)

// String names a mouse button: "MouseLeft", "MouseBack".
func (b MouseButton) String() string {
	mouseNames := [...]string{
		MouseLeft: "MouseLeft", MouseRight: "MouseRight", MouseMiddle: "MouseMiddle",
		MouseBack: "MouseBack", MouseForward: "MouseForward",
	}
	if b >= 0 && int(b) < len(mouseNames) && mouseNames[b] != "" {
		return mouseNames[b]
	}
	return fmt.Sprintf("Mouse%d", int(b)+1)
}

func (b MouseButton) level(r Reader, before bool) float32 { return r.mouseLevel(b, before) }
