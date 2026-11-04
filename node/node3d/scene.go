package node3d

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/merlindorin/ember/anim"
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/gltf"
	"github.com/merlindorin/ember/node"
	"github.com/merlindorin/ember/render"
	"github.com/merlindorin/ember/scene"
)

const lightsExtension = "KHR_lights_punctual"

// The KHR_lights_punctual types node3d reads and writes.
const (
	lightPoint       = "point"
	lightDirectional = "directional"
)

type lightDef struct {
	Type      string      `json:"type"`
	Color     *[3]float32 `json:"color,omitempty"`
	Intensity *float32    `json:"intensity,omitempty"`
	Range     float32     `json:"range,omitempty"`
}

type (
	transformed  interface{ GetTransform() *scene.Transform }
	viewer       interface{ GetCamera() *render.Camera }
	pointLit     interface{ GetPointLight() *render.PointLight }
	sunLit       interface{ GetSun() *Sun }
	meshInstance interface{ GetMesh() *Mesh }
)

func path(joints []anim.Joint, i int) string {
	var parts []string
	for ; i >= 0 && i < len(joints); i = joints[i].Parent {
		parts = append(parts, joints[i].Name)
	}
	slices.Reverse(parts)
	return strings.Join(parts, "/")
}

func lightsOf(m *gltf.Model) ([]lightDef, error) {
	raw, ok := m.Extensions[lightsExtension]
	if !ok {
		return nil, nil
	}
	var d struct {
		Lights []lightDef `json:"lights"`
	}
	if err := json.Unmarshal(raw, &d); err != nil {
		return nil, fmt.Errorf("node3d: %s: %w", lightsExtension, err)
	}
	return d.Lights, nil
}

func (l lightDef) color() color.Color {
	if l.Color == nil {
		return color.White
	}
	return color.RGB(l.Color[0], l.Color[1], l.Color[2])
}

func (l lightDef) intensity() float32 {
	if l.Intensity == nil {
		return 1
	}
	return *l.Intensity
}

// Instantiate builds a scene's node tree, owned by its root: each glTF
// node becomes the class its EMBER_node names, or else a Camera, a
// light, a Mesh or a Node from what glTF gives it. A scene
// with several roots gets a Node root, an animated one a Skeleton
// root. Instances and the base of an inherited scene come from resolve.
func Instantiate(s *node.Scene, resolve node.Resolver) (node.Noded, error) {
	return (&instancer{resolve: resolve}).instantiate(s)
}

// Instance instantiates s as an instance: its root and nodes are marked as
// coming from s, and the root no longer saves as s itself.
func Instance(s *node.Scene, resolve node.Resolver) (node.Noded, error) {
	return (&instancer{resolve: resolve}).instance(s, false)
}

type instancer struct {
	resolve node.Resolver
	stack   []string
}

func (in *instancer) instance(s *node.Scene, editable bool) (node.Noded, error) {
	e, err := in.instantiate(s)
	if err != nil {
		return nil, err
	}
	r := e.GetSceneNode()
	mark := &node.Instanced{Scene: s.ID, Root: r, EditableChildren: editable}
	r.Instance, r.File = mark, nil
	for _, d := range descendants(r) {
		d.GetSceneNode().Instance = mark
	}
	return e, nil
}

