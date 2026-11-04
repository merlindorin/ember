package main

import (
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/twod/light"
	"github.com/merlindorin/ember/world"
)

// Keys tries the torch's options: R steps its radius through 100, 150 and
// 220 points, X makes it shine through everything, or not.
type Keys struct {
	torch *light.Light
	in    *input.State
}

func (k *Keys) Update(world.Frame) {
	if k.in.Pressed(input.KeyR) {
		k.torch.SetRadius(nextRadius(k.torch.Radius()))
	}
	if k.in.Pressed(input.KeyX) {
		k.torch.SetShadows(!k.torch.Shadows())
	}
}

// nextRadius is the radius after r: 100, 150, 220, then 100 again.
func nextRadius(r float32) float32 {
	switch {
	case r < 150:
		return 150
	case r < 220:
		return 220
	}
	return 100
}
