package main

import (
	"math/rand/v2"

	"github.com/lantervale/ember/audio"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/world"
)

// stride is the time between footsteps.
const stride = 0.32

// Hero walks the dungeon and hears it.
type Hero struct {
	*twod.Sprite
	audio.Listener
}

func newHero() (*Hero, error) {
	sheet, err := twod.LoadTexture(files, "walker.png", twod.Pixelated())
	if err != nil {
		return nil, err
	}
	frames := twod.Strip(sheet, twod.Cell(16, 16))
	clips := twod.NewAnimations().
		Add("down", 8, frames[0:4], twod.Loop()).
		Add("up", 8, frames[4:8], twod.Loop()).
		Add("side", 8, frames[8:12], twod.Loop())
	body := twod.NewSprite(sheet, twod.Animate(clips), twod.At(40, 196), twod.Size(32, 32), twod.Pivot(0.5, 1))
	return &Hero{Sprite: body}, nil
}

// Walk moves the hero with WASD or the arrows, 120 points a second on the
// floor, with a footstep each stride, and claps on Space. Both play on the
// sfx bus, so they ring in the room he is in.
type Walk struct {
	hero   *Hero
	in     *input.State
	mixer  *audio.Mixer
	sounds *sounds
	since  float32
}

func (w *Walk) Update(f world.Frame) {
	if w.in.Pressed(input.KeySpace) {
		w.mixer.Play(w.sounds.clap, audio.OnBus("sfx"))
	}
	dir := w.in.Direction(input.KeyA, input.KeyD, input.KeyW, input.KeyS).
		Add(w.in.Direction(input.KeyLeft, input.KeyRight, input.KeyUp, input.KeyDown))
	h := w.hero
	if dir == (geom.Vec2{}) {
		h.Stop()
		w.since = stride
		return
	}
	step := dir.Normalize().Scale(120 * f.DT)
	for _, d := range []geom.Vec2{step, {X: step.X}, {Y: step.Y}} {
		if onFloor(h.Position().Add(d)) {
			h.Translate(d)
			break
		}
	}
	if w.since += f.DT; w.since >= stride {
		w.since = 0
		pitch := 0.9 + 0.2*rand.Float32() //nolint:gosec // a game's randomness, not a secret
		w.mixer.Play(w.sounds.step, audio.OnBus("sfx"), audio.Pitch(pitch), audio.Volume(0.7))
	}
	switch {
	case dir.X != 0:
		h.Play("side")
		h.SetFlip(dir.X < 0, false)
	case dir.Y < 0:
		h.Play("up")
	default:
		h.Play("down")
	}
}
