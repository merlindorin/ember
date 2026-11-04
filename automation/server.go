//go:build !js

package automation

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/grpcreflect"
	"github.com/merlindorin/ember/automation/automationpb"
	"github.com/merlindorin/ember/automation/automationpb/automationpbconnect"
	"github.com/merlindorin/ember/automation/scenepb/scenepbconnect"
	"github.com/merlindorin/ember/internal/wm"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/world"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

// serve serves ctl's World on addr, and gives what stops it and the
// address listened on.
func serve(addr, token string, ctl *controller) (func(), string, error) {
	lis, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", addr)
	if err != nil {
		return nil, "", fmt.Errorf("listen %s: %w", addr, err)
	}
	opts := connect.WithInterceptors(bearer(token))
	mux := http.NewServeMux()
	mux.Handle(automationpbconnect.NewAutomationHandler(&service{ctl: ctl}, opts))
	mux.Handle(scenepbconnect.NewSceneServiceHandler(&sceneService{ctl: ctl}, opts))
	reflector := grpcreflect.NewStaticReflector(automationpbconnect.AutomationName, scenepbconnect.SceneServiceName)
	mux.Handle(grpcreflect.NewHandlerV1(reflector, opts))
	mux.Handle(grpcreflect.NewHandlerV1Alpha(reflector, opts))
	var protocols http.Protocols
	protocols.SetHTTP1(true)
	protocols.SetUnencryptedHTTP2(true)
	srv := &http.Server{Handler: mux, Protocols: &protocols, ReadHeaderTimeout: 10 * time.Second}
	go func() { _ = srv.Serve(lis) }()
	return func() { srv.Close() }, lis.Addr().String(), nil
}

// bearer requires "Authorization: Bearer <token>" on every call, unary or
// streaming, when token is set.
func bearer(token string) connect.Interceptor { return bearerAuth(token) }

type bearerAuth string

func (b bearerAuth) check(h http.Header) error {
	if b != "" && h.Get("Authorization") != "Bearer "+string(b) {
		return connect.NewError(connect.CodeUnauthenticated, errors.New("automation: missing or wrong bearer token"))
	}
	return nil
}

func (b bearerAuth) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if err := b.check(req.Header()); err != nil {
			return nil, err
		}
		return next(ctx, req)
	}
}

func (b bearerAuth) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (b bearerAuth) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		if err := b.check(conn.RequestHeader()); err != nil {
			return err
		}
		return next(ctx, conn)
	}
}

func failure(code connect.Code, msg string) error { return connect.NewError(code, errors.New(msg)) }

func reply[T any](msg *T, err error) (*connect.Response[T], error) {
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(msg), nil
}

type service struct {
	automationpbconnect.UnimplementedAutomationHandler
	ctl *controller
}

func (s *service) GetUI(
	_ context.Context, r *connect.Request[automationpb.GetUIRequest],
) (*connect.Response[automationpb.GetUIResponse], error) {
	return connect.NewResponse(s.getUI(r.Msg)), nil
}

func (s *service) Act(
	ctx context.Context, r *connect.Request[automationpb.ActRequest],
) (*connect.Response[automationpb.ActResponse], error) {
	return reply(s.act(ctx, r.Msg))
}

func (s *service) ListActions(
	ctx context.Context, r *connect.Request[automationpb.ListActionsRequest],
) (*connect.Response[automationpb.ListActionsResponse], error) {
	return reply(s.listActions(ctx, r.Msg))
}

func (s *service) InvokeAction(
	ctx context.Context, r *connect.Request[automationpb.InvokeActionRequest],
) (*connect.Response[automationpb.InvokeActionResponse], error) {
	return reply(s.invokeAction(ctx, r.Msg))
}

func (s *service) ListEntities(
	ctx context.Context, r *connect.Request[automationpb.ListEntitiesRequest],
) (*connect.Response[automationpb.ListEntitiesResponse], error) {
	return reply(s.listEntities(ctx, r.Msg))
}

func (s *service) ListResources(
	ctx context.Context, r *connect.Request[automationpb.ListResourcesRequest],
) (*connect.Response[automationpb.ListResourcesResponse], error) {
	return reply(s.listResources(ctx, r.Msg))
}

func (s *service) GetProperty(
	ctx context.Context, r *connect.Request[automationpb.GetPropertyRequest],
) (*connect.Response[automationpb.GetPropertyResponse], error) {
	return reply(s.getProperty(ctx, r.Msg))
}

func (s *service) SetProperty(
	ctx context.Context, r *connect.Request[automationpb.SetPropertyRequest],
) (*connect.Response[automationpb.SetPropertyResponse], error) {
	return reply(s.setProperty(ctx, r.Msg))
}

func (s *service) SetPaused(
	ctx context.Context, r *connect.Request[automationpb.SetPausedRequest],
) (*connect.Response[automationpb.SetPausedResponse], error) {
	return reply(s.setPaused(ctx, r.Msg))
}

func (s *service) Step(
	ctx context.Context, r *connect.Request[automationpb.StepRequest],
) (*connect.Response[automationpb.StepResponse], error) {
	return reply(s.step(ctx, r.Msg))
}