func (in *instancer) instantiate(s *node.Scene) (node.Noded, error) {
	m := s.Model
	if slices.Contains(in.stack, s.ID) && s.ID != "" {
		return nil, fmt.Errorf(
			"node3d: scene %s instances itself: cycle %s",
			s.ID,
			strings.Join(append(in.stack, s.ID), " → "),
		)
	}
	in.stack = append(in.stack, s.ID)
	defer func() { in.stack = in.stack[:len(in.stack)-1] }()
	lights, err := lightsOf(m)
	if err != nil {
		return nil, err
	}
	instances := map[string]node.Instance{}
	for _, i := range s.Instances {
		instances[i.Node] = i
	}
	seen := map[string]bool{}
	for _, n := range s.Nodes {
		seen[n.ID] = true
	}
	for id := range instances {
		if !seen[id] {
			return nil, fmt.Errorf("node3d: instance at node %s, which the scene does not have", id)
		}
	}
	res, err := node.LoadResources(s.Resources)
	if err != nil {
		return nil, err
	}
	b := &builder{in: in, scene: s, lights: lights, instances: instances, res: res}
	var root node.Noded
	var byID map[string]node.Noded
	if animated(m) {
		root, byID, err = instantiateAnimated(s, b)
	} else {
		root, byID, err = instantiateStatic(s, b)
	}
	if err != nil {
		return nil, err
	}
	r := root.GetSceneNode()
	if s.Inherits != "" && (r.Instance == nil || r.Instance.Root != r || r.Instance.Scene != s.Inherits) {
		return nil, fmt.Errorf("node3d: scene %s inherits %s, but its root is not an instance of it", s.ID, s.Inherits)
	}
	for _, e := range descendants(r) {
		if e.GetSceneNode().Owner() == nil {
			_ = e.GetSceneNode().SetOwner(r)
		}
	}
	ext := maps.Clone(m.Extensions)
	delete(ext, lightsExtension)
	r.File = &node.File{
		ID:       s.ID,
		Inherits: s.Inherits,
		Raw:      gltf.Raw{Extensions: ext, Extras: m.Extras},
		Required: m.ExtensionsRequired,
	}
	for _, c := range s.Connections {
		from, to := byID[c.From], byID[c.To]
		if from == nil || to == nil {
			return nil, fmt.Errorf("node3d: connection %s from %s to %s: no such node", c.Signal, c.From, c.To)
		}
		if err = node.ConnectMethod(from, c.Signal, to, c.Method, c.Flags|node.Persist); err != nil {
			return nil, fmt.Errorf("node3d: connection %s: %w", c.Signal, err)
		}
	}
	return root, nil
}

// builder makes the node of each glTF node: an instance where the scene
// places one, else a node of its class.
type builder struct {
	res       *node.Resources
	in        *instancer
	scene     *node.Scene
	lights    []lightDef
	instances map[string]node.Instance
}

func (b *builder) isInstance(sn node.SceneNode) bool {
	_, ok := b.instances[sn.ID]
	return ok && sn.ID != ""
}

func (b *builder) node(i int, sn node.SceneNode, parts []*gltf.Part, toLocal geom.Mat4) (node.Noded, error) {
	m := b.scene.Model
	inst, ok := b.instances[sn.ID]
	if !ok || sn.ID == "" {
		e, err := build(m, i, sn, b.lights, parts, toLocal, b.res)
		if err == nil && sn.InstanceOf != "" {
			e.GetSceneNode().Instance = &node.Instanced{Scene: sn.InstanceOf, Root: e.GetSceneNode(), Embedded: true}
		}
		return e, err
	}
	if len(parts) > 0 {
		return nil, fmt.Errorf("an instance of %s holds no mesh of its own", inst.Scene)
	}
	if b.in.resolve == nil {
		return nil, fmt.Errorf("instance of scene %s, with nothing to find scenes by ID", inst.Scene)
	}
	sub, err := b.in.resolve(inst.Scene)
	if err != nil {
		return nil, fmt.Errorf("instance of scene %s: %w", inst.Scene, err)
	}
	e, err := b.in.instance(sub, inst.EditableChildren)
	if err != nil {
		return nil, fmt.Errorf("instance of scene %s: %w", inst.Scene, err)
	}
	n := e.GetSceneNode()
	n.SetName(m.Skeleton.Joints[i].Name)
	if i < len(m.Nodes) {
		n.Raw = gltf.Raw{Extensions: maps.Clone(m.Nodes[i].Extensions), Extras: m.Nodes[i].Extras}
	}
	if err = apply(e, sn, b.res); err != nil {
		return nil, err
	}
	if t, isTransformed := e.(transformed); isTransformed {
		r := m.Skeleton.Joints[i].Rest
		tr := t.GetTransform()
		tr.Position, tr.Rotation, tr.Scale = r.T, r.R, r.S
	}
	for _, o := range inst.Overrides {
		var target *node.Node
		target, err = inner(n, o.Node, o.Path)
		if err != nil {
			return nil, err
		}
		for _, k := range slices.Sorted(maps.Keys(o.Properties)) {
			if err = node.SetPropertyWith(target.Entity(), k, o.Properties[k], b.res); err != nil {
				return nil, fmt.Errorf("override of %s: %w", o.Path, err)
			}
		}
	}
	return e, nil
}

