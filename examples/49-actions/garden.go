package main

import (
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/world"
)

// garden is the trees planted so far and the ghost of the next one.
type garden struct {
	tree  twod.Frame
	tiles *twod.Texture
	trees *twod.Layer
	ghost *twod.Sprite
}

func newGarden(tiles *twod.Texture) *garden {
	tree := twod.Strip(tiles, twod.Cell(16, 16))[4]
	return &garden{
		tree:  tree,
		tiles: tiles,
		trees: twod.NewLayer(0),
		ghost: twod.NewSprite(
			tiles,
			twod.Still(tree),
			twod.Scale(3, 3),
			twod.Pivot(0.5, 1),
			twod.Tint(color.Hex("#ffffff80")),
		),
	}
}

// Plant keeps the ghost tree on the world under the pointer, plants a tree
// there on the plant action, and zooms the camera with the wheel.
type Plant struct {
	garden   *garden
	eye      *twod.Camera
	in       *input.State
	controls *input.Actions[Act]
}

func (p *Plant) Update(world.Frame) {
	at := p.eye.ScreenToWorld(p.in.Mouse())
	p.garden.ghost.SetPosition(at)
	if p.controls.Pressed(p.in, PlantTree) {
		p.garden.trees.Add(
			twod.NewSprite(
				p.garden.tiles,
				twod.Still(p.garden.tree),
				twod.At(at.X, at.Y),
				twod.Scale(3, 3),
				twod.Pivot(0.5, 1),
			),
		)
	}
	if wheel := p.in.Wheel().Y; wheel != 0 {
		p.eye.SetZoom(min(max(p.eye.Zoom()*(1-0.1*wheel), 0.5), 3))
	}
}
