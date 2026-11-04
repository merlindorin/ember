package main

import (
	"math"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/twod/light"
	"github.com/merlindorin/ember/world"
)

// newTorch is the warm light the hero carries, reaching 150 points, at full
// strength over its inner fifth, held inside his body so it casts no shadow
// of him.
func newTorch() *light.Light {
	return light.NewPoint(150, color.RGB(1, 0.75, 0.45), light.Falloff(0.2), light.At(0, -4))
}

// Flicker makes the torch waver, as a flame does: its intensity follows two
// slow waves of different speeds. A light has no flicker of its own; it's a
// few lines of the game's.
type Flicker struct{ torch *light.Light }

func (f *Flicker) Update(fr world.Frame) {
	t := float64(fr.Elapsed)
	f.torch.SetIntensity(float32(0.9 + 0.06*math.Sin(t*7) + 0.04*math.Sin(t*17)))
}