// inner finds a node of the instance rooted at r by its ID, or by its path
// when the ID is not unique there (nested instances of one scene).
func inner(r *node.Node, id, at string) (*node.Node, error) {
	var found []*node.Node
	for _, d := range descendants(r) {
		if n := d.GetSceneNode(); n.Instance == r.Instance && n.SceneID == id {
			found = append(found, n)
		}
	}
	if len(found) == 1 {
		return found[0], nil
	}
	if n := r.Get(at); n != nil && n.Instance == r.Instance {
		return n, nil
	}
	return nil, fmt.Errorf("the instance has no node %s (%s)", id, at)
}

func instantiateStatic(s *node.Scene, b *builder) (node.Noded, map[string]node.Noded, error) {
	m := s.Model
	joints := m.Skeleton.Joints
	globals := m.Skeleton.Globals(m.Skeleton.RestPose(), nil)
	parts := map[int][]*gltf.Part{}
	for _, p := range m.Parts {
		j := p.Joint
		if j < 0 || j >= len(joints) {
			j = -1
		}
		parts[j] = append(parts[j], p)
	}
	ents := make([]node.Noded, len(joints))
	byID := map[string]node.Noded{}
	for i, j := range joints {
		var sn node.SceneNode
		if i < len(s.Nodes) {
			sn = s.Nodes[i]
		}
		e, err := b.node(i, sn, parts[i], globals[i].Inverse())
		if err != nil {
			return nil, nil, fmt.Errorf("node3d: %s: %w", path(joints, i), err)
		}
		ents[i] = e
		if sn.ID != "" {
			byID[sn.ID] = e
		}
		if j.Parent >= 0 {
			ents[j.Parent].GetSceneNode().AddNode(e)
		}
	}

	var roots []node.Noded
	for i, j := range joints {
		if j.Parent < 0 {
			roots = append(roots, ents[i])
		}
	}
	if len(roots) == 1 && len(parts[-1]) == 0 && s.Root == nil {
		return roots[0], byID, nil
	}
	r, err := staticRoot(s, b, roots, parts[-1])
	if err != nil {
		return nil, nil, err
	}
	if s.Root != nil && s.Root.ID != "" {
		byID[s.Root.ID] = r
	}
	return r, byID, nil
}

// staticRoot is the node holding a static scene's roots and its loose
// surfaces.
func staticRoot(s *node.Scene, b *builder, roots []node.Noded, loose []*gltf.Part) (*Node, error) {
	r := NewNode()
	r.SetName(cmpOr(s.Model.Name, "Scene"))
	if s.Root != nil {
		if err := apply(r, *s.Root, b.res); err != nil {
			return nil, fmt.Errorf("node3d: root: %w", err)
		}
		r.SetName(cmpOr(s.Root.Name, r.Name()))
	}
	for _, e := range roots {
		r.AddNode(e)
	}
	if len(loose) > 0 {
		mi := NewMesh()
		mi.SetName("Mesh")
		mi.Surfaces = loose
		r.AddNode(mi)
	}
	return r, nil
}

