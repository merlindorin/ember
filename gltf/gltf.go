// Package gltf loads glTF 2.0 models (.glb, or .gltf with embedded data)
// into engine data: meshes, textured materials, a skeleton and animation
// clips. It is pure Go, so dedicated servers can load the same skeletons
// and clips as clients.
//
//	model, err := gltf.Load("assets/hero.glb")
//	hero := model.Spawn(world, scene.At(0, 0, 0))
//	hero.Play("Run", 0.2)
package gltf

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg" // JPEG textures
	_ "image/png"  // PNG textures
	"io/fs"
	"math"
	"net/url"
	"os"
	pathpkg "path"
	"path/filepath"
	"strings"

	"github.com/lantervale/ember/anim"
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/mesh"
	"github.com/lantervale/ember/render"
)

// Model is a loaded glTF scene.
type Model struct {
	Name  string
	Parts []*Part
	// Skeleton holds every node of the scene, parents first, so skinned
	// and rigidly attached parts follow animations alike.
	Skeleton *anim.Skeleton
	Clips    []*anim.Clip
	// Bounds encloses the parts in their rest pose.
	Bounds geom.AABB

	// Cameras are the glTF cameras, by the joint that holds them.
	Cameras map[int]render.Camera
	// Nodes holds each skeleton joint's glTF extensions and extras, which
	// the engine does not interpret and Encode writes back.
	Nodes      []Raw
	Extensions map[string]json.RawMessage
	Extras     json.RawMessage
	// ExtensionsRequired is the file's own list; Encode keeps the entries
	// it still writes.
	ExtensionsRequired []string
}

// Raw is glTF data kept as read.
type Raw struct {
	Extensions map[string]json.RawMessage
	Extras     json.RawMessage
}

func extensions(m map[string]json.RawMessage) map[string]json.RawMessage {
	if m == nil {
		return map[string]json.RawMessage{}
	}
	return m
}

// Part is one mesh primitive with its material. Skinned parts carry a
// Skin and per-vertex Influences; with no clips in the file, parts are
// baked in model space and Skin is nil.
type Part struct {
	Name       string
	Mesh       *mesh.Data
	Material   *render.Material
	Skin       *anim.Skin
	Influences []anim.Influence
	// Joint is the skeleton joint (glTF node) holding the part's mesh.
	Joint int
}

// Animated reports whether the model has clips to play.
func (m *Model) Animated() bool { return len(m.Clips) > 0 }

// Clip returns a clip by name.
func (m *Model) Clip(name string) *anim.Clip {
	for _, c := range m.Clips {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// Load reads a model from disk, with its external buffers and images.
func Load(path string) (*Model, error) {
	dir, file := filepath.Split(path)
	if dir == "" {
		dir = "."
	}
	return LoadFS(os.DirFS(dir), file)
}

// LoadFS reads a model from a file system (embed.FS, zip.Reader,
// os.DirFS). External buffers and images are read next to it.
func LoadFS(fsys fs.FS, name string) (*Model, error) {
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return nil, err
	}
	dir := pathpkg.Dir(name)
	return DecodeWith(data, func(uri string) ([]byte, error) {
		if u, uerr := url.PathUnescape(uri); uerr == nil {
			uri = u
		}
		return fs.ReadFile(fsys, pathpkg.Clean(pathpkg.Join(dir, uri)))
	})
}

// Resolver reads a file a .gltf document refers to by relative URI.
type Resolver func(uri string) ([]byte, error)

const (
	glbMagic  = 0x46546C67
	chunkJSON = 0x4E4F534A
	chunkBIN  = 0x004E4942
)

// Decode parses a .glb file or a .gltf document whose buffers and images
// are embedded as data URIs.
func Decode(data []byte) (*Model, error) { return DecodeWith(data, nil) }

// DecodeWith parses a model, reading external buffers and images through
// resolve.
func DecodeWith(data []byte, resolve Resolver) (*Model, error) {
	var doc document
	var bin []byte
	if len(data) >= 12 && binary.LittleEndian.Uint32(data) == glbMagic {
		js, b, err := splitGLB(data)
		if err != nil {
			return nil, err
		}
		if err = json.Unmarshal(js, &doc); err != nil {
			return nil, fmt.Errorf("gltf: %w", err)
		}
		bin = b
	} else if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("gltf: %w", err)
	}
	l := &loader{doc: &doc, textures: map[int]*render.Texture{}, resolve: resolve}
	for i, b := range doc.Buffers {
		switch {
		case b.URI == "" && i == 0:
			l.buffers = append(l.buffers, bin)
		case strings.HasPrefix(b.URI, "data:"):
			d, err := dataURI(b.URI)
			if err != nil {
				return nil, err
			}
			l.buffers = append(l.buffers, d)
		case resolve != nil:
			d, err := resolve(b.URI)
			if err != nil {
				return nil, fmt.Errorf("gltf: buffer %q: %w", b.URI, err)
			}
			l.buffers = append(l.buffers, d)
		default:
			return nil, fmt.Errorf("gltf: external buffer %q needs a file system, use Load or LoadFS", b.URI)
		}
	}
	return l.model()
}

