package main

import (
	"fmt"

	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/world"
)

// Plugs says which pad was plugged in or out last.
type Plugs struct {
	status       *twod.Text
	connected    world.Reader[input.GamepadConnected]
	disconnected world.Reader[input.GamepadDisconnected]
}

func (p *Plugs) Update(f world.Frame) {
	for e := range p.connected.Read(f.World) {
		p.status.SetText(fmt.Sprintf("Pad %d plugged in: %s", e.ID, e.Name))
	}
	for e := range p.disconnected.Read(f.World) {
		p.status.SetText(fmt.Sprintf("Pad %d unplugged", e.ID))
	}
}
