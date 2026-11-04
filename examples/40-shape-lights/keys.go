package main

import (
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/twod/light"
	"github.com/merlindorin/ember/world"
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
