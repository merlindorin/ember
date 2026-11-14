// Testing UI: a settings form that saves when Save is clicked. Its tests
// fill it in two ways, as a player would with the mouse and keyboard, and
// by the elements' automation IDs, with no window on screen.
//
//	go run ./examples/18-testing-ui
package main

import (
	"github.com/lantervale/ember"
	"github.com/lantervale/ember/platform"
)

func main() {
	win := platform.NewWindow(
		platform.Title("ember · testing ui"),
		platform.Size(520, 320),
		platform.FixedSize(),
	)
	win.Add(newForm(&Settings{}).doc)
	ember.New().AddEntities(win).MustRun()
}
