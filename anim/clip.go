package anim

import (
	"sort"

	"github.com/merlindorin/ember/geom"
)

type Path int

const (
	Translation Path = iota
	Rotation
	Scale
)

type Interpolation int

const (
	Linear Interpolation = iota
	Step
	// CubicSpline stores in-tangent, value, out-tangent per key (glTF).
	CubicSpline
)

// Channel animates one property of one joint. Values hold vec3s in XYZ or
// quaternions in XYZW.
type Channel struct {
	Joint         int
	Path          Path
	Interpolation Interpolation
	Times         []float32
	Values        []geom.Vec4
}

// Clip is a named animation. Its keys span [Start, Duration]: exporters
// often begin at the first frame's time rather than 0, and a loop plays
// that span, so the last key flows into the first.
type Clip struct {
	Name     string
	Start    float32
	Duration float32
	Channels []Channel
}

// Length is the playable span of the clip.
func (c *Clip) Length() float32 { return c.Duration - c.Start }

// Phase returns how far t is through the clip, 0..1.
func (c *Clip) Phase(t float32) float32 {
	if l := c.Length(); l > 0 {
		return geom.Clamp01((t - c.Start) / l)
	}
	return 0
}

// Sample overwrites the joints the clip animates with their value at t.
func (c *Clip) Sample(t float32, p Pose) {
	for i := range c.Channels {
		ch := &c.Channels[i]
		if ch.Joint < 0 || ch.Joint >= len(p) || len(ch.Times) == 0 {
			continue
		}
		v := ch.at(t)
		switch ch.Path {
		case Translation:
			p[ch.Joint].T = v.Vec3()
		case Rotation:
			p[ch.Joint].R = geom.Quat(v).Normalize()
		case Scale:
			p[ch.Joint].S = v.Vec3()
		}
	}
}

func (ch *Channel) value(i int) geom.Vec4 {
	if ch.Interpolation == CubicSpline {
		return ch.Values[i*3+1]
	}
	return ch.Values[i]
}

func (ch *Channel) at(t float32) geom.Vec4 {
	n := len(ch.Times)
	if t <= ch.Times[0] || n == 1 {
		return ch.value(0)
	}
	if t >= ch.Times[n-1] {
		return ch.value(n - 1)
	}
	i := sort.Search(n, func(i int) bool { return ch.Times[i] > t }) - 1
	t0, t1 := ch.Times[i], ch.Times[i+1]
	k := (t - t0) / (t1 - t0)
	a, b := ch.value(i), ch.value(i+1)
	switch ch.Interpolation {
	case Linear: // the code after the switch
	case Step:
		return a
	case CubicSpline:
		dt := t1 - t0
		m0, m1 := ch.Values[i*3+2].Scale(dt), ch.Values[(i+1)*3].Scale(dt)
		k2, k3 := k*k, k*k*k
		return a.Scale(2*k3 - 3*k2 + 1).Add(m0.Scale(k3 - 2*k2 + k)).Add(b.Scale(-2*k3 + 3*k2)).Add(m1.Scale(k3 - k2))
	}
	if ch.Path == Rotation {
		q := geom.Quat(a).Slerp(geom.Quat(b), k)
		return geom.V4(q.X, q.Y, q.Z, q.W)
	}
	return a.Scale(1 - k).Add(b.Scale(k))
}
