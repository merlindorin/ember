package main

import (
	"github.com/lantervale/ember/collide"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/world"
)

// Door opens the middle wall when something steps on the plate: the wall
// tile becomes floor, so it stops nothing any more.
type Door struct {
	walls   *twod.TileMap
	entered world.Reader[collide.BodyEntered]
}

func (d *Door) Update(f world.Frame) {
	for range d.entered.Read(f.World) {
		d.walls.Set(doorX, doorY, floor)
	}
}
