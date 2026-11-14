// Multiple windows: windows are entities. The first added is the primary,
// showing the scene through an orbit camera you drag with the mouse; the
// second shows it from above through its own fixed camera. Each holds its
// own label.
//
//	go run ./examples/10-multi-window
package main

import "github.com/lantervale/ember"

func main() {
	ember.New().
		AddEntities(
			newSceneWindow(),
			newGround(),
			newCrate(-2, 0),
			newCrate(2, 1),
			newMapWindow(),
		).
		MustRun()
}
