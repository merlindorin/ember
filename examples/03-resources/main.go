// ECS resources: a rising wind puts a candle and a campfire out.
//
// Weather is a resource (weather.go): one value in the World, not attached
// to any entity. Gusts writes it and Cooling (cooling.go) reads it, and both
// see the same value.
//
//	go run ./examples/03-resources
package main

import "github.com/merlindorin/ember/world"

func main() {
	world.New().
		Insert(&Weather{Wind: 0}).
		AddEntities(
			newFlame("candle", 1),
			newFlame("campfire", 4),
		).
		AddSystems(&Gusts{}, &Cooling{}).
		On(world.Update, "quit", quitWhenAllOut).
		MustRun()
}

func quitWhenAllOut(f world.Frame) {
	if f.World.Len() == 0 {
		f.Quit()
	}
}
