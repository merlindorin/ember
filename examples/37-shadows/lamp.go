package main

import (
	"fmt"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/twod/light"
	"github.com/lantervale/ember/world"
)

// newLamp is a warm lamp 100 points above the ground: what stands lower
// than that throws a shadow that ends, what stands higher one that runs to
// the edge of its light.
func newLamp() *light.Light {
	return light.NewPoint(260, color.RGB(1, 0.85, 0.6), light.Elevation(100))
}

// Lamp follows the mouse. Up and down, or the wheel, raise and lower it;
// S gives it a size of 12 points, softening its shadows away from what
// casts them; F steps through the curves its shadows fade by along their
// length: off, linear, smooth and soft.
type Lamp struct {
	lamp   *light.Light
	camera *twod.Camera
	in     *input.State
	label  *twod.Text
}

func (l *Lamp) Update(f world.Frame) {
	l.lamp.SetPosition(l.camera.ScreenToWorld(l.in.Mouse()))
	raise := l.in.Wheel().Y * 10
	if l.in.Down(input.KeyUp) {
		raise += 120 * f.DT
	}
	if l.in.Down(input.KeyDown) {
		raise -= 120 * f.DT
	}
	l.lamp.SetElevation(min(l.lamp.Elevation()+raise, 300))
	if l.in.Pressed(input.KeyS) {
		l.lamp.SetSize(12 - l.lamp.Size())
	}
	if l.in.Pressed(input.KeyF) {
		l.lamp.SetShadowFade(next(l.lamp.ShadowFade()))
	}
	l.label.SetText(
		fmt.Sprintf("elevation %.0f   size %.0f   fade %s", l.lamp.Elevation(), l.lamp.Size(), l.lamp.ShadowFade()),
	)
}

// next is the fade curve after f, back to none after the last.
func next(f light.Fade) light.Fade {
	switch f {
	case light.FadeNone:
		return light.FadeLinear
	case light.FadeLinear:
		return light.FadeSmooth
	case light.FadeSmooth:
		return light.FadeSoft
	case light.FadeSoft:
		return light.FadeNone
	}
	return light.FadeNone
}
