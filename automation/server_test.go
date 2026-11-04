package automation_test

import (
	"context"
	"net/http"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/grpcreflect"
	"github.com/merlindorin/ember/automation"
	"github.com/merlindorin/ember/automation/automationpb"
	"github.com/merlindorin/ember/automation/automationpb/automationpbconnect"
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/scene"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/world"
	"google.golang.org/protobuf/types/known/structpb"
)

type crate struct {
	*world.BasicEntity
	scene.Transform
	Name string
	Wood int
}

type harness struct {
	app    *world.World
	client *client
	http   *http.Client
	url    string
	crate  *crate
	clicks atomic.Int32
}

// hud draws a button, a meter and a second button, recording them as
// hud/craft-axe, hud/wood and hud/close.
type hud struct {
	twod.Item
	h *harness
}

func (d *hud) Draw(c *twod.Canvas) {
	c.Push("hud")
	defer c.Pop()
	craft := geom.Rect{X: 10, Y: 10, W: 120, H: 30}
	if in := c.Interact(craft, "craft-axe"); in.Clicked {
		d.h.clicks.Add(1)
	}
	c.Rect(craft, color.White)
	c.Record("craft-axe", automation.RoleButton, "Craft Axe", nil, true, craft, automation.ActPress)
	c.Record("wood", automation.RoleMeter, "Wood", d.h.crate.Wood, true, geom.Rect{})
	c.Record(
		"close", automation.RoleButton, "Fermer", nil, true, geom.Rect{X: 10, Y: 50, W: 120, H: 30}, automation.ActPress,
	)
}

// start runs a headless app with a HUD button on its own main loop and
// connects a client speaking the gRPC protocol, as grpcurl does, to its
// automation server.
func start(t *testing.T, token string) *harness {
	t.Helper()
	h := &harness{app: world.New()}
	auto := automation.New(automation.Addr("127.0.0.1:0"), automation.Token(token))
	h.app.AddPlugins(auto, twod.Plugin{})
	h.crate = &crate{BasicEntity: world.NewBasic(), Transform: scene.At(1, 0, 2), Name: "supply", Wood: 3}
	h.app.AddEntities(h.crate)
	h.app.AddEntities(twod.Build(&hud{Item: twod.Base(), h: h}, "hud"))
	auto.Define(automation.Action{ID: "craft.chop", Label: "add wood", Run: func(args map[string]any) (any, error) {
		n, _ := args["n"].(float64)
		h.crate.Wood += int(n)
		return float64(h.crate.Wood), nil
	}})

	var stop atomic.Bool
	done := make(chan struct{})
	go func() {
		defer close(done)
		for !stop.Load() {
			h.app.Frame(1.0 / 60)
			time.Sleep(time.Millisecond)
		}
	}()
	var protocols http.Protocols
	protocols.SetUnencryptedHTTP2(true)
	h.http = &http.Client{Transport: &http.Transport{Protocols: &protocols}}
	h.url = "http://" + auto.Addr()
	h.client = &client{automationpbconnect.NewAutomationClient(h.http, h.url, connect.WithGRPC())}
	t.Cleanup(func() {
		h.http.CloseIdleConnections()
		stop.Store(true)
		<-done
		auto.Stop()
	})
	return h
}

func ctx(t *testing.T) context.Context {
	c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return c
}

func TestServerReflectionListsTheService(t *testing.T) {
	h := start(t, "")
	stream := grpcreflect.NewClient(h.http, h.url, connect.WithGRPC()).NewStream(ctx(t))
	defer stream.Close()
	names, err := stream.ListServices()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, n := range names {
		found = found || n == "ember.automation.v1.Automation"
	}
	if !found {
		t.Fatalf("services: %v", names)
	}
}

func TestUISnapshotAndPressingAButton(t *testing.T) {
	h := start(t, "")
	var ui *automationpb.GetUIResponse
	for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); time.Sleep(5 * time.Millisecond) {
		var err error
		if ui, err = h.client.GetUI(ctx(t), &automationpb.GetUIRequest{Prefix: "hud"}); err != nil {
			t.Fatal(err)
		}
		if len(ui.Elements) == 3 {
			break
		}
	}
	if len(ui.GetElements()) != 3 {
		t.Fatalf("elements %v", ui.GetElements())
	}
	if b := ui.Elements[2]; b.Id != "hud/close" || b.Label != "Fermer" {
		t.Fatalf("a button with its own id is %v: its id must not follow its translated label", b)
	}
	button := ui.Elements[0]
	if button.Id != "hud/craft-axe" || button.Role != automation.RoleButton || button.Bounds.W != 120 {
		t.Fatalf("button %v", button)
	}
	if meter := ui.Elements[1]; meter.Id != "hud/wood" || meter.Value.GetNumberValue() != 3 {
		t.Fatalf("meter %v", meter)
	}

	resp, err := h.client.Act(ctx(t), &automationpb.ActRequest{Id: "hud/craft-axe", Act: automation.ActPress})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Element.Label != "Craft Axe" || h.clicks.Load() != 1 {
		t.Fatalf("clicks %d element %v", h.clicks.Load(), resp.Element)
	}
	if _, err = h.client.Act(ctx(t), &automationpb.ActRequest{
		Id: "hud/missing",
	}); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("acting on a missing element: %v", err)
	}
}

