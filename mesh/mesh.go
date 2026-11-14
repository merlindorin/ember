// Package mesh builds CPU-side geometry. Meshes are plain data: generate,
// transform, tint and merge them freely; the renderer uploads them to the
// GPU lazily the first time they are drawn and re-uploads after Touch.
//
//	hero := mesh.Merge(
//		mesh.Capsule(0.35, 1.0, 12).Tint(armor).Translate(geom.V3(0, 0.9, 0)),
//		mesh.Sphere(0.25, 12, 8).Tint(skin).Translate(geom.V3(0, 1.75, 0)),
//	)
package mesh

import (
	"math"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
)

type Vertex struct {
	Pos    geom.Vec3
	Normal geom.Vec3
	UV     geom.Vec2
	Color  color.Color
}

type Data struct {
	Vertices []Vertex
	Indices  []uint32

	version uint64
}

// Version changes whenever Touch is called; GPU caches compare it.
func (d *Data) Version() uint64 { return d.version }

// Touch marks the mesh as modified so the renderer re-uploads it.
func (d *Data) Touch() { d.version++ }

func (d *Data) Triangles() int { return len(d.Indices) / 3 }

// Clone returns a deep copy.
func (d *Data) Clone() *Data {
	return &Data{
		Vertices: append([]Vertex(nil), d.Vertices...),
		Indices:  append([]uint32(nil), d.Indices...),
	}
}

// Transform returns a copy with positions and normals transformed by m.
func (d *Data) Transform(m geom.Mat4) *Data {
	out := d.Clone()
	n := m.NormalMatrix()
	for i := range out.Vertices {
		v := &out.Vertices[i]
		v.Pos = m.MulPoint(v.Pos)
		v.Normal = n.MulDir(v.Normal).Normalize()
	}
	return out
}

func (d *Data) Translate(t geom.Vec3) *Data { return d.Transform(geom.Translate(t)) }

func (d *Data) Rotate(q geom.Quat) *Data { return d.Transform(q.Mat4()) }

func (d *Data) Scale(s geom.Vec3) *Data { return d.Transform(geom.Scale(s)) }

// Tint returns a copy with every vertex color multiplied by c. Generated
// meshes start white, so Tint effectively paints them.
func (d *Data) Tint(c color.Color) *Data {
	out := d.Clone()
	for i := range out.Vertices {
		out.Vertices[i].Color = out.Vertices[i].Color.Mul(c)
	}
	return out
}

// Flat returns a faceted copy: vertices are unshared and normals are per
// face. This is the low-poly look. Degenerate triangles are dropped.
func (d *Data) Flat() *Data {
	out := &Data{Vertices: make([]Vertex, 0, len(d.Indices)), Indices: make([]uint32, 0, len(d.Indices))}
	for i := 0; i+2 < len(d.Indices); i += 3 {
		a, b, c := d.Vertices[d.Indices[i]], d.Vertices[d.Indices[i+1]], d.Vertices[d.Indices[i+2]]
		cross := b.Pos.Sub(a.Pos).Cross(c.Pos.Sub(a.Pos))
		if cross.LenSq() < 1e-14 {
			continue
		}
		n := cross.Normalize()
		for _, v := range [3]Vertex{a, b, c} {
			v.Normal = n
			out.Indices = append(out.Indices, index(len(out.Vertices)))
			out.Vertices = append(out.Vertices, v)
		}
	}
	return out
}

// Merge concatenates meshes into one, which draws in a single call.
func Merge(parts ...*Data) *Data {
	out := &Data{}
	for _, p := range parts {
		base := index(len(out.Vertices))
		out.Vertices = append(out.Vertices, p.Vertices...)
		for _, i := range p.Indices {
			out.Indices = append(out.Indices, base+i)
		}
	}
	return out
}

// Bounds returns the axis-aligned bounds of the vertices.
func (d *Data) Bounds() geom.AABB {
	if len(d.Vertices) == 0 {
		return geom.AABB{}
	}
	b := geom.AABB{Min: d.Vertices[0].Pos, Max: d.Vertices[0].Pos}
	for _, v := range d.Vertices[1:] {
		b.Min = b.Min.Min(v.Pos)
		b.Max = b.Max.Max(v.Pos)
	}
	return b
}

// RecomputeNormals rebuilds smooth normals from triangle areas.
func (d *Data) RecomputeNormals() *Data {
	for i := range d.Vertices {
		d.Vertices[i].Normal = geom.Vec3{}
	}
	for i := 0; i+2 < len(d.Indices); i += 3 {
		ia, ib, ic := d.Indices[i], d.Indices[i+1], d.Indices[i+2]
		a, b, c := d.Vertices[ia].Pos, d.Vertices[ib].Pos, d.Vertices[ic].Pos
		n := b.Sub(a).Cross(c.Sub(a))
		for _, j := range [3]uint32{ia, ib, ic} {
			d.Vertices[j].Normal = d.Vertices[j].Normal.Add(n)
		}
	}
	for i := range d.Vertices {
		d.Vertices[i].Normal = d.Vertices[i].Normal.Normalize()
	}
	return d
}

func (d *Data) add(v Vertex) uint32 {
	if v.Color == (color.Color{}) {
		v.Color = color.White
	}
	d.Vertices = append(d.Vertices, v)
	return index(len(d.Vertices) - 1)
}

func (d *Data) quad(a, b, c, e uint32) {
	d.Indices = append(d.Indices, a, b, c, a, c, e)
}

// index is n as a vertex index; a mesh stays far under 2³² vertices.
func index(n int) uint32 {
	if n < 0 || n > math.MaxUint32 {
		panic("mesh: vertex index out of range")
	}
	return uint32(n)
}