func cmpOr(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func descendants(n *node.Node) []node.Noded {
	var out []node.Noded
	for _, c := range n.ChildNodes() {
		out = append(out, c.Entity())
		out = append(out, descendants(c)...)
	}
	return out
}

// apply sets the scene data of a node: ID, groups, process mode, unique
// name and properties.
func apply(e node.Noded, sn node.SceneNode, res *node.Resources) error {
	n := e.GetSceneNode()
	n.SceneID, n.ProcessMode = sn.ID, sn.ProcessMode
	for _, g := range sn.Groups {
		n.AddToGroup(g)
	}
	for _, k := range slices.Sorted(maps.Keys(sn.Properties)) {
		if err := node.SetPropertyWith(e, k, sn.Properties[k], res); err != nil {
			return err
		}
	}
	return nil
}

// build makes the node of joint i; toLocal takes its parts' meshes into
// the node's space.
func build(
	m *gltf.Model,
	i int,
	sn node.SceneNode,
	lights []lightDef,
	parts []*gltf.Part,
	toLocal geom.Mat4,
	res *node.Resources,
) (node.Noded, error) {
	var raw gltf.Raw
	if i < len(m.Nodes) {
		raw = gltf.Raw{Extensions: maps.Clone(m.Nodes[i].Extensions), Extras: m.Nodes[i].Extras}
	}
	light := -1
	if l, ok := raw.Extensions[lightsExtension]; ok {
		var ref struct {
			Light int `json:"light"`
		}
		if err := json.Unmarshal(l, &ref); err != nil || ref.Light < 0 || ref.Light >= len(lights) {
			return nil, fmt.Errorf("%s: bad light %s", lightsExtension, l)
		}
		light = ref.Light
		delete(raw.Extensions, lightsExtension)
	}
	camera, hasCamera := m.Cameras[i]

	class := sn.Type
	if class == "" {
		var err error
		if class, err = defaultClass(hasCamera, light, lights, len(parts) > 0); err != nil {
			return nil, err
		}
	}
	e, err := node.Create(class, sn.Behaviour)
	if err != nil {
		return nil, err
	}
	n := e.GetSceneNode()
	n.SetName(m.Skeleton.Joints[i].Name)
	n.Raw = raw
	if err = apply(e, sn, res); err != nil {
		return nil, err
	}
	if t, ok := e.(transformed); ok {
		r := m.Skeleton.Joints[i].Rest
		tr := t.GetTransform()
		tr.Position, tr.Rotation, tr.Scale = r.T, r.R, r.S
	}
	if hasCamera {
		v, ok := e.(viewer)
		if !ok {
			return nil, fmt.Errorf("it has a camera, but %s is not a camera", class)
		}
		*v.GetCamera() = camera
	}
	if light >= 0 {
		if err = setLight(e, lights[light], class); err != nil {
			return nil, err
		}
	}
	if len(parts) > 0 {
		mi, ok := e.(meshInstance)
		if !ok {
			return nil, fmt.Errorf("it has a mesh, but %s draws none", class)
		}
		for _, p := range parts {
			c := *p
			c.Mesh = p.Mesh.Transform(toLocal)
			mi.GetMesh().Surfaces = append(mi.GetMesh().Surfaces, &c)
		}
	}
	return e, nil
}

// defaultClass is the class of a glTF node that names none, from what it
// holds.
func defaultClass(hasCamera bool, light int, lights []lightDef, hasParts bool) (string, error) {
	switch {
	case hasCamera:
		return "node3d.Camera", nil
	case light >= 0 && lights[light].Type == lightPoint:
		return "node3d.PointLight", nil
	case light >= 0 && lights[light].Type == lightDirectional:
		return "node3d.Sun", nil
	case light >= 0:
		return "", fmt.Errorf("%s lights are not supported", lights[light].Type)
	case hasParts:
		return "node3d.Mesh", nil
	default:
		return "node3d.Node", nil
	}
}

func setLight(e node.Noded, l lightDef, class string) error {
	p, isPoint := e.(pointLit)
	sun, isSun := e.(sunLit)
	switch {
	case l.Type == lightPoint && isPoint:
		pl := p.GetPointLight()
		pl.Color, pl.Intensity, pl.Range = l.color(), l.intensity(), l.Range
	case l.Type == lightDirectional && isSun:
		d := sun.GetSun()
		d.Color, d.Energy = l.color(), l.intensity()
	default:
		return fmt.Errorf("it has a %s light, which %s cannot hold", l.Type, class)
	}
	return nil
}

// Pack turns root and the nodes it owns back into a scene, giving IDs to
// the scene and to nodes without one. Properties equal to their class's
// defaults are not written.
func Pack(root node.Noded, resolve node.Resolver) (*node.Scene, error) {
	r := root.GetSceneNode()
	if r.File == nil {
		r.File = &node.File{}
	}
	if r.File.ID == "" {
		r.File.ID = node.NewID()
	}
	return pack(root, &packer{root: r, resolve: resolve}, *r.File)
}

// Snapshot packs root and every node under it, owned or not, as a scene of
// its own, leaving the tree as it is but for IDs given to nodes without
// one. Instances stay instances, and unregistered nodes become Nodes. With
// keep, only the nodes it keeps are written, with their ancestors.
func Snapshot(root node.Noded, resolve node.Resolver, keep func(node.Noded) bool) (*node.Scene, error) {
	p := &packer{root: root.GetSceneNode(), resolve: resolve, live: true}
	if keep != nil {
		p.wanted = map[*node.Node]bool{}
		p.want(root, keep)
	}
	return pack(root, p, node.File{ID: node.NewID()})
}

// Within keeps the nodes whose world position is within radius of center.
func Within(center geom.Vec3, radius float32) func(node.Noded) bool {
	return func(e node.Noded) bool {
		t, ok := e.(transformed)
		return ok && t.GetTransform().World().Translation().Dist(center) <= radius
	}
}

func (p *packer) want(e node.Noded, keep func(node.Noded) bool) bool {
	n := e.GetSceneNode()
	in := keep(e)
	if n.Instance == nil || n.Instance.Root != n || n.Instance.Embedded {
		for _, c := range n.ChildNodes() {
			in = p.want(c.Entity(), keep) || in
		}
	}
	p.wanted[n] = in
	return in
}

func pack(root node.Noded, p *packer, file node.File) (*node.Scene, error) {
	r := root.GetSceneNode()
	m := &gltf.Model{
		Name: r.Name(), Skeleton: &anim.Skeleton{}, Cameras: map[int]render.Camera{},
		Extensions: maps.Clone(file.Raw.Extensions), Extras: file.Raw.Extras, ExtensionsRequired: file.Required,
	}
	if m.Extensions == nil {
		m.Extensions = map[string]json.RawMessage{}
	}
	s := &node.Scene{ID: file.ID, Model: m}
	if r.Instance != nil && r.Instance.Root == r && !r.Instance.Embedded {
		s.Inherits = r.Instance.Scene
	}
	p.scene = s
	var err error
	if k, ok := root.(interface{ GetSkeleton() *Skeleton }); ok && k.GetSkeleton().Model != nil {
		p.animated = true
		err = packAnimated(k.GetSkeleton(), root, s, p)
	} else {
		err = p.visit(root, -1)
	}
	if err != nil {
		return nil, err
	}
	globals := m.Skeleton.Globals(m.Skeleton.RestPose(), nil)
	for _, sp := range p.surfaces {
		c := *sp.part
		c.Name, c.Joint = m.Skeleton.Joints[sp.joint].Name, sp.joint
		if p.animated {
			c.Skin = &anim.Skin{Joints: []int{sp.joint}, InverseBind: []geom.Mat4{geom.Ident()}}
			c.Influences = make([]anim.Influence, len(c.Mesh.Vertices))
			for i := range c.Influences {
				c.Influences[i].Weights[0] = 1
			}
		} else {
			c.Mesh = sp.part.Mesh.Transform(globals[sp.joint])
		}
		m.Parts = append(m.Parts, &c)
	}
	s.Resources = p.res.Data()
	delete(m.Extensions, lightsExtension)
	if len(p.lights) > 0 {
		var b []byte
		b, err = json.Marshal(map[string][]lightDef{"lights": p.lights})
		if err != nil {
			return nil, err
		}
		m.Extensions[lightsExtension] = b
	}
	for _, e := range p.list {
		for _, c := range node.Connections(e) {
			if c.Target.Owner() != r && c.Target != r {
				continue
			}
			s.Connections = append(s.Connections, node.Connection{
				Signal: c.Signal, From: e.GetSceneNode().SceneID, To: c.Target.SceneID, Method: c.Method, Flags: c.Flags,
			})
		}
	}
	return s, nil
}

type packer struct {
	res     node.Resources
	resolve node.Resolver
	// live packs every node under the root, owned or not; wanted, when
	// set, limits it to the nodes kept and their ancestors.
	live     bool
	wanted   map[*node.Node]bool
	root     *node.Node
	scene    *node.Scene
	animated bool
	list     []node.Noded
	lights   []lightDef
	surfaces []struct {
		joint int
		part  *gltf.Part
	}
}

func (p *packer) sceneNode(e node.Noded) (node.SceneNode, error) {
	n := e.GetSceneNode()
	if n.SceneID == "" {
		n.SceneID = node.NewID()
	}
	class, behaviour, ok := node.ClassOf(e)
	switch {
	case !ok && p.live:
		return node.SceneNode{ID: n.SceneID, Type: "node.Empty", Groups: n.Groups(), ProcessMode: n.ProcessMode}, nil
	case !ok:
		return node.SceneNode{}, fmt.Errorf("node3d: %s is a %T, which is not a registered node type", n.Name(), e)
	}
	props, err := changed(e, class, behaviour, &p.res)
	if err != nil {
		return node.SceneNode{}, err
	}
	return node.SceneNode{
		ID:          n.SceneID,
		Type:        class,
		Behaviour:   behaviour,
		Groups:      n.Groups(),
		ProcessMode: n.ProcessMode,
		Properties:  props,
	}, nil
}

// visit appends e as a glTF node under joint parent, then the nodes under
// it that the root owns.
func (p *packer) visit(e node.Noded, parent int) error {
	m, s := p.scene.Model, p.scene
	n := e.GetSceneNode()
	placeholder := n.Instance != nil && n.Instance.Root == n && !n.Instance.Embedded
	if k, ok := e.(interface{ GetSkeleton() *Skeleton }); ok && !placeholder && k.GetSkeleton().Model != nil {
		return fmt.Errorf("node3d: %s is a node3d.Skeleton inside a scene; place its scene as an instance", n.Name())
	}
	var sn node.SceneNode
	var err error
	if placeholder {
		sn, err = p.instance(e)
	} else {
		sn, err = p.sceneNode(e)
	}
	if err != nil {
		return err
	}
	if n.Instance != nil && n.Instance.Root == n && n.Instance.Embedded {
		sn.InstanceOf = n.Instance.Scene
	}
	i := len(m.Skeleton.Joints)
	rest := anim.TRS{R: geom.QuatIdent(), S: geom.V3(1, 1, 1)}
	if _, socketed := e.(*Socket); !socketed {
		if t, ok := e.(transformed); ok {
			tr := t.GetTransform()
			rest = anim.TRS{T: tr.Position, R: tr.Rotation, S: tr.Scale}
		}
	}
	m.Skeleton.Joints = append(m.Skeleton.Joints, anim.Joint{Name: n.Name(), Parent: parent, Rest: rest})
	raw := gltf.Raw{Extensions: maps.Clone(n.Raw.Extensions), Extras: n.Raw.Extras}
	if raw.Extensions == nil {
		raw.Extensions = map[string]json.RawMessage{}
	}
	if v, ok := e.(viewer); ok && !placeholder {
		m.Cameras[i] = *v.GetCamera()
	}
	var light *lightDef
	switch lit := e.(type) {
	case pointLit:
		l := lit.GetPointLight()
		light = &lightDef{
			Type:      lightPoint,
			Color:     &[3]float32{l.Color.R, l.Color.G, l.Color.B},
			Intensity: &l.Intensity,
			Range:     l.Range,
		}
	case sunLit:
		sun := lit.GetSun()
		light = &lightDef{
			Type:      lightDirectional,
			Color:     &[3]float32{sun.Color.R, sun.Color.G, sun.Color.B},
			Intensity: &sun.Energy,
		}
	}
	if light != nil && !placeholder {
		raw.Extensions[lightsExtension] = json.RawMessage(fmt.Sprintf(`{"light":%d}`, len(p.lights)))
		p.lights = append(p.lights, *light)
	}
	if mi, ok := e.(meshInstance); ok && !placeholder {
		for _, part := range mi.GetMesh().Surfaces {
			p.surfaces = append(p.surfaces, struct {
				joint int
				part  *gltf.Part
			}{i, part})
		}
	}
	m.Nodes = append(m.Nodes, raw)
	s.Nodes = append(s.Nodes, sn)
	p.list = append(p.list, e)
	return p.visitChildren(n, i, placeholder)
}

// visitChildren visits the children of n, the node at joint i, that the
// pack takes.
func (p *packer) visitChildren(n *node.Node, i int, placeholder bool) error {
	for _, c := range n.ChildNodes() {
		switch {
		case placeholder && c.Instance == n.Instance:
			for _, d := range append([]node.Noded{c.Entity()}, descendants(c)...) {
				if d.GetSceneNode().Owner() == p.root && d.GetSceneNode().Instance != n.Instance {
					return fmt.Errorf(
						"node3d: %s is added inside instance %s, which is not supported yet",
						d.GetSceneNode().Name(),
						n.Name(),
					)
				}
			}
		case p.wanted != nil && !p.wanted[c]:
		case c.Owner() == p.root || p.live:
			if err := p.visit(c.Entity(), i); err != nil {
				return err
			}
		}
	}
	return nil
}

// instance writes the placeholder of an instance root and records the
// instance, with what differs from a fresh instance of its scene.
func (p *packer) instance(e node.Noded) (node.SceneNode, error) {
	n := e.GetSceneNode()
	if p.resolve == nil {
		return node.SceneNode{}, fmt.Errorf(
			"node3d: saving instance %s of scene %s needs a resolver",
			n.Name(),
			n.Instance.Scene,
		)
	}
	base, err := p.resolve(n.Instance.Scene)
	if err != nil {
		return node.SceneNode{}, fmt.Errorf("node3d: instance %s: %w", n.Name(), err)
	}
	fresh, err := Instance(base, p.resolve)
	if err != nil {
		return node.SceneNode{}, fmt.Errorf("node3d: instance %s: %w", n.Name(), err)
	}
	if n.SceneID == "" || n.SceneID == fresh.GetSceneNode().SceneID {
		n.SceneID = node.NewID()
	}
	props, err := differences(e, fresh, true, &p.res)
	if err != nil {
		return node.SceneNode{}, err
	}
	sn := node.SceneNode{ID: n.SceneID, Groups: n.Groups(), ProcessMode: n.ProcessMode, Properties: props}
	inst := node.Instance{Node: n.SceneID, Scene: n.Instance.Scene, EditableChildren: n.Instance.EditableChildren}
	fr := fresh.GetSceneNode()
	for _, d := range descendants(n) {
		m := d.GetSceneNode()
		if m.Instance != n.Instance {
			continue
		}
		at := relative(n, m)
		var twin *node.Node
		twin, err = inner(fr, m.SceneID, at)
		if err != nil {
			return node.SceneNode{}, fmt.Errorf("node3d: instance %s: %w", n.Name(), err)
		}
		var diff map[string]json.RawMessage
		diff, err = differences(d, twin.Entity(), false, &p.res)
		if err != nil {
			return node.SceneNode{}, err
		}
		if len(diff) > 0 {
			inst.Overrides = append(inst.Overrides,
				node.Override{Node: m.SceneID, Path: relative(fr, twin), Properties: diff})
		}
	}
	p.scene.Instances = append(p.scene.Instances, inst)
	return sn, nil
}

func relative(from, to *node.Node) string {
	var parts []string
	for n := to; n != nil && n != from; n = n.ParentNode() {
		parts = append(parts, n.Name())
	}
	slices.Reverse(parts)
	return strings.Join(parts, "/")
}

// inGLTF lists the properties of e that its glTF node stores: the TRS and
// a light's color, energy and range.
func inGLTF(e node.Noded) []string {
	var out []string
	if _, ok := e.(transformed); ok {
		out = append(out, "position", "rotation", "scale")
	}
	if _, ok := e.(pointLit); ok {
		out = append(out, "color", "intensity", "range")
	}
	if _, ok := e.(sunLit); ok {
		out = append(out, "color", "energy")
	}
	return out
}

// differences lists the properties of e that differ from those of base,
// leaving out those glTF holds when skipGLTF is set.
func differences(e, base node.Noded, skipGLTF bool, res *node.Resources) (map[string]json.RawMessage, error) {
	var out map[string]json.RawMessage
	var skip []string
	if skipGLTF {
		skip = inGLTF(e)
	}
	for _, k := range node.Properties(e) {
		if slices.Contains(skip, k) {
			continue
		}
		v, err := node.GetProperty(e, k)
		if err != nil {
			return nil, err
		}
		if d, baseErr := node.GetProperty(base, k); baseErr == nil && bytes.Equal(v, d) {
			continue
		}
		if v, err = node.GetPropertyWith(e, k, res); err != nil {
			return nil, err
		}
		if out == nil {
			out = map[string]json.RawMessage{}
		}
		out[k] = v
	}
	return out, nil
}

func changed(e node.Noded, class, behaviour string, res *node.Resources) (map[string]json.RawMessage, error) {
	fresh, err := node.Create(class, behaviour)
	if err != nil {
		return nil, err
	}
	return differences(e, fresh, true, res)
}
