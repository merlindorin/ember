package gltf

import (
	"fmt"

	"github.com/lantervale/ember/anim"
	"github.com/lantervale/ember/render"
	"github.com/lantervale/ember/scene"
	"github.com/lantervale/ember/world"
)

// Instance is a spawned model: the root carries the transform and the
// Animator, and each part is a child entity drawing one primitive.
type Instance struct {
	*world.BasicEntity
	scene.Transform
	anim.Animator
	Model       *Model
	Parts       []*PartEntity
	Attachments []*Attachment
}

// Attachment is a socket following a joint of an instance, carrying
// another instance: a sword in a hand, a hat on a head.
type Attachment struct {
	*world.BasicEntity
	scene.Transform
	anim.Socket
	Item *Instance
}

// PartEntity draws one part of an instance. Skinned parts own a copy of
// the mesh that the skinning system rewrites every frame.
type PartEntity struct {
	*world.BasicEntity
	scene.Transform
	render.Model
	anim.Skinned
	Name string
	Part *Part
}

// Spawn adds an instance of the model to the world. Animated models need
// anim.Plugin to move.
func (m *Model) Spawn(w *world.World, t scene.Transform) *Instance {
	in := &Instance{BasicEntity: world.NewBasic(), Transform: t, Model: m}
	in.Animator = anim.Animator{Skeleton: m.Skeleton, Clips: m.Clips}
	w.AddEntity(in)
	for _, p := range m.Parts {
		e := &PartEntity{BasicEntity: world.NewBasic(), Transform: scene.At(0, 0, 0), Name: p.Name, Part: p}
		e.Model = render.Model{Mesh: p.Mesh, Material: p.Material}
		if p.Skin != nil {
			e.Skinned, e.Mesh = anim.NewSkinned(&in.Animator, p.Skin, p.Mesh, p.Influences)
		}
		w.AddEntity(e)
		scene.Attach(w, in, e)
		in.Parts = append(in.Parts, e)
	}
	in.Update(0)
	for _, p := range in.Parts {
		p.Apply(p.Mesh)
	}
	return in
}

// Part returns the spawned part with a name, or nil.
func (in *Instance) Part(name string) *PartEntity {
	for _, p := range in.Parts {
		if p.Name == name {
			return p
		}
	}
	return nil
}

// Attach spawns model in the hand (or any joint) of the instance, offset
// in the joint's space.
func (in *Instance) Attach(w *world.World, model *Model, joint string, offset anim.TRS) (*Attachment, error) {
	j := in.Skeleton.Find(joint)
	if j < 0 {
		return nil, fmt.Errorf("gltf: %s has no joint %q", in.Model.Name, joint)
	}
	at := &Attachment{
		BasicEntity: world.NewBasic(),
		Transform:   scene.At(0, 0, 0),
		Socket:      anim.Socket{Animator: &in.Animator, Joint: j, Offset: offset},
	}
	if m, ok := at.Matrix(); ok {
		at.Position, at.Rotation = m.Translation(), anim.RotationOf(m)
	}
	w.AddEntity(at)
	scene.Attach(w, in, at)
	at.Item = model.Spawn(w, scene.At(0, 0, 0))
	scene.Attach(w, at, at.Item)
	in.Attachments = append(in.Attachments, at)
	return at, nil
}

// Detach removes an attachment and what it carries.
func (in *Instance) Detach(w *world.World, at *Attachment) {
	for i, a := range in.Attachments {
		if a == at {
			in.Attachments = append(in.Attachments[:i], in.Attachments[i+1:]...)
			break
		}
	}
	at.Item.Remove(w)
	w.RemoveEntity(at)
}

// Remove takes the instance, its parts and attachments out of the world.
func (in *Instance) Remove(w *world.World) {
	for len(in.Attachments) > 0 {
		in.Detach(w, in.Attachments[0])
	}
	for _, p := range in.Parts {
		w.RemoveEntity(p)
	}
	w.RemoveEntity(in)
}
