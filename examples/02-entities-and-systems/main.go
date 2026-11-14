// ECS basics: a candle and a campfire cool down until both are out.
//
// Components hold data (components.go), an entity is made of components
// (flame.go), and a system updates every entity that has the components it
// needs (cooling.go). The World runs them.
//
//	go run ./examples/02-entities-and-systems
package main

import "github.com/lantervale/ember/world"

func main() {
	world.New().
		AddEntities(
			newFlame("candle", 2),
			newFlame("campfire", 5),
		).
		AddSystems(&Cooling{}).
		On(world.Update, "quit", quitWhenAllOut).
		MustRun()
}

func quitWhenAllOut(f world.Frame) {
	if f.World.Len() == 0 {
		f.Quit()
	}
}
