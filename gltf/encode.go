package gltf

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"maps"
	"math"
	"slices"
	"strings"

	"github.com/lantervale/ember/anim"
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/render"
)

// Encode writes a model as a binary glTF (.glb) that Decode reads back
// alike: the skeleton as nodes, each part as a mesh primitive (skinned,
// rigidly following its joint, or static), materials with their textures
// embedded as PNG, and every clip.
func Encode(m *Model) ([]byte, error) {
	e := &encoder{materials: map[*render.Material]int{}, textures: map[*render.Texture]int{}}
	return e.encode(m)
}

type outDoc struct {
	Asset       map[string]string `json:"asset"`
	Scene       int               `json:"scene"`
	Scenes      []outScene        `json:"scenes"`
	Nodes       []outNode         `json:"nodes"`
	Meshes      []outMesh         `json:"meshes,omitempty"`
	Skins       []outSkin         `json:"skins,omitempty"`
	Animations  []outAnimation    `json:"animations,omitempty"`
	Materials   []outMaterial     `json:"materials,omitempty"`
	Textures    []outTexture      `json:"textures,omitempty"`
	Images      []outImage        `json:"images,omitempty"`
	Samplers    []outSampler      `json:"samplers,omitempty"`
	Accessors   []outAccessor     `json:"accessors,omitempty"`
	BufferViews []outView         `json:"bufferViews,omitempty"`
	Buffers     []outBuffer       `json:"buffers,omitempty"`
	Cameras     []cameraDef       `json:"cameras,omitempty"`

	ExtensionsUsed     []string                   `json:"extensionsUsed,omitempty"`
	ExtensionsRequired []string                   `json:"extensionsRequired,omitempty"`
	Extensions         map[string]json.RawMessage `json:"extensions,omitempty"`
	Extras             json.RawMessage            `json:"extras,omitempty"`
}

type outScene struct {
	Name  string `json:"name,omitempty"`
	Nodes []int  `json:"nodes"`
}

type outNode struct {
	Name        string      `json:"name,omitempty"`
	Children    []int       `json:"children,omitempty"`
	Mesh        *int        `json:"mesh,omitempty"`
	Skin        *int        `json:"skin,omitempty"`
	Camera      *int        `json:"camera,omitempty"`
	Translation *[3]float32 `json:"translation,omitempty"`
	Rotation    *[4]float32 `json:"rotation,omitempty"`
	Scale       *[3]float32 `json:"scale,omitempty"`

	Extensions map[string]json.RawMessage `json:"extensions,omitempty"`
	Extras     json.RawMessage            `json:"extras,omitempty"`
}

type outMesh struct {
	Name       string         `json:"name,omitempty"`
	Primitives []outPrimitive `json:"primitives"`
}

type outPrimitive struct {
	Attributes map[string]int `json:"attributes"`
	Indices    int            `json:"indices"`
	Material   int            `json:"material"`
}

type outSkin struct {
	InverseBindMatrices int   `json:"inverseBindMatrices"`
	Joints              []int `json:"joints"`
}

type outAnimation struct {
	Name     string          `json:"name,omitempty"`
	Channels []outChannel    `json:"channels"`
	Samplers []outAnimSample `json:"samplers"`
}

type outChannel struct {
	Sampler int `json:"sampler"`
	Target  struct {
		Node int    `json:"node"`
		Path string `json:"path"`
	} `json:"target"`
}

type outAnimSample struct {
	Input         int    `json:"input"`
	Output        int    `json:"output"`
	Interpolation string `json:"interpolation"`
}

type outMaterial struct {
	Name           string         `json:"name,omitempty"`
	AlphaMode      string         `json:"alphaMode,omitempty"`
	DoubleSided    bool           `json:"doubleSided,omitempty"`
	EmissiveFactor []float32      `json:"emissiveFactor,omitempty"`
	PBR            outPBR         `json:"pbrMetallicRoughness"`
	Extensions     map[string]any `json:"extensions,omitempty"`
}

type outPBR struct {
	BaseColorFactor  [4]float32  `json:"baseColorFactor"`
	BaseColorTexture *textureRef `json:"baseColorTexture,omitempty"`
	MetallicFactor   float32     `json:"metallicFactor"`
	RoughnessFactor  float32     `json:"roughnessFactor"`
}

