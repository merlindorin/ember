// Package input is a frame-coherent snapshot of keyboard, mouse and
// gamepad state. Each window has one; its platform backend feeds it, and
// games only read it, or simulate input on it in tests and captures.
//
//	in := win.Input()
//	if in.Pressed(input.KeySpace) { jump() }
//	move := in.Direction(input.KeyA, input.KeyD, input.KeyW, input.KeyS)
package input

import (
	"fmt"
	"slices"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/internal/inputfeed"
)

const keyCount = int(keyLast) + 1

type State struct {
	keys, prevKeys, liveKeys          [keyCount]bool
	repeated                          [keyCount]bool
	pendingRepeats                    []Key
	buttons, prevButtons, liveButtons [mouseButtonCount]bool
	// keyEdges and buttonEdges wait for a later frame: each is a change
	// to a key or button that already changed since the last frame, so a
	// tap quicker than a frame shows as a press, then a release.
	keyEdges, buttonEdges []edge

	mouse, prevMouse, liveMouse geom.Vec2
	wheel                       geom.Vec2
	pendingWheel                geom.Vec2
	text                        []rune
	pendingText                 []rune
	dropped                     []string
	pendingDrops                []string
	pads, livePads              [MaxGamepads]Gamepad
	joined, left                []int
	deadzone                    float32
	deadzoneSet                 bool
	touches                     []Touch
	liveTouches                 []liveTouch
	pinch, rotation             float32
	pendingPinch                float32
	pendingRotation             float32
	pan, swipe, pendingSwipe    geom.Vec2
	rumbleOff                   bool
	outRumbles                  [MaxGamepads]Rumble
	outSent                     [MaxGamepads]bool
	rumbles                     *RumbleRecorder

	pointerCaptured, nextPointer   bool
	keyboardCaptured, nextKeyboard bool
	capturedKeys, nextKeys         [keyCount]bool
	capturedPad, nextPadButtons    [gamepadButtonCount]bool
	lastDevice                     Device
}

// Button is anything held down: a Key, a MouseButton or a GamepadButton.
type Button interface {
	String() string
	level(r Reader, before bool) float32
}

// Reader is what buttons are read from: a window's *State, any pad on it,
// or one *Gamepad, one player's.
type Reader interface {
	keyLevel(k Key, before bool) float32
	mouseLevel(b MouseButton, before bool) float32
	padLevel(b GamepadButton, before bool) float32
}

// Down reports whether b is held: on any connected pad for a gamepad
// button, and past half way for a trigger or a stick direction. Keys read
// as up while a text field has the keyboard.
func (s *State) Down(b Button) bool { return down(s, b) }

// Pressed reports whether b went down this frame.
func (s *State) Pressed(b Button) bool { return pressed(s, b) }

// Released reports whether b went up this frame.
func (s *State) Released(b Button) bool { return released(s, b) }

// Value is how far b is held, from 0 to 1: a trigger's pull, a stick's
// lean that way; 0 or 1 for keys and buttons.
func (s *State) Value(b Button) float32 { return b.level(s, false) }

func down(r Reader, b Button) bool     { return b.level(r, false) > 0.5 }
func pressed(r Reader, b Button) bool  { return down(r, b) && b.level(r, true) <= 0.5 }
func released(r Reader, b Button) bool { return !down(r, b) && b.level(r, true) > 0.5 }

func axis(r Reader, neg, pos Button) float32 { return pos.level(r, false) - neg.level(r, false) }

func direction(r Reader, left, right, up, down Button) geom.Vec2 {
	v := geom.Vec2{X: axis(r, left, right), Y: axis(r, up, down)}
	if v.Len() > 1 {
		return v.Normalize()
	}
	return v
}

func (s *State) keyLevel(k Key, before bool) float32 {
	if s.keyCaptured(k) || !k.raw(s, before) {
		return 0
	}
	return 1
}

func (s *State) mouseLevel(b MouseButton, before bool) float32 {
	if b < 0 || int(b) >= mouseButtonCount {
		return 0
	}
	held := s.buttons[b]
	if before {
		held = s.prevButtons[b]
	}
	if held {
		return 1
	}
	return 0
}

func (s *State) padLevel(b GamepadButton, before bool) float32 {
	if s.padButtonCaptured(b) {
		return 0
	}
	var v float32
	for i := range s.pads {
		v = max(v, s.pads[i].padLevel(b, before))
	}
	return v
}

// PointerCaptured reports whether the UI holds the pointer: it is over an
// interactive element, so gameplay should ignore its clicks. The UI
// captures as it draws, late in a frame, and the capture holds through the
// next one, so every system sees it.
func (s *State) PointerCaptured() bool { return s.pointerCaptured }

