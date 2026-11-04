package camera

import (
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/platform"
)

// Each controller takes options of its own, typed so that one meant for
// another does not compile; some work on several.
type (
	OrbitOption  interface{ orbit(*Orbit) }
	FollowOption interface{ follow(*Follow) }
	FlyOption    interface{ fly(*Fly) }
)

// Option works on every controller.
type Option interface {
	OrbitOption
	FollowOption
	FlyOption
}

// DistanceOption works on the orbit and follow cameras.
type DistanceOption interface {
	OrbitOption
	FollowOption
}

// SensitivityOption works on the orbit and fly cameras.
type SensitivityOption interface {
	OrbitOption
	FlyOption
}

type (
	orbitOption  func(*Orbit)
	followOption func(*Follow)
	flyOption    func(*Fly)
)

func (o orbitOption) orbit(c *Orbit)    { o(c) }
func (o followOption) follow(c *Follow) { o(c) }
func (o flyOption) fly(c *Fly)          { o(c) }

type inOption struct{ win *platform.Window }

func (o inOption) orbit(c *Orbit)   { c.window = o.win }
func (o inOption) follow(c *Follow) { c.window = o.win }
func (o inOption) fly(c *Fly)       { c.window = o.win }

// In reads the input of win, and makes the camera draw in it. Without it,
// a camera uses the window it is under, else the primary: In is for a
// camera outside any window.
func In(win *platform.Window) Option { return inOption{win} }

type anglesOption geom.Vec2

func (o anglesOption) orbit(c *Orbit)   { c.SetAngles(o.X, o.Y) }
func (o anglesOption) follow(c *Follow) { c.SetAngles(o.X, o.Y) }
func (o anglesOption) fly(c *Fly)       { c.SetAngles(o.X, o.Y) }

// Angles is where the camera looks from: yaw around the vertical and pitch
// from the horizontal, in radians.
func Angles(yaw, pitch float32) Option { return anglesOption(geom.V2(yaw, pitch)) }

type distanceOption float32

func (o distanceOption) orbit(c *Orbit)   { c.SetDistance(float32(o)) }
func (o distanceOption) follow(c *Follow) { c.SetDistance(float32(o)) }

// Distance is how far the camera is from what it looks at.
func Distance(d float32) DistanceOption { return distanceOption(d) }

type sensitivityOption float32

func (o sensitivityOption) orbit(c *Orbit) {
	if o > 0 {
		c.sensitivity = float32(o)
	}
}

func (o sensitivityOption) fly(c *Fly) {
	if o > 0 {
		c.sensitivity = float32(o)
	}
}

// Sensitivity is the radians the camera turns per pixel dragged.
func Sensitivity(s float32) SensitivityOption { return sensitivityOption(s) }
