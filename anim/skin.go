package anim

import (
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/mesh"
)

// Influence binds a vertex to up to four joints of a Skin.
type Influence struct {
	Joints  [4]uint16
	Weights [4]float32
}

// Skin maps the joint slots used by influences to skeleton joints.
type Skin struct {
	Joints      []int
	InverseBind []geom.Mat4
}

// Palette returns the skinning matrix of every slot for model-space joint
// matrices.
func (s *Skin) Palette(globals []geom.Mat4, out []geom.Mat4) []geom.Mat4 {
	out = resize(out, len(s.Joints))
	for i, j := range s.Joints {
		out[i] = globals[j].Mul(s.InverseBind[i])
	}
	return out
}

// Deform writes bind posed through palette into out, which must have the
// same vertex count. Normals follow the rotation part of each matrix.
func Deform(bind *mesh.Data, influences []Influence, palette []geom.Mat4, out *mesh.Data) {
	for i, v := range bind.Vertices {
		inf := influences[i]
		var m geom.Mat4
		total := float32(0)
		for k := range 4 {
			w := inf.Weights[k]
			if w == 0 || int(inf.Joints[k]) >= len(palette) {
				continue
			}
			p := &palette[inf.Joints[k]]
			for c := range m {
				m[c] += p[c] * w
			}
			total += w
		}
		o := &out.Vertices[i]
		if total == 0 {
			o.Pos, o.Normal = v.Pos, v.Normal
			continue
		}
		if total != 1 {
			for c := range m {
				m[c] /= total
			}
		}
		o.Pos = m.MulPoint(v.Pos)
		o.Normal = m.MulDir(v.Normal).Normalize()
	}
}
