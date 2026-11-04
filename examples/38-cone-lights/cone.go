package main

import (
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/twod/light"
	"github.com/merlindorin/ember/world"
)

// newFlashlight is a cone of cold white light 260 points long, opening 0.6
// radians each side of where it points, a lamp 12 points across whose
// shadows soften behind what casts them. It sits inside the hero's body,
// so his body casts no shadow.
func newFlashlight() *light.Light {
	return light.NewCone(260, color.RGB(0.9, 0.95, 1), 0, 0.6, light.At(0, -4), light.Size(12))
}

// Flashlight points the beam at the mouse. F switches it off and on, S
// steps its spread through 0.3, 0.6 and 1 radian, and K softens its
// shadows, or not.
type Flashlight struct {
	beam   *light.Light
	camera *twod.Camera
	in     *input.State
}

func (f *Flashlight) Update(world.Frame) {
	if f.in.Pressed(input.KeyF) {
		f.beam.SetHidden(!f.beam.Hidden())
	}
	if f.in.Pressed(input.KeyS) {
		f.beam.SetSpread(nextSpread(f.beam.Spread()))
	}
	if f.in.Pressed(input.KeyK) {
		f.beam.SetSize(12 - f.beam.Size())
	}
	toward := f.camera.ScreenToWorld(f.in.Mouse()).Sub(f.beam.GlobalPosition())
	f.beam.SetAngle(toward.Angle())
}

// nextSpread is the spread after s: 0.3, 0.6, 1, then 0.3 again.
func nextSpread(s float32) float32 {
	switch {
	case s < 0.6:
		return 0.6
	case s < 1:
		return 1
	}
	return 0.3
}
