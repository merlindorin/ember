package node3d

import (
	"fmt"
	"slices"
	"strings"

	"github.com/merlindorin/ember/internal/netside"
	"github.com/merlindorin/ember/net"
	"github.com/merlindorin/ember/node"
	"github.com/merlindorin/ember/render"
	"github.com/merlindorin/ember/scene"
	"github.com/merlindorin/ember/world"
)

func init() { //nolint:gochecknoinits // registers the package's node types on import
	node.Register(node.Type{New: func() node.Noded { return NewEnvironment() }})
	node.Register(node.Type{New: func() node.Noded { return NewReplicator() }})
}

// Environment is a scene's environment: while it is in the World, its look
// lights the scene, the sun's direction coming from the scene's Sun.
type Environment struct {
	node.Empty
	Environment *render.Environment `node:"prop"`
}

func NewEnvironment() *Environment {
	return &Environment{Empty: *node.NewEmpty(), Environment: render.NewEnvironment()}
}

func (w *Environment) GetEnvironment() *render.Environment { return w.Environment }

type looks struct{ world.Group[render.Lit] }

// environment is the World's first environment, nil when it has none.
func environment(w *world.World) *render.Environment {
	l, ok := world.TryResource[looks](w)
	if !ok {
		l = world.SetResource(w, &looks{})
		w.AddView(l)
	}
	for e := range l.All() {
		return e.GetEnvironment()
	}
	return nil
}

// Replicator replicates the node at RootPath: it is the
// net.Replicated entity, sending the root's pose and net.Stateful state
// from the server and applying them to the root on clients. Its Kind is
// the root's scene ID, which SceneSpawner instantiates on clients.
type Replicator struct {
	node.Empty
	Network  net.Network
	RootPath string `node:"prop"`
	pose     scene.Transform
}

func NewReplicator() *Replicator {
	return &Replicator{Empty: *node.NewEmpty(), RootPath: "..", pose: scene.At(0, 0, 0)}
}

// GetTransform is the replicated pose, kept apart from the tree's
// transforms.
func (s *Replicator) GetTransform() *scene.Transform { return &s.pose }

func (s *Replicator) GetNetwork() *net.Network { return &s.Network }

func (s *Replicator) root() node.Noded {
	if n := s.Get(s.RootPath); n != nil && n != s.GetSceneNode() {
		return n.Entity()
	}
	return nil
}

func (s *Replicator) EnterTree() {
	if s.Network.NetID == 0 {
		if key, ok := s.placement(); ok {
			s.Network.NetID = net.PlacedID(key)
		}
	}
	if s.Network.NetID.Region() == net.PlacedRegion && s.client() {
		if t, ok := s.root().(transformed); ok {
			at := t.GetTransform()
			s.pose.Position, s.pose.Rotation, s.pose.Scale = at.Position, at.Rotation, at.Scale
		}
		c, _ := netside.Client(s.Tree().World()).(*net.Client)
		c.Place(s)
	}
	if s.Network.Kind != "" {
		return
	}
	if r := s.root(); r != nil {
		n := r.GetSceneNode()
		switch {
		case n.Instance != nil && !n.Instance.Embedded:
			s.Network.Kind = n.Instance.Scene
		case n.File != nil:
			s.Network.Kind = n.File.ID
		}
	}
}

// placement is the key of a synchronizer placed by a loaded scene file: the
// file's ID and the scene IDs down the owner chain, which server and clients
// loading the same level derive alike. Nodes added at runtime, whose owner
// chain ends without a file, are not placed.
func (s *Replicator) placement() (string, bool) {
	var ids []string
	n := s.GetSceneNode()
	for ; n != nil; n = n.Owner() {
		if n.SceneID == "" {
			return "", false
		}
		ids = append(ids, n.SceneID)
		if n.Owner() == nil {
			break
		}
	}
	if n == nil || n.File == nil || n.File.ID == "" {
		return "", false
	}
	slices.Reverse(ids)
	return n.File.ID + "/" + strings.Join(ids, "/"), true
}

func (s *Replicator) client() bool {
	w := s.Tree().World()
	return netside.Client(w) != nil && netside.Server(w) == nil
}

// LeaveTree frees the replicated scene when the World removes the
// replicator on a client, which is a despawn; otherwise only the
// replicator leaves.
func (s *Replicator) LeaveTree() {
	if s.Tree() != nil && s.client() {
		if r := s.root(); r != nil {
			r.GetSceneNode().QueueFree()
			return
		}
	}
	s.Empty.LeaveTree()
}

func (s *Replicator) Process(float32) {
	r := s.root()
	t, ok := r.(transformed)
	if !ok {
		return
	}
	at := t.GetTransform()
	if !s.client() {
		s.pose.Position, s.pose.Rotation, s.pose.Scale = at.Position, at.Rotation, at.Scale
		return
	}
	if !s.Tree().World().Has(s) {
		r.GetSceneNode().QueueFree()
		return
	}
	at.Position, at.Rotation, at.Scale = s.pose.Position, s.pose.Rotation, s.pose.Scale
}

func (s *Replicator) NetState() any {
	if st, ok := s.root().(net.Stateful); ok {
		return st.NetState()
	}
	return nil
}

func (s *Replicator) ApplyNetState(state any) {
	if st, ok := s.root().(net.Stateful); ok {
		st.ApplyNetState(state)
	}
}

// SceneSpawner returns the client-side factory of replicated scenes: it
// instances the scene of a kind under the tree's root and returns its
// Replicator, or nil when there is none.
func SceneSpawner(tree *node.Tree, resolve node.Resolver) func(kind string) net.Replicated {
	return func(kind string) net.Replicated {
		s, err := resolve(kind)
		if err != nil {
			return nil
		}
		e, err := Instance(s, resolve)
		if err != nil {
			return nil
		}
		for _, d := range descendants(e.GetSceneNode()) {
			if sync, ok := d.(*Replicator); ok {
				tree.Root().AddNode(e)
				return sync
			}
		}
		return nil
	}
}

// RegisterScenes registers each scene as a net kind spawned by
// SceneSpawner, checking first that it resolves and has a synchronizer.
func RegisterScenes(tree *node.Tree, resolve node.Resolver, ids ...string) error {
	for _, id := range ids {
		s, err := resolve(id)
		if err != nil {
			return fmt.Errorf("node3d: replicated scene %s: %w", id, err)
		}
		e, err := Instance(s, resolve)
		if err != nil {
			return fmt.Errorf("node3d: replicated scene %s: %w", id, err)
		}
		found := false
		for _, d := range descendants(e.GetSceneNode()) {
			_, isSync := d.(*Replicator)
			found = found || isSync
		}
		if !found {
			return fmt.Errorf("node3d: replicated scene %s has no node3d.Replicator", id)
		}
	}
	spawn := SceneSpawner(tree, resolve)
	for _, id := range ids {
		net.RegisterKind(id, func() net.Replicated { return spawn(id) })
	}
	return nil
}