type outTexture struct {
	Source  int `json:"source"`
	Sampler int `json:"sampler"`
}

type outImage struct {
	BufferView int    `json:"bufferView"`
	MimeType   string `json:"mimeType"`
}

type outSampler struct {
	MagFilter int `json:"magFilter"`
	MinFilter int `json:"minFilter"`
	WrapS     int `json:"wrapS"`
	WrapT     int `json:"wrapT"`
}

type outAccessor struct {
	BufferView    int       `json:"bufferView"`
	ComponentType int       `json:"componentType"`
	Count         int       `json:"count"`
	Type          string    `json:"type"`
	Min           []float32 `json:"min,omitempty"`
	Max           []float32 `json:"max,omitempty"`
}

type outView struct {
	Buffer     int `json:"buffer"`
	ByteOffset int `json:"byteOffset"`
	ByteLength int `json:"byteLength"`
}

type outBuffer struct {
	ByteLength int `json:"byteLength"`
}

type encoder struct {
	doc       outDoc
	bin       bytes.Buffer
	materials map[*render.Material]int
	textures  map[*render.Texture]int
}

const (
	componentFloat  = 5126
	componentUint   = 5125
	componentUshort = 5123
)

func (e *encoder) view(data []byte) int {
	for e.bin.Len()%4 != 0 {
		e.bin.WriteByte(0)
	}
	e.doc.BufferViews = append(e.doc.BufferViews, outView{ByteOffset: e.bin.Len(), ByteLength: len(data)})
	e.bin.Write(data)
	return len(e.doc.BufferViews) - 1
}

func (e *encoder) floats(v []float32, kind string, count int, bounds bool) int {
	buf := make([]byte, 4*len(v))
	for i, f := range v {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(f))
	}
	a := outAccessor{BufferView: e.view(buf), ComponentType: componentFloat, Count: count, Type: kind}
	if bounds && count > 0 {
		n := len(v) / count
		a.Min, a.Max = append([]float32(nil), v[:n]...), append([]float32(nil), v[:n]...)
		for i := n; i < len(v); i++ {
			a.Min[i%n] = min(a.Min[i%n], v[i])
			a.Max[i%n] = max(a.Max[i%n], v[i])
		}
	}
	e.doc.Accessors = append(e.doc.Accessors, a)
	return len(e.doc.Accessors) - 1
}

func (e *encoder) uints(v []uint32) int {
	buf := make([]byte, 4*len(v))
	for i, x := range v {
		binary.LittleEndian.PutUint32(buf[i*4:], x)
	}
	e.doc.Accessors = append(e.doc.Accessors, outAccessor{
		BufferView: e.view(buf), ComponentType: componentUint, Count: len(v), Type: typeScalar,
	})
	return len(e.doc.Accessors) - 1
}

func (e *encoder) joints(inf []anim.Influence) int {
	buf := make([]byte, 8*len(inf))
	for i, in := range inf {
		for k, j := range in.Joints {
			binary.LittleEndian.PutUint16(buf[i*8+k*2:], j)
		}
	}
	e.doc.Accessors = append(e.doc.Accessors, outAccessor{
		BufferView: e.view(buf), ComponentType: componentUshort, Count: len(inf), Type: typeVec4,
	})
	return len(e.doc.Accessors) - 1
}

func (e *encoder) encode(m *Model) ([]byte, error) {
	e.doc.Asset = map[string]string{"version": "2.0", "generator": "ember"}
	sk := m.Skeleton
	if sk == nil {
		sk = &anim.Skeleton{}
	}
	roots, err := e.meshes(m, sk, e.nodes(m, sk))
	if err != nil {
		return nil, err
	}
	for _, c := range m.Clips {
		e.clip(c)
	}
	e.doc.Scenes = []outScene{{Name: m.Name, Nodes: roots}}
	e.doc.Extensions, e.doc.Extras = m.Extensions, m.Extras
	e.listExtensions(m.ExtensionsRequired)
	if e.bin.Len() > 0 {
		e.doc.Buffers = []outBuffer{{ByteLength: e.bin.Len()}}
	}
	js, err := json.Marshal(e.doc)
	if err != nil {
		return nil, err
	}
	return glb(js, e.bin.Bytes())
}

