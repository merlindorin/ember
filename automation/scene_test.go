package automation_test

import (
	"errors"
	"net/http"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/grpcreflect"
	"github.com/lantervale/ember/automation"
	"github.com/lantervale/ember/automation/scenepb"
	"github.com/lantervale/ember/automation/scenepb/scenepbconnect"
	"github.com/lantervale/ember/node"
	"github.com/lantervale/ember/node/node3d"
	"github.com/lantervale/ember/world"
	"google.golang.org/protobuf/types/known/structpb"
)

type ticker struct {
	node.Timer
	Ticks int `node:"prop"`
}

func (k *ticker) Process(dt float32) {
	k.Timer.Process(dt)
	k.Ticks++
}

func TestMain(m *testing.M) {
	node.Register(node.Type{Name: "test.Ticker", Base: "node.Timer", New: func() node.Noded { return newTicker() }})
	os.Exit(m.Run())
}

func newTicker() *ticker { return &ticker{Timer: *node.NewTimer()} }

type sceneHarness struct {
	scene scenepbconnect.SceneServiceClient
	http  *http.Client
	url   string
	tree  *node.Tree
	add   atomic.Bool
}

func startScene(t *testing.T, token string) *sceneHarness {
	t.Helper()
	a := world.New()
	h := &sceneHarness{tree: node.New(a)}
	srv := automation.New(automation.Addr("127.0.0.1:0"), automation.Token(token), automation.Scenes(h.tree))
	a.AddPlugins(srv)
	camp := node3d.NewNode()
	camp.SetName("Camp")
	slow, fast := newTicker(), newTicker()
	slow.SetName("Slow")
	fast.SetName("Fast")
	slow.WaitTime, fast.WaitTime = 100, 100
	camp.AddNode(slow)
	camp.AddNode(fast)
	h.tree.Root().AddNode(camp)
	a.On(world.Update, "spawner", func(world.Frame) {
		if h.add.CompareAndSwap(true, false) {
			m := node3d.NewMarker()
			m.SetName("Flag")
			camp.AddNode(m)
		}
	})
	var stop atomic.Bool
	done := make(chan struct{})
	go func() {
		defer close(done)
		for !stop.Load() {
			a.Frame(1.0 / 60)
			time.Sleep(time.Millisecond)
		}
	}()
	var protocols http.Protocols
	protocols.SetUnencryptedHTTP2(true)
	h.http = &http.Client{Transport: &http.Transport{Protocols: &protocols}}
	h.url = "http://" + srv.Addr()
	h.scene = scenepbconnect.NewSceneServiceClient(h.http, h.url)
	t.Cleanup(func() {
		h.http.CloseIdleConnections()
		stop.Store(true)
		<-done
		srv.Stop()
	})
	return h
}

func names(n *scenepb.Node) []string {
	out := []string{n.GetName()}
	for _, c := range n.GetChildren() {
		out = append(out, names(c)...)
	}
	return out
}

func code(err error) connect.Code {
	var ce *connect.Error
	if errors.As(err, &ce) {
		return ce.Code()
	}
	return connect.CodeUnknown
}

func TestSceneServiceIsReflected(t *testing.T) {
	h := startScene(t, "")
	stream := grpcreflect.NewClient(h.http, h.url, connect.WithGRPC()).NewStream(ctx(t))
	defer stream.Close()
	list, err := stream.ListServices()
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range list {
		if n == "ember.scene.v1.SceneService" {
			return
		}
	}
	t.Fatalf("services: %v", list)
}

