package main

import (
	"math"

	"github.com/merlindorin/ember/audio"
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/twod"
)

// Hero walks the meadow and hears it. The listener is on him, so his
// rotation turns his ears: at 0 he faces up, and the angle grows
// clockwise. The arrow shows where he faces.
type Hero struct {
	twod.Item
	audio.Listener
	body *twod.Sprite
}

// Face turns the hero, keeping his sprite upright and mirrored to the side
// he faces.
func (h *Hero) Face(turn float32) {
	h.SetRotation(float32(math.Remainder(float64(turn), 2*math.Pi)))
	h.body.SetRotation(-h.Rotation())
	h.body.SetFlip(h.facing().X < -0.5, false)
}

// Side is the walk cycle that shows where he faces.
func (h *Hero) Side() string {
	switch d := h.facing(); {
	case d.X > 0.5 || d.X < -0.5:
		return "side"
	case d.Y < 0:
		return "up"
	}
	return "down"
}

// Degrees is where he faces, clockwise from up, 0 to 360.
func (h *Hero) Degrees() float32 {
	d := h.Rotation() * 180 / math.Pi
	if d < 0 {
		d += 360
	}
	return d
}

func (h *Hero) facing() geom.Vec2 {
	r := h.Rotation()
	return geom.V2(geom.Sin(r), -geom.Cos(r))
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
	body := twod.NewSprite(sheet, twod.Animate(clips), twod.Size(32, 32))
	arrow := twod.NewPolygon(
		[]geom.Vec2{{X: 0, Y: -30}, {X: -7, Y: -21}, {X: 7, Y: -21}},
		twod.Color(color.Hex("#ffe08a")),
	)
	h := twod.Build(&Hero{Item: twod.Base(), body: body}, "hero", twod.At(320, 204))
	h.Add(body, arrow)
	h.Face(0)
	return h, nil
}

// Source is a thing in the meadow that makes a sound where it is.
type Source struct {
	twod.Item
	*audio.Emitter
}

func newCampfire(crackle *audio.Stream) *Source {
	fire := twod.Build(&Source{Item: twod.Base(), Emitter: audio.NewEmitter(
		crackle,
		audio.Loop(),
		audio.Autoplay(),
		audio.OnBus("sfx"),
		audio.Volume(0.7),
		audio.Distance(30, 500),
		audio.Rolloff(1),
	)}, "campfire", twod.At(90, 220))
	fire.Add(
		twod.NewPolygon(
			[]geom.Vec2{{X: -18, Y: 6}, {X: 18, Y: 6}, {X: 14, Y: 12}, {X: -14, Y: 12}},
			twod.Color(color.Hex("#5a3a22")),
		),
		twod.NewPolygon([]geom.Vec2{{X: -12, Y: 6}, {X: 0, Y: -26}, {X: 12, Y: 6}}, twod.Color(color.Hex("#ff9a3c"))),
		twod.NewPolygon([]geom.Vec2{{X: -6, Y: 6}, {X: 0, Y: -12}, {X: 6, Y: 6}}, twod.Color(color.Hex("#ffe08a"))),
		twod.NewText("campfire", twod.At(0, 20), twod.Pivot(0.5, 0), twod.FontSize(13)),
	)
	return fire
}

func newRiver(flow *audio.Stream) *Source {
	river := twod.Build(&Source{Item: twod.Base(), Emitter: audio.NewEmitter(flow,
		audio.Loop(), audio.Autoplay(), audio.OnBus("sfx"), audio.Distance(80, 700), audio.Rolloff(0.7),
	)}, "river", twod.At(575, 180))
	river.Add(
		twod.NewPolygon(
			[]geom.Vec2{{X: -22, Y: -180}, {X: 22, Y: -180}, {X: 30, Y: 180}, {X: -14, Y: 180}},
			twod.Color(color.Hex("#3d7fb8")),
		),
		twod.NewText("river", twod.At(-34, 40), twod.Pivot(1, 0), twod.FontSize(13)),
	)
	return river
}