func TestActionsEntitiesAndProperties(t *testing.T) {
	h := start(t, "")
	c := h.client
	actions, err := c.ListActions(ctx(t), &automationpb.ListActionsRequest{Prefix: "craft"})
	if err != nil || len(actions.Actions) != 1 || actions.Actions[0].Id != "craft.chop" {
		t.Fatalf("%v %v", actions, err)
	}
	args, _ := structpb.NewStruct(map[string]any{"n": 4})
	out, err := c.InvokeAction(ctx(t), &automationpb.InvokeActionRequest{Id: "craft.chop", Args: args})
	if err != nil || out.Result.GetNumberValue() != 7 {
		t.Fatalf("%v %v", out, err)
	}

	list, err := c.ListEntities(ctx(t), &automationpb.ListEntitiesRequest{Component: "Transform"})
	if err != nil || list.Total != 1 || list.Entities[0].Name != "supply" {
		t.Fatalf("%v %v", list, err)
	}
	base := "entity/" + strconv.FormatUint(h.crate.ID(), 10)
	got, err := c.GetProperty(ctx(t), &automationpb.GetPropertyRequest{Path: base + "/Transform/Position"})
	if err != nil || got.Type != "geom.Vec3" || got.Value.GetStructValue().Fields["Z"].GetNumberValue() != 2 {
		t.Fatalf("%v %v", got, err)
	}
	if _, err = c.SetProperty(ctx(t), &automationpb.SetPropertyRequest{
		Path:  base + "/Wood",
		Value: structpb.NewNumberValue(42),
	}); err != nil {
		t.Fatal(err)
	}
	if v, _ := c.GetProperty(ctx(t), &automationpb.GetPropertyRequest{
		Path: base + "/Wood",
	}); v.Value.GetNumberValue() != 42 {
		t.Fatalf("wood %v", v)
	}
	if _, err = c.GetProperty(ctx(t), &automationpb.GetPropertyRequest{
		Path: "entity/0/Nope",
	}); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("bad path: %v", err)
	}
	res, err := c.ListResources(ctx(t), &automationpb.ListResourcesRequest{})
	if err != nil || len(res.Types) == 0 {
		t.Fatalf("%v %v", res, err)
	}
}

func TestPauseAndStepControlTime(t *testing.T) {
	h := start(t, "")
	c := h.client
	if _, err := c.SetPaused(ctx(t), &automationpb.SetPausedRequest{Paused: true}); err != nil {
		t.Fatal(err)
	}
	before := elapsed(t, c)
	frame := number(t, c, "time/frame")
	eventually(t, "frames to run while paused", func() bool { return number(t, c, "time/frame") >= frame+5 })
	if after := elapsed(t, c); after != before {
		t.Fatalf("time advanced while paused: %v -> %v", before, after)
	}
	if _, err := c.Step(ctx(t), &automationpb.StepRequest{Frames: 10}); err != nil {
		t.Fatal(err)
	}
	if after := elapsed(t, c); after <= before {
		t.Fatalf("step did not advance time: %v -> %v", before, after)
	}
}

func elapsed(t *testing.T, c *client) float64 { return number(t, c, "time/elapsed") }

func number(t *testing.T, c *client, path string) float64 {
	t.Helper()
	v, err := c.GetProperty(ctx(t), &automationpb.GetPropertyRequest{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	return v.Value.GetNumberValue()
}

// eventually polls cond until it holds, failing after a deadline: tests
// wait on what they need instead of sleeping a guessed time.
func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); !cond(); time.Sleep(2 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
	}
}

func TestTokenIsRequiredWhenConfigured(t *testing.T) {
	h := start(t, "s3cret")
	if _, err := h.client.ListResources(
		ctx(t), &automationpb.ListResourcesRequest{},
	); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("missing token: %v", err)
	}
	authed := withToken(ctx(t), "s3cret")
	if _, err := h.client.ListResources(authed, &automationpb.ListResourcesRequest{}); err != nil {
		t.Fatal(err)
	}
}

