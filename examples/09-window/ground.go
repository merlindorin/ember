package main

import (
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/mesh"
	"github.com/merlindorin/ember/render"
	"github.com/merlindorin/ember/scene"
	"github.com/merlindorin/ember/world"
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
