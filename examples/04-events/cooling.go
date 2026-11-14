package main

import "github.com/lantervale/ember/world"

type Burning interface {
	world.BasicFace
	GetHeat() *Heat
	GetName() *Name
}

// WentOut is an event: Cooling emits it, and any system can read it.
type WentOut struct{ Name string }

type Cooling struct{ world.Group[Burning] }

func (s *Cooling) Update(f world.Frame) {
	for flame := range s.All() {
		flame.GetHeat().Degrees -= f.DT

		if flame.GetHeat().Degrees <= 0 {
			world.Emit(f.World, WentOut{Name: flame.GetName().Value})
			f.World.RemoveEntity(flame)
		}
	}
}
