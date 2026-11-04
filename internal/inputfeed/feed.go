// Package inputfeed is the backend side of ember/input: what platform
// backends, ui and gui do to a window's input that games never do.
package inputfeed

// Feed drives one input state. Keys are input.Key values.
type Feed interface {
	// BeginFrame publishes what arrived since the last frame, keeping the
	// frame before as the baseline for presses and releases. The World
	// calls it for every window before any other system runs.
	BeginFrame()
	// AddRepeat records a key the system repeats while it is held.
	AddRepeat(key int)
	// RawDown, RawPressed, RawReleased and RawRepeated read keys even while
	// the keyboard is captured: for the layers that capture it.
	RawDown(key int) bool
	RawPressed(key int) bool
	RawReleased(key int) bool
	RawRepeated(key int) bool
	// GamepadChanges is the pads that connected and disconnected as this
	// frame began, by ID.
	GamepadChanges() (connected, disconnected []int)
	// CapturePointer and CaptureKeyboard mark the pointer or the whole
	// keyboard as the UI's for the rest of this frame and all of the next.
	CapturePointer()
	CaptureKeyboard()
	// CaptureKeys and CaptureGamepadButtons mark only these keys
	// (input.Key) or pad buttons (input.GamepadButton) as the UI's, for
	// the rest of this frame and all of the next: a focused menu takes
	// what it navigates by and leaves the rest to the game.
	CaptureKeys(keys ...int)
	CaptureGamepadButtons(buttons ...int)
	// RawGamepadDown reads whether a pad button, input.GamepadButton, is
	// held, even while the UI captures the pads.
	RawGamepadDown(button int) bool
	// RawGamepadPressed reads a pad button, input.GamepadButton, going
	// down on any pad, even while it is captured.
	RawGamepadPressed(button int) bool
	// Changed reports whether anything arrived this frame: a key, mouse or
	// pad button that went down, up or repeated, a stick that moved, the
	// pointer moved, the wheel turned, text typed, files dropped, a touch
	// or a gesture, a pad connecting or leaving. A quiet frame's UI can
	// draw as it did. A finger resting on a trackpad keeps it true.
	Changed() bool
	// Rumbles takes the rumbles asked for since it was last called, one
	// per pad at most, the latest; a zero one stops the pad.
	Rumbles() []Rumble
}

// Rumble is a pad's motors to play: Low the heavy left one, High the light
// right one, for Seconds.
type Rumble struct {
	Pad                int
	Low, High, Seconds float32
}

// Of is the feed of s, an *input.State; package input sets it.
var Of func(s any) Feed //nolint:gochecknoglobals // a hook another package fills at init, keeping the imports acyclic
