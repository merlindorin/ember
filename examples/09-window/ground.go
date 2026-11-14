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
		render.Model{
			Mesh:     mesh.Box(geom.V3(12, 0.2, 12)),
			Material: render.Mat(color.Hex("#7d828c")),
		},
	}
}
