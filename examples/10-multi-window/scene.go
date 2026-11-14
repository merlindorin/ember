package main

import (
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/mesh"
	"github.com/lantervale/ember/render"
	"github.com/lantervale/ember/scene"
	"github.com/lantervale/ember/world"
)

type Prop struct {
	*world.BasicEntity
	scene.Transform
	render.Model
}

func newGround() *Prop {
	return &Prop{
		world.NewBasic(),
		scene.At(0, -0.1, 0),
		render.Model{Mesh: mesh.Box(geom.V3(12, 0.2, 12)), Material: render.Mat(color.Hex("#7d828c"))},
	}
}

func newCrate(x, z float32) *Prop {
	return &Prop{
		world.NewBasic(),
		scene.At(x, 0.6, z),
		render.Model{Mesh: mesh.Cube(1.2), Material: render.Mat(color.Hex("#b5763a"))},
	}
}

// Eye is a fixed camera. Its Camera.Window picks the window it draws in,
// the primary one when nil.
type Eye struct {
	*world.BasicEntity
	scene.Transform
	render.Camera
}
