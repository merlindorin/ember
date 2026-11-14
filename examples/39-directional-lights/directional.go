package main

import (
	"math"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/twod/light"
	"github.com/lantervale/ember/world"
)

func moonlight() color.Color { return color.RGB(0.08, 0.09, 0.16) }
func daylight() color.Color  { return color.RGB(0.55, 0.55, 0.62) }

// A day lasts 12 seconds and the night after it 4.
const (
	dayLength   = 12
	nightLength = 4
)

// newSun is a warm light from far away: everything solid throws a shadow
// away from it.
func newSun() *light.Light {
	return light.NewDirectional(color.RGB(1, 0.9, 0.75), 0)
}

// Day runs the clock from dawn to dawn: the sun crosses the sky, brightest
// and with the shortest shadows at noon, and the ambient light goes from
// moonlight to daylight and back. At night the sun is hidden. Space stops
// the clock where it is, and starts it again.
type Day struct {
	sun    *light.Light
	camera *twod.Camera
	in     *input.State
	t      float32
	paused bool
}

func (d *Day) Update(f world.Frame) {
	if d.in != nil && d.in.Pressed(input.KeySpace) {
		d.paused = !d.paused
	}
	if d.paused {
		return
	}
	d.t = float32(math.Mod(float64(d.t+f.DT), dayLength+nightLength))
	day := d.t / dayLength
	d.sun.SetHidden(day >= 1)
	if day >= 1 {
		light.SetAmbient(d.camera, moonlight())
		return
	}
	high := float32(math.Sin(math.Pi * float64(day)))
	d.sun.SetAngle(math.Pi * (0.15 + 0.7*day))
	d.sun.SetIntensity(high)
	d.sun.SetShadowLength(60 + 260*(1-high))
	light.SetAmbient(d.camera, moonlight().Lerp(daylight(), high))
}
