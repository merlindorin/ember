package net

import (
	"fmt"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/scene"
)

// placed2D is a 2D entity, as a twod.Item is: its placement in its parent,
// and where it is in its layer. net can't name twod's types, which
// reach net through automation and node3d.
type placed2D interface {
	Position() geom.Vec2
	SetPosition(at geom.Vec2)
	Rotation() float32
	SetRotation(r float32)
	Scale() geom.Vec2
	SetScale(s geom.Vec2)
	GlobalPosition() geom.Vec2
}

// A 2D pose travels as a 3D one on the ground plane, the plane interest is
// measured on: (x, y) is (x, 0, y), the rotation a yaw, the scale (sx, 1,
// sy). A scene.Transform is used first when an entity has both.

func poseOf(e Replicated) Pose {
	switch e := e.(type) {
	case scene.Transformable:
		t := e.GetTransform()
		return Pose{t.Position, t.Rotation, t.Scale}
	case placed2D:
		at, s := e.Position(), e.Scale()
		return Pose{geom.V3(at.X, 0, at.Y), geom.QuatYaw(e.Rotation()), geom.V3(s.X, 1, s.Y)}
	}
	panic(posless(e))
}

func setPose(e Replicated, p Pose) {
	switch e := e.(type) {
	case scene.Transformable:
		t := e.GetTransform()
		t.Position, t.Rotation, t.Scale = p.Position, p.Rotation, p.Scale
	case placed2D:
		e.SetPosition(geom.V2(p.Position.X, p.Position.Z))
		e.SetRotation(2 * geom.Atan2(p.Rotation.Y, p.Rotation.W))
		e.SetScale(geom.V2(p.Scale.X, p.Scale.Z))
	default:
		panic(posless(e))
	}
}

// groundOf is where e is in the world, on the plane interest is measured
// on.
func groundOf(e Replicated) geom.Vec3 {
	switch e := e.(type) {
	case scene.Transformable:
		return e.GetTransform().WorldPosition()
	case placed2D:
		at := e.GlobalPosition()
		return geom.V3(at.X, 0, at.Y)
	}
	panic(posless(e))
}

func mustPose(e Replicated) {
	switch e.(type) {
	case scene.Transformable, placed2D:
	default:
		panic(posless(e))
	}
}

func posless(e Replicated) string {
	return fmt.Sprintf("net: %q (%T) has neither a scene.Transform nor a twod placement to replicate",
		e.GetNetwork().Kind, e)
}