// nodes writes the skeleton's joints as nodes and returns the roots.
func (e *encoder) nodes(m *Model, sk *anim.Skeleton) []int {
	var roots []int
	for i, j := range sk.Joints {
		n := outNode{Name: j.Name}
		if j.Rest.T != (geom.Vec3{}) {
			n.Translation = &[3]float32{j.Rest.T.X, j.Rest.T.Y, j.Rest.T.Z}
		}
		if r := j.Rest.R; r != (geom.Quat{}) && r != geom.QuatIdent() {
			n.Rotation = &[4]float32{r.X, r.Y, r.Z, r.W}
		}
		if s := j.Rest.S; s != (geom.Vec3{}) && s != geom.V3(1, 1, 1) {
			n.Scale = &[3]float32{s.X, s.Y, s.Z}
		}
		if i < len(m.Nodes) {
			n.Extensions, n.Extras = m.Nodes[i].Extensions, m.Nodes[i].Extras
		}
		if c, ok := m.Cameras[i]; ok {
			ci := len(e.doc.Cameras)
			e.doc.Cameras = append(e.doc.Cameras, cameraOf(c))
			n.Camera = &ci
		}
		e.doc.Nodes = append(e.doc.Nodes, n)
		if j.Parent < 0 || j.Parent >= i {
			roots = append(roots, i)
		} else {
			e.doc.Nodes[j.Parent].Children = append(e.doc.Nodes[j.Parent].Children, i)
		}
	}
	return roots
}

// meshes writes the parts as meshes on their nodes, adding a root node for
// each part that has none, and returns the roots.
func (e *encoder) meshes(m *Model, sk *anim.Skeleton, roots []int) ([]int, error) {
	rest := sk.Globals(sk.RestPose(), nil)
	skins := map[*anim.Skin]int{}
	type slot struct {
		mesh int
		skin *anim.Skin
	}
	meshAt := map[int]slot{}
	for _, p := range m.Parts {
		node, skin, space := place(sk, rest, p)
		if at, ok := meshAt[node]; node >= 0 && ok && at.skin != skin {
			node, skin, space = -1, p.Skin, geom.Ident()
		}
		prim, err := e.primitive(p, skin != nil, space)
		if err != nil {
			return nil, fmt.Errorf("gltf: part %q: %w", p.Name, err)
		}
		if at, ok := meshAt[node]; node >= 0 && ok {
			e.doc.Meshes[at.mesh].Primitives = append(e.doc.Meshes[at.mesh].Primitives, prim)
			continue
		}
		e.doc.Meshes = append(e.doc.Meshes, outMesh{Name: p.Name, Primitives: []outPrimitive{prim}})
		mi := len(e.doc.Meshes) - 1
		if node < 0 {
			e.doc.Nodes = append(e.doc.Nodes, outNode{Name: p.Name})
			node = len(e.doc.Nodes) - 1
			roots = append(roots, node)
		}
		e.doc.Nodes[node].Mesh = &mi
		meshAt[node] = slot{mi, skin}
		if skin != nil {
			si, ok := skins[skin]
			if !ok {
				si = e.skin(skin)
				skins[skin] = si
			}
			e.doc.Nodes[node].Skin = &si
		}
	}
	return roots, nil
}

// glb packs a JSON document and its binary buffer into a .glb file.
func glb(js, bin []byte) ([]byte, error) {
	for len(js)%4 != 0 {
		js = append(js, ' ')
	}
	for len(bin)%4 != 0 {
		bin = append(bin, 0)
	}
	total := 12 + 8 + len(js)
	if len(bin) > 0 {
		total += 8 + len(bin)
	}
	if total > math.MaxUint32 {
		return nil, errors.New("gltf: model too large for a glb")
	}
	var out bytes.Buffer
	w := func(v ...uint32) { _ = binary.Write(&out, binary.LittleEndian, v) }
	w(glbMagic, 2, uint32(total))
	w(uint32(len(js)), chunkJSON) //nolint:gosec // less than total, checked above
	out.Write(js)
	if len(bin) > 0 {
		w(uint32(len(bin)), chunkBIN) //nolint:gosec // less than total, checked above
		out.Write(bin)
	}
	return out.Bytes(), nil
}

