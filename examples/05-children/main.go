// Children: sparks belong to the campfire. Adding the campfire adds them,
// and when it goes out they leave with it, long before they would cool.
//
//	go run ./examples/05-children
package main

import (
	"fmt"
	"os"

	"github.com/lantervale/ember/world"
)

func main() {
	campfire := newFlame("campfire", 1)
	campfire.AppendChild(newFlame("spark", 5))
	campfire.AppendChild(newFlame("spark", 5))

	world.New().
		AddEntities(
			campfire,
			newFlame("candle", 2),
		).
		AddSystems(&Cooling{}).
		On(world.Startup, "count", countFlames).
		On(world.Update, "quit", quitWhenAllOut).
		MustRun()
}

func countFlames(f world.Frame) {
	fmt.Fprintf(os.Stdout, "%d flames burning\n", f.World.Len())
}

func quitWhenAllOut(f world.Frame) {
	if f.World.Len() == 0 {
		f.Quit()
	}
}