func splitGLB(data []byte) ([]byte, []byte, error) {
	var js, bin []byte
	total := int(binary.LittleEndian.Uint32(data[8:]))
	if total > len(data) {
		return nil, nil, errors.New("gltf: truncated glb")
	}
	for off := 12; off+8 <= total; {
		n := int(binary.LittleEndian.Uint32(data[off:]))
		kind := binary.LittleEndian.Uint32(data[off+4:])
		if off+8+n > total {
			return nil, nil, errors.New("gltf: truncated glb chunk")
		}
		chunk := data[off+8 : off+8+n]
		switch kind {
		case chunkJSON:
			js = chunk
		case chunkBIN:
			bin = chunk
		}
		off += 8 + (n+3)&^3
	}
	if js == nil {
		return nil, nil, errors.New("gltf: glb has no JSON chunk")
	}
	return js, bin, nil
}

func dataURI(uri string) ([]byte, error) {
	i := strings.Index(uri, ",")
	if i < 0 || !strings.Contains(uri[:i], ";base64") {
		return nil, errors.New("gltf: unsupported data URI")
	}
	return base64.StdEncoding.DecodeString(uri[i+1:])
}

type loader struct {
	doc      *document
	buffers  [][]byte
	textures map[int]*render.Texture
	mats     []*render.Material
	joint    map[int]int
	resolve  Resolver
}

func (l *loader) model() (*Model, error) {
	d := l.doc
	m := &Model{
		Skeleton:           &anim.Skeleton{},
		Extensions:         extensions(d.Extensions),
		Extras:             d.Extras,
		ExtensionsRequired: d.ExtensionsRequired,
	}
	sceneIdx := d.Scene
	if sceneIdx < 0 || sceneIdx >= len(d.Scenes) {
		sceneIdx = 0
	}
	var roots []int
	if len(d.Scenes) > 0 {
		roots = d.Scenes[sceneIdx].Nodes
		m.Name = d.Scenes[sceneIdx].Name
	} else {
		roots = d.rootNodes()
	}
	if len(roots) == 1 {
		m.Name = d.Nodes[roots[0]].Name
	}
	l.joint = map[int]int{}
	for _, r := range roots {
		if err := l.visit(m, r, -1); err != nil {
			return nil, err
		}
	}
	for i := range d.Materials {
		mat, err := l.material(i)
		if err != nil {
			return nil, err
		}
		l.mats = append(l.mats, mat)
	}
	for i := range d.Animations {
		c, err := l.clip(i)
		if err != nil {
			return nil, err
		}
		m.Clips = append(m.Clips, c)
	}
	rest := m.Skeleton.Globals(m.Skeleton.RestPose(), nil)
	first := true
	for node, j := range l.joint {
		n := &d.Nodes[node]
		if n.Mesh == nil {
			continue
		}
		parts, err := l.parts(*n.Mesh, n, j, rest, m.Animated())
		if err != nil {
			return nil, err
		}
		for _, p := range parts {
			b := restBounds(p, rest)
			if first {
				m.Bounds, first = b, false
			} else {
				m.Bounds = m.Bounds.Union(b)
			}
		}
		m.Parts = append(m.Parts, parts...)
	}
	sortParts(m.Parts)
	return m, nil
}

