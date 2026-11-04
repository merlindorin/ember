package main

import "github.com/merlindorin/ember/world"

// Weather is a resource: one value in the World, shared by every system.
type Weather struct{ Wind float32 }

// Gusts raises the wind by two every second.
type Gusts struct{}

func (s *Gusts) Update(f world.Frame) {
	world.Resource[Weather](f.World).Wind += 2 * f.DT
}
