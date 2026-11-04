package mesh

import (
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
)

// Box is centered on the origin with the given full extents.
func Box(size geom.Vec3) *Data {
	h := size.Scale(0.5)
	d := &Data{}
	faces := []struct{ n, v geom.Vec3 }{
		{geom.Right, geom.Up},
		{geom.Left, geom.Up},
		{geom.Up, geom.Forward},
		{geom.Down, geom.Backward},
		{geom.Backward, geom.Up},
		{geom.Forward, geom.Up},
	}
	for _, f := range faces {
		c := f.n.Mul(h)
		u := f.v.Cross(f.n).Mul(h)
		v := f.v.Mul(h)
		corner := func(su, sv float32, uv geom.Vec2) uint32 {
			return d.add(Vertex{Pos: c.Add(u.Scale(su)).Add(v.Scale(sv)), Normal: f.n, UV: uv})
		}
		a := corner(-1, -1, geom.V2(0, 1))
		b := corner(1, -1, geom.V2(1, 1))
		e := corner(1, 1, geom.V2(1, 0))
		g := corner(-1, 1, geom.V2(0, 0))
		d.quad(a, b, e, g)
	}
	return d
}

func Cube(size float32) *Data { return Box(geom.V3(size, size, size)) }

// Plane is a horizontal quad facing up, centered on the origin.
func Plane(width, depth float32) *Data {
	return Grid(width, depth, 1, 1, nil)
}

// Grid is a subdivided horizontal plane. height, when not nil, displaces
// each vertex and may return a vertex color; normals are recomputed.
func Grid(width, depth float32, cols, rows int, height func(x, z float32) (float32, color.Color)) *Data {
	d := &Data{}
	for r := 0; r <= rows; r++ {
		for c := 0; c <= cols; c++ {
			u, v := float32(c)/float32(cols), float32(r)/float32(rows)
			x, z := (u-0.5)*width, (v-0.5)*depth
			var y float32
			col := color.White
			if height != nil {
				y, col = height(x, z)
			}
			d.add(Vertex{Pos: geom.V3(x, y, z), Normal: geom.Up, UV: geom.V2(u, v), Color: col})
		}
	}
	stride := index(cols + 1)
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			i := index(r)*stride + index(c)
			d.quad(i, i+stride, i+stride+1, i+1)
		}
	}
	if height != nil {
		d.RecomputeNormals()
	}
	return d
}

// Sphere is a UV sphere.
func Sphere(radius float32, segments, rings int) *Data {
	d := &Data{}
	for r := 0; r <= rings; r++ {
		phi := geom.Pi * float32(r) / float32(rings)
		for s := 0; s <= segments; s++ {
			theta := geom.Tau * float32(s) / float32(segments)
			n := geom.V3(geom.Sin(phi)*geom.Cos(theta), geom.Cos(phi), geom.Sin(phi)*geom.Sin(theta))
			d.add(
				Vertex{
					Pos:    n.Scale(radius),
					Normal: n,
					UV:     geom.V2(float32(s)/float32(segments), float32(r)/float32(rings)),
				},
			)
		}
	}
	stride := index(segments + 1)
	for r := 0; r < rings; r++ {
		for s := 0; s < segments; s++ {
			i := index(r)*stride + index(s)
			d.quad(i, i+1, i+stride+1, i+stride)
		}
	}
	return d
}

// Cylinder spans y in [0, height]. A zero top radius makes a cone.
func Cylinder(bottom, top, height float32, segments int) *Data {
	d := &Data{}
	slope := (bottom - top) / height
	for s := 0; s <= segments; s++ {
		theta := geom.Tau * float32(s) / float32(segments)
		dir := geom.V3(geom.Cos(theta), 0, geom.Sin(theta))
		n := dir.Add(geom.V3(0, slope, 0)).Normalize()
		u := float32(s) / float32(segments)
		d.add(Vertex{Pos: dir.Scale(bottom), Normal: n, UV: geom.V2(u, 1)})
		d.add(Vertex{Pos: dir.Scale(top).WithY(height), Normal: n, UV: geom.V2(u, 0)})
	}
	for s := 0; s < segments; s++ {
		i := index(s * 2)
		d.quad(i, i+1, i+3, i+2)
	}
	disc := func(y, r float32, n geom.Vec3, flip bool) {
		if r <= 0 {
			return
		}
		center := d.add(Vertex{Pos: geom.V3(0, y, 0), Normal: n, UV: geom.V2(0.5, 0.5)})
		first := index(len(d.Vertices))
		for s := 0; s <= segments; s++ {
			theta := geom.Tau * float32(s) / float32(segments)
			c, sn := geom.Cos(theta), geom.Sin(theta)
			d.add(Vertex{Pos: geom.V3(c*r, y, sn*r), Normal: n, UV: geom.V2(0.5+c/2, 0.5+sn/2)})
		}
		for s := uint32(0); s < index(segments); s++ {
			if flip {
				d.Indices = append(d.Indices, center, first+s, first+s+1)
			} else {
				d.Indices = append(d.Indices, center, first+s+1, first+s)
			}
		}
	}
	disc(0, bottom, geom.Down, true)
	disc(height, top, geom.Up, false)
	return d
}

func Cone(radius, height float32, segments int) *Data {
	return Cylinder(radius, 0, height, segments)
}

