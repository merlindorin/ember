// Views: a Census follows every burning entity without being a system, so
// the game ends when nothing burns, even with a rock left in the world.
//
//	go run ./examples/06-views
package main

import (
	"fmt"
	"os"

	"github.com/merlindorin/ember/world"
)

func main() {
	census := &Census{}

	w := world.New().
		AddView(census).
		AddEntities(
			newFlame("candle", 1),
			newFlame("campfire", 2),
			newRock("rock"),
		).
		AddSystems(&Cooling{}).
		On(world.Update, "quit", quitWhenNothingBurns(census))
	w.MustRun()

	fmt.Fprintf(os.Stdout, "%d burning, %d in the world\n", census.Len(), w.Len())
}

func quitWhenNothingBurns(census *Census) func(world.Frame) {
	return func(f world.Frame) {
		if census.Len() == 0 {
			f.Quit()
		}
	}
}
