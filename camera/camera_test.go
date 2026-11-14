package camera //nolint:testpackage // white-box: the options set unexported fields (offset, smoothing, shake…)

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/internal/wm"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/render"
	"github.com/lantervale/ember/scene"
	"github.com/lantervale/ember/world"
)

type thing struct {
	*world.BasicEntity
	scene.Transform
}

func TestFollowLooksAtItsTargetFromItsAngles(t *testing.T) {
	a := world.New().AddPlugins(wm.Plugin{}, Plugin{})
	a.AddSystems(&scene.TransformSystem{})
	target := &thing{world.NewBasic(), scene.At(5, 0, 5)}
	cam := NewFollow(target, Angles(0, geom.Pi/4), Distance(10))
	a.AddEntities(target, cam)
	a.RunFrames(1)

	want := geom.V3(5, 10*geom.Sin(geom.Pi/4), 5+10*geom.Cos(geom.Pi/4))
	if !cam.Position.ApproxEq(want, 1e-3) {
		t.Fatalf("camera at %v want %v", cam.Position, want)
	}
	dir := target.Position.Sub(cam.Position).Normalize()
	if !cam.Forward().ApproxEq(dir, 1e-3) {
		t.Fatalf("camera looks %v want %v", cam.Forward(), dir)
	}
	other := &thing{world.NewBasic(), scene.At(-5, 0, 0)}
	a.AddEntities(other)
	cam.SetTarget(other)
	a.RunFrames(1)
	if cam.Target() != other || cam.Position.X != -5 {
		t.Fatalf("SetTarget moves the camera to the new target: %v", cam.Position)
	}
}

func TestFollowOptionsAndDefaults(t *testing.T) {
	win := platform.NewWindow()
	cam := NewFollow(
		nil,
		Angles(1, 0.5),
		Distance(30),
		Offset(geom.V3(0, 2, 0)),
		ZoomRange(10, 60),
		Smoothing(6),
		In(win),
	)
	yaw, pitch := cam.Angles()
	if yaw != 1 || pitch != 0.5 || cam.Distance() != 30 || cam.offset != geom.V3(0, 2, 0) ||
		cam.nearest != 10 || cam.farthest != 60 || cam.smoothing != 6 || cam.Window != win {
		t.Fatalf("options: %+v", cam.Follow)
	}
	def := NewFollow(nil)
	if def.Distance() != 18 || def.smoothing != 0 || def.Window != nil {
		t.Fatalf("defaults: %+v", def.Follow)
	}
	def.AddShake(0.7)
	def.AddShake(0.7)
	if def.shake != 1 {
		t.Fatal("shake saturates at 1")
	}
}

func TestFollowSmoothingConverges(t *testing.T) {
	a := world.New().AddPlugins(wm.Plugin{}, Plugin{})
	target := &thing{world.NewBasic(), scene.At(0, 0, 0)}
	cam := NewFollow(target, Angles(0, 1), Distance(10), Smoothing(8))
	a.AddEntities(target, cam)
	a.RunFrames(1)
	target.Position = geom.V3(10, 0, 0)
	a.RunFrames(2)
	if cam.focus.X <= 0 || cam.focus.X >= 10 {
		t.Fatalf("focus should be in flight, got %v", cam.focus)
	}
	a.RunFrames(240)
	if !cam.focus.ApproxEq(target.Position, 1e-2) {
		t.Fatalf("focus %v did not converge", cam.focus)
	}
}

func TestGroundDirectionFollowsTheView(t *testing.T) {
	for _, yaw := range []float32{0, geom.Pi / 4, 2} {
		f := FollowBase(nil)
		f.SetAngles(yaw, 1)
		eye := geom.V3(geom.Sin(yaw), 0, geom.Cos(yaw))
		if up := f.GroundDirection(geom.V2(0, -1)); !up.ApproxEq(eye.Neg(), 1e-5) {
			t.Fatalf("yaw %v: up walks away from the camera, got %v", yaw, up)
		}
		right := f.GroundDirection(geom.V2(0.5, 0))
		if geom.Abs(right.Dot(eye)) > 1e-5 || geom.Abs(right.Len()-0.5) > 1e-5 || right.Y != 0 {
			t.Fatalf("yaw %v: right is across the view and keeps length, got %v", yaw, right)
		}
	}
	f := FollowBase(nil)
	if r := f.GroundDirection(geom.V2(1, 0)); !r.ApproxEq(geom.V3(1, 0, 0), 1e-5) {
		t.Fatalf("looking down -Z, right is +X: %v", r)
	}
}

