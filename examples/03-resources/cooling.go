package main

import (
	"fmt"
	"os"

	"github.com/lantervale/ember/world"
)

type Burning interface {
	world.BasicFace
	GetHeat() *Heat
	GetName() *Name
}

// Cooling takes from every burning entity as many degrees per second as
// the wind blows.
type Cooling struct{ world.Group[Burning] }

func (s *Cooling) Update(f world.Frame) {
	wind := world.Resource[Weather](f.World).Wind
	for flame := range s.All() {
		flame.GetHeat().Degrees -= wind * f.DT
		if flame.GetHeat().Degrees <= 0 {
			fmt.Fprintf(os.Stdout, "%.1fs  %s went out in a wind of %.0f\n", f.Elapsed, flame.GetName().Value, wind)
			f.World.RemoveEntity(flame)
		}
	}
}
