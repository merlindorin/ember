package gltf //nolint:testpackage // white-box: round-trips rigGLB, the white-box builder of gltf_test.go

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lantervale/ember/anim"
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/mesh"
	"github.com/lantervale/ember/render"
)

// posed returns every part's vertices in model space at time t of a clip
// (the rest pose without one), as Spawn would draw them.
func posed(m *Model, clip string, t float32) [][]geom.Vec3 {
	p := m.Skeleton.RestPose()
	if c := m.Clip(clip); c != nil {
		c.Sample(t, p)
	}
	g := m.Skeleton.Globals(p, nil)
	var out [][]geom.Vec3
	for _, part := range m.Parts {
		d := part.Mesh
		if part.Skin != nil {
			d = part.Mesh.Clone()
			anim.Deform(part.Mesh, part.Influences, part.Skin.Palette(g, nil), d)
		}
		var pts []geom.Vec3
		for _, v := range d.Vertices {
			pts = append(pts, v.Pos)
		}
		out = append(out, pts)
	}
	return out
}

func samePose(t *testing.T, a, b *Model, clip string, at float32) {
	t.Helper()
	pa, pb := posed(a, clip, at), posed(b, clip, at)
	if len(pa) != len(pb) {
		t.Fatalf("%d parts, then %d", len(pa), len(pb))
	}
	for i := range pa {
		if len(pa[i]) != len(pb[i]) {
			t.Fatalf("part %s: %d vertices, then %d", a.Parts[i].Name, len(pa[i]), len(pb[i]))
		}
		for k := range pa[i] {
			if !pa[i][k].ApproxEq(pb[i][k], 1e-4) {
				t.Fatalf("part %s vertex %d at %s %.2fs: %v, then %v", a.Parts[i].Name, k, clip, at, pa[i][k], pb[i][k])
			}
		}
	}
}

func roundTrip(t *testing.T, m *Model) *Model {
	t.Helper()
	data, err := Encode(m)
	if err != nil {
		t.Fatal(err)
	}
	back, err := Decode(data)
	if err != nil {
		t.Fatalf("decoding what Encode wrote: %v", err)
	}
	return back
}

func TestEncodeRoundTripsARig(t *testing.T) {
	m, err := Decode(rigGLB(t, true))
	if err != nil {
		t.Fatal(err)
	}
	back := roundTrip(t, m)
	if len(back.Skeleton.Joints) != len(m.Skeleton.Joints) {
		t.Fatalf("parts go back on their nodes: %d joints, then %d", len(m.Skeleton.Joints), len(back.Skeleton.Joints))
	}
	for i, j := range m.Skeleton.Joints {
		if b := back.Skeleton.Joints[i]; b.Name != j.Name || b.Parent != j.Parent ||
			!b.Rest.T.ApproxEq(j.Rest.T, 1e-6) || !b.Rest.S.ApproxEq(j.Rest.S, 1e-6) {
			t.Fatalf("joint %d: %+v, then %+v", i, j, b)
		}
	}
	if len(back.Clips) != 1 || back.Clips[0].Name != "Wave" || back.Clips[0].Duration != 1 {
		t.Fatalf("clips %+v", back.Clips)
	}
	for _, at := range []float32{0, 0.3, 0.8} {
		samePose(t, m, back, "Wave", at)
	}
	body, glass := back.Parts[0], back.Parts[1]
	if body.Name != "Body" || body.Skin == nil || len(body.Influences) != 3 {
		t.Fatalf("the skinned body: %+v", body)
	}
	tex := body.Material.Texture
	if tex == nil || tex.Image.Bounds().Dx() != 2 || tex.Filter != render.FilterNearest || !body.Material.DoubleSided ||
		geom.Abs(body.Material.Specular-m.Parts[0].Material.Specular) > 1e-5 {
		t.Fatalf("the textured material: %+v", body.Material)
	}
	if !glass.Material.Transparent || geom.Abs(glass.Material.Albedo.A-0.4) > 1e-6 ||
		glass.Material.Emissive != color.RGB(1, 0.5, 0) {
		t.Fatalf("the glass: %+v", glass.Material)
	}
	again := roundTrip(t, back)
	if len(again.Skeleton.Joints) != len(m.Skeleton.Joints) || len(again.Parts) != len(m.Parts) {
		t.Fatal("writing again changes nothing")
	}
}

