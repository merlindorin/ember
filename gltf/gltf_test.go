package gltf //nolint:testpackage // white-box: builds GLB files with the unexported chunk constants

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"image"
	stdcolor "image/color"
	"image/png"
	"math"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/lantervale/ember/anim"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/render"
	"github.com/lantervale/ember/scene"
	"github.com/lantervale/ember/world"
)

// builder assembles a glTF document and its binary buffer.
type builder struct {
	bin   bytes.Buffer
	views []map[string]any
	accs  []map[string]any
}

func (b *builder) add(data any, count int, kind string, ctype int, extra map[string]any) int {
	for b.bin.Len()%4 != 0 {
		b.bin.WriteByte(0)
	}
	off := b.bin.Len()
	_ = binary.Write(&b.bin, binary.LittleEndian, data)
	b.views = append(b.views, map[string]any{"buffer": 0, "byteOffset": off, "byteLength": b.bin.Len() - off})
	acc := map[string]any{"bufferView": len(b.views) - 1, "count": count, "type": kind, "componentType": ctype}
	for k, v := range extra {
		acc[k] = v
	}
	b.accs = append(b.accs, acc)
	return len(b.accs) - 1
}

func (b *builder) raw(data []byte) int {
	for b.bin.Len()%4 != 0 {
		b.bin.WriteByte(0)
	}
	off := b.bin.Len()
	b.bin.Write(data)
	b.views = append(b.views, map[string]any{"buffer": 0, "byteOffset": off, "byteLength": len(data)})
	return len(b.views) - 1
}

