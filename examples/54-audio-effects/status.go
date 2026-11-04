package main

import (
	"fmt"
	"math"

	"github.com/merlindorin/ember/audio"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/world"
)

// HUD shows the controls and what the effects are doing, since a GIF can't
// play them.
type HUD struct {
	layer *twod.Layer
	lines []*twod.Text
}

func newHUD() *HUD {
	h := &HUD{layer: twod.NewLayer(1, twod.Fixed())}
	for i := range 3 {
		t := twod.NewText("", twod.At(16, 12+20*float32(i)), twod.Pivot(0, 0), twod.FontSize(15))
		h.lines = append(h.lines, t)
		h.layer.Add(t)
	}
	return h
}

// Status writes the reverb's and the door filter's settings on the HUD.
type Status struct {
	hud  *HUD
	hero *Hero
	room *audio.Reverb
	door *Door
}

func (s *Status) Update(world.Frame) {
	where := "corridor"
	if inHall(s.hero.Position().X) > 0.5 {
		where = "great hall"
	}
	state := "shut"
	if s.door.opened {
		state = "open"
	}
	s.hud.lines[0].SetText("WASD to walk · Space to clap · E at the door")
	s.hud.lines[1].SetText(fmt.Sprintf("%s · reverb size %.2f, wet %.2f", where, s.room.RoomSize(), s.room.Wet()))
	s.hud.lines[2].SetText(fmt.Sprintf("door %s · drums through a low-pass at %.0f Hz", state, s.door.filter.Cutoff()))
}

func pow(x, y float32) float32 { return float32(math.Pow(float64(x), float64(y))) }