// visit adds a node and its subtree to the model's skeleton.
func (l *loader) visit(m *Model, node, parent int) error {
	d := l.doc
	if node < 0 || node >= len(d.Nodes) {
		return fmt.Errorf("gltf: node %d out of range", node)
	}
	if _, seen := l.joint[node]; seen {
		return fmt.Errorf("gltf: node %d appears twice in the hierarchy", node)
	}
	n := &d.Nodes[node]
	l.joint[node] = len(m.Skeleton.Joints)
	m.Skeleton.Joints = append(m.Skeleton.Joints, anim.Joint{Name: n.Name, Parent: parent, Rest: n.trs()})
	m.Nodes = append(m.Nodes, Raw{Extensions: extensions(n.Extensions), Extras: n.Extras})
	if n.Camera != nil {
		if *n.Camera < 0 || *n.Camera >= len(d.Cameras) {
			return fmt.Errorf("gltf: camera %d out of range", *n.Camera)
		}
		if m.Cameras == nil {
			m.Cameras = map[int]render.Camera{}
		}
		m.Cameras[len(m.Nodes)-1] = d.Cameras[*n.Camera].camera()
	}
	self := l.joint[node]
	for _, c := range n.Children {
		if err := l.visit(m, c, self); err != nil {
			return err
		}
	}
	return nil
}

func restBounds(p *Part, rest []geom.Mat4) geom.AABB {
	if p.Skin == nil {
		return p.Mesh.Bounds()
	}
	out := p.Mesh.Clone()
	anim.Deform(p.Mesh, p.Influences, p.Skin.Palette(rest, nil), out)
	return out.Bounds()
}

func (l *loader) parts(meshIdx int, n *node, joint int, rest []geom.Mat4, animated bool) ([]*Part, error) {
	d := l.doc
	if meshIdx < 0 || meshIdx >= len(d.Meshes) {
		return nil, fmt.Errorf("gltf: mesh %d out of range", meshIdx)
	}
	gm := &d.Meshes[meshIdx]
	var skin *anim.Skin
	if n.Skin != nil {
		s, err := l.skin(*n.Skin)
		if err != nil {
			return nil, err
		}
		skin = s
	} else if animated {
		skin = &anim.Skin{Joints: []int{joint}, InverseBind: []geom.Mat4{geom.Ident()}}
	}
	var out []*Part
	for pi, prim := range gm.Primitives {
		if prim.Mode != nil && *prim.Mode != 4 {
			continue
		}
		data, influences, err := l.primitive(&prim)
		if err != nil {
			return nil, fmt.Errorf("gltf: mesh %q primitive %d: %w", gm.Name, pi, err)
		}
		p := &Part{Name: n.Name, Mesh: data, Skin: skin, Joint: joint}
		if len(gm.Primitives) > 1 {
			p.Name = fmt.Sprintf("%s.%d", n.Name, pi)
		}
		switch {
		case n.Skin != nil:
			if influences == nil {
				return nil, fmt.Errorf("gltf: skinned mesh %q has no JOINTS_0/WEIGHTS_0", gm.Name)
			}
			p.Influences = influences
		case animated:
			p.Influences = make([]anim.Influence, len(data.Vertices))
			for i := range p.Influences {
				p.Influences[i].Weights[0] = 1
			}
		default:
			p.Mesh = data.Transform(rest[joint])
		}
		if prim.Material != nil && *prim.Material < len(l.mats) {
			p.Material = l.mats[*prim.Material]
		} else {
			p.Material = render.Mat(color.Gray(0.8))
		}
		out = append(out, p)
	}
	return out, nil
}

func (l *loader) primitive(p *primitive) (*mesh.Data, []anim.Influence, error) {
	d, err := l.vertices(p)
	if err != nil {
		return nil, nil, err
	}
	n := len(d.Vertices)
	if d.Indices, err = l.indices(p, n); err != nil {
		return nil, nil, err
	}
	if _, ok := p.Attributes["NORMAL"]; !ok {
		d.RecomputeNormals()
	}
	inf, err := l.influences(p, n)
	if err != nil {
		return nil, nil, err
	}
	return d, inf, nil
}

// attribute reads a vertex attribute, reporting whether the primitive has it.
func (l *loader) attribute(p *primitive, name string, comps int) ([]float32, bool, error) {
	idx, ok := p.Attributes[name]
	if !ok {
		return nil, false, nil
	}
	v, err := l.accessor(idx, comps)
	return v, true, err
}

