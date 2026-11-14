package main

import (
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/world"
)

// newHero is the walker of 28-y-sort, 32 points tall, standing in the
// middle of the window.
func newHero() (*twod.Sprite, error) {
	sheet, err := twod.LoadTexture(files, "walker.png", twod.Pixelated())
	if err != nil {
		return nil, err
	}
	frames := twod.Strip(sheet, twod.Cell(16, 16))
	clips := twod.NewAnimations().
		Add("down", 8, frames[0:4], twod.Loop()).
		Add("up", 8, frames[4:8], twod.Loop()).
		Add("side", 8, frames[8:12], twod.Loop())
	return twod.NewSprite(sheet,
		twod.Animate(clips),
		twod.At(320, 200),
		twod.Size(32, 32),
		twod.Pivot(0.5, 1),
	), nil
}

// Walk moves the hero where WASD or the arrows point, 120 points a second,
// twice that while either Shift is held. It leaves him be while he is
// being named, or typing would walk him.
type Walk struct {
	hero   *twod.Sprite
	in     *input.State
	naming *Naming
}

func (w *Walk) Update(f world.Frame) {
	dir := w.in.Direction(input.KeyA, input.KeyD, input.KeyW, input.KeyS).
		Add(w.in.Direction(input.KeyLeft, input.KeyRight, input.KeyUp, input.KeyDown))
	if w.naming.active || dir == (geom.Vec2{}) {
		w.hero.Stop()
		return
	}
	speed := float32(120)
	if w.in.Down(input.KeyShift) {
		speed *= 2
	}
	w.hero.Translate(dir.Normalize().Scale(speed * f.DT))
	switch {
	case dir.X != 0:
		w.hero.Play("side")
		w.hero.SetFlip(dir.X < 0, false)
	case dir.Y < 0:
		w.hero.Play("up")
	default:
		w.hero.Play("down")
	}
}
