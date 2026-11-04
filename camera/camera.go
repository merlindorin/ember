// Package camera provides ready-made cameras: NewOrbit circles a point,
// NewFollow trails a target and NewFly flies freely. Each is an entity with
// a render.Camera, set up with options, that reads the input of its own
// window; a game's own camera entity embeds OrbitBase, FollowBase or
// FlyBase instead. Each controller's system drives the entity's Transform
// in PostUpdate, after transforms are resolved.
//
//	eye := camera.NewFollow(hero, camera.Angles(geom.Pi/4, 0.9), camera.Distance(30))
package camera

import (
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/internal/wm"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/world"
)

// Plugin installs every controller system; they only act on entities that
// embed their component. ember.New adds it.
type Plugin struct{}

func (Plugin) Build(a *world.World) {
	a.AddSystems(&FollowSystem{}, &OrbitSystem{}, &FlySystem{})
}

// inputOf is the input of e's window: win when set, else the window e is
// under, else the primary.
func inputOf(w *world.World, e interface{ GetBasicEntity() *world.BasicEntity }, win *platform.Window) *input.State {
	if win = wm.Home(w, e, win); win != nil {
		return win.Input()
	}
	return nil
}