// KeyboardCaptured reports whether the UI has the whole keyboard, as a
// text field does: keys then read as up, so typing triggers no shortcuts.
func (s *State) KeyboardCaptured() bool { return s.keyboardCaptured }

// Captured reports whether the UI holds b: the keys and pad buttons a
// focused menu navigates by, or every key while a text field has the
// keyboard. A captured button reads as up; every other one still reaches
// the game. Mouse buttons are never captured, PointerCaptured says when
// the pointer is over the UI.
func (s *State) Captured(b Button) bool {
	switch b := b.(type) {
	case Key:
		return s.keyCaptured(b)
	case GamepadButton:
		return s.padButtonCaptured(b)
	}
	return false
}

func (s *State) keyCaptured(k Key) bool {
	if s.keyboardCaptured {
		return true
	}
	if pair, ok := sides(k); ok {
		return s.keyCaptured(pair[0]) || s.keyCaptured(pair[1])
	}
	return valid(k) && s.capturedKeys[k]
}

func (s *State) padButtonCaptured(b GamepadButton) bool {
	return b >= 0 && b < gamepadButtonCount && s.capturedPad[b]
}

// Mouse is the pointer's position in window points, from the window's
// top-left corner. A 2D game finds the world under it with its camera:
// cam.ScreenToWorld(in.Mouse()).
func (s *State) Mouse() geom.Vec2 { return s.mouse }

// MouseDelta is the pointer's movement since the previous frame.
func (s *State) MouseDelta() geom.Vec2 { return s.mouse.Sub(s.prevMouse) }

// Wheel is this frame's wheel movement, a trackpad's two-finger scroll
// too, in lines: Y up is negative.
func (s *State) Wheel() geom.Vec2 { return s.wheel }

// Text is the characters typed this frame, for text fields: what a key
// writes in the player's layout, where a Key is a position.
func (s *State) Text() []rune { return s.text }

// Dropped is the paths of the files and folders dropped on the window
// this frame.
func (s *State) Dropped() []string { return s.dropped }

// Axis is pos's value less neg's: -1, 0 or 1 for keys, in between for a
// stick or a trigger.
func (s *State) Axis(neg, pos Button) float32 { return axis(s, neg, pos) }

// Direction is where four buttons point, Y down the screen (so up is
// negative), as on screen and on the ground: unit length for keys, a
// stick's lean kept, never longer than 1.
func (s *State) Direction(left, right, up, down Button) geom.Vec2 {
	return direction(s, left, right, up, down)
}

// The rest simulates input: backends call it as events arrive, and tests
// and scripted captures call it to play a player. What it sets shows from
// the next frame, as a real event would.

// SetKey holds k down or lets it go.
func (s *State) SetKey(k Key, down bool) {
	if valid(k) {
		setLevel(s.liveKeys[:], s.keys[:], &s.keyEdges, int(k), down)
	}
}

// SetMouseButton holds b down or lets it go.
func (s *State) SetMouseButton(b MouseButton, down bool) {
	if b >= 0 && int(b) < mouseButtonCount {
		setLevel(s.liveButtons[:], s.buttons[:], &s.buttonEdges, int(b), down)
	}
}

// edge is one waiting change: i goes down or up.
type edge struct {
	i    int
	down bool
}

// setLevel changes live[i] to down at once unless i already changed since
// the frame that saw seen, or has changes waiting: then it waits in edges.
func setLevel(live, seen []bool, edges *[]edge, i int, down bool) {
	last, waiting := live[i], false
	for _, e := range *edges {
		if e.i == i {
			last, waiting = e.down, true
		}
	}
	switch {
	case last == down:
	case waiting || live[i] != seen[i]:
		*edges = append(*edges, edge{i, down})
	default:
		live[i] = down
	}
}

// applyEdges applies each key's or button's first waiting change, after a
// frame has taken live; the rest wait for the next frames.
func applyEdges(live []bool, edges *[]edge) {
	rest := (*edges)[:0]
	var done []int
	for _, e := range *edges {
		if slices.Contains(done, e.i) {
			rest = append(rest, e)
			continue
		}
		done = append(done, e.i)
		live[e.i] = e.down
	}
	*edges = rest
}

// SetMouse moves the pointer to p, in window points.
func (s *State) SetMouse(p geom.Vec2) { s.liveMouse = p }

// AddWheel turns the wheel by d, seen next frame.
func (s *State) AddWheel(d geom.Vec2) { s.pendingWheel = s.pendingWheel.Add(d) }

// AddText types r, seen next frame.
func (s *State) AddText(r rune) { s.pendingText = append(s.pendingText, r) }

// AddDrop drops paths on the window, seen next frame.
func (s *State) AddDrop(paths ...string) { s.pendingDrops = append(s.pendingDrops, paths...) }

func valid(k Key) bool { return k >= 0 && int(k) < keyCount }

