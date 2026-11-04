package input

// Device is a kind of input a player uses.
type Device int

const (
	KeyboardDevice Device = iota
	MouseDevice
	GamepadDevice
	TouchDevice
)

func (d Device) String() string {
	return [...]string{"keyboard", "mouse", "gamepad", "touch"}[d]
}

// LastDevice is the kind of input the player used last: a key or typed
// text, a mouse button, the wheel or a move of the pointer, a pad's button,
// stick or trigger, or a finger. A footer reads it to show "Enter" or the
// pad's button. It is the keyboard until something is used.
func (s *State) LastDevice() Device { return s.lastDevice }

// noteDevice records the device used this frame.
func (s *State) noteDevice() {
	for k := range s.keys {
		if s.keys[k] && !s.prevKeys[k] {
			s.lastDevice = KeyboardDevice
		}
	}
	if len(s.text) > 0 {
		s.lastDevice = KeyboardDevice
	}
	for b := range s.buttons {
		if s.buttons[b] && !s.prevButtons[b] {
			s.lastDevice = MouseDevice
		}
	}
	if s.wheel.X != 0 || s.wheel.Y != 0 || s.mouse.Dist(s.prevMouse) > 3 {
		s.lastDevice = MouseDevice
	}
	for _, t := range s.touches {
		if t.Phase == TouchBegan {
			s.lastDevice = TouchDevice
		}
	}
	for i := range s.pads {
		p := &s.pads[i]
		for b := GamepadButton(0); b < gamepadButtonCount; b++ {
			if p.rawLevel(b, false) > 0.5 && p.rawLevel(b, true) <= 0.5 {
				s.lastDevice = GamepadDevice
			}
		}
	}
}