// Capsule spans y in [0, height] with hemispherical ends.
func Capsule(radius, height float32, segments int) *Data {
	rings := max(segments/2, 2)
	if rings%2 == 1 {
		rings++
	}
	mid := max(height-2*radius, 0)
	d := &Data{}
	for r := 0; r <= rings; r++ {
		phi := geom.Pi * float32(r) / float32(rings)
		yOff := radius + mid
		if r > rings/2 {
			yOff = radius
		}
		if r == rings/2 {
			for _, y := range [2]float32{radius + mid, radius} {
				for s := 0; s <= segments; s++ {
					theta := geom.Tau * float32(s) / float32(segments)
					n := geom.V3(geom.Cos(theta), 0, geom.Sin(theta))
					d.add(Vertex{Pos: n.Scale(radius).WithY(y), Normal: n})
				}
			}
			continue
		}
		for s := 0; s <= segments; s++ {
			theta := geom.Tau * float32(s) / float32(segments)
			n := geom.V3(geom.Sin(phi)*geom.Cos(theta), geom.Cos(phi), geom.Sin(phi)*geom.Sin(theta))
			d.add(Vertex{Pos: n.Scale(radius).Add(geom.V3(0, yOff, 0)), Normal: n})
		}
	}
	stride := index(segments + 1)
	rowsTotal := index(len(d.Vertices)) / stride
	for r := uint32(0); r+1 < rowsTotal; r++ {
		for s := uint32(0); s < index(segments); s++ {
			i := r*stride + s
			d.quad(i, i+1, i+stride+1, i+stride)
		}
	}
	return d
}

// Torus lies in the XZ plane.
func Torus(radius, tube float32, segments, sides int) *Data {
	d := &Data{}
	for i := 0; i <= segments; i++ {
		u := geom.Tau * float32(i) / float32(segments)
		center := geom.V3(geom.Cos(u)*radius, 0, geom.Sin(u)*radius)
		for j := 0; j <= sides; j++ {
			v := geom.Tau * float32(j) / float32(sides)
			n := geom.V3(geom.Cos(u)*geom.Cos(v), geom.Sin(v), geom.Sin(u)*geom.Cos(v))
			d.add(
				Vertex{
					Pos:    center.Add(n.Scale(tube)),
					Normal: n,
					UV:     geom.V2(float32(i)/float32(segments), float32(j)/float32(sides)),
				},
			)
		}
	}
	stride := index(sides + 1)
	for i := 0; i < segments; i++ {
		for j := 0; j < sides; j++ {
			a := index(i)*stride + index(j)
			d.quad(a, a+1, a+stride+1, a+stride)
		}
	}
	return d
}

// Quad is a unit quad in the XY plane facing +Z, used for billboards.
func Quad() *Data {
	d := &Data{}
	a := d.add(Vertex{Pos: geom.V3(-0.5, -0.5, 0), Normal: geom.Backward, UV: geom.V2(0, 1)})
	b := d.add(Vertex{Pos: geom.V3(0.5, -0.5, 0), Normal: geom.Backward, UV: geom.V2(1, 1)})
	c := d.add(Vertex{Pos: geom.V3(0.5, 0.5, 0), Normal: geom.Backward, UV: geom.V2(1, 0)})
	e := d.add(Vertex{Pos: geom.V3(-0.5, 0.5, 0), Normal: geom.Backward, UV: geom.V2(0, 0)})
	d.quad(a, b, c, e)
	return d
}

// Icosphere is a low-poly sphere, great for rocks and orbs once Flat.
func Icosphere(radius float32, subdivisions int) *Data {
	t := (1 + geom.Sqrt(5)) / 2
	pts := []geom.Vec3{
		geom.V3(-1, t, 0), geom.V3(1, t, 0), geom.V3(-1, -t, 0), geom.V3(1, -t, 0),
		geom.V3(0, -1, t), geom.V3(0, 1, t), geom.V3(0, -1, -t), geom.V3(0, 1, -t),
		geom.V3(t, 0, -1), geom.V3(t, 0, 1), geom.V3(-t, 0, -1), geom.V3(-t, 0, 1),
	}
	idx := []uint32{
		0, 11, 5, 0, 5, 1, 0, 1, 7, 0, 7, 10, 0, 10, 11,
		1, 5, 9, 5, 11, 4, 11, 10, 2, 10, 7, 6, 7, 1, 8,
		3, 9, 4, 3, 4, 2, 3, 2, 6, 3, 6, 8, 3, 8, 9,
		4, 9, 5, 2, 4, 11, 6, 2, 10, 8, 6, 7, 9, 8, 1,
	}
	for i := range pts {
		pts[i] = pts[i].Normalize()
	}
	for s := 0; s < subdivisions; s++ {
		cache := map[[2]uint32]uint32{}
		midpoint := func(a, b uint32) uint32 {
			key := [2]uint32{min(a, b), max(a, b)}
			if m, ok := cache[key]; ok {
				return m
			}
			pts = append(pts, pts[a].Add(pts[b]).Normalize())
			cache[key] = index(len(pts) - 1)
			return cache[key]
		}
		next := make([]uint32, 0, len(idx)*4)
		for i := 0; i < len(idx); i += 3 {
			a, b, c := idx[i], idx[i+1], idx[i+2]
			ab, bc, ca := midpoint(a, b), midpoint(b, c), midpoint(c, a)
			next = append(next, a, ab, ca, b, bc, ab, c, ca, bc, ab, bc, ca)
		}
		idx = next
	}
	d := &Data{Indices: idx}
	for _, p := range pts {
		d.add(Vertex{Pos: p.Scale(radius), Normal: p})
	}
	return d
}
