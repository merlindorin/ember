package capture_test

import (
	"testing"

	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/internal/inputfeed"
	"github.com/lantervale/ember/platform/capture"
)

func padAt(t *testing.T, p capture.Pads, frame int) *input.State {
	t.Helper()
	var in input.State
	p.Apply(&in, frame)
	inputfeed.Of(&in).BeginFrame()
	return &in
}

func TestAScriptedPadHoldsButtonsAndMovesItsSticksAndTriggers(t *testing.T) {
	p, err := capture.ParsePads(
		"south: 10-20; leftstick: 0,0@0; leftstick: 1,-1@40; righttrigger: 0@0; righttrigger: 1@20",
	)
	if err != nil {
		t.Fatal(err)
	}
	in := padAt(t, p, 15)
	pad := in.Gamepad(0)
	if !pad.Connected() || !pad.Down(input.GamepadSouth) {
		t.Fatal("frame 15: pad 0 is plugged in, south held")
	}
	if v := pad.Value(input.GamepadRightTrigger); v < 0.74 || v > 0.76 {
		t.Fatalf("frame 15: the trigger is three quarters pulled: %v", v)
	}
	if s := in.Gamepad(0).Stick(input.LeftStick); s.X <= 0.2 || s.Y >= -0.2 {
		t.Fatalf("frame 15: the stick leans right and up, past the deadzone: %v", s)
	}
	in = padAt(t, p, 20)
	if in.Gamepad(0).Down(input.GamepadSouth) {
		t.Fatal("frame 20 lets go of south")
	}
	if s := padAt(t, p, 60).Gamepad(0).Stick(input.LeftStick); s.X < 0.6 || s.Y > -0.6 {
		t.Fatalf("after its last waypoint the stick stays there: %v", s)
	}
}

func TestEachPadIsPluggedInForItsRange(t *testing.T) {
	p, err := capture.ParsePads("connect:0-50; 1/connect:20-40; 1/north:20-40")
	if err != nil {
		t.Fatal(err)
	}
	if in := padAt(t, p, 10); !in.Gamepad(0).Connected() || in.Gamepad(1).Connected() {
		t.Fatal("frame 10: pad 0 only")
	}
	in := padAt(t, p, 30)
	if !in.Gamepad(1).Connected() || !in.Gamepad(1).Down(input.GamepadNorth) || len(in.Gamepads()) != 2 {
		t.Fatal("frame 30: both pads, north held on pad 1")
	}
	if later := padAt(t, p, 45); later.Gamepad(1).Connected() {
		t.Fatal("frame 45: pad 1 is unplugged")
	}
}

func TestBadPadScriptsAreRefused(t *testing.T) {
	for _, s := range []string{
		"jump:0-4", "south:4-4", "leftstick:1@3", "lefttrigger:a@3",
		"16/south:0-4", "x/south:0-4", "leftstick:0,0@5;leftstick:1,1@2",
	} {
		if _, err := capture.ParsePads(s); err == nil {
			t.Errorf("%q was accepted", s)
		}
	}
}

func TestAScriptedPadHidesARealPadInItsSlot(t *testing.T) {
	p, err := capture.ParsePads("east:0-10")
	if err != nil {
		t.Fatal(err)
	}
	var in input.State
	in.ConnectGamepad(0, "Wireless Controller")
	in.SetGamepadButton(0, input.GamepadSouth, true)
	in.SetGamepadTrigger(0, input.GamepadLeftTrigger, 1)
	p.Apply(&in, 5)
	inputfeed.Of(&in).BeginFrame()
	pad := in.Gamepad(0)
	if pad.Down(input.GamepadSouth) || pad.Value(input.GamepadLeftTrigger) != 0 || !pad.Down(input.GamepadEast) {
		t.Fatal("only the script's state shows, not the real pad's")
	}
}

func TestACaptureClicksTheTouchpad(t *testing.T) {
	p, err := capture.ParsePads("touchpad: 5-10")
	if err != nil {
		t.Fatal(err)
	}
	if !padAt(t, p, 7).Gamepad(0).Down(input.GamepadTouchpad) {
		t.Fatal("frame 7: the touchpad held")
	}
}

func TestOneStickKeyframeHoldsFromItsFrameOn(t *testing.T) {
	p, err := capture.ParsePads("leftstick: 1,0@10")
	if err != nil {
		t.Fatal(err)
	}
	var in input.State
	for frame := range 60 {
		p.Apply(&in, frame)
		inputfeed.Of(&in).BeginFrame()
		held := in.Gamepad(0).Stick(input.LeftStick).X > 0.9 && in.Down(input.GamepadLeftStickRight)
		switch {
		case frame < 10 && held:
			t.Fatalf("frame %d: at rest before its keyframe", frame)
		case frame >= 10 && !held:
			t.Fatalf("frame %d: one keyframe holds the stick from its frame on", frame)
		case frame == 10 && !in.Pressed(input.GamepadLeftStickRight):
			t.Fatal("so leaning it presses that direction once, as a menu reads it")
		}
	}
}
