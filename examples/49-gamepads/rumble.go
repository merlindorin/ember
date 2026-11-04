package main

import (
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/world"
)

// Rumble shakes the pads. South gives a 0.3 s jolt, the heavy motor full
// and the light one at 0.4. The triggers drive the motors live, each alone:
// the left trigger the heavy left motor, the right trigger the light right
// one, as hard as they are pulled. Letting both go stops the pad.
type Rumble struct {
	in      *input.State
	pulling [input.MaxGamepads]bool
}

func (r *Rumble) Update(world.Frame) {
	for _, pad := range r.in.Gamepads() {
		low, high := pad.Value(input.GamepadLeftTrigger), pad.Value(input.GamepadRightTrigger)
		switch {
		case pad.Pressed(input.GamepadSouth):
			pad.Rumble(1, 0.4, 0.3)
		case low > 0 || high > 0:
			pad.Rumble(low, high, 0.1)
			r.pulling[pad.ID()] = true
		case r.pulling[pad.ID()]:
			pad.StopRumble()
			r.pulling[pad.ID()] = false
		}
	}
}