//nolint:gochecknoinits // backends reach a State's feed through inputfeed, which cannot import input
func init() {
	inputfeed.Of = func(s any) inputfeed.Feed {
		st, ok := s.(*State)
		if !ok {
			panic(fmt.Sprintf("input: a feed is for a *input.State, not %T", s))
		}
		return feed{st}
	}
}

// feed is a state's backend side.
type feed struct{ s *State }

func (f feed) BeginFrame() {
	s := f.s
	s.prevKeys, s.keys = s.keys, s.liveKeys
	s.prevButtons, s.buttons = s.buttons, s.liveButtons
	applyEdges(s.liveKeys[:], &s.keyEdges)
	applyEdges(s.liveButtons[:], &s.buttonEdges)
	s.prevMouse, s.mouse = s.mouse, s.liveMouse
	s.pointerCaptured, s.nextPointer = s.nextPointer, false
	s.keyboardCaptured, s.nextKeyboard = s.nextKeyboard, false
	s.capturedKeys, s.nextKeys = s.nextKeys, [keyCount]bool{}
	s.capturedPad, s.nextPadButtons = s.nextPadButtons, [gamepadButtonCount]bool{}
	s.beginPads()
	s.beginTouches()
	s.wheel, s.pendingWheel = s.pendingWheel, geom.Vec2{}
	s.text, s.pendingText = append(s.text[:0], s.pendingText...), s.pendingText[:0]
	s.dropped, s.pendingDrops = append(s.dropped[:0], s.pendingDrops...), s.pendingDrops[:0]
	clear(s.repeated[:])
	for _, k := range s.pendingRepeats {
		if valid(k) {
			s.repeated[k] = true
		}
	}
	s.pendingRepeats = s.pendingRepeats[:0]
	s.noteDevice()
}

func (f feed) Changed() bool {
	s := f.s
	if s.keys != s.prevKeys || s.buttons != s.prevButtons || s.mouse != s.prevMouse ||
		s.wheel != (geom.Vec2{}) || len(s.text) > 0 || len(s.dropped) > 0 || len(s.joined)+len(s.left) > 0 ||
		len(s.touches) > 0 || s.pinch != 0 || s.rotation != 0 || s.pan != (geom.Vec2{}) || s.swipe != (geom.Vec2{}) {
		return true
	}
	if slices.Contains(s.repeated[:], true) {
		return true
	}
	for i := range s.pads {
		p := &s.pads[i]
		if p.buttons != p.prevButtons {
			return true
		}
		for a := range p.axes {
			if abs(p.axes[a]-p.prevAxes[a]) > stickNoise {
				return true
			}
		}
	}
	return false
}

// stickNoise is how far a stick or trigger moves between frames before it
// counts as moved, not a resting pad's jitter.
const stickNoise = 0.01

func abs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

func (f feed) AddRepeat(key int) { f.s.pendingRepeats = append(f.s.pendingRepeats, Key(key)) }

func (f feed) RawDown(key int) bool { return Key(key).raw(f.s, false) }

func (f feed) RawPressed(key int) bool {
	return Key(key).raw(f.s, false) && !Key(key).raw(f.s, true)
}

func (f feed) RawReleased(key int) bool {
	return !Key(key).raw(f.s, false) && Key(key).raw(f.s, true)
}

func (f feed) RawRepeated(key int) bool       { return valid(Key(key)) && f.s.repeated[key] }
func (f feed) GamepadChanges() ([]int, []int) { return f.s.joined, f.s.left }
func (f feed) CapturePointer()                { f.s.pointerCaptured, f.s.nextPointer = true, true }
func (f feed) CaptureKeyboard()               { f.s.keyboardCaptured, f.s.nextKeyboard = true, true }

func (f feed) CaptureKeys(keys ...int) {
	for _, k := range keys {
		if valid(Key(k)) {
			f.s.capturedKeys[k], f.s.nextKeys[k] = true, true
		}
	}
}

func (f feed) CaptureGamepadButtons(buttons ...int) {
	for _, b := range buttons {
		if b >= 0 && b < int(gamepadButtonCount) {
			f.s.capturedPad[b], f.s.nextPadButtons[b] = true, true
		}
	}
}

func (f feed) RawGamepadDown(button int) bool {
	b := GamepadButton(button)
	var now float32
	for i := range f.s.pads {
		now = max(now, f.s.pads[i].rawLevel(b, false))
	}
	return now > 0.5
}

func (f feed) RawGamepadPressed(button int) bool {
	b := GamepadButton(button)
	var now, before float32
	for i := range f.s.pads {
		now, before = max(now, f.s.pads[i].rawLevel(b, false)), max(before, f.s.pads[i].rawLevel(b, true))
	}
	return now > 0.5 && before <= 0.5
}
