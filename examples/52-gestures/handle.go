package main

import (
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/world"
)

// home is where the crate starts and goes back to.
func home() geom.Vec2 {
	return geom.V2(320, 200)
}

// panSpeed is how many points the crate moves when two fingers cross the
// whole trackpad.
const panSpeed = 640

// Handle zooms the crate by the pinch, turns it by the rotation and moves
// it by the pan, between a quarter and four times its size.
type Handle struct {
	crate *twod.Sprite
	in    *input.State
}

func (h *Handle) Update(world.Frame) {
	if h.in.Swipe() != (geom.Vec2{}) || h.in.Pressed(input.KeyR) {
		h.crate.SetPosition(home())
		h.crate.SetRotation(0)
		h.crate.SetScale(geom.V2(8, 8))
		return
	}
	scale := geom.Clamp(h.crate.Scale().X*(1+h.in.Pinch()), 2, 32)
	h.crate.SetScale(geom.V2(scale, scale))
	h.crate.SetRotation(h.crate.Rotation() + h.in.Rotation())
	h.crate.Translate(h.in.Pan().Scale(panSpeed))
}
