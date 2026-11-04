package mesh_test

import (
	"testing"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/mesh"
)

// checkWinding verifies every triangle faces the same way as its vertex
// normals, which is what back-face culling relies on.
func checkWinding(t *testing.T, name string, d *mesh.Data) {
	t.Helper()
	for i := 0; i+2 < len(d.Indices); i += 3 {
		a, b, c := d.Vertices[d.Indices[i]], d.Vertices[d.Indices[i+1]], d.Vertices[d.Indices[i+2]]
		face := b.Pos.Sub(a.Pos).Cross(c.Pos.Sub(a.Pos))
		if face.LenSq() < 1e-12 {
			continue
		}
		if face.Dot(a.Normal.Add(b.Normal).Add(c.Normal)) <= 0 {
			t.Fatalf("%s: triangle %d is wound against its normals", name, i/3)
		}
	}
}

func checkIndices(t *testing.T, name string, d *mesh.Data) {
	t.Helper()
	if len(d.Indices)%3 != 0 || len(d.Indices) == 0 {
		t.Fatalf("%s: %d indices", name, len(d.Indices))
	}
	for _, i := range d.Indices {
		if int(i) >= len(d.Vertices) {
			t.Fatalf("%s: index %d out of range", name, i)
		}
	}
}

func TestGeneratorsProduceValidCCWGeometry(t *testing.T) {
	shapes := map[string]*mesh.Data{
		"box":       mesh.Box(geom.V3(1, 2, 3)),
		"plane":     mesh.Plane(4, 4),
		"sphere":    mesh.Sphere(1, 16, 8),
		"cylinder":  mesh.Cylinder(1, 0.5, 2, 12),
		"cone":      mesh.Cone(1, 2, 12),
		"capsule":   mesh.Capsule(0.5, 2, 12),
		"torus":     mesh.Torus(1, 0.25, 16, 8),
		"quad":      mesh.Quad(),
		"icosphere": mesh.Icosphere(1, 2),
	}
	for name, d := range shapes {
		checkIndices(t, name, d)
		checkWinding(t, name, d)
		checkWinding(t, name+"/flat", d.Flat())
	}
}

func TestBoxBounds(t *testing.T) {
	b := mesh.Box(geom.V3(2, 4, 6)).Bounds()
	if b.Min != geom.V3(-1, -2, -3) || b.Max != geom.V3(1, 2, 3) {
		t.Fatalf("got %v", b)
	}
}

func TestCapsuleSpansHeight(t *testing.T) {
	b := mesh.Capsule(0.5, 2, 12).Bounds()
	if geom.Abs(b.Min.Y) > 1e-4 || geom.Abs(b.Max.Y-2) > 1e-4 {
		t.Fatalf("got %v", b)
	}
}

func TestMergeOffsetsIndices(t *testing.T) {
	a, b := mesh.Cube(1), mesh.Cube(1).Translate(geom.V3(5, 0, 0))
	m := mesh.Merge(a, b)
	if len(m.Vertices) != 48 || len(m.Indices) != 72 {
		t.Fatalf("got %d verts %d indices", len(m.Vertices), len(m.Indices))
	}
	checkIndices(t, "merge", m)
	if m.Bounds().Max.X != 5.5 {
		t.Fatalf("bounds %v", m.Bounds())
	}
}

func TestTintAndTransformDoNotMutateSource(t *testing.T) {
	src := mesh.Cube(1)
	_ = src.Tint(color.Red).Translate(geom.V3(1, 0, 0))
	if src.Vertices[0].Color != color.White || src.Bounds().Max.X != 0.5 {
		t.Fatal("source mutated")
	}
}

func TestTransformRotatesNormals(t *testing.T) {
	d := mesh.Plane(1, 1).Rotate(geom.QuatAxisAngle(geom.Right, geom.Pi/2))
	if n := d.Vertices[0].Normal; !n.ApproxEq(geom.Backward, 1e-4) {
		t.Fatalf("normal %v", n)
	}
}

func TestGridHeightfield(t *testing.T) {
	d := mesh.Grid(10, 10, 4, 4, func(x, _ float32) (float32, color.Color) { return x * 0.1, color.Green })
	checkIndices(t, "grid", d)
	checkWinding(t, "grid", d)
	if len(d.Vertices) != 25 || d.Vertices[0].Color != color.Green {
		t.Fatal("grid layout wrong")
	}
}

func TestTouchBumpsVersion(t *testing.T) {
	d := mesh.Cube(1)
	v := d.Version()
	d.Touch()
	if d.Version() == v {
		t.Fatal("version unchanged")
	}
}

func TestFlatDropsDegenerateTriangles(t *testing.T) {
	for _, v := range mesh.Cone(1, 2, 8).Flat().Vertices {
		if geom.Abs(v.Normal.Len()-1) > 1e-4 {
			t.Fatalf("flat normal %v is not unit length", v.Normal)
		}
	}
}