func TestTheRightStickOfItsWindowZooms(t *testing.T) {
	a := world.New().AddPlugins(wm.Plugin{}, Plugin{})
	win := platform.NewWindow()
	target := &thing{world.NewBasic(), scene.At(0, 0, 0)}
	cam := NewFollow(target, Angles(0, 1), Distance(20), ZoomRange(10, 30))
	a.AddEntities(win, target, cam)
	win.Input().ConnectGamepad(0, "pad")
	win.Input().SetGamepadStick(0, input.RightStick, geom.V2(0, 1))
	a.RunFrames(10)
	if cam.Distance() <= 20 {
		t.Fatalf("pulling the right stick back zooms out, distance %v", cam.Distance())
	}
	a.RunFrames(600)
	if cam.Distance() != 30 {
		t.Fatalf("zoom is clamped, distance %v", cam.Distance())
	}
}

type ownFollow struct {
	*world.BasicEntity
	scene.Transform
	render.Camera
	Follow
}

func TestAGamesOwnCameraEmbedsFollowBase(t *testing.T) {
	win := platform.NewWindow()
	target := &thing{world.NewBasic(), scene.At(0, 0, 0)}
	cam := BuildFollow(
		&ownFollow{world.NewBasic(), scene.At(0, 0, 0), render.Camera{}, FollowBase(target)},
		Distance(5),
		In(win),
	)
	if cam.Distance() != 5 || cam.Window != win {
		t.Fatal("BuildFollow applies the options to the embedded Follow")
	}
}

func TestFlyMovesWithItsWindowsKeysAndLooksWithTheMouse(t *testing.T) {
	a := world.New().AddPlugins(wm.Plugin{}, Plugin{})
	win, other := platform.NewWindow(), platform.NewWindow()
	cam := NewFly(Speed(10), Sensitivity(0.01), In(win))
	a.AddEntities(win, other, cam)
	other.Input().SetKey(input.KeyW, true)
	a.Tick(1)
	if cam.Position != (geom.Vec3{}) {
		t.Fatal("another window's keys do nothing")
	}
	win.Input().SetKey(input.KeyW, true)
	for range 30 {
		a.Tick(1.0 / 60)
	}
	if !cam.Position.ApproxEq(geom.V3(0, 0, -5), 1e-4) {
		t.Fatalf("W flies forward at its speed: %v", cam.Position)
	}
	win.Input().SetKey(input.KeyW, false)
	in := win.Input()
	in.SetMouse(geom.V2(100, 100))
	a.Tick(1.0 / 60)
	in.SetMouseButton(input.MouseRight, true)
	in.SetMouse(geom.V2(120, 100))
	a.Tick(1.0 / 60)
	if yaw, _ := cam.Angles(); geom.Abs(yaw+0.2) > 1e-5 {
		t.Fatalf("dragging with the right button turns it: yaw %v", yaw)
	}
	if d := NewFly(); d.Speed() != 8 || d.sensitivity != 0.004 {
		t.Fatal("defaults")
	}
}

func TestACameraReadsTheInputOfTheWindowItIsUnder(t *testing.T) {
	a := world.New().AddPlugins(wm.Plugin{}, Plugin{})
	main, tool := platform.NewWindow(), platform.NewWindow()
	cam := NewFly(Speed(10))
	tool.AppendChild(cam)
	a.AddEntities(main, tool)
	main.Input().SetKey(input.KeyW, true)
	a.Tick(1)
	if cam.Position != (geom.Vec3{}) {
		t.Fatal("the primary window's keys do nothing")
	}
	tool.Input().SetKey(input.KeyW, true)
	for range 30 {
		a.Tick(1.0 / 60)
	}
	if !cam.Position.ApproxEq(geom.V3(0, 0, -5), 1e-4) {
		t.Fatalf("its own window's do: %v", cam.Position)
	}
}
