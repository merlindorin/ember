package main

import (
	"fmt"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/twod"
)

func shell() color.Color { return color.Hex("#2f3542") }
func idle() color.Color  { return color.Hex("#4b5366") }
func lit() color.Color   { return color.Hex("#ffb547") }
func label() color.Color { return color.Hex("#c9d1e0") }

// PadView draws the pad in slot id as it is held, from the window's input,
// every frame; it draws nothing while no pad is in its slot, but for the
// first, which asks for one.
type PadView struct {
	twod.Item
	id int
}

func newPadView(id int, opts ...twod.Option) *PadView {
	return twod.Build(&PadView{Item: twod.Base(), id: id}, fmt.Sprintf("pad-%d", id), opts...)
}

func (v *PadView) Draw(c *twod.Canvas) {
	pad := c.Input().Gamepad(v.id)
	if !pad.Connected() {
		if v.id == 0 {
			c.Text(geom.V2(-150, -10), "Plug in a pad", twod.Color(label()))
		}
		return
	}
	c.Text(
		geom.V2(-150, -118),
		fmt.Sprintf("Pad %d · %s", pad.ID(), pad.Name()),
		twod.FontSize(14),
		twod.Color(label()),
	)

	trigger := func(x float32, b input.GamepadButton) {
		box := geom.Rect{X: x, Y: -96, W: 44, H: 14}
		c.Rect(box, idle())
		c.Rect(geom.Rect{X: x, Y: -96, W: 44 * pad.Value(b), H: 14}, lit())
	}
	trigger(-130, input.GamepadLeftTrigger)
	trigger(86, input.GamepadRightTrigger)

	button := func(r geom.Rect, b input.GamepadButton) {
		c.RoundRect(r, 4, glow(pad, b))
	}
	button(geom.Rect{X: -130, Y: -76, W: 44, H: 10}, input.GamepadLeftShoulder)
	button(geom.Rect{X: 86, Y: -76, W: 44, H: 10}, input.GamepadRightShoulder)

	c.RoundRect(geom.Rect{X: -150, Y: -62, W: 300, H: 130}, 48, shell())

	button(geom.Rect{X: -93, Y: -42, W: 16, H: 20}, input.GamepadUp)
	button(geom.Rect{X: -93, Y: -6, W: 16, H: 20}, input.GamepadDown)
	button(geom.Rect{X: -113, Y: -22, W: 20, H: 16}, input.GamepadLeft)
	button(geom.Rect{X: -77, Y: -22, W: 20, H: 16}, input.GamepadRight)

	face := []struct {
		button input.GamepadButton
		at     geom.Vec2
	}{
		{input.GamepadNorth, geom.V2(85, -36)}, {input.GamepadSouth, geom.V2(85, 4)},
		{input.GamepadWest, geom.V2(65, -16)}, {input.GamepadEast, geom.V2(105, -16)},
	}
	for _, f := range face {
		c.Circle(f.at, 9, glow(pad, f.button))
	}
	c.Circle(geom.V2(-24, -34), 5, glow(pad, input.GamepadBack))
	c.Circle(geom.V2(24, -34), 5, glow(pad, input.GamepadStart))
	c.Circle(geom.V2(0, -12), 8, glow(pad, input.GamepadGuide))

	stick := func(at geom.Vec2, s input.Stick, press input.GamepadButton) {
		c.Ring(at, 22, 2, idle())
		c.Circle(at.Add(pad.Stick(s).Scale(14)), 10, glow(pad, press))
	}
	stick(geom.V2(-42, 34), input.LeftStick, input.GamepadLeftStickPress)
	stick(geom.V2(42, 34), input.RightStick, input.GamepadRightStickPress)
}

func glow(pad *input.Gamepad, b input.GamepadButton) color.Color {
	if pad.Down(b) {
		return lit()
	}
	return idle()
}
