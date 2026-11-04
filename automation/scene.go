//go:build !js

package automation

import (
	"context"
	"encoding/json"
	"slices"

	"connectrpc.com/connect"
	"github.com/merlindorin/ember/automation/scenepb"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/node"
	"github.com/merlindorin/ember/node/node3d"
	"google.golang.org/protobuf/types/known/structpb"
)

// sceneService serves ember.scene.v1.SceneService on the app's node.Tree.
type sceneService struct {
	ctl      *controller
	isolated *isolation
}

type isolation struct {
	node   *node.Node
	mode   node.ProcessMode
	paused bool
}

func (s *sceneService) tree() (*node.Tree, error) {
	t := s.ctl.scenes
	if t == nil {
		return nil, failure(connect.CodeFailedPrecondition, "automation: this app has no scene tree")
	}
	return t, nil
}

func (s *sceneService) find(path string) (*node.Node, error) {
	t, err := s.tree()
	if err != nil {
		return nil, err
	}
	if path == "" || path == "/root" {
		return t.Root(), nil
	}
	n := t.Root().Get(path)
	if n == nil {
		return nil, failure(connect.CodeNotFound, "automation: no node at "+path)
	}
	return n, nil
}

// nodeProto describes n with depth levels of nodes, n included; 0 means all.
func nodeProto(n *node.Node, depth uint32) *scenepb.Node {
	class, behaviour, _ := node.ClassOf(n.Entity())
	mode, _ := n.ProcessMode.MarshalText()
	out := &scenepb.Node{
		Id:          n.SceneID,
		Name:        n.Name(),
		Path:        n.Path(),
		Type:        class,
		Behaviour:   behaviour,
		Groups:      n.Groups(),
		ProcessMode: string(mode),
	}
	if n.Instance != nil && n.Instance.Root == n {
		out.Scene = n.Instance.Scene
	}
	if depth != 1 {
		for _, c := range n.ChildNodes() {
			out.Children = append(out.Children, nodeProto(c, max(depth, 1)-1))
		}
	}
	return out
}

func (s *sceneService) GetTree(
	ctx context.Context, r *connect.Request[scenepb.GetTreeRequest],
) (*connect.Response[scenepb.GetTreeResponse], error) {
	return reply(do(ctx, s.ctl, func() (*scenepb.GetTreeResponse, error) {
		n, err := s.find(r.Msg.GetPath())
		if err != nil {
			return nil, err
		}
		levels := r.Msg.GetDepth()
		if levels > 0 {
			levels++
		}
		return &scenepb.GetTreeResponse{Root: nodeProto(n, levels)}, nil
	}))
}

func (s *sceneService) GetNode(
	ctx context.Context, r *connect.Request[scenepb.GetNodeRequest],
) (*connect.Response[scenepb.GetNodeResponse], error) {
	return reply(do(ctx, s.ctl, func() (*scenepb.GetNodeResponse, error) {
		n, err := s.find(r.Msg.GetPath())
		if err != nil {
			return nil, err
		}
		e := n.Entity()
		out := &scenepb.GetNodeResponse{
			Node:       nodeProto(n, 1),
			Properties: map[string]*structpb.Value{},
			Signals:    node.SignalNames(e),
		}
		for _, p := range node.Properties(e) {
			if out.Properties[p], err = propertyValue(e, p); err != nil {
				return nil, failure(connect.CodeInternal, err.Error())
			}
		}
		return out, nil
	}))
}

func propertyValue(e node.Noded, name string) (*structpb.Value, error) {
	raw, err := node.GetProperty(e, name)
	if err != nil {
		return nil, err
	}
	var v any
	if err = json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}
	return structpb.NewValue(v)
}

func (s *sceneService) SetProperty(
	ctx context.Context, r *connect.Request[scenepb.SetPropertyRequest],
) (*connect.Response[scenepb.SetPropertyResponse], error) {
	return reply(do(ctx, s.ctl, func() (*scenepb.SetPropertyResponse, error) {
		n, err := s.find(r.Msg.GetPath())
		if err != nil {
			return nil, err
		}
		raw, err := r.Msg.GetValue().MarshalJSON()
		if err != nil {
			return nil, failure(connect.CodeInvalidArgument, err.Error())
		}
		if err = node.SetProperty(n.Entity(), r.Msg.GetName(), raw); err != nil {
			return nil, failure(connect.CodeInvalidArgument, err.Error())
		}
		return &scenepb.SetPropertyResponse{}, nil
	}))
}

