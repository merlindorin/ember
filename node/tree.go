package node

import (
	"reflect"

	"github.com/lantervale/ember/world"
)

type (
	EnterTreer       interface{ EnterTree() }
	Readier          interface{ Ready() }
	ExitTreer        interface{ ExitTree() }
	Processor        interface{ Process(dt float32) }
	PhysicsProcessor interface{ PhysicsProcess(dt float32) }
)

type root struct {
	*world.BasicEntity
	Node
}

// Tree is the scene tree of a world: its root is /root, and the nodes under
// it are in the world.
type Tree struct {
	Paused bool
	// Resolve finds scenes by ID for tools, such as SceneService snapshots.
	Resolve Resolver

	world    *world.World
	root     *root
	deferred []func()
	free     []*Node
}

// New creates a tree in w and installs its systems:
// Process in Update and PhysicsProcess in FixedUpdate, at priority 0, then
// deferred calls and frees early in PostUpdate.
func New(w *world.World) *Tree {
	t := &Tree{world: w, root: &root{BasicEntity: world.NewBasic()}}
	t.root.SetName("root")
	t.root.bind(t.root)
	t.root.tree = t
	t.root.ready = true
	w.AddEntity(t.root)
	w.AddSystems(&processSystem{tree: t}, &processSystem{tree: t, physics: true}, &flushSystem{tree: t})
	return t
}

func (t *Tree) Root() *Node { return &t.root.Node }

func (t *Tree) World() *world.World { return t.world }

func (t *Tree) enter(n *Node) {
	t.propagateEnter(n)
	t.propagateReady(n)
}

func (t *Tree) propagateEnter(n *Node) {
	n.tree = t
	bindSignals(n)
	if e, ok := n.self.(EnterTreer); ok {
		e.EnterTree()
	}
	t.world.AddEntity(n.self)
	for _, c := range n.ChildNodes() {
		t.propagateEnter(c)
	}
}

func (t *Tree) propagateReady(n *Node) {
	for _, c := range n.ChildNodes() {
		t.propagateReady(c)
	}
	if !n.ready && n.tree == t {
		n.ready = true
		if r, ok := n.self.(Readier); ok {
			r.Ready()
		}
	}
}

func (t *Tree) exit(n *Node, leaveWorld bool) {
	t.propagateExit(n)
	if leaveWorld {
		t.world.RemoveEntity(n.self)
	}
}

func (t *Tree) propagateExit(n *Node) {
	children := n.ChildNodes()
	for i := len(children) - 1; i >= 0; i-- {
		t.propagateExit(children[i])
	}
	if e, ok := n.self.(ExitTreer); ok {
		e.ExitTree()
	}
	n.tree = nil
}

// Walk visits the tree's nodes in tree order, root first.
func (t *Tree) Walk(fn func(*Node)) {
	fn(t.Root())
	t.Root().walk(fn)
}

// Group lists the group's nodes in tree order.
func (t *Tree) Group(g string) []*Node {
	var out []*Node
	t.Walk(func(n *Node) {
		if n.IsInGroup(g) {
			out = append(out, n)
		}
	})
	return out
}

// CallGroup calls method with args on every node of the group whose entity
// has it.
func (t *Tree) CallGroup(g, method string, args ...any) {
	in := make([]reflect.Value, len(args))
	for i, a := range args {
		in[i] = reflect.ValueOf(a)
	}
	for _, n := range t.Group(g) {
		if m := reflect.ValueOf(n.self).MethodByName(method); m.IsValid() {
			m.Call(in)
		}
	}
}

func (t *Tree) deferCall(fn func()) { t.deferred = append(t.deferred, fn) }

func (t *Tree) flush() {
	for len(t.deferred) > 0 || len(t.free) > 0 {
		calls := t.deferred
		t.deferred = nil
		for _, fn := range calls {
			fn()
		}
		free := t.free
		t.free = nil
		for _, n := range free {
			if n.parent != nil {
				n.parent.detach(n, true)
				n.cleanOwners()
			}
		}
	}
}

func (t *Tree) processable(n *Node, inherited ProcessMode) (bool, ProcessMode) {
	mode := n.ProcessMode
	if mode == Inherit {
		mode = inherited
	}
	switch mode {
	case Always:
		return true, mode
	case WhenPaused:
		return t.Paused, mode
	case Disabled:
		return false, mode
	case Inherit, Pausable:
		// runs unless the tree is paused
	}
	return !t.Paused, mode
}

type processSystem struct {
	tree    *Tree
	physics bool
}

func (s *processSystem) Update(frame world.Frame) {
	dt := frame.DT
	var run []Noded
	var visit func(*Node, ProcessMode)
	visit = func(n *Node, inherited ProcessMode) {
		ok, mode := s.tree.processable(n, inherited)
		if ok {
			run = append(run, n.self)
		}
		for _, c := range n.children {
			visit(c, mode)
		}
	}
	visit(s.tree.Root(), Pausable)
	for _, e := range run {
		if e.GetSceneNode().tree != s.tree {
			continue
		}
		if s.physics {
			if p, ok := e.(PhysicsProcessor); ok {
				p.PhysicsProcess(dt)
			}
		} else if p, ok := e.(Processor); ok {
			p.Process(dt)
		}
	}
}

func (s *processSystem) Remove(world.Identifier) {}

func (s *processSystem) Phase() world.Phase {
	if s.physics {
		return world.FixedUpdate
	}
	return world.Update
}

type flushSystem struct{ tree *Tree }

func (s *flushSystem) Update(world.Frame)      { s.tree.flush() }
func (s *flushSystem) Remove(world.Identifier) {}
func (s *flushSystem) Phase() world.Phase      { return world.PostUpdate }
func (s *flushSystem) Priority() int           { return 2000 }
