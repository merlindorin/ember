package input

import (
	"fmt"

	"github.com/merlindorin/ember/geom"
)

// GamepadButton is a controller's button, named by its position on a
// standard pad. Triggers and stick directions are analog: their Value is
// how far they are pulled or leant, and they count as down past half way.
type GamepadButton int

const (
	GamepadSouth GamepadButton = iota
	GamepadEast
	GamepadWest
	GamepadNorth
	GamepadLeftShoulder
	GamepadRightShoulder
	GamepadBack
	GamepadStart
	GamepadGuide
	GamepadLeftStickPress
	GamepadRightStickPress
	GamepadUp
	GamepadRight
	GamepadDown
	GamepadLeft
	// GamepadTouchpad is a PlayStation pad's touchpad click.
	GamepadTouchpad
	GamepadLeftTrigger
	GamepadRightTrigger
	GamepadLeftStickLeft
	GamepadLeftStickRight
	GamepadLeftStickUp
	GamepadLeftStickDown
	GamepadRightStickLeft
	GamepadRightStickRight
	GamepadRightStickUp
	GamepadRightStickDown

	gamepadButtonCount
	digitalCount = int(GamepadTouchpad) + 1
)

// String names a gamepad button: "GamepadSouth", "GamepadLeftStickUp".
func (b GamepadButton) String() string {
	padNames := [...]string{
		"South", "East", "West", "North", "LeftShoulder", "RightShoulder", "Back", "Start", "Guide",
		"LeftStickPress", "RightStickPress", "Up", "Right", "Down", "Left", "Touchpad", "LeftTrigger", "RightTrigger",
		"LeftStickLeft", "LeftStickRight", "LeftStickUp", "LeftStickDown",
		"RightStickLeft", "RightStickRight", "RightStickUp", "RightStickDown",
	}
	if b >= 0 && b < gamepadButtonCount {
		return "Gamepad" + padNames[b]
	}
	return fmt.Sprintf("GamepadButton(%d)", int(b))
}

func (b GamepadButton) level(r Reader, before bool) float32 { return r.padLevel(b, before) }

// Stick is one of a pad's two sticks.
type Stick int

const (
	LeftStick Stick = iota
	RightStick
)

// MaxGamepads is how many controllers are tracked at once; a pad's ID is
// its slot, below it.
const MaxGamepads = 16

const defaultDeadzone = 0.2

const (
	leftX = iota
	leftY
	rightX
	rightY
	leftTrigger
	rightTrigger
	axisCount
)

// Gamepad is one controller's frame, read like the input it arrives on:
// keys and mouse buttons read as up on it.
type Gamepad struct {
	id                   int
	name                 string
	connected            bool
	buttons, prevButtons [digitalCount]bool
	axes, prevAxes       [axisCount]float32
	deadzone             float32
	state                *State
}

// ID is the pad's slot, stable while it stays connected.
func (g *Gamepad) ID() int { return g.id }

// Name is the controller's name, such as "PS5 Controller".
func (g *Gamepad) Name() string { return g.name }

// Connected reports whether a controller is in this slot.
func (g *Gamepad) Connected() bool { return g.connected }

func (g *Gamepad) Down(b Button) bool                    { return down(g, b) }
func (g *Gamepad) Pressed(b Button) bool                 { return pressed(g, b) }
func (g *Gamepad) Released(b Button) bool                { return released(g, b) }
func (g *Gamepad) Value(b Button) float32                { return b.level(g, false) }
func (g *Gamepad) Axis(neg, pos Button) float32          { return axis(g, neg, pos) }
func (g *Gamepad) Direction(l, r, u, d Button) geom.Vec2 { return direction(g, l, r, u, d) }

// Stick is where s leans, each axis in [-1, 1] with Y down, past a
// radial deadzone rescaled so a slight lean starts slow.
func (g *Gamepad) Stick(s Stick) geom.Vec2 {
	first := GamepadLeftStickLeft
	if s == RightStick {
		first = GamepadRightStickLeft
	}
	for b := first; b < first+4; b++ {
		if g.capturedButton(b) {
			return geom.Vec2{}
		}
	}
	return g.stick(s, false)
}

func (g *Gamepad) capturedButton(b GamepadButton) bool {
	return g.state != nil && g.state.padButtonCaptured(b)
}

func (g *Gamepad) stick(s Stick, before bool) geom.Vec2 {
	if !g.connected {
		return geom.Vec2{}
	}
	axes := g.axes
	if before {
		axes = g.prevAxes
	}
	x := leftX
	if s == RightStick {
		x = rightX
	}
	v := geom.V2(axes[x], axes[x+1])
	dz := g.deadzone
	l := v.Len()
	if l <= dz {
		return geom.Vec2{}
	}
	return v.Scale(geom.Clamp01((l-dz)/(1-dz)) / l)
}

func (*Gamepad) keyLevel(Key, bool) float32           { return 0 }
func (*Gamepad) mouseLevel(MouseButton, bool) float32 { return 0 }

func (g *Gamepad) padLevel(b GamepadButton, before bool) float32 {
	if g.capturedButton(b) {
		return 0
	}
	return g.rawLevel(b, before)
}

