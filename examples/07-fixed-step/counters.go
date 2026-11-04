package main

import "github.com/merlindorin/ember/world"

// Steps runs in FixedUpdate: sixty times per simulated second, whatever the
// frame rate.
type Steps struct{ count int }

func (s *Steps) Phase() world.Phase { return world.FixedUpdate }

func (s *Steps) Update(world.Frame) { s.count++ }

// Frames runs in Update: once per frame.
type Frames struct{ count int }

func (s *Frames) Update(world.Frame) { s.count++ }
