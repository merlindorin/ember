// Package ember is the batteries-included entry point.
//
//	func main() {
//		win := platform.NewWindow(platform.Title("My Game"))
//		ember.New().
//			AddEntities(win, NewCrate()).
//			AddSystems(&SpinSystem{}).
//			MustRun()
//	}
//
// New wires the desktop windows (a default one when the game adds none),
// the GPU renderer, UI, transforms, cameras, skeletal
// animation, sound (the device opens when the game adds an audio.Mixer) and the
// automation gRPC API (off unless EMBER_AUTOMATION is set), and returns the
// World they set up. Dedicated servers build theirs from world.New and pure-Go
// plugins instead.
package ember

import (
	"github.com/lantervale/ember/anim"
	"github.com/lantervale/ember/audio"
	"github.com/lantervale/ember/automation"
	"github.com/lantervale/ember/camera"
	"github.com/lantervale/ember/ease"
	"github.com/lantervale/ember/gpu"
	"github.com/lantervale/ember/internal/wm"
	"github.com/lantervale/ember/render"
	"github.com/lantervale/ember/world"
)

// Option sets up the World New returns.
type Option func(*options)

type options struct {
	headless   bool
	automation *automation.Automation
	crashLog   string
}

// Headless runs the World on the headless ticker instead of a desktop
// window: the stack is set up, but nothing is presented.
func Headless() Option { return func(c *options) { c.headless = true } }

// Automation records the World into auto, which the game keeps to
// define its actions. Without it, New makes one, served only when
// EMBER_AUTOMATION is set.
func Automation(auto *automation.Automation) Option {
	return func(c *options) { c.automation = auto }
}

// CrashLog is where the desktop runner writes what went wrong when the
// game fails to start or panics, such as the game's save folder. Without
// it, crash.log goes in the user's config folder under the slug of the
// primary window's title.
func CrashLog(path string) Option { return func(c *options) { c.crashLog = path } }

// New returns a World with the full presentation stack.
func New(opts ...Option) *world.World {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	if o.automation == nil {
		o.automation = automation.New()
	}
	w := world.New().AddPlugins(
		runnerPlugin(o.crashLog),
		wm.Plugin{},
		render.Plugin{},
		anim.Plugin{},
		camera.Plugin{},
		ease.Plugin{},
		audio.Plugin{},
		gpu.Plugin{},
		o.automation,
	)
	if o.headless {
		w.SetRunner(world.Headless)
	}
	return w
}