// place picks the node a part's mesh goes on (-1: a new root node), the
// skin it is written with (nil: none) and the transform that takes its
// vertices from the model's space to that node's. Parts go back on the
// node they were loaded from (Joint, or the first joint of their name), so
// a file read and written again keeps its nodes: a part rigidly following
// a joint on that joint, unskinned, and a static part (baked in model
// space) on its node, unbaked.
func place(sk *anim.Skeleton, rest []geom.Mat4, p *Part) (int, *anim.Skin, geom.Mat4) {
	named := sk.Find(p.Name)
	if i := strings.LastIndex(p.Name, "."); named < 0 && i > 0 {
		named = sk.Find(p.Name[:i])
	}
	if named >= 0 && p.Joint != named && p.Joint < len(sk.Joints) && sk.Joints[p.Joint].Name == sk.Joints[named].Name {
		named = p.Joint
	}
	switch {
	case p.Skin == nil && named >= 0:
		return named, nil, rest[named].Inverse()
	case p.Skin == nil:
		return -1, nil, geom.Ident()
	case rigid(p.Skin):
		return p.Skin.Joints[0], nil, geom.Ident()
	}
	return named, p.Skin, geom.Ident()
}

func rigid(s *anim.Skin) bool {
	return len(s.Joints) == 1 && s.InverseBind[0] == geom.Ident()
}

func (e *encoder) skin(s *anim.Skin) int {
	ibm := make([]float32, 0, 16*len(s.InverseBind))
	for _, m := range s.InverseBind {
		ibm = append(ibm, m[:]...)
	}
	e.doc.Skins = append(e.doc.Skins, outSkin{
		InverseBindMatrices: e.floats(ibm, typeMat4, len(s.InverseBind), false),
		Joints:              append([]int(nil), s.Joints...),
	})
	return len(e.doc.Skins) - 1
}

func (e *encoder) primitive(p *Part, skinned bool, space geom.Mat4) (outPrimitive, error) {
	d := p.Mesh
	if d == nil || len(d.Vertices) == 0 {
		return outPrimitive{}, fmt.Errorf("no vertices")
	}
	if space != geom.Ident() {
		d = d.Transform(space)
	}
	n := len(d.Vertices)
	pos := make([]float32, 0, 3*n)
	nrm := make([]float32, 0, 3*n)
	uv := make([]float32, 0, 2*n)
	col := make([]float32, 0, 4*n)
	colored := false
	for _, v := range d.Vertices {
		pos = append(pos, v.Pos.X, v.Pos.Y, v.Pos.Z)
		nrm = append(nrm, v.Normal.X, v.Normal.Y, v.Normal.Z)
		uv = append(uv, v.UV.X, v.UV.Y)
		col = append(col, v.Color.R, v.Color.G, v.Color.B, v.Color.A)
		colored = colored || v.Color != color.White
	}
	attrs := map[string]int{
		"POSITION":   e.floats(pos, typeVec3, n, true),
		"NORMAL":     e.floats(nrm, typeVec3, n, false),
		"TEXCOORD_0": e.floats(uv, typeVec2, n, false),
	}
	if colored {
		attrs["COLOR_0"] = e.floats(col, typeVec4, n, false)
	}
	if skinned {
		if len(p.Influences) != n {
			return outPrimitive{}, fmt.Errorf("%d influences for %d vertices", len(p.Influences), n)
		}
		w := make([]float32, 0, 4*n)
		for _, in := range p.Influences {
			w = append(w, in.Weights[:]...)
		}
		attrs["JOINTS_0"] = e.joints(p.Influences)
		attrs["WEIGHTS_0"] = e.floats(w, typeVec4, n, false)
	}
	return outPrimitive{Attributes: attrs, Indices: e.uints(d.Indices), Material: e.material(p.Material)}, nil
}