func (l *loader) vertices(p *primitive) (*mesh.Data, error) {
	pos, ok, err := l.attribute(p, "POSITION", 3)
	if !ok {
		return nil, errors.New("no POSITION")
	}
	if err != nil {
		return nil, err
	}
	n := len(pos) / 3
	d := &mesh.Data{Vertices: make([]mesh.Vertex, n)}
	for i := range d.Vertices {
		d.Vertices[i] = mesh.Vertex{Pos: geom.V3(pos[i*3], pos[i*3+1], pos[i*3+2]), Color: color.White}
	}
	nrm, _, err := l.attribute(p, "NORMAL", 3)
	if err != nil {
		return nil, err
	}
	for i := range min(n, len(nrm)/3) {
		d.Vertices[i].Normal = geom.V3(nrm[i*3], nrm[i*3+1], nrm[i*3+2])
	}
	uv, _, err := l.attribute(p, "TEXCOORD_0", 2)
	if err != nil {
		return nil, err
	}
	for i := range min(n, len(uv)/2) {
		d.Vertices[i].UV = geom.V2(uv[i*2], uv[i*2+1])
	}
	if err = l.colors(p, d); err != nil {
		return nil, err
	}
	return d, nil
}

func (l *loader) colors(p *primitive, d *mesh.Data) error {
	idx, ok := p.Attributes["COLOR_0"]
	if !ok {
		return nil
	}
	comps := l.doc.Accessors[idx].components()
	v, err := l.accessor(idx, comps)
	if err != nil {
		return err
	}
	for i := range min(len(d.Vertices), len(v)/comps) {
		c := color.RGB(v[i*comps], v[i*comps+1], v[i*comps+2])
		if comps == 4 {
			c.A = v[i*comps+3]
		}
		d.Vertices[i].Color = c
	}
	return nil
}

func (l *loader) indices(p *primitive, n int) ([]uint32, error) {
	if p.Indices == nil {
		out := make([]uint32, n)
		for i := range out {
			out[i] = uint32(i)
		}
		return out, nil
	}
	v, err := l.accessor(*p.Indices, 1)
	if err != nil {
		return nil, err
	}
	out := make([]uint32, len(v))
	for i, x := range v {
		if int(x) >= n {
			return nil, fmt.Errorf("index %d out of range", int(x))
		}
		out[i] = uint32(x)
	}
	return out, nil
}

func (l *loader) influences(p *primitive, n int) ([]anim.Influence, error) {
	ji, hasJ := p.Attributes["JOINTS_0"]
	wi, hasW := p.Attributes["WEIGHTS_0"]
	if !hasJ || !hasW {
		return nil, nil
	}
	joints, err := l.accessorRaw(ji, 4)
	if err != nil {
		return nil, err
	}
	weights, err := l.accessor(wi, 4)
	if err != nil {
		return nil, err
	}
	inf := make([]anim.Influence, n)
	for i := range inf {
		for k := range 4 {
			if i*4+k < len(joints) && i*4+k < len(weights) {
				inf[i].Joints[k] = uint16(joints[i*4+k])
				inf[i].Weights[k] = weights[i*4+k]
			}
		}
	}
	return inf, nil
}

func (l *loader) skin(i int) (*anim.Skin, error) {
	d := l.doc
	if i < 0 || i >= len(d.Skins) {
		return nil, fmt.Errorf("gltf: skin %d out of range", i)
	}
	s := &d.Skins[i]
	out := &anim.Skin{}
	for _, node := range s.Joints {
		j, ok := l.joint[node]
		if !ok {
			return nil, fmt.Errorf("gltf: skin joint node %d is not in the scene", node)
		}
		out.Joints = append(out.Joints, j)
	}
	out.InverseBind = make([]geom.Mat4, len(out.Joints))
	if s.InverseBindMatrices == nil {
		for k := range out.InverseBind {
			out.InverseBind[k] = geom.Ident()
		}
		return out, nil
	}
	v, err := l.accessor(*s.InverseBindMatrices, 16)
	if err != nil {
		return nil, err
	}
	for k := range out.InverseBind {
		copy(out.InverseBind[k][:], v[k*16:])
	}
	return out, nil
}

