package camera

import (
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/internal/inputfeed"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/render"
	"github.com/merlindorin/ember/scene"
	"github.com/merlindorin/ember/world"
)

// Orbit circles a point with the mouse and zooms with the wheel: the left
// or right button orbits, and the wheel zooms. It reads the input of its
// window only.
type Orbit struct {
	center               geom.Vec3
	yaw, pitch, distance float32
	sensitivity          float32
	tools                bool
	window               *platform.Window
}

// OrbitCamera is a ready-made orbiting camera entity.
type OrbitCamera struct {
	*world.BasicEntity
	scene.Transform
	render.Camera
	Orbit
}

// NewOrbit returns an orbiting camera set up by opts.
//
//	cam := camera.NewOrbit(camera.Around(geom.V3(0, 1, 0)), camera.Distance(6), camera.In(win))
func NewOrbit(opts ...OrbitOption) *OrbitCamera {
	return BuildOrbit(
		&OrbitCamera{BasicEntity: world.NewBasic(), Transform: scene.At(0, 0, 0), Orbit: OrbitBase()},
		opts...)
}

// OrbitBase is the Orbit a game's own camera entity embeds before
// BuildOrbit.
func OrbitBase() Orbit { return Orbit{distance: 10, sensitivity: 0.006} }

// BuildOrbit applies opts to a game's own camera entity, which embeds an
// Orbit made by OrbitBase. A camera drawing in the orbit's window when it
// has one, it is placed on its orbit right away.
//
//	eye := camera.BuildOrbit(&Eye{world.NewBasic(), scene.At(0, 0, 0), render.Camera{}, camera.OrbitBase()},
//		camera.Distance(6))
func BuildOrbit[T Orbiter](e T, opts ...OrbitOption) T {
	o := e.GetOrbit()
	for _, opt := range opts {
		opt.orbit(o)
	}
	if c, ok := any(e).(interface{ GetCamera() *render.Camera }); ok && o.window != nil {
		c.GetCamera().Window = o.window
	}
	o.place(e.GetTransform())
	return e
}

// Around orbits center (default the origin).
func Around(center geom.Vec3) OrbitOption { return orbitOption(func(o *Orbit) { o.center = center }) }

// ToolControls leaves the left button to the application, as in modelling
// tools: the right button orbits, the middle button (or Shift and the right
// button) pans, the wheel zooms.
func ToolControls() OrbitOption { return orbitOption(func(o *Orbit) { o.tools = true }) }

func (o *Orbit) GetOrbit() *Orbit { return o }

func (o *Orbit) Center() geom.Vec3 { return o.center }

func (o *Orbit) SetCenter(c geom.Vec3) { o.center = c }

func (o *Orbit) Distance() float32 { return o.distance }

// SetDistance keeps the distance above half a unit.
func (o *Orbit) SetDistance(d float32) { o.distance = max(d, 0.5) }

// Angles are the yaw and the pitch.
func (o *Orbit) Angles() (float32, float32) { return o.yaw, o.pitch }

// SetAngles keeps the pitch short of straight up or down.
func (o *Orbit) SetAngles(yaw, pitch float32) { o.yaw, o.pitch = yaw, geom.Clamp(pitch, -1.5, 1.5) }

func (o *Orbit) direction() geom.Vec3 {
	return geom.V3(geom.Sin(o.yaw)*geom.Cos(o.pitch), geom.Sin(o.pitch), geom.Cos(o.yaw)*geom.Cos(o.pitch))
}

func (o *Orbit) drive(in *input.State) {
	if in.PointerCaptured() {
		return
	}
	d := in.MouseDelta()
	shift := inputfeed.Of(in).RawDown(int(input.KeyLeftShift)) || inputfeed.Of(in).RawDown(int(input.KeyRightShift))
	pan := o.tools && (in.Down(input.MouseMiddle) || (shift && in.Down(input.MouseRight)))
	look := in.Down(input.MouseRight) || (!o.tools && in.Down(input.MouseLeft))
	switch {
	case pan:
		dir := o.direction()
		right := geom.Up.Cross(dir).Normalize()
		up := dir.Cross(right)
		k := o.distance * 0.0015
		o.center = o.center.Add(right.Scale(-d.X * k)).Add(up.Scale(d.Y * k))
	case look:
		o.SetAngles(o.yaw-d.X*o.sensitivity, o.pitch+d.Y*o.sensitivity)
	}
	o.SetDistance(o.distance * (1 - in.Wheel().Y*0.08))
}

func (o *Orbit) place(t *scene.Transform) {
	t.Position = o.center.Add(o.direction().Scale(o.distance))
	t.Rotation = geom.QuatLookRotation(o.center.Sub(t.Position), geom.Up)
}

type Orbiter interface {
	scene.Transformable
	GetOrbit() *Orbit
}

type OrbitSystem struct {
	world.Group[Orbiter]
	world *world.World
}

func (s *OrbitSystem) New(w *world.World) { s.world = w }
func (s *OrbitSystem) Phase() world.Phase { return world.PostUpdate }
func (s *OrbitSystem) Priority() int      { return 500 }

func (s *OrbitSystem) Update(world.Frame) {
	for c := range s.All() {
		o := c.GetOrbit()
		if in := inputOf(s.world, c, o.window); in != nil {
			o.drive(in)
		}
		o.place(c.GetTransform())
	}
}
