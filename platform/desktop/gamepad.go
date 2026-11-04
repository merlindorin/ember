package desktop

import (
	"fmt"
	"os"
	"runtime"
	"slices"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
)

// loadGamepadMappings adds SDL game controller mappings from the file
// named by EMBER_GAMEPAD_DB (gamecontrollerdb.txt format) on top of the
// ones built into GLFW, which already cover PS4, PS5 and Xbox pads.
func loadGamepadMappings() {
	path := os.Getenv("EMBER_GAMEPAD_DB")
	if path == "" {
		return
	}
	db, err := os.ReadFile(path) //nolint:gosec // the file EMBER_GAMEPAD_DB names
	if err == nil && !glfw.UpdateGamepadMappings(string(db)) {
		err = fmt.Errorf("invalid mappings")
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "desktop: EMBER_GAMEPAD_DB %s: %v\n", path, err)
	}
}

// touchpadButton is the raw button of a pad's touchpad click, which GLFW
// 3.3's gamepad mappings leave out, -1 for a pad without one: button 13 of
// Sony's DualShock 4 and DualSense on macOS and Windows, as in SDL's
// controller database. On Linux the kernel reports the touchpad apart.
func touchpadButton(guid string) int {
	if runtime.GOOS == "linux" || !touchpadPad(guid) {
		return -1
	}
	return 13
}

// sony is Sony's vendor id, little endian as in a GUID.
const sony = "4c05"

// touchpads are the products, little endian as in a GUID, of Sony's pads
// with a touchpad: the DualShock 4, its V2 and wireless adaptor, the
// DualSense and the DualSense Edge.
var touchpads = []string{"c405", "cc09", "a00b", "e60c", "f20d"} //nolint:gochecknoglobals // a lookup table

// touchpadPad reports whether an SDL-style joystick GUID, the vendor then
// the product little endian in its third and fifth groups of four hex
// digits, is one of Sony's (054c) pads with a touchpad.
func touchpadPad(guid string) bool {
	return len(guid) >= 20 && guid[8:12] == sony && slices.Contains(touchpads, guid[16:20])
}

// joystick is a controller the backend reads: GLFW's, or a test's.
type joystick interface {
	IsGamepad() bool
	GetGamepadState() *glfw.GamepadState
	GetGamepadName() string
	GetGUID() string
	GetButtons() []glfw.Action
}

// joysticks are GLFW's, by slot.
var joysticks = func() []joystick { //nolint:gochecknoglobals // GLFW's slots, which tests replace
	out := make([]joystick, input.MaxGamepads)
	for i := range out {
		out[i] = glfw.Joystick1 + glfw.Joystick(i)
	}
	return out
}()

// devices is the real controllers a run reads: none when scripted, a
// capture or a hidden run, which only its script plays.
func devices(scripted bool, plugged []joystick) []joystick {
	if scripted {
		return nil
	}
	return plugged
}

// pollGamepads feeds every connected pad of joys into in, its slot the
// joystick's.
func pollGamepads(in *input.State, joys []joystick) {
	for i, joy := range joys {
		if !joy.IsGamepad() {
			in.DisconnectGamepad(i)
			continue
		}
		st := joy.GetGamepadState()
		if st == nil {
			continue
		}
		in.ConnectGamepad(i, joy.GetGamepadName())
		for b, a := range st.Buttons {
			in.SetGamepadButton(i, input.GamepadButton(b), a == glfw.Press)
		}
		if b := touchpadButton(joy.GetGUID()); b >= 0 {
			raw := joy.GetButtons()
			in.SetGamepadButton(i, input.GamepadTouchpad, b < len(raw) && raw[b] == glfw.Press)
		}
		in.SetGamepadStick(i, input.LeftStick, geom.V2(st.Axes[glfw.AxisLeftX], st.Axes[glfw.AxisLeftY]))
		in.SetGamepadStick(i, input.RightStick, geom.V2(st.Axes[glfw.AxisRightX], st.Axes[glfw.AxisRightY]))
		in.SetGamepadTrigger(i, input.GamepadLeftTrigger, (st.Axes[glfw.AxisLeftTrigger]+1)/2)
		in.SetGamepadTrigger(i, input.GamepadRightTrigger, (st.Axes[glfw.AxisRightTrigger]+1)/2)
	}
}
