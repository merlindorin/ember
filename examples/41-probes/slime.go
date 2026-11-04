package main

import (
	"image"
	"image/color"

	ecolor "github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/twod/light"
	"github.com/merlindorin/ember/world"
)

// A slime lurks once the light where it stands falls below a quarter, and
// shows again only above 27%, so a flickering torch doesn't make it blink.
const (
	lurkBelow = 0.25
	showAbove = 0.27
)

// Slime is three entities at one spot: its lit body, its unlit eyes, and
// a probe that asks the light there. They're siblings, because hiding an
// entity hides its children too.
type Slime struct {
	body, eyes *twod.Sprite
	probe      *light.Probe
	lurking    bool
}

// newSlime starts lurking, until its probe has an answer.
func newSlime(at geom.Vec2) *Slime {
	return &Slime{
		body:    twod.NewSprite(slimePicture(false), twod.At(at.X, at.Y), twod.Scale(2, 2), twod.Hidden()),
		eyes:    twod.NewSprite(slimePicture(true), twod.At(at.X, at.Y), twod.Scale(2, 2), light.Unlit()),
		probe:   light.NewProbe(twod.At(at.X, at.Y)),
		lurking: true,
	}
}

// lurks says whether a slime lurks in the light l, given whether it did.
func lurks(l ecolor.Color, lurking bool) bool {
	v := max(l.R, l.G, l.B)
	if lurking {
		return v <= showAbove
	}
	return v < lurkBelow
}

// Lurk shows each slime's body or only its eyes, from what its probe saw.
type Lurk struct {
	slimes []*Slime
}

func (l *Lurk) Update(world.Frame) {
	for _, s := range l.slimes {
		if !s.probe.Ready() {
			continue
		}
		s.lurking = lurks(s.probe.Light(), s.lurking)
		s.body.SetHidden(s.lurking)
		s.eyes.SetHidden(!s.lurking)
	}
}

// slimePicture draws a violet dome 14 pixels wide with two eyes, or the
// eyes alone, glowing yellow, at the same places.
func slimePicture(eyesOnly bool) *twod.Texture {
	const w, h = 14, 10
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			dx, dy := (float64(x)+0.5-w/2)/(w/2), (float64(y)+0.5-h)/h
			if eyesOnly || dx*dx+dy*dy > 1 {
				continue
			}
			img.Set(x, y, color.RGBA{uint8(150 + 70*(-dy)), 80, 210, 255})
		}
	}
	eye := color.RGBA{20, 30, 20, 255}
	if eyesOnly {
		eye = color.RGBA{255, 230, 80, 255}
	}
	for _, x := range []int{4, 9} {
		img.Set(x, 4, eye)
		img.Set(x, 5, eye)
	}
	return twod.NewTexture(img, twod.Pixelated())
}
