package input_test

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/internal/inputfeed"
)

func TestAMenuCapturesOnlyTheKeysItUses(t *testing.T) {
	var s input.State
	s.SetKey(input.KeyEnter, true)
	s.SetKey(input.KeyW, true)
	s.SetKey(input.KeyLeftShift, true)
	inputfeed.Of(&s).BeginFrame()
	inputfeed.Of(&s).CaptureKeys(int(input.KeyEnter), int(input.KeyTab), int(input.KeyLeftShift))

	assert.True(t, s.Captured(input.KeyEnter))
	assert.False(t, s.Down(input.KeyEnter) || s.Pressed(input.KeyEnter), "a captured key reads as up")
	assert.True(t, s.Pressed(input.KeyW), "every other key reaches the game")
	assert.False(t, s.Captured(input.KeyW))
	assert.True(t, s.Captured(input.KeyShift), "either side captured captures the pair")
	assert.False(t, s.KeyboardCaptured(), "a menu holds keys, not the keyboard")
	assert.True(t, inputfeed.Of(&s).RawPressed(int(input.KeyEnter)), "raw reads ignore capture")
	assert.False(t, s.Captured(input.MouseLeft))

	inputfeed.Of(&s).BeginFrame()
	assert.True(t, s.Captured(input.KeyEnter), "capture holds through the next frame")
	inputfeed.Of(&s).BeginFrame()
	assert.False(t, s.Captured(input.KeyEnter))
	assert.True(t, s.Down(input.KeyEnter))
}

func TestATextFieldCapturesEveryKey(t *testing.T) {
	var s input.State
	s.SetKey(input.KeyW, true)
	inputfeed.Of(&s).BeginFrame()
	inputfeed.Of(&s).CaptureKeyboard()
	assert.True(t, s.KeyboardCaptured())
	assert.True(t, s.Captured(input.KeyW))
	assert.False(t, s.Down(input.KeyW))
}

func TestAMenuCapturesOnlyThePadButtonsItUses(t *testing.T) {
	var s input.State
	s.ConnectGamepad(0, "pad")
	s.SetGamepadButton(0, input.GamepadSouth, true)
	s.SetGamepadButton(0, input.GamepadNorth, true)
	s.SetGamepadStick(0, input.LeftStick, geom.V2(1, 0))
	s.SetGamepadStick(0, input.RightStick, geom.V2(0, 1))
	inputfeed.Of(&s).BeginFrame()
	inputfeed.Of(&s).CaptureGamepadButtons(
		int(input.GamepadSouth),
		int(input.GamepadLeftStickLeft), int(input.GamepadLeftStickRight),
		int(input.GamepadLeftStickUp), int(input.GamepadLeftStickDown),
	)

	pad := s.Gamepad(0)
	assert.True(t, s.Captured(input.GamepadSouth))
	assert.False(t, s.Down(input.GamepadSouth) || pad.Down(input.GamepadSouth))
	assert.True(t, s.Pressed(input.GamepadNorth) && pad.Pressed(input.GamepadNorth))
	assert.Equal(t, geom.Vec2{}, s.Stick(input.LeftStick), "the stick's four directions capture it")
	assert.Equal(t, float32(0), s.Value(input.GamepadLeftStickRight))
	assert.NotEqual(t, geom.Vec2{}, pad.Stick(input.RightStick))
	assert.True(t, inputfeed.Of(&s).RawGamepadPressed(int(input.GamepadSouth)))

	inputfeed.Of(&s).BeginFrame()
	assert.True(t, s.Captured(input.GamepadSouth))
	inputfeed.Of(&s).BeginFrame()
	assert.True(t, s.Down(input.GamepadSouth))
}
