package input_test

import (
	"encoding/json"
	"testing"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/internal/inputfeed"
)

func TestAPadsButtonsReadLikeKeys(t *testing.T) {
	var s input.State
	s.ConnectGamepad(2, "PS5 Controller")
	s.SetGamepadButton(2, input.GamepadSouth, true)
	inputfeed.Of(&s).BeginFrame()
	if !s.Pressed(input.GamepadSouth) || !s.Down(input.GamepadSouth) || s.Value(input.GamepadSouth) != 1 {
		t.Fatal("the input reads any pad's south button")
	}
	inputfeed.Of(&s).BeginFrame()
	if s.Pressed(input.GamepadSouth) || !s.Down(input.GamepadSouth) {
		t.Fatal("a held button is not pressed again")
	}
	s.SetGamepadButton(2, input.GamepadSouth, false)
	inputfeed.Of(&s).BeginFrame()
	if !s.Released(input.GamepadSouth) {
		t.Fatal("letting go releases it")
	}
	if input.GamepadSouth.String() != "GamepadSouth" || input.GamepadRightStickDown.String() != "GamepadRightStickDown" {
		t.Fatal("buttons print their position")
	}
}

func TestEachPadIsAPlayersOwn(t *testing.T) {
	var s input.State
	s.ConnectGamepad(0, "one")
	s.ConnectGamepad(3, "two")
	s.SetGamepadButton(3, input.GamepadEast, true)
	s.SetKey(input.KeySpace, true)
	inputfeed.Of(&s).BeginFrame()
	pads := s.Gamepads()
	if len(pads) != 2 || pads[0].ID() != 0 || pads[1].ID() != 3 || pads[1].Name() != "two" {
		t.Fatalf("connected pads by ID: %v", pads)
	}
	one, two := s.Gamepad(0), s.Gamepad(3)
	if one.Down(input.GamepadEast) || !two.Pressed(input.GamepadEast) {
		t.Fatal("each pad reads its own buttons")
	}
	if two.Down(input.KeySpace) {
		t.Fatal("keys read as up on a pad")
	}
	if g := s.Gamepad(5); g.Connected() || g.Down(input.GamepadEast) || s.Gamepad(99).Connected() {
		t.Fatal("an empty slot reads as idle")
	}
}

func TestSticksLeanPastARescaledDeadzone(t *testing.T) {
	var s input.State
	s.ConnectGamepad(0, "pad")
	s.SetGamepadStick(0, input.LeftStick, geom.V2(0.1, 0))
	inputfeed.Of(&s).BeginFrame()
	if v := s.Stick(input.LeftStick); v != (geom.Vec2{}) {
		t.Fatalf("inside the deadzone: %v", v)
	}
	s.SetGamepadStick(0, input.LeftStick, geom.V2(0, -0.6))
	inputfeed.Of(&s).BeginFrame()
	if v := s.Stick(input.LeftStick); geom.Abs(v.Y+0.5) > 1e-5 || v.X != 0 {
		t.Fatalf("halfway past the deadzone reads 0.5: %v", v)
	}
	if v := s.Value(input.GamepadLeftStickUp); geom.Abs(v-0.5) > 1e-5 || s.Value(input.GamepadLeftStickDown) != 0 {
		t.Fatalf("a stick direction's value is the lean that way: %v", v)
	}
	s.SetGamepadStick(0, input.LeftStick, geom.V2(0, -0.5))
	inputfeed.Of(&s).BeginFrame()
	if s.Down(input.GamepadLeftStickUp) || s.Value(input.GamepadLeftStickUp) == 0 {
		t.Fatal("a direction leant less than half way has a value but is not down")
	}
	s.SetGamepadStick(0, input.LeftStick, geom.V2(0, -0.6))
	s.SetDeadzone(0.5)
	inputfeed.Of(&s).BeginFrame()
	if v := s.Stick(input.LeftStick); geom.Abs(v.Y+0.2) > 1e-5 || s.Deadzone() != 0.5 {
		t.Fatalf("a wider deadzone: %v", v)
	}
	if (&input.State{}).Deadzone() != 0.2 {
		t.Fatal("the default deadzone")
	}
}

func TestTriggersAndStickDirectionsAreAnalogButtons(t *testing.T) {
	var s input.State
	s.ConnectGamepad(1, "pad")
	s.SetGamepadTrigger(1, input.GamepadRightTrigger, 0.3)
	inputfeed.Of(&s).BeginFrame()
	if s.Value(input.GamepadRightTrigger) != 0.3 || s.Down(input.GamepadRightTrigger) {
		t.Fatal("a light pull is a value, not yet down")
	}
	s.SetGamepadTrigger(1, input.GamepadRightTrigger, 0.8)
	inputfeed.Of(&s).BeginFrame()
	if !s.Pressed(input.GamepadRightTrigger) {
		t.Fatal("past half way it is pressed")
	}
	s.SetGamepadButton(1, input.GamepadLeftStickRight, true)
	s.SetGamepadButton(1, input.GamepadLeftTrigger, true)
	inputfeed.Of(&s).BeginFrame()
	if s.Stick(input.LeftStick) != geom.V2(1, 0) || s.Value(input.GamepadLeftTrigger) != 1 {
		t.Fatal("a direction set as a button leans all the way, a trigger pulls fully")
	}
}

