package main

import (
	"math"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/world"
)

const (
	// turnSpeed is how fast Q and E turn the hero: a half turn a second.
	turnSpeed = math.Pi
	// faceSpeed is how fast he turns to face where he walks.
	faceSpeed = 3 * math.Pi
)

// Walk turns the hero with Q and E, and moves him where WASD or the arrows
// point, 140 points a second inside the window, turning him to face that
// way. His rotation turns his ears; his sprite stays upright and shows the
// side he faces.
type Walk struct {
	hero *Hero
	in   *input.State
}

func (w *Walk) Update(f world.Frame) {
	turn := w.hero.Rotation() + w.in.Axis(input.KeyQ, input.KeyE)*turnSpeed*f.DT
	dir := w.in.Direction(input.KeyA, input.KeyD, input.KeyW, input.KeyS).
		Add(w.in.Direction(input.KeyLeft, input.KeyRight, input.KeyUp, input.KeyDown))
	moving := dir != (geom.Vec2{})
	if moving {
		at := w.hero.Position().Add(dir.Normalize().Scale(140 * f.DT))
		w.hero.SetPosition(geom.V2(min(max(at.X, 16), 624), min(max(at.Y, 24), 336)))
		turn = towards(turn, geom.Atan2(dir.X, -dir.Y), faceSpeed*f.DT)
	}
	w.hero.Face(turn)
	if moving || turn != w.hero.Rotation() || w.in.Axis(input.KeyQ, input.KeyE) != 0 {
		w.hero.body.Play(w.hero.Side())
	} else {
		w.hero.body.Stop()
	}
}

// towards turns from by at most step to reach to, the short way round.
func towards(from, to, step float32) float32 {
	d := float32(math.Remainder(float64(to-from), 2*math.Pi))
	return from + min(max(d, -step), step)
}
