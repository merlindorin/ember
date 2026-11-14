// Package anim is skeletal animation as plain data: skeletons, clips
// sampled into poses, crossfades, and CPU skinning of meshes. It has no GPU
// dependency, so servers can evaluate the same poses (hitboxes, attachment
// points) as clients.
//
//	type Character struct {
//		*world.BasicEntity
//		scene.Transform
//		anim.Animator
//	}
//	hero.Play("Run", 0.25)
package anim

import (
	"github.com/lantervale/ember/geom"
)

// TRS is a joint's local transform.
type TRS struct {
	T geom.Vec3
	R geom.Quat
	S geom.Vec3
}

func (t TRS) Mat4() geom.Mat4 { return geom.TRS(t.T, t.R, t.S) }

// Lerp interpolates translation and scale linearly and rotation spherically.
func (t TRS) Lerp(o TRS, k float32) TRS {
	return TRS{T: t.T.Lerp(o.T, k), R: t.R.Slerp(o.R, k), S: t.S.Lerp(o.S, k)}
}

// Joint is a node of a skeleton. Parent is -1 for roots.
type Joint struct {
	Name   string
	Parent int
	Rest   TRS
}

// Skeleton is a joint hierarchy where every parent comes before its
// children.
type Skeleton struct {
	Joints []Joint
	index  map[string]int
}

// Find returns the index of a joint by name, or -1.
func (s *Skeleton) Find(name string) int {
	if s.index == nil {
		s.index = make(map[string]int, len(s.Joints))
		for i, j := range s.Joints {
			if _, dup := s.index[j.Name]; !dup {
				s.index[j.Name] = i
			}
		}
	}
	if i, ok := s.index[name]; ok {
		return i
	}
	return -1
}

// Pose holds one local transform per joint.
type Pose []TRS

// RestPose returns a new pose with every joint at rest.
func (s *Skeleton) RestPose() Pose {
	p := make(Pose, len(s.Joints))
	s.Reset(p)
	return p
}

// Reset puts every joint of p back at rest.
func (s *Skeleton) Reset(p Pose) {
	for i, j := range s.Joints {
		p[i] = j.Rest
	}
}

// Globals resolves a pose into model-space joint matrices.
func (s *Skeleton) Globals(p Pose, out []geom.Mat4) []geom.Mat4 {
	out = resize(out, len(s.Joints))
	for i, j := range s.Joints {
		m := p[i].Mat4()
		if j.Parent >= 0 {
			m = out[j.Parent].Mul(m)
		}
		out[i] = m
	}
	return out
}

// Blend writes the mix of a and b (weight w on b) into out.
func Blend(a, b Pose, w float32, out Pose) {
	for i := range out {
		out[i] = a[i].Lerp(b[i], w)
	}
}

func resize[T any](s []T, n int) []T {
	if cap(s) < n {
		return make([]T, n)
	}
	return s[:n]
}
