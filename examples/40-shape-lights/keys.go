package main

import (
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/twod/light"
	"github.com/lantervale/ember/world"
)

// Keys tries the crystal's shadows: X makes its light shine through the
// trees and pillars, or be blocked by them again.
type Keys struct {
	crystal *light.Light
	in      *input.State
}

func (k *Keys) Update(world.Frame) {
	if k.in.Pressed(input.KeyX) {
		k.crystal.SetShadows(!k.crystal.Shadows())
	}
}
