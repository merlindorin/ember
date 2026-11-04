package main

import "github.com/merlindorin/ember/world"

// Rock has a Name but no Heat: it never burns.
type Rock struct {
	*world.BasicEntity
	Name
}

func newRock(name string) *Rock {
	return &Rock{BasicEntity: world.NewBasic(), Name: Name{name}}
}