func TestEncodeStaticModelsAndBrightEmission(t *testing.T) {
	sk := &anim.Skeleton{Joints: []anim.Joint{
		{Name: "Root", Parent: -1, Rest: anim.TRS{T: geom.V3(0, 1, 0), R: geom.QuatIdent(), S: geom.V3(2, 2, 2)}},
		{Name: "Lamp", Parent: 0, Rest: anim.TRS{T: geom.V3(1, 0, 0), R: geom.QuatYaw(1), S: geom.V3(1, 1, 1)}},
	}}
	tri := &mesh.Data{Vertices: []mesh.Vertex{
		{Pos: geom.V3(0, 0, 0), Color: color.Red},
		{Pos: geom.V3(1, 0, 0), Color: color.White},
		{Pos: geom.V3(0, 1, 0), Color: color.White},
	}, Indices: []uint32{0, 1, 2}}
	tri.RecomputeNormals()
	glow := &render.Material{Albedo: color.White, Emissive: color.RGB(4, 2, 0), Unlit: true}
	m := &Model{Name: "Props", Skeleton: sk, Parts: []*Part{
		{Name: "Lamp", Mesh: tri, Material: glow},
		{Name: "Loose", Mesh: tri, Material: glow},
	}}
	back := roundTrip(t, m)
	if back.Name != "Props" && back.Name != "Root" {
		t.Fatalf("name %q", back.Name)
	}
	samePose(t, m, back, "", 0)
	if len(back.Skeleton.Joints) != 3 {
		t.Fatalf("the lamp goes back on its node, the loose part on a new one: %d joints", len(back.Skeleton.Joints))
	}
	if c := back.Parts[0].Mesh.Vertices[0].Color; c != color.Red {
		t.Fatalf("vertex colors: %v", c)
	}
	if mat := back.Parts[0].Material; mat.Emissive != color.RGB(4, 2, 0) || !mat.Unlit {
		t.Fatalf("bright emission and unlit come back: %+v", mat)
	}
	data, _ := Encode(m)
	if !strings.Contains(string(data), "KHR_materials_emissive_strength") ||
		!strings.Contains(string(data), "KHR_materials_unlit") {
		t.Fatal("emission above 1 and unlit materials use the standard extensions")
	}
}

func TestEncodeRoundTripsTheApocalypsePack(t *testing.T) {
	if os.Getenv("EMBER_REAL_PACKS") != "1" {
		t.Skip("real asset packs are opt-in: EMBER_REAL_PACKS=1")
	}
	path := os.Getenv("EMBER_TEST_GLB_ZIP")
	if path == "" {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, "Downloads", "Apocalypse_Free_glb.zip")
	}
	z, zerr := zip.OpenReader(path)
	if zerr != nil {
		t.Skip("asset pack not available:", zerr)
	}
	defer z.Close()
	tested := 0
	for _, f := range z.File {
		base := filepath.Base(f.Name)
		if base != "Adult_Survivor.glb" && base != "Doberman.glb" && base != "Katana.glb" && base != "Apocalypse_Car.glb" {
			continue
		}
		m, err := LoadFS(z, f.Name)
		if err != nil {
			t.Fatal(err)
		}
		back := roundTrip(t, m)
		if len(back.Skeleton.Joints) != len(m.Skeleton.Joints) || len(back.Clips) != len(m.Clips) {
			t.Fatalf("%s: %d joints %d clips, then %d and %d",
				base, len(m.Skeleton.Joints), len(m.Clips), len(back.Skeleton.Joints), len(back.Clips))
		}
		clip := ""
		if len(m.Clips) > 0 {
			clip = m.Clips[len(m.Clips)/2].Name
		}
		samePose(t, m, back, clip, 0.4)
		tested++
	}
	if tested == 0 {
		t.Skip("no known model in the pack")
	}
}