func (s *service) Screenshot(
	ctx context.Context, r *connect.Request[automationpb.ScreenshotRequest],
) (*connect.Response[automationpb.ScreenshotResponse], error) {
	return reply(s.screenshot(ctx, r.Msg))
}

func (s *service) getUI(req *automationpb.GetUIRequest) *automationpb.GetUIResponse {
	frame, elements := s.ctl.reg.Snapshot(req.GetWindow(), req.GetPrefix())
	return &automationpb.GetUIResponse{Frame: frame, Elements: toProto(elements)}
}

func (s *service) WatchUI(
	ctx context.Context,
	r *connect.Request[automationpb.WatchUIRequest],
	stream *connect.ServerStream[automationpb.WatchUIResponse],
) error {
	req := r.Msg
	wake, stop := s.ctl.reg.watch()
	defer stop()
	var last []*automationpb.Element
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-wake:
		}
		frame, elements := s.ctl.reg.Snapshot(req.GetWindow(), req.GetPrefix())
		pb := toProto(elements)
		if sameElements(pb, last) {
			continue
		}
		last = pb
		if err := stream.Send(&automationpb.WatchUIResponse{Frame: frame, Elements: pb}); err != nil {
			return err
		}
	}
}

func (s *service) act(ctx context.Context, req *automationpb.ActRequest) (*automationpb.ActResponse, error) {
	if req.GetId() == "" {
		return nil, failure(connect.CodeInvalidArgument, "id is required")
	}
	kind := req.GetAct()
	if kind == "" {
		kind = ActPress
	}
	done := s.ctl.reg.Queue(req.GetId(), kind, req.GetValue().AsInterface())
	select {
	case r := <-done:
		if r.Err != nil {
			return nil, failure(connect.CodeNotFound, r.Err.Error())
		}
		return &automationpb.ActResponse{Element: elementProto(r.Element)}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (s *service) listActions(
	ctx context.Context, req *automationpb.ListActionsRequest,
) (*automationpb.ListActionsResponse, error) {
	return do(ctx, s.ctl, func() (*automationpb.ListActionsResponse, error) {
		resp := &automationpb.ListActionsResponse{}
		for _, a := range s.ctl.reg.Actions(req.GetPrefix()) {
			resp.Actions = append(resp.Actions, &automationpb.Action{
				Id:      a.ID,
				Label:   a.Label,
				Enabled: a.Enabled == nil || a.Enabled(),
			})
		}
		return resp, nil
	})
}

func (s *service) invokeAction(
	ctx context.Context, req *automationpb.InvokeActionRequest,
) (*automationpb.InvokeActionResponse, error) {
	return do(ctx, s.ctl, func() (*automationpb.InvokeActionResponse, error) {
		out, err := s.ctl.reg.Invoke(req.GetId(), req.GetArgs().AsMap())
		if err != nil {
			return nil, failure(connect.CodeFailedPrecondition, err.Error())
		}
		v, err := toValue(out)
		if err != nil {
			return nil, failure(connect.CodeInternal, err.Error())
		}
		return &automationpb.InvokeActionResponse{Result: v}, nil
	})
}

func (s *service) listEntities(
	ctx context.Context, req *automationpb.ListEntitiesRequest,
) (*automationpb.ListEntitiesResponse, error) {
	return do(ctx, s.ctl, func() (*automationpb.ListEntitiesResponse, error) {
		infos, total := Entities(s.ctl.world, EntityFilter{
			Type:      req.GetType(),
			Component: req.GetComponent(),
			Name:      req.GetName(),
			Limit:     int(req.GetLimit()),
		})
		resp := &automationpb.ListEntitiesResponse{Total: int32(total)} //nolint:gosec // entity counts stay far below 2^31
		for _, e := range infos {
			resp.Entities = append(resp.Entities, &automationpb.Entity{
				Id:         e.ID,
				Type:       e.Type,
				Name:       e.Name,
				Components: e.Components,
				Parent:     e.Parent,
				Children:   e.Children,
			})
		}
		return resp, nil
	})
}

func (s *service) listResources(
	ctx context.Context, _ *automationpb.ListResourcesRequest,
) (*automationpb.ListResourcesResponse, error) {
	return do(ctx, s.ctl, func() (*automationpb.ListResourcesResponse, error) {
		return &automationpb.ListResourcesResponse{Types: Resources(s.ctl.world)}, nil
	})
}

func (s *service) getProperty(
	ctx context.Context, req *automationpb.GetPropertyRequest,
) (*automationpb.GetPropertyResponse, error) {
	return do(ctx, s.ctl, func() (*automationpb.GetPropertyResponse, error) {
		v, typ, err := Get(s.ctl.world, req.GetPath(), int(req.GetDepth()))
		if err != nil {
			return nil, failure(connect.CodeNotFound, err.Error())
		}
		pv, err := toValue(v)
		if err != nil {
			return nil, failure(connect.CodeInternal, err.Error())
		}
		return &automationpb.GetPropertyResponse{Value: pv, Type: typ}, nil
	})
}

func (s *service) setProperty(
	ctx context.Context, req *automationpb.SetPropertyRequest,
) (*automationpb.SetPropertyResponse, error) {
	return do(ctx, s.ctl, func() (*automationpb.SetPropertyResponse, error) {
		v, err := Set(s.ctl.world, req.GetPath(), req.GetValue().AsInterface())
		if err != nil {
			return nil, failure(connect.CodeInvalidArgument, err.Error())
		}
		pv, err := toValue(v)
		if err != nil {
			return nil, failure(connect.CodeInternal, err.Error())
		}
		return &automationpb.SetPropertyResponse{Value: pv}, nil
	})
}

func (s *service) setPaused(
	ctx context.Context, req *automationpb.SetPausedRequest,
) (*automationpb.SetPausedResponse, error) {
	return do(ctx, s.ctl, func() (*automationpb.SetPausedResponse, error) {
		s.ctl.paused = req.GetPaused()
		return &automationpb.SetPausedResponse{Paused: s.ctl.paused}, nil
	})
}

func (s *service) step(ctx context.Context, req *automationpb.StepRequest) (*automationpb.StepResponse, error) {
	n := max(int(req.GetFrames()), 1)
	done, err := do(ctx, s.ctl, func() (chan uint64, error) {
		ch := make(chan uint64, 1)
		s.ctl.steps += n
		s.ctl.stepDone = append(s.ctl.stepDone, ch)
		return ch, nil
	})
	if err != nil {
		return nil, err
	}
	select {
	case frame := <-done:
		return &automationpb.StepResponse{Frame: frame}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (s *service) screenshot(
	ctx context.Context, req *automationpb.ScreenshotRequest,
) (*automationpb.ScreenshotResponse, error) {
	done, err := do(ctx, s.ctl, func() (chan shotResult, error) {
		win := findWindow(s.ctl.world, req.GetWindow())
		if win == nil {
			return nil, failure(connect.CodeNotFound, fmt.Sprintf("no window %q", req.GetWindow()))
		}
		path := filepath.Join(os.TempDir(), fmt.Sprintf("ember-shot-%d.png", time.Now().UnixNano()))
		win.Capture(path)
		sh := &shot{path: path, done: make(chan shotResult, 1)}
		s.ctl.shots = append(s.ctl.shots, sh)
		return sh.done, nil
	})
	if err != nil {
		return nil, err
	}
	select {
	case r := <-done:
		if r.err != nil {
			return nil, failure(connect.CodeUnavailable, r.err.Error())
		}
		return &automationpb.ScreenshotResponse{Png: r.png}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func findWindow(w *world.World, title string) *platform.Window {
	if title == "" {
		return wm.Resolve(w, nil)
	}
	for _, e := range w.Entities() {
		if win, ok := e.(platform.Windowed); ok && win.GetWindow().Title() == title {
			return win.GetWindow()
		}
	}
	return nil
}

func toValue(v any) (*structpb.Value, error) {
	return structpb.NewValue(v)
}

func elementProto(e Element) *automationpb.Element {
	v, err := toValue(plain(e.Value))
	if err != nil {
		v = structpb.NewStringValue(fmt.Sprint(e.Value))
	}
	return &automationpb.Element{
		Id: e.ID, Role: e.Role, Label: e.Label, Value: v,
		Enabled: e.Enabled, Window: e.Window, Acts: e.Acts, Focused: e.Focused,
		Bounds: &automationpb.Rect{X: e.Bounds.X, Y: e.Bounds.Y, W: e.Bounds.W, H: e.Bounds.H},
	}
}

func plain(v any) any {
	if v == nil {
		return nil
	}
	switch x := v.(type) {
	case bool, string, float64:
		return x
	}
	return Encode(reflectValue(v), 4)
}

func toProto(elements []Element) []*automationpb.Element {
	out := make([]*automationpb.Element, len(elements))
	for i, e := range elements {
		out[i] = elementProto(e)
	}
	return out
}

func sameElements(a, b []*automationpb.Element) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !proto.Equal(a[i], b[i]) {
			return false
		}
	}
	return true
}

// do runs fn on the main loop and waits for it.
func do[T any](ctx context.Context, c *controller, fn func() (T, error)) (T, error) {
	type result struct {
		v   T
		err error
	}
	ch := make(chan result, 1)
	select {
	case c.jobs <- func() { v, err := fn(); ch <- result{v, err} }:
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	}
	select {
	case r := <-ch:
		return r.v, r.err
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	}
}

// watch gives a channel woken when a frame publishes, and what stops it.
func (r *Registry) watch() (<-chan struct{}, func()) {
	ch := make(chan struct{}, 1)
	r.mu.Lock()
	r.watchers = append(r.watchers, ch)
	r.mu.Unlock()
	return ch, func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		for i, w := range r.watchers {
			if w == ch {
				r.watchers = append(r.watchers[:i], r.watchers[i+1:]...)
				return
			}
		}
	}
}
