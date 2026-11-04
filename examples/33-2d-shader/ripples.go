package main

import (
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/render"
	"github.com/merlindorin/ember/world"
)

const (
	maxRipples = 4
	lifetime   = 1.6 // seconds, as in the shader's fade
)

// Ripples starts a ripple where the window is clicked, ages the ripples,
// and hands them to the shader as uniforms every frame. With all four
// alive, a click replaces the oldest.
type Ripples struct {
	effect *render.Effect
	window *platform.Window
	centre [maxRipples]geom.Vec2
	age    [maxRipples]float32
	alive  [maxRipples]bool
	next   int
}

func (r *Ripples) Update(f world.Frame) {
	in := r.window.Input()
	if in.Pressed(input.MouseLeft) {
		size := r.window.Size()
		r.centre[r.next] = geom.V2(in.Mouse().X/size.X, in.Mouse().Y/size.Y)
		r.age[r.next], r.alive[r.next] = 0, true
		r.next = (r.next + 1) % maxRipples
	}

	ripples := make([]geom.Vec4, maxRipples)
	for i := range maxRipples {
		if r.alive[i] {
			r.age[i] += f.DT
			r.alive[i] = r.age[i] < lifetime
		}
		live := float32(0)
		if r.alive[i] {
			live = 1
		}
		ripples[i] = geom.V4(r.centre[i].X, r.centre[i].Y, r.age[i], live)
	}
	r.effect.SetVec4s("uRipples", ripples)
}
