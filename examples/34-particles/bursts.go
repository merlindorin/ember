package main

import (
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/world"
)

// Bursts throws a one-shot burst of embers where the window is clicked,
// and removes each burst once its last ember is gone.
type Bursts struct {
	window *platform.Window
	layer  *twod.Layer
	done   world.Reader[twod.ParticlesFinished]
}

func (b *Bursts) Update(f world.Frame) {
	in := b.window.Input()
	if in.Pressed(input.MouseLeft) {
		at := in.Mouse()
		b.layer.Add(twod.NewParticles(nil,
			twod.At(at.X, at.Y),
			twod.Burst(80),
			twod.OneShot(),
			twod.Lifetime(0.6, 1.4),
			twod.Velocity(60, 220),
			twod.Direction(-geom.Pi/2, geom.Pi),
			twod.Gravity(geom.V2(0, 260)),
			twod.Drag(0.6),
			twod.ColorOverLife(color.Hex("#fff3b0"), color.RGBA(1, 0.3, 0.05, 0)),
			twod.SizeOverLife(5, 1),
			twod.Blend(twod.Add),
		))
	}
	for ev := range b.done.Read(f.World) {
		f.World.RemoveEntity(ev.Particles)
	}
}
