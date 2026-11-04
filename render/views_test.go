package render_test

import (
	"testing"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/render"
	"github.com/merlindorin/ember/scene"
	"github.com/merlindorin/ember/world"
)

type eye struct {
	*world.BasicEntity
	scene.Transform
	render.Camera
}

func newEye(pos geom.Vec3, c render.Camera) *eye {
	return &eye{world.NewBasic(), scene.AtVec(pos).LookingAt(geom.Zero3), c}
}

func multiWindowApp() (*world.World, *platform.Window, *platform.Window) {
	a := world.New().AddPlugins(render.Plugin{})
	primary := platform.NewWindow(platform.Size(800, 600))
	a.AddEntities(primary)
	tool := platform.NewWindow(platform.Size(300, 300))
	return a, primary, tool
}

func TestEachWindowPresentsItsOwnCamera(t *testing.T) {
	a, _, tool := multiWindowApp()
	main := newEye(geom.V3(0, 10, 10), render.Camera{})
	high := newEye(geom.V3(0, 5, 20), render.Camera{Priority: 5})
	mapCam := newEye(geom.V3(0, 50, 0.01), render.Camera{OrthoHeight: 40, Window: tool})
	a.AddEntities(main, high, mapCam)
	a.RunFrames(1)

	if !high.Presenting() || main.Presenting() {
		t.Fatal("the primary window presents its highest-priority camera")
	}
	if !main.View().Active || !main.View().Position.ApproxEq(main.Position, 1e-4) {
		t.Fatal("every camera resolves its view")
	}
	if v := mapCam.View(); !mapCam.Presenting() || v.Viewport != geom.V2(300, 300) || v.Projection.OrthoHeight != 40 {
		t.Fatalf("tool window view %+v", v)
	}
}

func TestACameraRendersInTheWindowItIsUnder(t *testing.T) {
	a, primary, tool := multiWindowApp()
	inTool := newEye(geom.V3(0, 5, 5), render.Camera{})
	tool.AppendChild(inTool)
	elsewhere := newEye(geom.V3(0, 5, 5), render.Camera{Window: primary})
	tool.AppendChild(elsewhere)
	loose := newEye(geom.V3(0, 5, 5), render.Camera{Priority: -1})
	a.AddEntities(tool, loose)
	a.RunFrames(1)
	if inTool.RendersIn() != tool || !inTool.Presenting() || inTool.View().Viewport != geom.V2(300, 300) {
		t.Fatal("a camera under a window renders in it")
	}
	if elsewhere.RendersIn() != primary || loose.RendersIn() != primary {
		t.Fatal("Window wins, and a camera under no window renders in the primary")
	}
}

func TestGizmosAreEntitiesClearedAtFrameStart(t *testing.T) {
	a, _, _ := multiWindowApp()
	g := render.NewGizmos()
	a.AddEntities(g)
	a.On(world.Update, "draw", func(world.Frame) { g.Line(geom.Zero3, geom.Up, color.Red) })
	a.Tick(1.0 / 60)
	if n := len(g.Lines); n != 1 {
		t.Fatalf("after tick: %d lines", n)
	}
	a.Present()
	a.Present()
	if n := len(g.Lines); n != 1 {
		t.Fatal("rendering several windows must not consume the buffers")
	}
	a.Tick(1.0 / 60)
	if n := len(g.Lines); n != 1 {
		t.Fatalf("buffers must be reset at frame start, got %d", n)
	}
	b := render.NewBillboards()
	a.AddEntities(b)
	b.Glow(geom.Zero3, 1, color.Red)
	a.Tick(1.0 / 60)
	if len(b.Items) != 0 {
		t.Fatal("billboards are cleared too")
	}
}
