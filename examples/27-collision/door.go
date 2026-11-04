package main

import (
	"github.com/merlindorin/ember/collide"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/world"
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
