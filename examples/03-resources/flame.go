package main

import "github.com/merlindorin/ember/world"

// Flame is an entity: an ID plus the components it is made of.
type Flame struct {
	*world.BasicEntity
	Heat
	Name
}

func newFlame(name string, degrees float32) *Flame {
	return &Flame{BasicEntity: world.NewBasic(), Heat: Heat{degrees}, Name: Name{name}}
}
