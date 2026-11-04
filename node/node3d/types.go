// Package node3d holds the 3D node types and turns static scenes into
// node trees and back.
package node3d

import (
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/gltf"
	"github.com/merlindorin/ember/node"
	"github.com/merlindorin/ember/render"
	"github.com/merlindorin/ember/scene"
	"github.com/merlindorin/ember/world"
)

func init() { //nolint:gochecknoinits // registers the package's node types on import
	for _, t := range []node.Type{
		{New: func() node.Noded { return NewNode() }},
		{New: func() node.Noded { return NewMesh() }},
		{New: func() node.Noded { return NewCamera() }},
		{New: func() node.Noded { return NewPointLight() }},
		{New: func() node.Noded { return NewSun() }},
		{New: func() node.Noded { return NewMarker() }},
	} {
		node.Register(t)
	}
}

type Node struct {
	*world.BasicEntity
	node.Node
	scene.Transform
}

func NewNode() *Node { return &Node{BasicEntity: world.NewBasic(), Transform: scene.At(0, 0, 0)} }

// Mesh draws its surfaces, meshes in the node's space, each
// through a child entity.
type Mesh struct {
	Node
	Surfaces []*gltf.Part
	drawn    []*surface
}

type surface struct {
	*world.BasicEntity
	scene.Transform
	render.Model
}

func NewMesh() *Mesh { return &Mesh{Node: *NewNode()} }

func (m *Mesh) GetMesh() *Mesh { return m }

func (m *Mesh) EnterTree() {
	if m.drawn == nil {
		for _, p := range m.Surfaces {
			s := &surface{
				BasicEntity: world.NewBasic(),
				Transform:   scene.At(0, 0, 0),
				Model:       render.Model{Mesh: p.Mesh, Material: p.Material},
			}
			m.AppendChild(s.BasicEntity)
			m.drawn = append(m.drawn, s)
		}
	}
	for _, s := range m.drawn {
		m.Tree().World().AddEntity(s)
	}
}

type Camera struct {
	Node
	render.Camera
}

func NewCamera() *Camera { return &Camera{Node: *NewNode()} }

type PointLight struct {
	Node
	render.PointLight
}

func NewPointLight() *PointLight { return &PointLight{Node: *NewNode()} }

// Sun is the sun: it drives the render.Environment resource,
// shining along its -Z.
type Sun struct {
	Node
	Color  color.Color `node:"prop"`
	Energy float32     `node:"prop"`
}

func NewSun() *Sun {
	return &Sun{Node: *NewNode(), Color: color.White, Energy: 1}
}

func (d *Sun) GetSun() *Sun { return d }

func (d *Sun) Process(float32) {
	env := environment(d.Tree().World())
	if env == nil {
		return
	}
	env.SetSun(d.World().MulDir(geom.V3(0, 0, -1)).Normalize(), d.Color, d.Energy)
}

type Marker struct{ Node }

func NewMarker() *Marker { return &Marker{Node: *NewNode()} }
