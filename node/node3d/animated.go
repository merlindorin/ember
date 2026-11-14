package node3d

import (
	"fmt"
	"maps"

	"github.com/lantervale/ember/anim"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/gltf"
	"github.com/lantervale/ember/node"
	"github.com/lantervale/ember/render"
	"github.com/lantervale/ember/scene"
	"github.com/lantervale/ember/world"
)

func init() { //nolint:gochecknoinits // registers the package's node types on import
	node.Register(node.Type{New: func() node.Noded { return NewSkeleton() }})
	node.Register(node.Type{New: func() node.Noded { return NewAnimator() }})
	node.Register(node.Type{New: func() node.Noded { return NewSocket() }})
}

// Skeleton animates the bones of an animated file and draws its skinned
// and bone-held meshes. Bones are not nodes.
type Skeleton struct {
	Node
	anim.Animator
	// Model holds the bones, their meshes and the clips.
	Model *gltf.Model
	bones []node.SceneNode
	drawn []*skinnedSurface
}

type skinnedSurface struct {
	*world.BasicEntity
	scene.Transform
	render.Model
	anim.Skinned
}

func NewSkeleton() *Skeleton { return &Skeleton{Node: *NewNode()} }

func (s *Skeleton) GetSkeleton() *Skeleton { return s }

func (s *Skeleton) EnterTree() {
	if s.drawn == nil && s.Model != nil {
		for _, p := range s.Model.Parts {
			e := &skinnedSurface{
				BasicEntity: world.NewBasic(),
				Transform:   scene.At(0, 0, 0),
				Model:       render.Model{Mesh: p.Mesh, Material: p.Material},
			}
			if p.Skin != nil {
				e.Skinned, e.Mesh = anim.NewSkinned(&s.Animator, p.Skin, p.Mesh, p.Influences)
			}
			s.AppendChild(e.BasicEntity)
			s.drawn = append(s.drawn, e)
		}
		s.Update(0)
		for _, e := range s.drawn {
			e.Apply(e.Mesh)
		}
	}
	for _, e := range s.drawn {
		s.Tree().World().AddEntity(e)
	}
}

// Animator plays the clips of the Skeleton it is a child of.
type Animator struct {
	node.Empty
	Autoplay          string `node:"prop"`
	AnimationFinished node.Signal[string]

	once     string
	finished bool
}

func NewAnimator() *Animator { return &Animator{Empty: *node.NewEmpty()} }

func (p *Animator) animator() *anim.Animator {
	if parent := p.ParentNode(); parent != nil {
		if s, ok := parent.Entity().(*Skeleton); ok {
			return &s.Animator
		}
	}
	return nil
}

func (p *Animator) Ready() {
	if p.Autoplay != "" {
		p.Play(p.Autoplay)
	}
}

// Play loops a clip, reporting whether it exists.
func (p *Animator) Play(name string) bool {
	a := p.animator()
	p.once = ""
	return a != nil && a.Play(name, 0.2)
}

// PlayOnce plays a clip to its end, then emits animation_finished.
func (p *Animator) PlayOnce(name string) bool {
	a := p.animator()
	if a == nil || !a.PlayOnce(name, 0.2) {
		return false
	}
	p.once, p.finished = name, false
	return true
}

func (p *Animator) Stop() {
	if a := p.animator(); a != nil {
		a.Stop()
	}
	p.once = ""
}

func (p *Animator) CurrentAnimation() string {
	if a := p.animator(); a != nil {
		if c, _ := a.Clip(); c != nil {
			return c.Name
		}
	}
	return ""
}

func (p *Animator) Process(float32) {
	a := p.animator()
	if p.once == "" || p.finished || a == nil {
		return
	}
	if c, t := a.Clip(); c != nil && c.Name == p.once && t >= c.Duration {
		p.finished = true
		p.AnimationFinished.Emit(p.once)
	}
}

// Socket follows a bone of the Skeleton it is a child of.
type Socket struct {
	Node
	anim.Socket
	BoneName string `node:"prop"`
}

func NewSocket() *Socket {
	return &Socket{Node: *NewNode(), Socket: anim.Socket{Joint: -1}}
}

func (b *Socket) EnterTree() {
	b.Animator, b.Joint = nil, -1
	if parent := b.ParentNode(); parent != nil {
		if s, ok := parent.Entity().(*Skeleton); ok && s.Skeleton != nil {
			b.Animator, b.Joint = &s.Animator, s.Skeleton.Find(b.BoneName)
		}
	}
}

func animated(m *gltf.Model) bool {
	if m.Animated() {
		return true
	}
	for _, p := range m.Parts {
		if p.Skin != nil && (len(p.Skin.Joints) != 1 || p.Skin.InverseBind[0] != geom.Ident()) {
			return true
		}
	}
	return false
}

