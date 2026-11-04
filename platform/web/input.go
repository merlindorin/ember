//go:build js

package web

import (
	"strconv"
	"strings"
	"syscall/js"
	"unicode/utf8"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/internal/inputfeed"
)

// keys maps KeyboardEvent.code, a key's place on the keyboard whatever
// its layout prints, to ember's Keys, which are places too.
//
//nolint:gochecknoglobals // a lookup table, built once
var keys = func() map[string]input.Key {
	m := map[string]input.Key{
		"Space":          input.KeySpace,
		"Quote":          input.KeyApostrophe,
		"Comma":          input.KeyComma,
		"Minus":          input.KeyMinus,
		"Period":         input.KeyPeriod,
		"Slash":          input.KeySlash,
		"Semicolon":      input.KeySemicolon,
		"Equal":          input.KeyEqual,
		"BracketLeft":    input.KeyLeftBracket,
		"Backslash":      input.KeyBackslash,
		"BracketRight":   input.KeyRightBracket,
		"Backquote":      input.KeyGraveAccent,
		"Escape":         input.KeyEscape,
		"Enter":          input.KeyEnter,
		"Tab":            input.KeyTab,
		"Backspace":      input.KeyBackspace,
		"Insert":         input.KeyInsert,
		"Delete":         input.KeyDelete,
		"ArrowRight":     input.KeyRight,
		"ArrowLeft":      input.KeyLeft,
		"ArrowDown":      input.KeyDown,
		"ArrowUp":        input.KeyUp,
		"PageUp":         input.KeyPageUp,
		"PageDown":       input.KeyPageDown,
		"Home":           input.KeyHome,
		"End":            input.KeyEnd,
		"CapsLock":       input.KeyCapsLock,
		"ScrollLock":     input.KeyScrollLock,
		"NumLock":        input.KeyNumLock,
		"PrintScreen":    input.KeyPrintScreen,
		"Pause":          input.KeyPause,
		"NumpadDecimal":  input.KeyPadDecimal,
		"NumpadDivide":   input.KeyPadDivide,
		"NumpadMultiply": input.KeyPadMultiply,
		"NumpadSubtract": input.KeyPadSubtract,
		"NumpadAdd":      input.KeyPadAdd,
		"NumpadEnter":    input.KeyPadEnter,
		"NumpadEqual":    input.KeyPadEqual,
		"ShiftLeft":      input.KeyLeftShift,
		"ControlLeft":    input.KeyLeftControl,
		"AltLeft":        input.KeyLeftAlt,
		"MetaLeft":       input.KeyLeftSuper,
		"ShiftRight":     input.KeyRightShift,
		"ControlRight":   input.KeyRightControl,
		"AltRight":       input.KeyRightAlt,
		"MetaRight":      input.KeyRightSuper,
		"ContextMenu":    input.KeyMenu,
	}
	for c := 'A'; c <= 'Z'; c++ {
		m["Key"+string(c)] = input.KeyA + input.Key(c-'A')
	}
	for d := 0; d <= 9; d++ {
		m["Digit"+string(rune('0'+d))] = input.Key0 + input.Key(d)
		m["Numpad"+string(rune('0'+d))] = input.KeyPad0 + input.Key(d)
	}
	for f := 1; f <= 24; f++ {
		m["F"+strconv.Itoa(f)] = input.KeyF1 + input.Key(f-1)
	}
	return m
}()

// mouseButtons maps PointerEvent.button to ember's.
//
//nolint:gochecknoglobals // a lookup table
var mouseButtons = map[int]input.MouseButton{
	0: input.MouseLeft,
	1: input.MouseMiddle,
	2: input.MouseRight,
	3: input.MouseBack,
	4: input.MouseForward,
}

