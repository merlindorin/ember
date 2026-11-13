package main

import (
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/world"
)

// Act is where the player means to go.
type Act int

const (
	Left Act = iota
	Right
	Up
	Down
)

func (a Act) String() string { return [...]string{"left", "right", "up", "down"}[a] }

// newControls binds each way to the arrows, a pad's left stick and its
// cross.
func newControls() *input.Actions[Act] {
	return input.NewActions(
		input.Bind(Left, input.KeyLeft, input.GamepadLeftStickLeft, input.GamepadLeft),
		input.Bind(Right, input.KeyRight, input.GamepadLeftStickRight, input.GamepadRight),
		input.Bind(Up, input.KeyUp, input.GamepadLeftStickUp, input.GamepadUp),
		input.Bind(Down, input.KeyDown, input.GamepadLeftStickDown, input.GamepadDown),
	)
}

// newHero is the walker of 47-keyboard, in the middle of the field.
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
		twod.At(field().X+field().W/2, field().Y+field().H/2),
		twod.Size(32, 32),
		twod.Pivot(0.5, 1),
	), nil
}

// Walk moves the hero up to 160 points a second where the actions point:
// a stick leant half way walks at half speed. He stays in the field.
type Walk struct {
	hero     *twod.Sprite
	in       *input.State
	controls *input.Actions[Act]
}

func (w *Walk) Update(f world.Frame) {
	dir := w.controls.Direction(w.in, Left, Right, Up, Down)
	if dir == (geom.Vec2{}) {
		w.hero.Stop()
		return
	}
	at := w.hero.Position().Add(dir.Scale(160 * f.DT))
	w.hero.SetPosition(geom.V2(
		min(max(at.X, field().X+16), field().X+field().W-16),
		min(max(at.Y, field().Y+32), field().Y+field().H),
	))
	switch {
	case dir.X > 0.3 || dir.X < -0.3:
		w.hero.Play("side")
		w.hero.SetFlip(dir.X < 0, false)
	case dir.Y < 0:
		w.hero.Play("up")
	default:
		w.hero.Play("down")
	}
}