func TestDisconnectingClearsThePadAndIsReported(t *testing.T) {
	var s input.State
	s.ConnectGamepad(0, "pad")
	s.SetGamepadButton(0, input.GamepadStart, true)
	inputfeed.Of(&s).BeginFrame()
	in, out := inputfeed.Of(&s).GamepadChanges()
	if len(in) != 1 || in[0] != 0 || len(out) != 0 {
		t.Fatalf("connected: %v %v", in, out)
	}
	inputfeed.Of(&s).BeginFrame()
	if again, _ := inputfeed.Of(&s).GamepadChanges(); len(again) != 0 {
		t.Fatal("a connection is reported once")
	}
	s.DisconnectGamepad(0)
	inputfeed.Of(&s).BeginFrame()
	if _, gone := inputfeed.Of(&s).GamepadChanges(); len(gone) != 1 || gone[0] != 0 {
		t.Fatalf("disconnected: %v", gone)
	}
	if s.Down(input.GamepadStart) || len(s.Gamepads()) != 0 {
		t.Fatal("a gone pad holds nothing")
	}
}

func TestPadsIgnoreKeyboardCapture(t *testing.T) {
	var s input.State
	s.ConnectGamepad(0, "pad")
	s.SetGamepadButton(0, input.GamepadStart, true)
	inputfeed.Of(&s).BeginFrame()
	inputfeed.Of(&s).CaptureKeyboard()
	if !s.Pressed(input.GamepadStart) {
		t.Fatal("a text field has the keyboard, not the pad")
	}
}

func TestOneActionMapMovesByKeysOrStick(t *testing.T) {
	type act int
	const (
		left act = iota
		right
		up
		down
		jump
	)
	controls := input.NewActions(
		input.Bind(left, input.KeyA, input.GamepadLeftStickLeft),
		input.Bind(right, input.KeyD, input.GamepadLeftStickRight),
		input.Bind(up, input.KeyW, input.GamepadLeftStickUp),
		input.Bind(down, input.KeyS, input.GamepadLeftStickDown),
		input.Bind(jump, input.KeySpace, input.GamepadSouth),
	)
	var s input.State
	s.SetKey(input.KeyD, true)
	s.SetKey(input.KeyW, true)
	inputfeed.Of(&s).BeginFrame()
	if v := controls.Direction(&s, left, right, up, down); geom.Abs(v.Len()-1) > 1e-5 || v.X <= 0 || v.Y >= 0 {
		t.Fatalf("keys point at unit length: %v", v)
	}
	s.SetKey(input.KeyD, false)
	s.SetKey(input.KeyW, false)
	s.ConnectGamepad(1, "pad")
	s.SetGamepadStick(1, input.LeftStick, geom.V2(0.6, 0))
	s.SetGamepadButton(1, input.GamepadSouth, true)
	inputfeed.Of(&s).BeginFrame()
	if v := controls.Direction(&s, left, right, up, down); geom.Abs(v.X-0.5) > 1e-5 || v.Y != 0 {
		t.Fatalf("a stick keeps its lean: %v", v)
	}
	if !controls.Pressed(&s, jump) || !controls.Pressed(s.Gamepad(1), jump) || controls.Down(s.Gamepad(0), jump) {
		t.Fatal("the map reads the input or one player's pad")
	}

	data, err := json.Marshal(controls)
	if err != nil {
		t.Fatal(err)
	}
	back := input.NewActions(input.Bind(jump))
	if err = json.Unmarshal(data, back); err != nil {
		t.Fatal(err)
	}
	if b := back.Bindings(jump); len(b) != 2 || b[1] != input.GamepadSouth {
		t.Fatalf("pad buttons save by name: %s", data)
	}
}

func TestTheTouchpadClickIsADigitalButtonAndTheAnalogOnesStillAre(t *testing.T) {
	var s input.State
	s.ConnectGamepad(0, "PS4 Controller")
	s.SetGamepadButton(0, input.GamepadTouchpad, true)
	s.SetGamepadTrigger(0, input.GamepadLeftTrigger, 0.3)
	inputfeed.Of(&s).BeginFrame()
	if !s.Pressed(input.GamepadTouchpad) || input.GamepadTouchpad.String() != "GamepadTouchpad" {
		t.Fatal("the touchpad click presses like any button")
	}
	if v := s.Value(input.GamepadLeftTrigger); v < 0.29 || v > 0.31 || s.Down(input.GamepadLeftTrigger) {
		t.Fatalf("L2 stays analog, up below half way: %v", v)
	}
}
