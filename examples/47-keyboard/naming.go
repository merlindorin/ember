package main

import (
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/world"
)

// Naming edits the hero's name tag. Enter starts and ends it; meanwhile
// typed characters add to the name, Backspace takes the last one off, and
// Ctrl or Cmd with V pastes the clipboard.
type Naming struct {
	window *platform.Window
	tag    *twod.Text
	name   string
	active bool
}

func (n *Naming) Update(world.Frame) {
	in := n.window.Input()
	if in.Pressed(input.KeyEnter) {
		n.active = !n.active
	}
	if !n.active {
		n.tag.SetText(n.name)
		return
	}
	n.name += string(in.Text())
	if in.Pressed(input.KeyBackspace) && n.name != "" {
		r := []rune(n.name)
		n.name = string(r[:len(r)-1])
	}
	if (in.Down(input.KeyControl) || in.Down(input.KeySuper)) && in.Pressed(input.KeyV) {
		n.name += n.window.Clipboard()
	}
	n.tag.SetText(n.name + "_")
}
