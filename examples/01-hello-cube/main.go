// Hello cube, the classic RGB cube: each corner coloured by its position, black at (0,0,0)
// and white at (1,1,1), blended across the faces, unlit and spinning.
//
//	go run ./examples/01-hello-cube
package main

import (
	"github.com/lantervale/ember"
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/mesh"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/render"
	"github.com/lantervale/ember/scene"
	"github.com/lantervale/ember/world"
)

type Cube struct {
	*world.BasicEntity
	scene.Transform
	render.Model
}

type Eye struct {
	*world.BasicEntity
	scene.Transform
	render.Camera
}

func rgbCube() *mesh.Data {
	m := mesh.Cube(1)
	for i, v := range m.Vertices {
		p := v.Pos.Add(geom.V3(0.5, 0.5, 0.5))
		m.Vertices[i].Color = color.RGB(p.X, p.Y, p.Z)
	}
	return m
}

func main() {
	helloCube().MustRun()
}

func helloCube() *world.World {
	cube := &Cube{
		world.NewBasic(),
		scene.At(0, 0, 0),
		render.Model{Mesh: rgbCube(), Material: &render.Material{Albedo: color.White, Unlit: true}},
	}

	win := platform.NewWindow(
		platform.Title("ember · hello cube"),
		platform.Size(800, 600),
		platform.Background(color.Hex("#202028")),
	)

	return ember.New().
		AddEntities(
			win,
			cube,
			&Eye{world.NewBasic(), scene.At(0, 0, 3).LookingAt(geom.V3(0, 0, 0)), render.Camera{}},
		).
		On(world.Update, "spin", onUpdate(cube))
}

func onUpdate(cube *Cube) func(world.Frame) {
	return func(f world.Frame) {
		cube.Rotation = geom.QuatEuler(0.4*f.DT, 0.7*f.DT, 0).Mul(cube.Rotation)
	}
}