func pngBytes(t *testing.T) []byte {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, stdcolor.RGBA{255, 0, 0, 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// rigGLB is a triangle skinned to "Bone" under an "Armature" node scaled
// by 2, a static "Hat" attached to the bone, and a "Wave" clip rotating the
// bone a quarter turn around Z.
func rigGLB(t *testing.T, glb bool) []byte {
	var b builder
	pos := b.add([]float32{0, 1, 0, 1, 1, 0, 0, 2, 0}, 3, "VEC3", 5126,
		map[string]any{"min": []float32{0, 1, 0}, "max": []float32{1, 2, 0}})
	nrm := b.add([]float32{0, 0, 1, 0, 0, 1, 0, 0, 1}, 3, "VEC3", 5126, nil)
	uv := b.add([]float32{0, 0, 1, 0, 0, 1}, 3, "VEC2", 5126, nil)
	joints := b.add([]uint8{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, 3, "VEC4", 5121, nil)
	weights := b.add([]float32{1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0}, 3, "VEC4", 5126, nil)
	idx := b.add([]uint16{0, 1, 2, 0}, 3, "SCALAR", 5123, nil)
	ibm := b.add([]float32{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, -1, 0, 1}, 1, "MAT4", 5126, nil)
	times := b.add([]float32{0, 1}, 2, "SCALAR", 5126, nil)
	q := float32(math.Sqrt(0.5))
	rots := b.add([]float32{0, 0, 0, 1, 0, 0, q, q}, 2, "VEC4", 5126, nil)
	hatPos := b.add([]float32{0, 0, 0, 0.1, 0, 0, 0, 0.1, 0}, 3, "VEC3", 5126, nil)
	img := b.raw(pngBytes(t))

	doc := map[string]any{
		"asset":  map[string]any{"version": "2.0"},
		"scene":  0,
		"scenes": []any{map[string]any{"nodes": []int{0}}},
		"nodes": []any{
			map[string]any{"name": "Armature", "children": []int{1, 2}, "scale": []float32{2, 2, 2}},
			map[string]any{"name": "Bone", "translation": []float32{0, 1, 0}, "children": []int{3}},
			map[string]any{"name": "Body", "mesh": 0, "skin": 0},
			map[string]any{"name": "Hat", "mesh": 1, "translation": []float32{0, 0.5, 0}},
		},
		"meshes": []any{
			map[string]any{"name": "Body", "primitives": []any{map[string]any{
				"attributes": map[string]int{
					"POSITION": pos, "NORMAL": nrm, "TEXCOORD_0": uv, "JOINTS_0": joints, "WEIGHTS_0": weights,
				},
				"indices": idx, "material": 0,
			}}},
			map[string]any{"name": "Hat", "primitives": []any{
				map[string]any{"attributes": map[string]int{"POSITION": hatPos}, "material": 1},
			}},
		},
		"skins": []any{map[string]any{"joints": []int{1}, "inverseBindMatrices": ibm}},
		"animations": []any{map[string]any{
			"name":     "Wave",
			"samplers": []any{map[string]any{"input": times, "output": rots}},
			"channels": []any{map[string]any{"sampler": 0, "target": map[string]any{"node": 1, "path": "rotation"}}},
		}},
		"materials": []any{
			map[string]any{"name": "Skin", "doubleSided": true, "pbrMetallicRoughness": map[string]any{
				"baseColorTexture": map[string]any{"index": 0}, "roughnessFactor": 0.5,
			}},
			map[string]any{"name": "Glass", "alphaMode": "BLEND", "emissiveFactor": []float32{1, 0.5, 0},
				"pbrMetallicRoughness": map[string]any{"baseColorFactor": []float32{1, 1, 1, 0.4}}},
		},
		"textures":    []any{map[string]any{"source": 0, "sampler": 0}},
		"samplers":    []any{map[string]any{"magFilter": 9728}},
		"images":      []any{map[string]any{"bufferView": img, "mimeType": "image/png"}},
		"accessors":   b.accs,
		"bufferViews": b.views,
		"buffers":     []any{map[string]any{"byteLength": b.bin.Len()}},
	}
	if !glb {
		doc["buffers"] = []any{map[string]any{"byteLength": b.bin.Len(), "uri": dataBuffer(b.bin.Bytes())}}
	}
	js, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if !glb {
		return js
	}
	for len(js)%4 != 0 {
		js = append(js, ' ')
	}
	bin := b.bin.Bytes()
	for len(bin)%4 != 0 {
		bin = append(bin, 0)
	}
	var out bytes.Buffer
	w := func(v ...uint32) { _ = binary.Write(&out, binary.LittleEndian, v) }
	w(glbMagic, 2, uint32(12+8+len(js)+8+len(bin)))
	w(uint32(len(js)), chunkJSON)
	out.Write(js)
	w(uint32(len(bin)), chunkBIN)
	out.Write(bin)
	return out.Bytes()
}

// dataBuffer embeds a buffer as a data URI.
func dataBuffer(b []byte) string {
	return "data:application/octet-stream;base64," + base64.StdEncoding.EncodeToString(b)
}

func TestDecodeGLBAndEmbeddedGLTF(t *testing.T) {
	for _, glb := range []bool{true, false} {
		m, err := Decode(rigGLB(t, glb))
		if err != nil {
			t.Fatalf("glb=%v: %v", glb, err)
		}
		checkRig(t, m, glb)
	}
}

// checkRig checks what Decode read of rigGLB.
func checkRig(t *testing.T, m *Model, glb bool) {
	t.Helper()
	if m.Name != "Armature" || len(m.Parts) != 2 || len(m.Skeleton.Joints) != 4 || len(m.Clips) != 1 {
		t.Fatalf("glb=%v: %+v", glb, m)
	}
	body, hat := m.Parts[0], m.Parts[1]
	if body.Name != "Body" || hat.Name != "Hat" {
		t.Fatalf("parts sorted by name: %s %s", body.Name, hat.Name)
	}
	if len(body.Mesh.Vertices) != 3 || len(body.Mesh.Indices) != 3 || body.Mesh.Vertices[1].UV != geom.V2(1, 0) {
		t.Fatalf("body mesh %+v", body.Mesh)
	}
	if body.Skin == nil || body.Skin.Joints[0] != m.Skeleton.Find("Bone") || body.Influences[2].Weights[0] != 1 {
		t.Fatalf("body skin %+v", body.Skin)
	}
	mat := body.Material
	if !mat.DoubleSided || mat.Texture == nil || mat.Texture.Filter != render.FilterNearest ||
		mat.Texture.Image.Bounds().Dx() != 2 {
		t.Fatalf("textured material %+v", mat)
	}
	if g := hat.Material; !g.Transparent || g.Albedo.A != 0.4 || g.Emissive.G != 0.5 {
		t.Fatalf("glass material %+v", g)
	}
	if hat.Skin == nil || hat.Skin.Joints[0] != m.Skeleton.Find("Hat") || hat.Mesh.Vertices[0].Normal == (geom.Vec3{}) {
		t.Fatalf("animated files attach static meshes rigidly to their node, with computed normals: %+v", hat.Skin)
	}
	if c := m.Clip("Wave"); c == nil || c.Duration != 1 || len(c.Channels) != 1 ||
		c.Channels[0].Joint != m.Skeleton.Find("Bone") {
		t.Fatalf("clip %+v", c)
	}
	if b := m.Bounds; !b.Max.ApproxEq(geom.V3(2, 4, 0), 1e-4) || !b.Min.ApproxEq(geom.V3(0, 2, 0), 1e-4) {
		t.Fatalf("rest bounds include the armature scale: %v", b)
	}
}

func TestSpawnedInstancesAnimate(t *testing.T) {
	m, err := Decode(rigGLB(t, true))
	if err != nil {
		t.Fatal(err)
	}
	a := world.New().AddPlugins(render.Plugin{}, anim.Plugin{})
	in := m.Spawn(a, scene.At(0, 0, 0))
	body := in.Part("Body")
	if body == nil || in.Part("Nope") != nil || body.Mesh == body.Part.Mesh {
		t.Fatal("skinned parts draw their own mesh copy")
	}
	tip := func() geom.Vec3 { return body.Mesh.Vertices[2].Pos }
	if !tip().ApproxEq(geom.V3(0, 4, 0), 1e-4) {
		t.Fatalf("spawned in rest pose: %v", tip())
	}
	if !in.PlayOnce("Wave", 0) {
		t.Fatal("play")
	}
	for range 10 {
		a.Tick(0.2)
	}
	if !tip().ApproxEq(geom.V3(-2, 2, 0), 1e-3) {
		t.Fatalf("the bone rotated the tip around its pivot: %v", tip())
	}
	hat := in.Part("Hat")
	if p := hat.Mesh.Vertices[0].Pos; !p.ApproxEq(geom.V3(-1, 2, 0), 1e-3) {
		t.Fatalf("the hat follows the bone: %v", p)
	}
	if n := len(a.Entities()); n != 3 {
		t.Fatalf("root and two parts: %d", n)
	}
	in.Remove(a)
	a.Tick(0.1)
	if n := len(a.Entities()); n != 0 {
		t.Fatalf("remove takes every entity: %d", n)
	}
}

func TestStaticModelsAreBakedAndErrorsAreReported(t *testing.T) {
	var b builder
	pos := b.add([]float32{0, 0, 0, 1, 0, 0, 0, 1, 0}, 3, "VEC3", 5126, nil)
	doc := map[string]any{
		"nodes": []any{map[string]any{"name": "Rock", "mesh": 0, "translation": []float32{5, 0, 0}}},
		"meshes": []any{map[string]any{"primitives": []any{
			map[string]any{"attributes": map[string]int{"POSITION": pos}},
		}}},
		"accessors":   b.accs,
		"bufferViews": b.views,
		"buffers":     []any{map[string]any{"byteLength": b.bin.Len(), "uri": dataBuffer(b.bin.Bytes())}},
	}
	js, _ := json.Marshal(doc)
	m, err := Decode(js)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Parts) != 1 || m.Parts[0].Skin != nil || m.Parts[0].Mesh.Vertices[1].Pos != geom.V3(6, 0, 0) || m.Animated() {
		t.Fatalf("static parts are baked in model space: %+v", m.Parts[0])
	}
	if m.Parts[0].Material == nil {
		t.Fatal("primitives without material get a default one")
	}
	for name, bad := range map[string][]byte{
		"not json":        []byte("nope"),
		"truncated glb":   {0x67, 0x6c, 0x54, 0x46, 2, 0, 0, 0, 0xff, 0, 0, 0},
		"external buffer": []byte(`{"buffers":[{"uri":"model.bin","byteLength":4}]}`),
		"bad accessor":    []byte(`{"nodes":[{"mesh":0}],"meshes":[{"primitives":[{"attributes":{"POSITION":7}}]}]}`),
	} {
		if _, err = Decode(bad); err == nil {
			t.Fatalf("%s must fail", name)
		}
	}
}

func TestAttachmentsFollowAJointAndGoAwayWithTheirHolder(t *testing.T) {
	rig, err := Decode(rigGLB(t, true))
	if err != nil {
		t.Fatal(err)
	}
	a := world.New().AddPlugins(render.Plugin{}, anim.Plugin{})
	in := rig.Spawn(a, scene.At(10, 0, 0))
	if _, err = in.Attach(a, rig, "Nope", anim.TRS{}); err == nil {
		t.Fatal("unknown joints are errors")
	}
	at, err := in.Attach(a, rig, "Bone", anim.TRS{T: geom.V3(0, 0.5, 0)})
	if err != nil {
		t.Fatal(err)
	}
	a.Tick(0.1)
	if p := at.WorldPosition(); !p.ApproxEq(geom.V3(10, 3, 0), 1e-3) {
		t.Fatalf("attachment at %v", p)
	}
	in.PlayOnce("Wave", 0)
	for range 10 {
		a.Tick(0.2)
	}
	if p := at.WorldPosition(); !p.ApproxEq(geom.V3(9, 2, 0), 1e-3) {
		t.Fatalf("attachment follows the animation: %v", p)
	}
	in.Remove(a)
	a.Tick(0.1)
	if n := len(a.Entities()); n != 0 {
		t.Fatalf("holder, parts, socket and item are gone: %d left", n)
	}
}

func TestExternalBuffersAndImagesAreReadNextToTheFile(t *testing.T) {
	glb := rigGLB(t, false)
	var doc map[string]any
	if err := json.Unmarshal(glb, &doc); err != nil {
		t.Fatal(err)
	}
	bufs, ok := doc["buffers"].([]any)
	if !ok {
		t.Fatal("no buffers")
	}
	buf, ok := bufs[0].(map[string]any)
	if !ok {
		t.Fatal("no buffer")
	}
	uri, ok := buf["uri"].(string)
	if !ok {
		t.Fatal("no buffer URI")
	}
	bin, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(uri, "data:application/octet-stream;base64,"))
	if err != nil {
		t.Fatal(err)
	}
	buf["uri"] = "rig%20data.bin"
	doc["images"] = []any{map[string]any{"uri": "textures/skin.png"}}
	js, _ := json.Marshal(doc)
	fsys := fstest.MapFS{
		"models/rig.gltf":          {Data: js},
		"models/rig data.bin":      {Data: bin},
		"models/textures/skin.png": {Data: pngBytes(t)},
	}
	m, err := LoadFS(fsys, "models/rig.gltf")
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Parts) != 2 || m.Parts[0].Material.Texture == nil || m.Parts[0].Material.Texture.Image.Bounds().Dx() != 2 {
		t.Fatalf("parts %d texture %+v", len(m.Parts), m.Parts[0].Material.Texture)
	}
	if _, err = Decode(js); err == nil || !strings.Contains(err.Error(), "LoadFS") {
		t.Fatalf("decoding alone explains it needs a file system: %v", err)
	}
	delete(fsys, "models/rig data.bin")
	if _, err = LoadFS(fsys, "models/rig.gltf"); err == nil {
		t.Fatal("a missing buffer is an error")
	}
}
