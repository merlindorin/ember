// Package scene provides the spatial backbone: the Transform component and
// the system that resolves world matrices through the BasicEntity hierarchy.
//
//	type Crate struct {
//		*world.BasicEntity
//		scene.Transform
//		render.Model
//	}
//
// Transform is the authored, local pose. Its world matrix is derived every
// frame in PostUpdate and is what renderers, physics and networking read.
package scene

import (
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/world"
)

type Transform struct {
	Position geom.Vec3 `node:"prop"`
	Rotation geom.Quat `node:"prop"`
	Scale    geom.Vec3 `node:"prop"`

	world    geom.Mat4
	resolved bool
}

// Transformable is any entity with a Transform.
type Transformable interface {
	world.BasicFace
	GetTransform() *Transform
}

func (t *Transform) GetTransform() *Transform { return t }

// At returns an identity transform positioned at (x, y, z).
func At(x, y, z float32) Transform { return AtVec(geom.V3(x, y, z)) }

func AtVec(p geom.Vec3) Transform {
	return Transform{Position: p, Rotation: geom.QuatIdent(), Scale: geom.One3}
}

func (t Transform) WithScale(s float32) Transform { t.Scale = geom.V3(s, s, s); return t }

func (t Transform) WithScaleVec(s geom.Vec3) Transform { t.Scale = s; return t }

func (t Transform) WithRotation(q geom.Quat) Transform { t.Rotation = q; return t }

func (t Transform) WithYaw(yaw float32) Transform { t.Rotation = geom.QuatYaw(yaw); return t }

// LookingAt orients the transform so that -Z faces target.
func (t Transform) LookingAt(target geom.Vec3) Transform {
	t.Rotation = geom.QuatLookRotation(target.Sub(t.Position), geom.Up)
	return t
}

// Local returns the local TRS matrix.
func (t *Transform) Local() geom.Mat4 { return geom.TRS(t.Position, t.Rotation, t.Scale) }

// World returns the world matrix resolved during the last PostUpdate, or the
// local matrix if the entity has not been resolved yet.
func (t *Transform) World() geom.Mat4 {
	if !t.resolved {
		return t.Local()
	}
	return t.world
}

func (t *Transform) WorldPosition() geom.Vec3 { return t.World().Translation() }

func (t *Transform) Forward() geom.Vec3 { return t.Rotation.Rotate(geom.Forward) }

func (t *Transform) Right() geom.Vec3 { return t.Rotation.Rotate(geom.Right) }

// Yaw returns the heading around the up axis.
func (t *Transform) Yaw() float32 {
	f := t.Forward()
	return geom.Atan2(-f.X, -f.Z)
}

func (t *Transform) resolve(parent *geom.Mat4) geom.Mat4 {
	if parent == nil {
		t.world = t.Local()
	} else {
		t.world = parent.Mul(t.Local())
	}
	t.resolved = true
	return t.world
}

// TransformSystem resolves world matrices for every Transformable, walking
// BasicEntity children so that parented entities follow their parents. An
// entity whose parent has no Transform is top level.
type TransformSystem struct {
	world.Group[Transformable]
}

func (s *TransformSystem) Phase() world.Phase { return world.PostUpdate }

// Priority runs it first in PostUpdate so cameras and followers read fresh
// matrices.
func (s *TransformSystem) Priority() int { return 1000 }

func (s *TransformSystem) Update(world.Frame) {
	for _, e := range s.Slice() {
		if p := e.GetBasicEntity().Parent(); p == nil {
			s.propagate(e, nil)
		} else if _, ok := s.Get(p.ID()); !ok {
			s.propagate(e, nil)
		}
	}
}

func (s *TransformSystem) propagate(e Transformable, parent *geom.Mat4) {
	m := e.GetTransform().resolve(parent)
	for _, c := range e.GetBasicEntity().Children() {
		if child, ok := s.Get(c.ID()); ok {
			s.propagate(child, &m)
		}
	}
}

// Attach parents child under parent and adds it to the world.
func Attach(w *world.World, parent world.BasicFace, child world.BasicFace) {
	parent.GetBasicEntity().AppendChild(child.GetBasicEntity())
	w.AddEntity(child)
}
