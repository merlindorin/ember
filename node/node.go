package node

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"iter"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/lantervale/ember/gltf"
	"github.com/lantervale/ember/world"
)

// Noded is any entity carrying a Node.
type Noded interface {
	world.BasicFace
	GetSceneNode() *Node
}

type ProcessMode uint8

const (
	Inherit ProcessMode = iota
	Pausable
	WhenPaused
	Always
	Disabled
)

var processModes = Words[ProcessMode]{ //nolint:gochecknoglobals // word table
	"inherit", "pausable", "when_paused", "always", "disabled",
}

func (v ProcessMode) MarshalText() ([]byte, error)  { return processModes.Text(v) }
func (v *ProcessMode) UnmarshalText(b []byte) error { return processModes.Parse(v, b) }

// Node is a place in the scene tree. Embed it in an entity next to
// *world.BasicEntity; the entity's parent and children follow the tree.
type Node struct {
	// SceneID is the node's stable ID in scene files, 16 hex characters,
	// empty for nodes that no scene saved.
	SceneID     string
	ProcessMode ProcessMode
	// AutoTranslateMode says whether the node's texts are translated.
	AutoTranslateMode AutoTranslateMode `node:"prop"`
	// Raw is the node's glTF data that no node type interprets, written
	// back when its scene is saved.
	Raw gltf.Raw
	// File is set on a scene's root: what saving it again keeps.
	File *File
	// Instance marks the nodes made from another scene.
	Instance *Instanced

	name     string
	byName   map[string]*Node
	serial   map[string]int
	groups   []string
	owner    *Node
	self     Noded
	parent   *Node
	children []*Node
	tree     *Tree
	ready    bool
	// kin counts the changes to children, so what keeps its own record of
	// them knows when to look again.
	kin uint64
}

func (n *Node) GetSceneNode() *Node { return n }

// Instanced marks an instance's root and every node of the instanced scene
// under it, which keep that scene's IDs. An embedded instance marks only
// its root, and its nodes are saved in the instancing file.
type Instanced struct {
	Scene            string
	Root             *Node
	Embedded         bool
	EditableChildren bool
}

// Resolver finds a scene by ID.
type Resolver func(id string) (*Scene, error)

// AutoTranslateMode says whether texts are translated: Inherit follows the parent, and
// the root translates.
type AutoTranslateMode int

const (
	AutoTranslateInherit AutoTranslateMode = iota
	AutoTranslateAlways
	AutoTranslateDisabled
)

var autoTranslateModes = Words[AutoTranslateMode]{ //nolint:gochecknoglobals // word table
	"inherit", "always", "disabled",
}

func (v AutoTranslateMode) MarshalText() ([]byte, error)  { return autoTranslateModes.Text(v) }
func (v *AutoTranslateMode) UnmarshalText(b []byte) error { return autoTranslateModes.Parse(v, b) }

// CanAutoTranslate resolves the node's auto_translate_mode.
func (n *Node) CanAutoTranslate() bool {
	for at := n; at != nil; at = at.parent {
		switch at.AutoTranslateMode {
		case AutoTranslateAlways:
			return true
		case AutoTranslateDisabled:
			return false
		case AutoTranslateInherit:
			// the parent decides
		}
	}
	return true
}

// File is the scene-level data of a scene's root.
type File struct {
	ID, Inherits string
	// Raw holds the document's extensions and extras.
	Raw      gltf.Raw
	Required []string
}

// NewID returns a fresh scene ID.
func NewID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func (n *Node) Name() string       { return n.name }
func (n *Node) Entity() Noded      { return n.self }
func (n *Node) ParentNode() *Node  { return n.parent }
func (n *Node) Tree() *Tree        { return n.tree }
func (n *Node) IsInsideTree() bool { return n.tree != nil }

func (n *Node) ChildNodes() []*Node { return slices.Clone(n.children) }

// ChildrenChanged counts the times a child was added to n or taken from
// it: the same count, the same children in the same order.
func (n *Node) ChildrenChanged() uint64 { return n.kin }

// EachChild yields n's children in order without copying them, for walks
// that add or remove none.
func (n *Node) EachChild() iter.Seq[*Node] {
	return func(yield func(*Node) bool) {
		for _, c := range n.children {
			if !yield(c) {
				return
			}
		}
	}
}

// SetName renames the node, replacing the path separator / with _ and
// numbering it Name2, Name3… when a sibling holds the name.
func (n *Node) SetName(name string) {
	name = strings.ReplaceAll(name, "/", "_")
	if n.parent != nil {
		name = n.parent.freeName(n, name)
		n.parent.unindex(n)
		n.name = name
		n.parent.index(n)
		return
	}
	n.name = name
}

func (n *Node) freeName(child *Node, want string) string {
	taken := func(name string) bool {
		c, ok := n.byName[name]
		return ok && c != child
	}
	if !taken(want) {
		return want
	}
	if n.serial == nil {
		n.serial = map[string]int{}
	}
	for i := max(n.serial[want], 2); ; i++ {
		if s := want + strconv.Itoa(i); !taken(s) {
			n.serial[want] = i + 1
			return s
		}
	}
}

func (n *Node) index(c *Node) {
	if n.byName == nil {
		n.byName = map[string]*Node{}
	}
	n.byName[c.name] = c
}

func (n *Node) unindex(c *Node) {
	if n.byName[c.name] == c {
		delete(n.byName, c.name)
	}
}

// Bind ties n to its entity before it is added anywhere, as a root that
// a layout or a lookup walks up to. AddNode does it for children.
func (n *Node) Bind(self Noded) { n.bind(self) }

