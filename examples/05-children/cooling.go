package main

import (
	"fmt"
	"os"

	"github.com/lantervale/ember/world"
)

// Burning is what the Cooling system needs: any entity with Heat and a Name.
type Burning interface {
	world.BasicFace
	GetHeat() *Heat
	GetName() *Name
}

// Cooling takes one degree per second from every burning entity, and puts
// it out at zero.
type Cooling struct{ world.Group[Burning] }

func (s *Cooling) Update(f world.Frame) {
	for flame := range s.All() {
		flame.GetHeat().Degrees -= f.DT

		if flame.GetHeat().Degrees <= 0 {
			fmt.Fprintf(os.Stdout, "%.1fs  %s went out\n", f.Elapsed, flame.GetName().Value)
			f.World.RemoveEntity(flame)
		}
	}
}
