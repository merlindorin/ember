package main

import (
	"fmt"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/world"
)

// finger is one touch drawn: a spot and its label.
type finger struct {
	spot  *twod.Polygon
	label *twod.Text
}

// Fingers adds a finger when a touch starts, places it where the touch is
// every frame, and removes it when the touch ends.
type Fingers struct {
	in      *input.State
	layer   *twod.Layer
	fingers map[int]finger
	started world.Reader[input.TouchStarted]
	ended   world.Reader[input.TouchEnded]
}

func newFingers(in *input.State, layer *twod.Layer) *Fingers {
	return &Fingers{in: in, layer: layer, fingers: map[int]finger{}}
}

func (f *Fingers) Update(fr world.Frame) {
	for e := range f.started.Read(fr.World) {
		spot := twod.NewPolygon(disc(22), twod.Color(color.Hex("#f0b44c")))
		label := twod.NewText("", twod.FontSize(13))
		f.layer.Add(spot, label)
		f.fingers[e.Touch.ID] = finger{spot, label}
	}
	for _, t := range f.in.Touches() {
		if g, ok := f.fingers[t.ID]; ok {
			at := onPad(t.At)
			g.spot.SetPosition(at)
			g.label.SetPosition(at.Add(geom.V2(0, -36)))
			g.label.SetText(fmt.Sprintf("%d · %s", t.ID, t.Phase))
		}
	}
	for e := range f.ended.Read(fr.World) {
		if g, ok := f.fingers[e.Touch.ID]; ok {
			fr.World.RemoveEntity(g.spot)
			fr.World.RemoveEntity(g.label)
			delete(f.fingers, e.Touch.ID)
		}
	}
}