func (n *Node) bind(self Noded) {
	if n.self != nil {
		return
	}
	n.self = self
	if n.name == "" {
		n.name = reflect.Indirect(reflect.ValueOf(self)).Type().Name()
	}
	for _, c := range n.children {
		n.basic().AppendChild(c.basic())
	}
}

func (n *Node) basic() *world.BasicEntity { return n.self.GetBasicEntity() }

// AddNode appends e under n; if n is in the tree, e's subtree joins the
// world and enters the tree.
func (n *Node) AddNode(e Noded) {
	c := e.GetSceneNode()
	if c.parent != nil {
		panic(fmt.Sprintf("node: %s already has a parent; use Reparent", c.name))
	}
	c.bind(e)
	n.link(c)
	if n.tree != nil {
		n.tree.enter(c)
	}
}

func (n *Node) link(c *Node) {
	c.name = n.freeName(c, c.name)
	c.parent = n
	n.children = append(n.children, c)
	n.kin++
	n.index(c)
	if n.self != nil {
		n.basic().AppendChild(c.basic())
	}
}

// RemoveNode takes e's subtree out of the tree and the world; the nodes
// stay usable and can be added again.
func (n *Node) RemoveNode(e Noded) {
	if c := e.GetSceneNode(); c.parent == n {
		n.detach(c, true)
		c.cleanOwners()
	}
}

func (n *Node) detach(c *Node, leaveWorld bool) {
	if c.tree != nil {
		c.tree.exit(c, leaveWorld)
	}
	n.children = slices.DeleteFunc(n.children, func(x *Node) bool { return x == c })
	n.kin++
	n.unindex(c)
	c.parent = nil
	if n.self != nil {
		n.basic().RemoveChild(c.basic())
	}
}

// LeaveTree takes the node out of its parent, keeping its subtree whole;
// the World calls it when it removes the entity.
func (n *Node) LeaveTree() {
	if p := n.parent; p != nil {
		p.detach(n, false)
		n.cleanOwners()
	}
}

// cleanOwners clears the owners in n's subtree that are no longer above
// the nodes they own after a node moves.
func (n *Node) cleanOwners() {
	n.walk(func(m *Node) {
		if m.owner != nil && !m.owner.isAncestorOf(m) {
			m.owner = nil
		}
	})
	if n.owner != nil && !n.owner.isAncestorOf(n) {
		n.owner = nil
	}
}

// Reparent moves n under parent. Within the tree, n exits and enters again
// but stays in the world. Owners still above the moved nodes are kept.
func (n *Node) Reparent(parent *Node) {
	if n.parent != nil {
		n.parent.detach(n, parent.tree == nil)
	}
	parent.link(n)
	n.cleanOwners()
	if parent.tree != nil {
		parent.tree.enter(n)
	}
}

// RequestReady makes Ready run again the next time n enters the tree.
func (n *Node) RequestReady() { n.ready = false }

// QueueFree removes n from the tree and the world at the end of the frame.
func (n *Node) QueueFree() {
	if n.tree != nil {
		n.tree.free = append(n.tree.free, n)
	}
}

func (n *Node) isAncestorOf(m *Node) bool {
	for p := m.parent; p != nil; p = p.parent {
		if p == n {
			return true
		}
	}
	return false
}

func (n *Node) walk(fn func(*Node)) {
	for _, c := range slices.Clone(n.children) {
		fn(c)
		c.walk(fn)
	}
}

func (n *Node) Owner() *Node { return n.owner }

// SetOwner makes owner's scene hold n; owner must be one of n's ancestors.
func (n *Node) SetOwner(owner *Node) error {
	if owner != nil && !owner.isAncestorOf(n) {
		return fmt.Errorf("node: %s cannot own %s, which is not its descendant", owner.name, n.name)
	}
	n.owner = owner
	return nil
}

func (n *Node) AddToGroup(g string) {
	if !n.IsInGroup(g) {
		n.groups = append(n.groups, g)
	}
}

func (n *Node) RemoveFromGroup(g string) {
	n.groups = slices.DeleteFunc(n.groups, func(x string) bool { return x == g })
}

func (n *Node) IsInGroup(g string) bool { return slices.Contains(n.groups, g) }

func (n *Node) Groups() []string { return slices.Clone(n.groups) }

// Path is n's absolute path, such as /root/World/Player, or empty outside
// the tree.
func (n *Node) Path() string {
	if n.tree == nil {
		return ""
	}
	var parts []string
	for p := n; p != nil; p = p.parent {
		parts = append(parts, p.name)
	}
	slices.Reverse(parts)
	return "/" + strings.Join(parts, "/")
}

// Get resolves a node path from n: "A/B", "..", "." or "/root/A". It
// returns nil when nothing matches.
func (n *Node) Get(path string) *Node {
	if path == "" {
		return nil
	}
	cur := n
	parts := strings.Split(path, "/")
	if parts[0] == "" {
		if n.tree == nil || len(parts) < 2 || parts[1] != n.tree.root.name {
			return nil
		}
		cur, parts = &n.tree.root.Node, parts[2:]
	}
	for _, p := range parts {
		switch {
		case cur == nil:
			return nil
		case p == "." || p == "":
		case p == "..":
			cur = cur.parent
		default:
			cur = cur.child(p)
		}
	}
	return cur
}

func (n *Node) child(name string) *Node { return n.byName[name] }

// Find resolves path from n and returns the entity there as a T.
func Find[T any](from *Node, path string) (T, bool) {
	var zero T
	n := from.Get(path)
	if n == nil {
		return zero, false
	}
	t, ok := n.self.(T)
	return t, ok
}
