// Events: Cooling tells the rest of the game a flame went out, and Log
// hears it, without either knowing the other.
//
//	go run ./examples/04-events
package main

import "github.com/merlindorin/ember/world"

func main() {
	world.New().
		AddEntities(
			newFlame("candle", 1),
			newFlame("campfire", 2),
		).
		AddSystems(&Cooling{}, &Log{}).
		On(world.Update, "quit", quitWhenAllOut).
		MustRun()
}

func quitWhenAllOut(f world.Frame) {
	if f.World.Len() == 0 {
		f.Quit()
	}
}