func (e *encoder) material(m *render.Material) int {
	if m == nil {
		m = render.Mat(color.Gray(0.8))
	}
	if i, ok := e.materials[m]; ok {
		return i
	}
	out := outMaterial{DoubleSided: m.DoubleSided}
	out.PBR.BaseColorFactor = [4]float32{m.Albedo.R, m.Albedo.G, m.Albedo.B, m.Albedo.A}
	out.PBR.RoughnessFactor = geom.Clamp01(1 - m.Specular/0.6)
	if m.Transparent {
		out.AlphaMode = "BLEND"
	}
	if c := m.Emissive; c.R != 0 || c.G != 0 || c.B != 0 {
		strength := max(c.R, c.G, c.B, 1)
		out.EmissiveFactor = []float32{c.R / strength, c.G / strength, c.B / strength}
		if strength > 1 {
			out.Extensions = map[string]any{"KHR_materials_emissive_strength": map[string]float32{"emissiveStrength": strength}}
		}
	}
	if m.Unlit {
		if out.Extensions == nil {
			out.Extensions = map[string]any{}
		}
		out.Extensions["KHR_materials_unlit"] = map[string]any{}
	}
	if m.Texture != nil && m.Texture.Image != nil {
		out.PBR.BaseColorTexture = &textureRef{Index: e.texture(m.Texture)}
	}
	e.doc.Materials = append(e.doc.Materials, out)
	e.materials[m] = len(e.doc.Materials) - 1
	return e.materials[m]
}

func (e *encoder) texture(t *render.Texture) int {
	if i, ok := e.textures[t]; ok {
		return i
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, t.Image)
	e.doc.Images = append(e.doc.Images, outImage{BufferView: e.view(buf.Bytes()), MimeType: "image/png"})
	s := outSampler{MagFilter: 9729, MinFilter: 9987, WrapS: 10497, WrapT: 10497}
	if t.Filter == render.FilterNearest {
		s.MagFilter, s.MinFilter = 9728, 9728
	} else if !t.Mipmaps {
		s.MinFilter = 9729
	}
	if !t.Repeat {
		s.WrapS, s.WrapT = 33071, 33071
	}
	e.doc.Samplers = append(e.doc.Samplers, s)
	e.doc.Textures = append(e.doc.Textures, outTexture{Source: len(e.doc.Images) - 1, Sampler: len(e.doc.Samplers) - 1})
	e.textures[t] = len(e.doc.Textures) - 1
	return e.textures[t]
}

func (e *encoder) clip(c *anim.Clip) {
	a := outAnimation{Name: c.Name}
	for _, ch := range c.Channels {
		if len(ch.Times) == 0 || ch.Joint < 0 || ch.Joint >= len(e.doc.Nodes) {
			continue
		}
		path, kind, comps := "translation", typeVec3, 3
		switch ch.Path {
		case anim.Rotation:
			path, kind, comps = "rotation", typeVec4, 4
		case anim.Scale:
			path = "scale"
		case anim.Translation: // translation is the default above
		}
		values := make([]float32, 0, comps*len(ch.Values))
		for _, v := range ch.Values {
			values = append(values, []float32{v.X, v.Y, v.Z, v.W}[:comps]...)
		}
		interp := "LINEAR"
		switch ch.Interpolation {
		case anim.Step:
			interp = "STEP"
		case anim.CubicSpline:
			interp = "CUBICSPLINE"
		case anim.Linear: // LINEAR is the default above
		}
		s := outAnimSample{
			Input:         e.floats(ch.Times, typeScalar, len(ch.Times), true),
			Output:        e.floats(values, kind, len(ch.Values), false),
			Interpolation: interp,
		}
		a.Samplers = append(a.Samplers, s)
		out := outChannel{Sampler: len(a.Samplers) - 1}
		out.Target.Node, out.Target.Path = ch.Joint, path
		a.Channels = append(a.Channels, out)
	}
	if len(a.Channels) > 0 {
		e.doc.Animations = append(e.doc.Animations, a)
	}
}

// listExtensions fills extensionsUsed with every extension the document
// carries, and extensionsRequired with the required ones it still uses.
// Ember's own extensions are never required, so other tools open the file.
func (e *encoder) listExtensions(required []string) {
	used := map[string]bool{}
	for k := range e.doc.Extensions {
		used[k] = true
	}
	for _, n := range e.doc.Nodes {
		for k := range n.Extensions {
			used[k] = true
		}
	}
	for _, m := range e.doc.Materials {
		for k := range m.Extensions {
			used[k] = true
		}
	}
	e.doc.ExtensionsUsed = slices.Sorted(maps.Keys(used))
	for _, r := range required {
		if used[r] && !strings.HasPrefix(r, "EMBER_") {
			e.doc.ExtensionsRequired = append(e.doc.ExtensionsRequired, r)
		}
	}
}