// listen feeds the page's keyboard, the canvas's pointer and wheel into
// in, as GLFW's callbacks do on the desktop: events arrive between
// frames, and the World reads them at its next one. The handlers are
// kept for the page's life.
func listen(canvas js.Value, in *input.State) {
	on := func(target js.Value, event string, fn func(e js.Value)) {
		target.Call("addEventListener", event, js.FuncOf(func(_ js.Value, args []js.Value) any {
			fn(args[0])
			return nil
		}), map[string]any{"passive": false})
	}
	window := js.Global()
	canvas.Set("tabIndex", 0)
	canvas.Call("focus")
	on(window, "keydown", func(e js.Value) {
		k, ok := keys[e.Get("code").String()]
		if !ok {
			return
		}
		if e.Get("repeat").Bool() {
			inputfeed.Of(in).AddRepeat(int(k))
		} else {
			in.SetKey(k, true)
		}
		ctrl := e.Get("ctrlKey").Bool() || e.Get("metaKey").Bool()
		if text := e.Get("key").String(); !ctrl && utf8.RuneCountInString(text) == 1 {
			r, _ := utf8.DecodeRuneInString(text)
			in.AddText(r)
		}
		// The page mustn't scroll or leave the game on Space, arrows or Tab.
		if !ctrl || strings.HasPrefix(e.Get("code").String(), "Arrow") {
			e.Call("preventDefault")
		}
	})
	on(window, "keyup", func(e js.Value) {
		if k, ok := keys[e.Get("code").String()]; ok {
			in.SetKey(k, false)
		}
	})
	point := func(e js.Value) geom.Vec2 {
		return geom.V2(float32(e.Get("offsetX").Float()), float32(e.Get("offsetY").Float()))
	}
	on(canvas, "pointermove", func(e js.Value) { in.SetMouse(point(e)) })
	on(canvas, "pointerdown", func(e js.Value) {
		canvas.Call("focus")
		canvas.Call("setPointerCapture", e.Get("pointerId"))
		in.SetMouse(point(e))
		if b, ok := mouseButtons[e.Get("button").Int()]; ok {
			in.SetMouseButton(b, true)
		}
		e.Call("preventDefault")
	})
	on(canvas, "pointerup", func(e js.Value) {
		in.SetMouse(point(e))
		if b, ok := mouseButtons[e.Get("button").Int()]; ok {
			in.SetMouseButton(b, false)
		}
	})
	on(canvas, "contextmenu", func(e js.Value) { e.Call("preventDefault") })
	on(canvas, "wheel", func(e js.Value) {
		// GLFW reports lines, up and left positive; a wheel's notch is
		// about 100 pixels in Chrome, a line about 16.
		unit := map[int]float64{0: 100, 1: 3, 2: 0.3}[e.Get("deltaMode").Int()]
		in.AddWheel(geom.V2(float32(-e.Get("deltaX").Float()/unit), float32(-e.Get("deltaY").Float()/unit)))
		e.Call("preventDefault")
	})
}

// standard is the Gamepad API's standard mapping, button by button.
//
//nolint:gochecknoglobals // a lookup table
var standard = []input.GamepadButton{
	input.GamepadSouth, input.GamepadEast, input.GamepadWest, input.GamepadNorth,
	input.GamepadLeftShoulder, input.GamepadRightShoulder, input.GamepadLeftTrigger, input.GamepadRightTrigger,
	input.GamepadBack, input.GamepadStart, input.GamepadLeftStickPress, input.GamepadRightStickPress,
	input.GamepadUp, input.GamepadDown, input.GamepadLeft, input.GamepadRight, input.GamepadGuide,
}

// pads polls the Gamepad API once a frame, as browsers only give pads'
// state by polling: pads joining and leaving, buttons, triggers, sticks.
type pads struct{ known map[int]bool }

func (p *pads) poll(in *input.State) {
	if p.known == nil {
		p.known = map[int]bool{}
	}
	list := js.Global().Get("navigator").Call("getGamepads")
	seen := map[int]bool{}
	for i := range list.Length() {
		g := list.Index(i)
		if g.IsNull() || g.IsUndefined() || !g.Get("connected").Bool() {
			continue
		}
		id := g.Get("index").Int()
		seen[id] = true
		if !p.known[id] {
			in.ConnectGamepad(id, g.Get("id").String())
			p.known[id] = true
		}
		buttons := g.Get("buttons")
		for b, name := range standard {
			if b >= buttons.Length() {
				break
			}
			v := buttons.Index(b)
			switch name {
			case input.GamepadLeftTrigger, input.GamepadRightTrigger:
				in.SetGamepadTrigger(id, name, float32(v.Get("value").Float()))
			case input.GamepadSouth, input.GamepadEast, input.GamepadWest, input.GamepadNorth, input.GamepadLeftShoulder,
				input.GamepadRightShoulder, input.GamepadBack, input.GamepadStart, input.GamepadGuide,
				input.GamepadLeftStickPress, input.GamepadRightStickPress, input.GamepadUp, input.GamepadRight,
				input.GamepadDown, input.GamepadLeft, input.GamepadTouchpad, input.GamepadLeftStickLeft,
				input.GamepadLeftStickRight, input.GamepadLeftStickUp, input.GamepadLeftStickDown, input.GamepadRightStickLeft,
				input.GamepadRightStickRight, input.GamepadRightStickUp, input.GamepadRightStickDown:
				in.SetGamepadButton(id, name, v.Get("pressed").Bool())
			}
		}
		axes := g.Get("axes")
		if axes.Length() >= 4 {
			in.SetGamepadStick(
				id,
				input.LeftStick,
				geom.V2(float32(axes.Index(0).Float()), float32(axes.Index(1).Float())),
			)
			in.SetGamepadStick(
				id,
				input.RightStick,
				geom.V2(float32(axes.Index(2).Float()), float32(axes.Index(3).Float())),
			)
		}
	}
	for id := range p.known {
		if !seen[id] {
			in.DisconnectGamepad(id)
			delete(p.known, id)
		}
	}
}