func (g *Gamepad) rawLevel(b GamepadButton, before bool) float32 {
	if !g.connected {
		return 0
	}
	switch {
	case b >= 0 && int(b) < digitalCount:
		buttons := g.buttons
		if before {
			buttons = g.prevButtons
		}
		if buttons[b] { //nolint:gosec // b < digitalCount, the array's length
			return 1
		}
		return 0
	case b == GamepadLeftTrigger || b == GamepadRightTrigger:
		axes := g.axes
		if before {
			axes = g.prevAxes
		}
		return geom.Clamp01(axes[leftTrigger+int(b-GamepadLeftTrigger)])
	case b >= GamepadLeftStickLeft && b < gamepadButtonCount:
		s, dir := LeftStick, b-GamepadLeftStickLeft
		if b >= GamepadRightStickLeft {
			s, dir = RightStick, b-GamepadRightStickLeft
		}
		v := g.stick(s, before)
		return max([...]float32{-v.X, v.X, -v.Y, v.Y}[dir], 0)
	}
	return 0
}

// Gamepads is the connected pads, by ID.
func (s *State) Gamepads() []*Gamepad {
	var out []*Gamepad
	for i := range s.pads {
		if s.pads[i].connected {
			out = append(out, &s.pads[i])
		}
	}
	return out
}

// Gamepad is the pad with id, idle when none is connected there.
func (s *State) Gamepad(id int) *Gamepad {
	if id < 0 || id >= MaxGamepads {
		return &Gamepad{id: id}
	}
	return &s.pads[id]
}

// Stick is where s leans on whichever connected pad leans it most.
func (s *State) Stick(st Stick) geom.Vec2 {
	var best geom.Vec2
	for i := range s.pads {
		if v := s.pads[i].Stick(st); v.Len() > best.Len() {
			best = v
		}
	}
	return best
}

// Deadzone is the stick radius below which a stick reads as centred, 0.2
// unless SetDeadzone changed it.
func (s *State) Deadzone() float32 {
	if s.deadzoneSet {
		return s.deadzone
	}
	return defaultDeadzone
}

// SetDeadzone sets the radius below which sticks read as centred, from
// the next frame.
func (s *State) SetDeadzone(r float32) { s.deadzone, s.deadzoneSet = geom.Clamp(r, 0, 0.99), true }

// GamepadConnected is emitted when a pad connects, as its first frame
// begins.
type GamepadConnected struct {
	ID   int
	Name string
}

// GamepadDisconnected is emitted when a pad goes, as the frame without it
// begins.
type GamepadDisconnected struct{ ID int }

// ConnectGamepad plugs a pad called name into slot id.
func (s *State) ConnectGamepad(id int, name string) {
	if p := s.livePad(id); p != nil {
		p.connected, p.name = true, name
	}
}

// DisconnectGamepad unplugs the pad in slot id.
func (s *State) DisconnectGamepad(id int) {
	if p := s.livePad(id); p != nil {
		*p = Gamepad{}
	}
}

// SetGamepadButton holds b down on pad id or lets it go: a trigger fully
// pulled, a stick direction leant all the way.
func (s *State) SetGamepadButton(id int, b GamepadButton, down bool) {
	p := s.livePad(id)
	if p == nil {
		return
	}
	v := float32(0)
	if down {
		v = 1
	}
	switch {
	case b >= 0 && int(b) < digitalCount:
		p.buttons[b] = down
	case b == GamepadLeftTrigger || b == GamepadRightTrigger:
		p.axes[leftTrigger+int(b-GamepadLeftTrigger)] = v
	case b >= GamepadLeftStickLeft && b < gamepadButtonCount:
		x, dir := leftX, int(b-GamepadLeftStickLeft)
		if b >= GamepadRightStickLeft {
			x, dir = rightX, int(b-GamepadRightStickLeft)
		}
		p.axes[x+dir/2] = float32(dir%2*2-1) * v
	}
}

// SetGamepadStick leans stick st of pad id to v, each axis in [-1, 1],
// Y down.
func (s *State) SetGamepadStick(id int, st Stick, v geom.Vec2) {
	if p := s.livePad(id); p != nil {
		x := leftX
		if st == RightStick {
			x = rightX
		}
		p.axes[x], p.axes[x+1] = geom.Clamp(v.X, -1, 1), geom.Clamp(v.Y, -1, 1)
	}
}

// SetGamepadTrigger pulls trigger, GamepadLeftTrigger or
// GamepadRightTrigger, of pad id by pull, from 0 to 1.
func (s *State) SetGamepadTrigger(id int, trigger GamepadButton, pull float32) {
	if p := s.livePad(id); p != nil && (trigger == GamepadLeftTrigger || trigger == GamepadRightTrigger) {
		p.axes[leftTrigger+int(trigger-GamepadLeftTrigger)] = geom.Clamp01(pull)
	}
}

func (s *State) livePad(id int) *Gamepad {
	if id < 0 || id >= MaxGamepads {
		return nil
	}
	return &s.livePads[id]
}

// beginPads publishes the pads set since the last frame and notes which
// connected and disconnected.
func (s *State) beginPads() {
	s.joined, s.left = s.joined[:0], s.left[:0]
	for i := range s.pads {
		old, live := &s.pads[i], s.livePads[i]
		switch {
		case live.connected && !old.connected:
			s.joined = append(s.joined, i)
		case !live.connected && old.connected:
			s.left = append(s.left, i)
		}
		prevButtons, prevAxes := old.buttons, old.axes
		*old = live
		old.id, old.prevButtons, old.prevAxes, old.deadzone, old.state = i, prevButtons, prevAxes, s.Deadzone(), s
	}
}
