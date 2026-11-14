// Command fails is a scene that never draws: it panics in its first frame,
// or earlier on a machine that can't open its window, for the test that a
// failed scene's crash log stays out of the player's config folder.
package main

import (
	"github.com/lantervale/ember"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/world"
)

type panicker struct{}

func (panicker) Update(world.Frame) { panic("the scene was made to fail") }

func main() {
	ember.New().
		AddEntities(platform.NewWindow(platform.Title("Golden Fails"), platform.Size(160, 100))).
		AddSystems(panicker{}).
		MustRun()
}