func (l *loader) clip(i int) (*anim.Clip, error) {
	a := &l.doc.Animations[i]
	c := &anim.Clip{Name: a.Name, Start: float32(math.Inf(1))}
	if c.Name == "" {
		c.Name = fmt.Sprintf("Animation %d", i)
	}
	for _, ch := range a.Channels {
		if ch.Target.Node == nil || ch.Sampler >= len(a.Samplers) {
			continue
		}
		j, ok := l.joint[*ch.Target.Node]
		if !ok {
			continue
		}
		path, comps, ok := channelPath(ch.Target.Path)
		if !ok {
			continue
		}
		s := a.Samplers[ch.Sampler]
		times, err := l.accessor(s.Input, 1)
		if err != nil {
			return nil, err
		}
		values, err := l.accessor(s.Output, comps)
		if err != nil {
			return nil, err
		}
		out := anim.Channel{Joint: j, Path: path, Times: times}
		switch s.Interpolation {
		case "STEP":
			out.Interpolation = anim.Step
		case "CUBICSPLINE":
			out.Interpolation = anim.CubicSpline
		}
		per := 1
		if out.Interpolation == anim.CubicSpline {
			per = 3
		}
		if len(values)/comps < len(times)*per {
			return nil, fmt.Errorf("gltf: animation %q has fewer values than keys", c.Name)
		}
		out.Values = make([]geom.Vec4, len(times)*per)
		for k := range out.Values {
			v := values[k*comps:]
			if comps == 4 {
				out.Values[k] = geom.V4(v[0], v[1], v[2], v[3])
			} else {
				out.Values[k] = geom.V4(v[0], v[1], v[2], 0)
			}
		}
		if len(times) > 0 {
			c.Duration = max(c.Duration, times[len(times)-1])
			c.Start = min(c.Start, times[0])
		}
		c.Channels = append(c.Channels, out)
	}
	if len(c.Channels) == 0 {
		c.Start = 0
	}
	return c, nil
}

// channelPath maps a glTF target path to the joint property it animates
// and its number of components.
func channelPath(p string) (anim.Path, int, bool) {
	switch p {
	case "translation":
		return anim.Translation, 3, true
	case "rotation":
		return anim.Rotation, 4, true
	case "scale":
		return anim.Scale, 3, true
	}
	return 0, 0, false
}

func (l *loader) material(i int) (*render.Material, error) {
	gm := &l.doc.Materials[i]
	pbr := gm.PBR
	base := color.White
	if f := pbr.BaseColorFactor; len(f) == 4 {
		base = color.Color{R: f[0], G: f[1], B: f[2], A: f[3]}
	}
	rough := float32(1)
	if pbr.RoughnessFactor != nil {
		rough = *pbr.RoughnessFactor
	}
	m := &render.Material{
		Albedo:      base,
		Specular:    (1 - rough) * 0.6,
		Transparent: gm.AlphaMode == "BLEND",
		DoubleSided: gm.DoubleSided,
	}
	if f := gm.EmissiveFactor; len(f) == 3 {
		m.Emissive = color.RGB(f[0], f[1], f[2])
		if s := gm.Extensions.EmissiveStrength; s != nil {
			m.Emissive = m.Emissive.Scale(s.EmissiveStrength)
		}
	}
	m.Unlit = gm.Extensions.Unlit != nil
	if pbr.BaseColorTexture != nil {
		t, err := l.texture(pbr.BaseColorTexture.Index)
		if err != nil {
			return nil, fmt.Errorf("gltf: material %q: %w", gm.Name, err)
		}
		m.Texture = t
	}
	return m, nil
}

