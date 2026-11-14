package camera

import (
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/render"
	"github.com/lantervale/ember/scene"
	"github.com/lantervale/ember/world"
)

// Fly is a free camera: WASD to move, Q and E down and up, the right mouse
// button to look around, Shift to go faster. For inspecting a level.
type Fly struct {
	speed       float32
	yaw, pitch  float32
	sensitivity float32
	window      *platform.Window
}

// FlyCamera is a ready-made flying camera entity.
type FlyCamera struct {
	*world.BasicEntity
	scene.Transform
	render.Camera
	Fly
}

// NewFly returns a flying camera set up by opts, at the origin until
// placed with its Transform.
func NewFly(opts ...FlyOption) *FlyCamera {
	return BuildFly(&FlyCamera{BasicEntity: world.NewBasic(), Transform: scene.At(0, 0, 0), Fly: FlyBase()}, opts...)
}

// FlyBase is the Fly a game's own camera entity embeds before BuildFly.
func FlyBase() Fly { return Fly{speed: 8, sensitivity: 0.004} }

// BuildFly applies opts to a game's own camera entity, which embeds a Fly
// made by FlyBase, and draws it in the fly's window.
func BuildFly[T Flyer](e T, opts ...FlyOption) T {
	f := e.GetFly()
	for _, opt := range opts {
		opt.fly(f)
	}
	if c, ok := any(e).(interface{ GetCamera() *render.Camera }); ok && f.window != nil {
		c.GetCamera().Window = f.window
	}
	return e
}

// Speed is how fast the camera flies, in units per second (default 8).
func Speed(s float32) FlyOption {
	return flyOption(func(f *Fly) {
		if s > 0 {
			f.speed = s
		}
	})
}

func (f *Fly) GetFly() *Fly { return f }

func (f *Fly) Speed() float32 { return f.speed }

// Angles are the yaw and the pitch.
func (f *Fly) Angles() (float32, float32) { return f.yaw, f.pitch }

// SetAngles keeps the pitch short of straight up or down.
func (f *Fly) SetAngles(yaw, pitch float32) { f.yaw, f.pitch = yaw, geom.Clamp(pitch, -1.5, 1.5) }

type Flyer interface {
	scene.Transformable
	GetFly() *Fly
}

type FlySystem struct {
	world.Group[Flyer]
	world *world.World
}

func (s *FlySystem) New(w *world.World) { s.world = w }
func (s *FlySystem) Phase() world.Phase { return world.PostUpdate }
func (s *FlySystem) Priority() int      { return 500 }

func (s *FlySystem) Update(frame world.Frame) {
	for c := range s.All() {
		f := c.GetFly()
		t := c.GetTransform()
		if in := inputOf(s.world, c, f.window); in != nil {
			if in.Down(input.MouseRight) {
				d := in.MouseDelta()
				f.SetAngles(f.yaw-d.X*f.sensitivity, f.pitch-d.Y*f.sensitivity)
			}
			speed := f.speed
			if in.Down(input.KeyLeftShift) {
				speed *= 4
			}
			t.Rotation = geom.QuatEuler(f.pitch, f.yaw, 0)
			move := t.Forward().Scale(in.Axis(input.KeyS, input.KeyW)).
				Add(t.Right().Scale(in.Axis(input.KeyA, input.KeyD))).
				Add(geom.Up.Scale(in.Axis(input.KeyQ, input.KeyE)))
			t.Position = t.Position.Add(move.ClampLen(1).Scale(speed * frame.DT))
			continue
		}
		t.Rotation = geom.QuatEuler(f.pitch, f.yaw, 0)
	}
}
