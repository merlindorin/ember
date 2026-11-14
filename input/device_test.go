package input_test

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/internal/inputfeed"
)

func TestLastDeviceFollowsWhatThePlayerUses(t *testing.T) {
	var s input.State
	step := func() { inputfeed.Of(&s).BeginFrame() }
	if s.LastDevice() != input.KeyboardDevice {
		t.Fatal("the keyboard until anything is used")
	}
	s.ConnectGamepad(0, "PS4 Controller")
	s.SetGamepadButton(0, input.GamepadSouth, true)
	step()
	if s.LastDevice() != input.GamepadDevice {
		t.Fatal("a pad button")
	}
	step()
	if s.LastDevice() != input.GamepadDevice {
		t.Fatal("holding it keeps the pad")
	}
	s.SetMouse(geom.V2(1, 1))
	step()
	if s.LastDevice() != input.GamepadDevice {
		t.Fatal("a twitch of the mouse is not using it")
	}
	s.SetMouse(geom.V2(40, 30))
	step()
	if s.LastDevice() != input.MouseDevice {
		t.Fatal("moving the mouse")
	}
	s.SetKey(input.KeyW, true)
	step()
	if s.LastDevice() != input.KeyboardDevice {
		t.Fatal("a key")
	}
	s.SetGamepadStick(0, input.LeftStick, geom.V2(0.9, 0))
	step()
	if s.LastDevice() != input.GamepadDevice || input.GamepadDevice.String() != "gamepad" {
		t.Fatal("a stick leant past half way")
	}
	s.AddWheel(geom.V2(0, 1))
	step()
	if s.LastDevice() != input.MouseDevice {
		t.Fatal("the wheel")
	}
	s.AddText('a')
	step()
	if s.LastDevice() != input.KeyboardDevice {
		t.Fatal("typed text")
	}
}
