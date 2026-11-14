package main

import "github.com/lantervale/ember/world"

// Ball moves one unit each fixed step, and remembers where it was before.
type Ball struct{ x, before float32 }

func (b *Ball) Phase() world.Phase { return world.FixedUpdate }

func (b *Ball) Update(world.Frame) { b.before, b.x = b.x, b.x+1 }

// Drawn is where to draw the ball in a frame: between its last two steps,
// alpha of the way.
func (b *Ball) Drawn(alpha float32) float32 { return b.before + (b.x-b.before)*alpha }
