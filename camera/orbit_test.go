package camera //nolint:testpackage // white-box: the options set unexported fields (window, sensitivity, tools)

import (
	"testing"

	"github.com/lantervale/ember/internal/wm"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/render"
	"github.com/lantervale/ember/scene"
	"github.com/lantervale/ember/world"
)

func drag(a *world.World, in *input.State, b input.MouseButton, shift bool) {
	in.SetMouse(geom.V2(100, 100))
	a.RunFrames(1)
	in.SetKey(input.KeyLeftShift, shift)
	in.SetMouseButton(b, true)
	in.SetMouse(geom.V2(140, 120))
	a.RunFrames(1)
	in.SetMouseButton(b, false)
	in.SetKey(input.KeyLeftShift, false)
}

func TestNewOrbitIsSetUpByItsOptions(t *testing.T) {
	win := platform.NewWindow()
	cam := NewOrbit(
		Around(geom.V3(1, 2, 3)),
		Distance(6),
		Angles(0.5, 0.25),
		Sensitivity(0.01),
		ToolControls(),
		In(win),
	)
	yaw, pitch := cam.Angles()
	if cam.Center() != geom.V3(1, 2, 3) || cam.Distance() != 6 || yaw != 0.5 || pitch != 0.25 ||
		cam.sensitivity != 0.01 || !cam.tools || cam.window != win || cam.Window != win {
		t.Fatalf("options: %+v", cam.Orbit)
	}
	if d := cam.Position.Sub(cam.Center()).Len(); geom.Abs(d-6) > 1e-4 {
		t.Fatalf("placed on its orbit right away: %v", d)
	}
	if f := cam.Forward(); !f.ApproxEq(cam.Center().Sub(cam.Position).Normalize(), 1e-4) {
		t.Fatalf("looking at the center: %v", f)
	}
	def := NewOrbit(Sensitivity(0))
	if def.Distance() != 10 || def.sensitivity != 0.006 || def.Center() != geom.Zero3 || def.tools ||
		def.Window != nil {
		t.Fatalf("defaults: %+v", def.Orbit)
	}
}

func TestTheOrbitKeepsItsDistanceAndPitchInRange(t *testing.T) {
	o := OrbitBase()
	o.SetDistance(0.1)
	o.SetAngles(1, 3)
	if _, pitch := o.Angles(); o.Distance() != 0.5 || pitch != 1.5 {
		t.Fatalf("distance %v pitch %v", o.Distance(), pitch)
	}
	o.SetAngles(0, geom.Pi/2-1.5)
	o.SetDistance(2)
	var tr scene.Transform
	o.place(&tr)
	want := geom.V3(0, geom.Sin(geom.Pi/2-1.5), geom.Cos(geom.Pi/2-1.5)).Scale(2)
	if !tr.Position.ApproxEq(want, 1e-4) {
		t.Fatalf("position %v, want %v", tr.Position, want)
	}
}

func TestTheOrbitReadsOnlyItsOwnWindow(t *testing.T) {
	a := world.New().AddPlugins(wm.Plugin{}, Plugin{})
	primary, tool := platform.NewWindow(), platform.NewWindow()
	main, side := NewOrbit(Distance(5)), NewOrbit(Distance(5), In(tool))
	a.AddEntities(primary, tool, main, side)
	a.RunFrames(1)
	drag(a, tool.Input(), input.MouseLeft, false)
	if yaw, _ := side.Angles(); yaw == 0 {
		t.Fatal("dragging in the tool window orbits its camera")
	}
	if yaw, _ := main.Angles(); yaw != 0 {
		t.Fatal("the primary's camera ignores the tool window")
	}
	drag(a, primary.Input(), input.MouseLeft, false)
	if yaw, _ := main.Angles(); yaw == 0 {
		t.Fatal("without In, the camera reads the primary window")
	}
	in := primary.Input()
	in.AddWheel(geom.V2(0, 1))
	a.RunFrames(1)
	if main.Distance() >= 5 {
		t.Fatal("the wheel zooms in")
	}
}

type eye struct {
	*world.BasicEntity
	scene.Transform
	render.Camera
	Orbit
}

func TestOrbitToolControlsLeaveTheLeftButton(t *testing.T) {
	a := world.New().AddPlugins(wm.Plugin{}, Plugin{})
	win := platform.NewWindow()
	in := win.Input()
	cam := BuildOrbit(
		&eye{world.NewBasic(), scene.At(0, 0, 0), render.Camera{}, OrbitBase()},
		Distance(5),
		ToolControls(),
		In(win),
	)
	a.AddEntities(win, cam)
	a.RunFrames(1)
	if cam.Window != win {
		t.Fatal("a game's own camera draws in the orbit's window")
	}
	drag(a, in, input.MouseLeft, false)
	if yaw, pitch := cam.Angles(); yaw != 0 || pitch != 0 || cam.Center() != (geom.Vec3{}) {
		t.Fatal("the left button is the tool's: no orbit")
	}
	drag(a, in, input.MouseRight, false)
	yaw, pitch := cam.Angles()
	if yaw == 0 || pitch == 0 || cam.Center() != (geom.Vec3{}) {
		t.Fatal("the right button orbits")
	}
	for _, shift := range []bool{false, true} {
		before := cam.Center()
		b := input.MouseMiddle
		if shift {
			b = input.MouseRight
		}
		drag(a, in, b, shift)
		if y, _ := cam.Angles(); cam.Center() == before || y != yaw {
			t.Fatalf("middle or Shift+right pans, shift=%v", shift)
		}
		if d := cam.Position.Sub(cam.Center()).Len(); geom.Abs(d-5) > 1e-3 {
			t.Fatalf("panning keeps the distance: %v", d)
		}
	}
}
