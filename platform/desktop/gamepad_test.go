package desktop //nolint:testpackage // tests the unexported joystick polling

import (
	"runtime"
	"testing"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/internal/inputfeed"
	"github.com/lantervale/ember/platform/capture"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGLFWsPadButtonsAreEmbersByPosition(t *testing.T) {
	for glfwButton, want := range map[glfw.GamepadButton]input.GamepadButton{
		glfw.ButtonCross:       input.GamepadSouth,
		glfw.ButtonCircle:      input.GamepadEast,
		glfw.ButtonSquare:      input.GamepadWest,
		glfw.ButtonTriangle:    input.GamepadNorth,
		glfw.ButtonLeftBumper:  input.GamepadLeftShoulder,
		glfw.ButtonRightBumper: input.GamepadRightShoulder,
		glfw.ButtonBack:        input.GamepadBack,
		glfw.ButtonStart:       input.GamepadStart,
		glfw.ButtonGuide:       input.GamepadGuide,
		glfw.ButtonLeftThumb:   input.GamepadLeftStickPress,
		glfw.ButtonRightThumb:  input.GamepadRightStickPress,
		glfw.ButtonDpadUp:      input.GamepadUp,
		glfw.ButtonDpadRight:   input.GamepadRight,
		glfw.ButtonDpadDown:    input.GamepadDown,
		glfw.ButtonDpadLeft:    input.GamepadLeft,
	} {
		assert.Equal(t, want, input.GamepadButton(glfwButton), "Cross is South, Circle East, Share Back, Options Start")
	}
}

func TestSonysPadsWithATouchpadReadItsClick(t *testing.T) {
	ds4 := "030000004c050000cc09000000010000"
	dualSense := "050000004c050000e60c000000010000"
	ps3 := "030000004c0500006802000000010000"
	xbox := "030000005e040000e002000000000000"
	want := 13
	if runtime.GOOS == "linux" {
		want = -1
	}
	assert.Equal(t, want, touchpadButton(ds4))
	assert.Equal(t, want, touchpadButton(dualSense))
	assert.Equal(t, -1, touchpadButton(ps3), "a PS3 pad has no touchpad")
	assert.Equal(t, -1, touchpadButton(xbox))
	assert.Equal(t, -1, touchpadButton("short"))
}

// realPad is a controller in a player's hands: connected, South held, the
// left stick leant right.
type realPad struct{}

func (realPad) IsGamepad() bool        { return true }
func (realPad) GetGamepadName() string { return "PS4 Controller" }
func (realPad) GetGUID() string        { return "030000004c050000cc09000000010000" }
func (realPad) GetButtons() []glfw.Action {
	return make([]glfw.Action, 16)
}

func (realPad) GetGamepadState() *glfw.GamepadState {
	var st glfw.GamepadState
	st.Buttons[glfw.ButtonCross] = glfw.Press
	st.Axes[glfw.AxisLeftX] = 1
	st.Axes[glfw.AxisLeftTrigger], st.Axes[glfw.AxisRightTrigger] = -1, -1
	return &st
}

func TestALiveRunReadsTheRealPad(t *testing.T) {
	var in input.State
	pollGamepads(&in, devices(false, []joystick{realPad{}}))
	inputfeed.Of(&in).BeginFrame()
	assert.True(t, in.Down(input.GamepadSouth))
	assert.Equal(t, float32(1), in.Stick(input.LeftStick).X)
}

func TestAScriptedRunReadsOnlyItsScriptedPad(t *testing.T) {
	pads, err := capture.ParsePads("north:0-10")
	require.NoError(t, err)
	var in input.State
	pollGamepads(&in, devices(true, []joystick{realPad{}}))
	pads.Apply(&in, 0)
	inputfeed.Of(&in).BeginFrame()
	assert.False(t, in.Down(input.GamepadSouth), "the real pad's South never reaches a capture")
	assert.Zero(t, in.Stick(input.LeftStick), "nor its stick")
	assert.True(t, in.Down(input.GamepadNorth), "the script's button does")
}