// instantiateAnimated makes a Skeleton root whose bones are the file's
// untyped glTF nodes. Typed nodes become nodes under it: at the top level,
// or as Sockets on a bone, with their subtrees.
func instantiateAnimated(s *node.Scene, b *builder) (node.Noded, map[string]node.Noded, error) {
	m := s.Model
	joints := m.Skeleton.Joints
	typed := make([]bool, len(joints))
	for i := range joints {
		typed[i] = i < len(s.Nodes) && (s.Nodes[i].Type != "" || b.isInstance(s.Nodes[i]))
	}
	bones, err := m.Without(typed)
	if err != nil {
		return nil, nil, fmt.Errorf("node3d: bones must not be under typed nodes: %w", err)
	}
	root, skel, err := animatedRoot(s, b)
	if err != nil {
		return nil, nil, err
	}
	skel.Model = bones
	skel.Animator = anim.Animator{Skeleton: bones.Skeleton, Clips: bones.Clips}
	skel.bones = append(skel.bones, boneNodes(s, typed)...)

	byID := map[string]node.Noded{}
	if s.Root != nil && s.Root.ID != "" {
		byID[s.Root.ID] = root
	}
	ents := make([]node.Noded, len(joints))
	hasPlayer := false
	for i, j := range joints {
		if !typed[i] {
			continue
		}
		sn := s.Nodes[i]
		var e node.Noded
		e, err = b.node(i, sn, unskinnedParts(m, i), geom.Ident())
		if err != nil {
			return nil, nil, fmt.Errorf("node3d: %s: %w", path(joints, i), err)
		}
		ents[i] = e
		if sn.ID != "" {
			byID[sn.ID] = e
		}
		_, isPlayer := e.(*Animator)
		hasPlayer = hasPlayer || isPlayer
		switch {
		case j.Parent < 0:
			skel.AddNode(e)
		case !typed[j.Parent]:
			socket, ok := e.(*Socket)
			if !ok {
				return nil, nil, fmt.Errorf(
					"node3d: %s is a %s on bone %s; nodes on bones go in a node3d.Socket",
					path(joints, i),
					sn.Type,
					joints[j.Parent].Name,
				)
			}
			socket.onBone(joints[j.Parent].Name)
			skel.AddNode(e)
		default:
			ents[j.Parent].GetSceneNode().AddNode(e)
		}
	}
	if !hasPlayer && len(bones.Clips) > 0 {
		p := NewAnimator()
		p.SetName("Animator")
		skel.AddNode(p)
	}
	return root, byID, nil
}

// animatedRoot is the root of an animated scene: its Skeleton, or the
// node the scene's root names, which holds one.
func animatedRoot(s *node.Scene, b *builder) (node.Noded, *Skeleton, error) {
	skel := NewSkeleton()
	name := cmpOr(s.Model.Name, "Skeleton")
	if s.Root == nil {
		skel.SetName(name)
		return skel, skel, nil
	}
	e, err := node.Create(cmpOr(s.Root.Type, "node3d.Skeleton"), s.Root.Behaviour)
	if err != nil {
		return nil, nil, fmt.Errorf("node3d: root: %w", err)
	}
	k, ok := e.(interface{ GetSkeleton() *Skeleton })
	if !ok {
		return nil, nil, fmt.Errorf("node3d: the root of an animated scene is a node3d.Skeleton, not %s", s.Root.Type)
	}
	if err = apply(e, *s.Root, b.res); err != nil {
		return nil, nil, fmt.Errorf("node3d: root: %w", err)
	}
	skel = k.GetSkeleton()
	skel.SetName(cmpOr(s.Root.Name, name))
	return e, skel, nil
}

// onBone places the socket at the bone it follows, bone unless it names
// its own.
func (b *Socket) onBone(bone string) {
	if b.BoneName == "" {
		b.BoneName = bone
	}
	b.Position, b.Rotation, b.Scale = geom.Vec3{}, geom.QuatIdent(), geom.V3(1, 1, 1)
}

// boneNodes are the scene data of the joints that are not typed nodes.
func boneNodes(s *node.Scene, typed []bool) []node.SceneNode {
	var out []node.SceneNode
	for i, t := range typed {
		if !t {
			var sn node.SceneNode
			if i < len(s.Nodes) {
				sn = s.Nodes[i]
			}
			out = append(out, sn)
		}
	}
	return out
}

// unskinnedParts are copies of the parts on joint i without their skins.
func unskinnedParts(m *gltf.Model, i int) []*gltf.Part {
	var parts []*gltf.Part
	for _, p := range m.Parts {
		if p.Joint == i {
			c := *p
			c.Skin, c.Influences = nil, nil
			parts = append(parts, &c)
		}
	}
	return parts
}

func packAnimated(skel *Skeleton, root node.Noded, s *node.Scene, p *packer) error {
	r := root.GetSceneNode()
	bones := skel.Model
	m := s.Model
	m.Skeleton = &anim.Skeleton{Joints: append([]anim.Joint(nil), bones.Skeleton.Joints...)}
	m.Nodes = append([]gltf.Raw(nil), bones.Nodes...)
	for len(m.Nodes) < len(m.Skeleton.Joints) {
		m.Nodes = append(m.Nodes, gltf.Raw{})
	}
	m.Parts, m.Clips, m.Cameras = append([]*gltf.Part(nil), bones.Parts...), bones.Clips, maps.Clone(bones.Cameras)
	if m.Cameras == nil {
		m.Cameras = map[int]render.Camera{}
	}
	s.Nodes = append([]node.SceneNode(nil), skel.bones...)
	for len(s.Nodes) < len(m.Skeleton.Joints) {
		s.Nodes = append(s.Nodes, node.SceneNode{})
	}
	sn, err := p.sceneNode(root)
	if err != nil {
		return err
	}
	sn.Name = r.Name()
	s.Root = &sn
	p.list = append(p.list, root)
	for _, c := range r.ChildNodes() {
		if c.Owner() != r {
			continue
		}
		parent := -1
		if b, ok := c.Entity().(*Socket); ok {
			if parent = skel.Skeleton.Find(b.BoneName); parent < 0 {
				return fmt.Errorf("node3d: %s follows bone %q, which %s does not have", c.Name(), b.BoneName, r.Name())
			}
		}
		if err = p.visit(c.Entity(), parent); err != nil {
			return err
		}
	}
	return nil
}