func TestGetTreeAndNode(t *testing.T) {
	h := startScene(t, "")
	tree, err := h.scene.GetTree(ctx(t), connect.NewRequest(&scenepb.GetTreeRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if got := names(tree.Msg.GetRoot()); len(got) != 4 || got[0] != "root" || got[1] != "Camp" || got[3] != "Fast" {
		t.Fatalf("tree %v", got)
	}
	shallow, err := h.scene.GetTree(ctx(t), connect.NewRequest(&scenepb.GetTreeRequest{Path: "/root/Camp", Depth: 1}))
	if err != nil {
		t.Fatal(err)
	}
	if c := shallow.Msg.GetRoot().GetChildren(); len(c) != 2 || len(c[0].GetChildren()) != 0 ||
		c[0].GetType() != "node.Timer" || c[0].GetPath() != "/root/Camp/Slow" {
		t.Fatalf("shallow %v", shallow.Msg)
	}

	n, err := h.scene.GetNode(ctx(t), connect.NewRequest(&scenepb.GetNodeRequest{Path: "/root/Camp/Slow"}))
	if err != nil {
		t.Fatal(err)
	}
	if n.Msg.GetProperties()["wait_time"].GetNumberValue() != 100 || len(n.Msg.GetSignals()) != 1 ||
		n.Msg.GetSignals()[0] != "timeout" {
		t.Fatalf("node %v", n.Msg)
	}
	_, err = h.scene.GetNode(ctx(t), connect.NewRequest(&scenepb.GetNodeRequest{Path: "/root/Nope"}))
	if code(err) != connect.CodeNotFound {
		t.Fatalf("missing node: %v", err)
	}
}

func TestSetProperty(t *testing.T) {
	h := startScene(t, "")
	set := func(name string, v *structpb.Value) error {
		_, err := h.scene.SetProperty(ctx(t), connect.NewRequest(&scenepb.SetPropertyRequest{
			Path:  "/root/Camp/Slow",
			Name:  name,
			Value: v,
		}))
		return err
	}
	if err := set("wait_time", structpb.NewNumberValue(2)); err != nil {
		t.Fatal(err)
	}
	n, err := h.scene.GetNode(ctx(t), connect.NewRequest(&scenepb.GetNodeRequest{Path: "/root/Camp/Slow"}))
	if err != nil || n.Msg.GetProperties()["wait_time"].GetNumberValue() != 2 {
		t.Fatalf("after set: %v %v", n, err)
	}
	if err = set("speed", structpb.NewNumberValue(1)); code(err) != connect.CodeInvalidArgument {
		t.Fatalf("unknown property: %v", err)
	}
	if err = set("wait_time", structpb.NewStringValue("soon")); code(err) != connect.CodeInvalidArgument {
		t.Fatalf("bad value: %v", err)
	}
}

func TestIsolateRunsOnlyASubtree(t *testing.T) {
	h := startScene(t, "")
	if _, err := h.scene.Isolate(ctx(t), connect.NewRequest(&scenepb.IsolateRequest{
		Path: "/root/Camp/Fast",
	})); err != nil {
		t.Fatal(err)
	}
	slow, fast := ticks(t, h, "Slow"), ticks(t, h, "Fast")
	eventually(t, "the isolated subtree to run", func() bool { return ticks(t, h, "Fast") >= fast+5 })
	if now := ticks(t, h, "Slow"); now != slow {
		t.Fatalf("the rest stays paused: slow %v→%v", slow, now)
	}
	if _, err := h.scene.Isolate(ctx(t), connect.NewRequest(&scenepb.IsolateRequest{})); err != nil {
		t.Fatal(err)
	}
	slow = ticks(t, h, "Slow")
	eventually(t, "ending the isolation to resume everything", func() bool { return ticks(t, h, "Slow") > slow })
}

func ticks(t *testing.T, h *sceneHarness, name string) float64 {
	t.Helper()
	n, err := h.scene.GetNode(ctx(t), connect.NewRequest(&scenepb.GetNodeRequest{Path: "/root/Camp/" + name}))
	if err != nil {
		t.Fatal(err)
	}
	return n.Msg.GetProperties()["ticks"].GetNumberValue()
}

func TestWatchTreeReportsChanges(t *testing.T) {
	h := startScene(t, "")
	stream, err := h.scene.WatchTree(ctx(t), connect.NewRequest(&scenepb.WatchTreeRequest{Path: "/root/Camp"}))
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	if !stream.Receive() || len(stream.Msg().GetChanges()) != 3 {
		t.Fatalf("first message lists the subtree: %v %v", stream.Msg(), stream.Err())
	}
	h.add.Store(true)
	if !stream.Receive() {
		t.Fatal(stream.Err())
	}
	c := stream.Msg().GetChanges()
	if len(c) != 1 || c[0].GetKind() != scenepb.WatchTreeResponse_KIND_ADDED ||
		c[0].GetNode().GetPath() != "/root/Camp/Flag" {
		t.Fatalf("changes %v", c)
	}
}

func TestSnapshotStreamsAScene(t *testing.T) {
	h := startScene(t, "")
	stream, err := h.scene.Snapshot(ctx(t), connect.NewRequest(&scenepb.SnapshotRequest{
		Path:      "/root/Camp",
		ChunkSize: 256,
	}))
	if err != nil {
		t.Fatal(err)
	}
	var data []byte
	chunks := 0
	var total uint64
	for stream.Receive() {
		data = append(data, stream.Msg().GetData()...)
		total = stream.Msg().GetTotal()
		chunks++
	}
	if err = stream.Err(); err != nil {
		t.Fatal(err)
	}
	if chunks < 2 || uint64(len(data)) != total {
		t.Fatalf("%d chunks, %d of %d bytes", chunks, len(data), total)
	}
	s, err := node.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if j := s.Model.Skeleton.Joints; len(j) != 3 || j[0].Name != "Camp" {
		t.Fatalf("snapshot joints %v", j)
	}
}

func TestSceneServiceNeedsATreeAndTheToken(t *testing.T) {
	h := start(t, "")
	c := scenepbconnect.NewSceneServiceClient(h.http, h.url)
	if _, err := c.GetTree(ctx(t), connect.NewRequest(
		&scenepb.GetTreeRequest{},
	)); code(err) != connect.CodeFailedPrecondition {
		t.Fatalf("no tree: %v", err)
	}
	s := startScene(t, "secret")
	if _, err := s.scene.GetTree(ctx(t), connect.NewRequest(
		&scenepb.GetTreeRequest{},
	)); code(err) != connect.CodeUnauthenticated {
		t.Fatalf("no token: %v", err)
	}
}
