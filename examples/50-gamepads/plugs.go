package main

import (
	"fmt"

	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/world"
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
