package input_test

import (
	"testing"

	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/internal/inputfeed"
)

// A press and its release that both arrive before a frame, as a quick tap
// or a scripted click does, show as a press that frame and a release the
// next, never as nothing.
func TestATapBetweenFramesIsPressedThenReleased(t *testing.T) {
	var s input.State
	s.SetMouseButton(input.MouseLeft, true)
	s.SetMouseButton(input.MouseLeft, false)
	s.SetKey(input.KeySpace, true)
	s.SetKey(input.KeySpace, false)
	inputfeed.Of(&s).BeginFrame()
	if !s.Pressed(input.MouseLeft) || !s.Pressed(input.KeySpace) {
		t.Fatal("the tap's press is lost")
	}
	inputfeed.Of(&s).BeginFrame()
	if !s.Released(input.MouseLeft) || !s.Released(input.KeySpace) {
		t.Fatal("the tap's release is lost")
	}
	inputfeed.Of(&s).BeginFrame()
	if s.Down(input.MouseLeft) || s.Released(input.MouseLeft) {
		t.Fatal("the tap outlives its release")
	}
}

// A release of a button a frame has already seen down applies at once.
func TestAHeldButtonsReleaseIsNotDelayed(t *testing.T) {
	var s input.State
	s.SetMouseButton(input.MouseLeft, true)
	inputfeed.Of(&s).BeginFrame()
	s.SetMouseButton(input.MouseLeft, false)
	inputfeed.Of(&s).BeginFrame()
	if !s.Released(input.MouseLeft) {
		t.Fatal("a held button's release waits a frame")
	}
}

// Taps quicker than frames each get their frames: three of them before
// one frame are three presses, a frame apart from their releases.
func TestTapsBeforeOneFrameAreEachPressed(t *testing.T) {
	var s input.State
	for range 3 {
		s.SetMouseButton(input.MouseLeft, true)
		s.SetMouseButton(input.MouseLeft, false)
	}
	s.SetMouseButton(input.MouseLeft, true)
	presses := 0
	for range 10 {
		inputfeed.Of(&s).BeginFrame()
		if s.Pressed(input.MouseLeft) {
			presses++
		}
	}
	if presses != 4 || !s.Down(input.MouseLeft) {
		t.Fatalf("three taps and a hold: %d presses, down %v", presses, s.Down(input.MouseLeft))
	}
}
