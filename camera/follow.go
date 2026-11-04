package camera

import (
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/render"
	"github.com/merlindorin/ember/scene"
	"github.com/merlindorin/ember/world"
)

// Follow is the action-RPG camera: it trails a target from a fixed pitch
// and heading, with smoothing, and zooms on the mouse wheel or the right
// stick.
type Follow struct {
	target            scene.Transformable
	offset            geom.Vec3
	yaw, pitch        float32
	distance          float32
	nearest, farthest float32
	smoothing         float32
	shake             float32
	window            *platform.Window
	focus             geom.Vec3
	zoom              float32
	placed            bool
}

// FollowCamera is a ready-made following camera entity.
type FollowCamera struct {
	*world.BasicEntity
	scene.Transform
	render.Camera
	Follow
}

// NewFollow returns a camera following target, set up by opts.
//
//	eye := camera.NewFollow(hero, camera.Angles(geom.Pi/4, 0.9), camera.Distance(30), camera.Smoothing(6))
func NewFollow(target scene.Transformable, opts ...FollowOption) *FollowCamera {
	return BuildFollow(
		&FollowCamera{BasicEntity: world.NewBasic(), Transform: scene.At(0, 0, 0), Follow: FollowBase(target)},
		opts...)
}

// FollowBase is the Follow a game's own camera entity embeds before
// BuildFollow.
func FollowBase(target scene.Transformable) Follow {
	return Follow{target: target, distance: 18, pitch: 0.8}
}

// BuildFollow applies opts to a game's own camera entity, which embeds a
// Follow made by FollowBase, and draws it in the follow's window.
func BuildFollow[T Follower](e T, opts ...FollowOption) T {
	f := e.GetFollow()
	for _, opt := range opts {
		opt.follow(f)
	}
	if c, ok := any(e).(interface{ GetCamera() *render.Camera }); ok && f.window != nil {
		c.GetCamera().Window = f.window
	}
	return e
}

// Offset looks at the target plus v, such as its chest rather than its
// feet.
func Offset(v geom.Vec3) FollowOption { return followOption(func(f *Follow) { f.offset = v }) }

// ZoomRange keeps the distance within [nearest, farthest] when the player
// zooms.
func ZoomRange(nearest, farthest float32) FollowOption {
	return followOption(func(f *Follow) { f.nearest, f.farthest = nearest, farthest })
}

// Smoothing eases the camera towards its target at speed per second,
// instead of snapping.
func Smoothing(speed float32) FollowOption {
	return followOption(func(f *Follow) { f.smoothing = speed })
}

func (f *Follow) GetFollow() *Follow { return f }

func (f *Follow) Target() scene.Transformable     { return f.target }
func (f *Follow) SetTarget(t scene.Transformable) { f.target = t }
func (f *Follow) Distance() float32               { return f.distance }

// SetDistance keeps the distance positive.
func (f *Follow) SetDistance(d float32) { f.distance = max(d, 0.1) }

// Angles are the yaw and the pitch.
func (f *Follow) Angles() (float32, float32) { return f.yaw, f.pitch }

// SetAngles keeps the pitch short of straight up or down.
func (f *Follow) SetAngles(yaw, pitch float32) { f.yaw, f.pitch = yaw, geom.Clamp(pitch, -1.5, 1.5) }

// AddShake adds screen-shake trauma, saturating at 1; it decays by itself.
func (f *Follow) AddShake(amount float32) { f.shake = min(f.shake+amount, 1) }

// GroundDirection maps a screen-space direction (a stick or WASD vector, Y
// down) to the ground plane as seen through this camera, keeping its length:
// pushing up walks away from the camera whatever its heading.
func (f *Follow) GroundDirection(v geom.Vec2) geom.Vec3 {
	sin, cos := geom.Sin(f.yaw), geom.Cos(f.yaw)
	forward := geom.V3(-sin, 0, -cos)
	right := geom.V3(cos, 0, -sin)
	return right.Scale(v.X).Add(forward.Scale(-v.Y))
}

type Follower interface {
	scene.Transformable
	GetFollow() *Follow
}

type FollowSystem struct {
	world.Group[Follower]
	world *world.World
}

func (s *FollowSystem) New(w *world.World) { s.world = w }
func (s *FollowSystem) Phase() world.Phase { return world.PostUpdate }
func (s *FollowSystem) Priority() int      { return 500 }

func (s *FollowSystem) Update(frame world.Frame) {
	dt := frame.DT
	elapsed := s.world.Time().Elapsed
	for c := range s.All() {
		f := c.GetFollow()
		if f.target == nil {
			continue
		}
		if !f.placed {
			f.zoom = f.distance
			f.focus = f.target.GetTransform().WorldPosition().Add(f.offset)
			f.placed = true
		}
		if in := inputOf(s.world, c, f.window); in != nil {
			if !in.PointerCaptured() {
				f.distance -= in.Wheel().Y * f.distance * 0.08
			}
			f.distance += in.Stick(input.RightStick).Y * f.distance * 1.5 * dt
		}
		if f.farthest > 0 {
			f.distance = geom.Clamp(f.distance, f.nearest, f.farthest)
		}
		target := f.target.GetTransform().WorldPosition().Add(f.offset)
		if f.smoothing > 0 {
			f.focus = geom.DampVec3(f.focus, target, f.smoothing, dt)
			f.zoom = geom.Damp(f.zoom, f.distance, 10, dt)
		} else {
			f.focus, f.zoom = target, f.distance
		}
		dir := geom.V3(geom.Sin(f.yaw)*geom.Cos(f.pitch), geom.Sin(f.pitch), geom.Cos(f.yaw)*geom.Cos(f.pitch))
		eye := f.focus.Add(dir.Scale(f.zoom))
		look := f.focus
		if f.shake > 0 {
			k := f.shake * f.shake * 0.35
			jitter := geom.V3(geom.Sin(elapsed*47), geom.Sin(elapsed*39+1.3), geom.Sin(elapsed*53+2.1)).Scale(k)
			eye, look = eye.Add(jitter), look.Add(jitter.Scale(0.5))
			f.shake = max(f.shake-dt*1.8, 0)
		}
		t := c.GetTransform()
		t.Position = eye
		t.Rotation = geom.QuatLookRotation(look.Sub(eye), geom.Up)
	}
}
