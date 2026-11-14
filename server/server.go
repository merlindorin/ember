// Package server builds the World a game server runs: the engine's
// simulation on the headless ticker, with no window, GPU or sound device,
// so a server binary builds with CGO_ENABLED=0 (decision 0064).
//
//	w := server.New().AddPlugins(net.NewServer(net.WithListener(l)), game.Plugin{})
//	w.MustRun()
package server

import (
	"github.com/lantervale/ember/anim"
	"github.com/lantervale/ember/audio"
	"github.com/lantervale/ember/automation"
	"github.com/lantervale/ember/camera"
	"github.com/lantervale/ember/ease"
	"github.com/lantervale/ember/internal/wm"
	"github.com/lantervale/ember/render"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/twod/light"
	"github.com/lantervale/ember/world"
)

// Option sets up the World New returns.
type Option func(*options)

type options struct {
	automation *automation.Automation
}

// Automation records the World into auto, which the game keeps to define
// its actions. Without it, New makes one, served only when
// EMBER_AUTOMATION is set.
func Automation(auto *automation.Automation) Option {
	return func(o *options) { o.automation = auto }
}

// New returns a World with the same simulation as ember.New: transforms,
// 2D entities and their steps, lights, animation, tweens, cameras,
// windows' input frames and a mixer's null output, without the GPU or the
// desktop runner. It runs on the headless ticker, in real time at the
// fixed step.
func New(opts ...Option) *world.World {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	if o.automation == nil {
		o.automation = automation.New()
	}
	return world.New().AddPlugins(
		wm.Plugin{},
		render.Plugin{},
		twod.Plugin{},
		light.Plugin{},
		anim.Plugin{},
		camera.Plugin{},
		ease.Plugin{},
		audio.Plugin{},
		o.automation,
	)
}