func (l *loader) texture(i int) (*render.Texture, error) {
	if t, ok := l.textures[i]; ok {
		return t, nil
	}
	d := l.doc
	if i < 0 || i >= len(d.Textures) || d.Textures[i].Source == nil || *d.Textures[i].Source >= len(d.Images) {
		return nil, fmt.Errorf("texture %d has no image", i)
	}
	tex := d.Textures[i]
	im := d.Images[*tex.Source]
	var raw []byte
	switch {
	case im.BufferView != nil:
		b, err := l.view(*im.BufferView)
		if err != nil {
			return nil, err
		}
		raw = b
	case strings.HasPrefix(im.URI, "data:"):
		b, err := dataURI(im.URI)
		if err != nil {
			return nil, err
		}
		raw = b
	case l.resolve != nil && im.URI != "":
		b, err := l.resolve(im.URI)
		if err != nil {
			return nil, fmt.Errorf("image %q: %w", im.URI, err)
		}
		raw = b
	default:
		return nil, fmt.Errorf("external image %q needs a file system, use Load or LoadFS", im.URI)
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("image %q: %w", im.Name, err)
	}
	t := &render.Texture{Image: img, Mipmaps: true, Repeat: true}
	if tex.Sampler != nil && *tex.Sampler < len(d.Samplers) {
		s := d.Samplers[*tex.Sampler]
		if s.MagFilter == 9728 {
			t.Filter, t.Mipmaps = render.FilterNearest, false
		}
		t.Repeat = s.WrapS != 33071 && s.WrapT != 33071
	}
	l.textures[i] = t
	return t, nil
}

func (l *loader) view(i int) ([]byte, error) {
	d := l.doc
	if i < 0 || i >= len(d.BufferViews) {
		return nil, fmt.Errorf("buffer view %d out of range", i)
	}
	v := d.BufferViews[i]
	if v.Buffer >= len(l.buffers) || v.ByteOffset+v.ByteLength > len(l.buffers[v.Buffer]) {
		return nil, fmt.Errorf("buffer view %d exceeds its buffer", i)
	}
	return l.buffers[v.Buffer][v.ByteOffset : v.ByteOffset+v.ByteLength], nil
}

// accessor reads an accessor as floats, normalizing integer data when the
// accessor says so.
func (l *loader) accessor(i, comps int) ([]float32, error) { return l.read(i, comps, true) }

// accessorRaw reads integer data without normalization (joint indices).
func (l *loader) accessorRaw(i, comps int) ([]float32, error) { return l.read(i, comps, false) }

func (l *loader) read(i, comps int, normalize bool) ([]float32, error) {
	d := l.doc
	if i < 0 || i >= len(d.Accessors) {
		return nil, fmt.Errorf("accessor %d out of range", i)
	}
	a := d.Accessors[i]
	if c := a.components(); c != comps {
		return nil, fmt.Errorf("accessor %d is %s, want %d components", i, a.Type, comps)
	}
	out := make([]float32, a.Count*comps)
	if a.BufferView == nil {
		return out, nil
	}
	size := componentSize(a.ComponentType)
	if size == 0 {
		return nil, fmt.Errorf("accessor %d has unknown component type %d", i, a.ComponentType)
	}
	data, err := l.view(*a.BufferView)
	if err != nil {
		return nil, err
	}
	stride := d.BufferViews[*a.BufferView].ByteStride
	if stride == 0 {
		stride = size * comps
	}
	if a.Count > 0 && a.ByteOffset+(a.Count-1)*stride+size*comps > len(data) {
		return nil, fmt.Errorf("accessor %d exceeds its buffer view", i)
	}
	norm := normalize && a.Normalized
	for e := range a.Count {
		base := a.ByteOffset + e*stride
		for c := range comps {
			out[e*comps+c] = component(data[base+c*size:], a.ComponentType, norm)
		}
	}
	return out, nil
}

func componentSize(t int) int {
	switch t {
	case 5120, 5121:
		return 1
	case 5122, 5123:
		return 2
	case 5125, 5126:
		return 4
	}
	return 0
}

func component(b []byte, t int, normalized bool) float32 {
	switch t {
	case 5126:
		return math.Float32frombits(binary.LittleEndian.Uint32(b))
	case 5121:
		if normalized {
			return float32(b[0]) / 255
		}
		return float32(b[0])
	case 5123:
		v := binary.LittleEndian.Uint16(b)
		if normalized {
			return float32(v) / 65535
		}
		return float32(v)
	case 5125:
		return float32(binary.LittleEndian.Uint32(b))
	case 5120:
		v := int8(b[0]) //nolint:gosec // the format stores signed bytes; this reads their bits
		if normalized {
			return max(float32(v)/127, -1)
		}
		return float32(v)
	case 5122:
		v := int16(binary.LittleEndian.Uint16(b)) //nolint:gosec // the format stores signed shorts; this reads their bits
		if normalized {
			return max(float32(v)/32767, -1)
		}
		return float32(v)
	}
	return 0
}
