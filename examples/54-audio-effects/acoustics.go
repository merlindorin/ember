package main

import (
	"github.com/merlindorin/ember/audio"
	"github.com/merlindorin/ember/ease"
	"github.com/merlindorin/ember/world"
)

// A room's sound: how large its reverb is, and how much of it is heard.
type room struct{ size, wet float32 }

func corridor() room { return room{size: 0.25, wet: 0.12} }
func hall() room     { return room{size: 0.92, wet: 0.45} }

// Acoustics blends the sfx bus's reverb from the corridor's to the hall's
// over the 80 points around the archway, so it changes as he walks
// through. The reverb glides to each new value by itself, so setting it
// every frame never clicks.
type Acoustics struct {
	hero *Hero
	room *audio.Reverb
}

// inHall is how far the hero is into the hall, from 0 to 1.
func inHall(x float32) float32 { return ease.InOutQuad(min(max((x-230)/80, 0), 1)) }

func (a *Acoustics) Update(world.Frame) {
	t := inHall(a.hero.Position().X)
	a.room.SetRoomSize(corridor().size + (hall().size-corridor().size)*t)
	a.room.SetWet(corridor().wet + (hall().wet-corridor().wet)*t)
}