func (s *sceneService) Isolate(
	ctx context.Context, r *connect.Request[scenepb.IsolateRequest],
) (*connect.Response[scenepb.IsolateResponse], error) {
	return reply(do(ctx, s.ctl, func() (*scenepb.IsolateResponse, error) {
		t, err := s.tree()
		if err != nil {
			return nil, err
		}
		var target *node.Node
		if r.Msg.GetPath() != "" {
			if target, err = s.find(r.Msg.GetPath()); err != nil {
				return nil, err
			}
		}
		if i := s.isolated; i != nil {
			i.node.ProcessMode, t.Paused = i.mode, i.paused
			s.isolated = nil
		}
		if target != nil {
			s.isolated = &isolation{node: target, mode: target.ProcessMode, paused: t.Paused}
			target.ProcessMode, t.Paused = node.Always, true
		}
		return &scenepb.IsolateResponse{}, nil
	}))
}

type watched struct {
	proto *scenepb.Node
	key   string
}

func (s *sceneService) WatchTree(
	ctx context.Context,
	r *connect.Request[scenepb.WatchTreeRequest],
	stream *connect.ServerStream[scenepb.WatchTreeResponse],
) error {
	wake, stop := s.ctl.reg.watch()
	defer stop()
	var last map[*node.Node]watched
	for {
		now, err := do(ctx, s.ctl, func() (map[*node.Node]watched, error) {
			n, err := s.find(r.Msg.GetPath())
			if err != nil {
				return nil, err
			}
			out := map[*node.Node]watched{}
			var visit func(*node.Node)
			visit = func(n *node.Node) {
				p := nodeProto(n, 1)
				b, _ := json.Marshal(p)
				out[n] = watched{p, string(b)}
				for _, c := range n.ChildNodes() {
					visit(c)
				}
			}
			visit(n)
			return out, nil
		})
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		var changes []*scenepb.WatchTreeResponse_Change
		for n, w := range now {
			switch was, ok := last[n]; {
			case !ok:
				changes = append(changes, &scenepb.WatchTreeResponse_Change{
					Kind: scenepb.WatchTreeResponse_KIND_ADDED,
					Node: w.proto,
				})
			case was.key != w.key:
				changes = append(changes, &scenepb.WatchTreeResponse_Change{
					Kind: scenepb.WatchTreeResponse_KIND_CHANGED,
					Node: w.proto,
				})
			}
		}
		for n, w := range last {
			if _, ok := now[n]; !ok {
				changes = append(changes, &scenepb.WatchTreeResponse_Change{
					Kind: scenepb.WatchTreeResponse_KIND_REMOVED,
					Node: &scenepb.Node{Path: w.proto.GetPath()},
				})
			}
		}
		slices.SortFunc(changes, func(a, b *scenepb.WatchTreeResponse_Change) int {
			switch {
			case a.GetNode().GetPath() < b.GetNode().GetPath():
				return -1
			case a.GetNode().GetPath() > b.GetNode().GetPath():
				return 1
			}
			return 0
		})
		last = now
		if len(changes) > 0 {
			if err = stream.Send(&scenepb.WatchTreeResponse{Changes: changes}); err != nil {
				return err
			}
		}
		select {
		case <-ctx.Done():
			return nil
		case <-wake:
		}
	}
}

func (s *sceneService) Snapshot(
	ctx context.Context,
	r *connect.Request[scenepb.SnapshotRequest],
	stream *connect.ServerStream[scenepb.SnapshotResponse],
) error {
	data, err := do(ctx, s.ctl, func() ([]byte, error) {
		t, err := s.tree()
		if err != nil {
			return nil, err
		}
		n, err := s.find(r.Msg.GetPath())
		if err != nil {
			return nil, err
		}
		var keep func(node.Noded) bool
		if radius := r.Msg.GetRadius(); radius > 0 {
			c := r.Msg.GetCenter()
			keep = node3d.Within(geom.V3(c.GetX(), c.GetY(), c.GetZ()), radius)
		}
		scene, err := node3d.Snapshot(n.Entity(), t.Resolve, keep)
		if err != nil {
			return nil, failure(connect.CodeFailedPrecondition, err.Error())
		}
		return node.Encode(scene)
	})
	if err != nil {
		return err
	}
	size := int(r.Msg.GetChunkSize())
	if size <= 0 {
		size = 64 << 10
	}
	for at := 0; at < len(data); at += size {
		chunk := data[at:min(at+size, len(data))]
		if err = stream.Send(&scenepb.SnapshotResponse{Data: chunk, Total: uint64(len(data))}); err != nil {
			return err
		}
	}
	return nil
}
