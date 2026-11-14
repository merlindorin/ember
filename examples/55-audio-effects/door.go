package main

import (
	"github.com/lantervale/ember/audio"
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/ease"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/world"
)

const (
	muffled = 350
	open    = 16000
	// reach is how close the hero must stand to work the door.
	reach = 100
)

// Door is the door in the hall's north wall. The band behind it plays
// through a low-pass that muffles them while it is shut.
type Door struct {
	*twod.Polygon
	band   *Band
	filter *audio.Filter
	opened bool
}

// Band is the drums behind the door, heard from where they are.
type Band struct {
	twod.Item
	*audio.Emitter
}

func newDoor(drums *audio.Clip) *Door {
	filter := audio.NewLowPass(audio.Cutoff(muffled))
	band := twod.Build(&Band{Item: twod.Base(), Emitter: audio.NewEmitter(drums,
		audio.Loop(), audio.Autoplay(), audio.Effects(filter), audio.Distance(60, 700), audio.Rolloff(0.6),
	)}, "band", twod.At(500, 30))
	band.Add(twod.NewText("drums behind the door", twod.At(0, 0), twod.FontSize(13)))
	door := twod.NewPolygon(box(geom.Rect{X: 470, Y: 54, W: 60, H: 16}), twod.Color(color.Hex("#8a5a32")))
	return &Door{Polygon: door, band: band, filter: filter}
}

// Doorway opens and shuts the door on E when the hero is near it, and
// sweeps the band's low-pass open or shut with the door's swing.
type Doorway struct {
	door *Door
	hero *Hero
	in   *input.State
}

func (d *Doorway) Update(f world.Frame) {
	if !d.in.Pressed(input.KeyE) || d.hero.Position().Dist(geom.V2(500, 70)) > reach {
		return
	}
	door := d.door
	door.opened = !door.opened
	door.SetHidden(door.opened)
	from, to := door.filter.Cutoff(), float32(muffled)
	if door.opened {
		to = open
	}
	f.World.AddEntity(ease.NewTween(0.5, func(t float32) {
		door.filter.SetCutoff(from * pow(to/from, t))
	}, ease.Easing(ease.OutQuad)))
}