func TestWatchUIStreamsChanges(t *testing.T) {
	h := start(t, "")
	stream, err := h.client.WatchUI(ctx(t), &automationpb.WatchUIRequest{Prefix: "hud/wood"})
	if err != nil {
		t.Fatal(err)
	}
	first, err := stream.Recv()
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Elements) != 1 {
		t.Fatalf("first update %v", first)
	}
	args, _ := structpb.NewStruct(map[string]any{"n": 1})
	if _, err = h.client.InvokeAction(ctx(t), &automationpb.InvokeActionRequest{
		Id:   "craft.chop",
		Args: args,
	}); err != nil {
		t.Fatal(err)
	}
	next, err := stream.Recv()
	if err != nil {
		t.Fatal(err)
	}
	if next.Elements[0].Value.GetNumberValue() != first.Elements[0].Value.GetNumberValue()+1 {
		t.Fatalf("stream must deliver the change: %v then %v", first.Elements[0].Value, next.Elements[0].Value)
	}
}

func TestScreenshotFailsCleanlyWithoutAWindow(t *testing.T) {
	h := start(t, "")
	if _, err := h.client.Screenshot(
		ctx(t), &automationpb.ScreenshotRequest{},
	); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("headless screenshot: %v", err)
	}
}

// client calls the Connect client the way the tests did a gRPC one.
type client struct {
	c automationpbconnect.AutomationClient
}

type tokenKey struct{}

func withToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, tokenKey{}, token)
}

func call[Req, Resp any](
	ctx context.Context, f func(context.Context, *connect.Request[Req]) (*connect.Response[Resp], error), msg *Req,
) (*Resp, error) {
	req := connect.NewRequest(msg)
	if token, ok := ctx.Value(tokenKey{}).(string); ok {
		req.Header().Set("Authorization", "Bearer "+token)
	}
	resp, err := f(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

func (c *client) GetUI(ctx context.Context, m *automationpb.GetUIRequest) (*automationpb.GetUIResponse, error) {
	return call(ctx, c.c.GetUI, m)
}

func (c *client) Act(ctx context.Context, m *automationpb.ActRequest) (*automationpb.ActResponse, error) {
	return call(ctx, c.c.Act, m)
}

func (c *client) ListActions(
	ctx context.Context, m *automationpb.ListActionsRequest,
) (*automationpb.ListActionsResponse, error) {
	return call(ctx, c.c.ListActions, m)
}

func (c *client) InvokeAction(
	ctx context.Context, m *automationpb.InvokeActionRequest,
) (*automationpb.InvokeActionResponse, error) {
	return call(ctx, c.c.InvokeAction, m)
}

func (c *client) ListEntities(
	ctx context.Context, m *automationpb.ListEntitiesRequest,
) (*automationpb.ListEntitiesResponse, error) {
	return call(ctx, c.c.ListEntities, m)
}

func (c *client) ListResources(
	ctx context.Context, m *automationpb.ListResourcesRequest,
) (*automationpb.ListResourcesResponse, error) {
	return call(ctx, c.c.ListResources, m)
}

func (c *client) GetProperty(
	ctx context.Context, m *automationpb.GetPropertyRequest,
) (*automationpb.GetPropertyResponse, error) {
	return call(ctx, c.c.GetProperty, m)
}

func (c *client) SetProperty(
	ctx context.Context, m *automationpb.SetPropertyRequest,
) (*automationpb.SetPropertyResponse, error) {
	return call(ctx, c.c.SetProperty, m)
}

func (c *client) SetPaused(
	ctx context.Context, m *automationpb.SetPausedRequest,
) (*automationpb.SetPausedResponse, error) {
	return call(ctx, c.c.SetPaused, m)
}

func (c *client) Step(ctx context.Context, m *automationpb.StepRequest) (*automationpb.StepResponse, error) {
	return call(ctx, c.c.Step, m)
}

func (c *client) Screenshot(
	ctx context.Context, m *automationpb.ScreenshotRequest,
) (*automationpb.ScreenshotResponse, error) {
	return call(ctx, c.c.Screenshot, m)
}

type watch struct {
	s *connect.ServerStreamForClient[automationpb.WatchUIResponse]
}

func (w watch) Recv() (*automationpb.WatchUIResponse, error) {
	if !w.s.Receive() {
		return nil, w.s.Err()
	}
	return w.s.Msg(), nil
}

func (c *client) WatchUI(ctx context.Context, m *automationpb.WatchUIRequest) (watch, error) {
	s, err := c.c.WatchUI(ctx, connect.NewRequest(m))
	return watch{s}, err
}

func TestTheConnectProtocolWorksToo(t *testing.T) {
	h := start(t, "")
	c := automationpbconnect.NewAutomationClient(h.http, h.url)
	if res, err := c.ListResources(ctx(t), connect.NewRequest(&automationpb.ListResourcesRequest{})); err != nil ||
		len(res.Msg.Types) == 0 {
		t.Fatalf("%v %v", res, err)
	}
}
