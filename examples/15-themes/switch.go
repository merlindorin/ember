package main

import (
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/world"
)

// look is a theme's name and the class that turns it on; no class is the
// kit's own look.
type look struct {
	name, class string
}

func looks() []look {
	return []look{
		{"Default", ""},
		{"Ember, from Go", "ember"},
		{"Frost, from a file", "frost"},
		{"Wood, a skin", "wood"},
	}
}

// Switch gives the dialog the next look each time T is pressed.
type Switch struct {
	window       *platform.Window
	name, dialog *ui.Node
	next         int
}

func (s *Switch) Update(world.Frame) {
	if !s.window.Input().Pressed(input.KeyT) {
		return
	}
	if c := looks()[s.next].class; c != "" {
		s.dialog.RemoveClass(c)
	}
	s.next = (s.next + 1) % len(looks())
	l := looks()[s.next]
	if l.class != "" {
		s.dialog.AddClass(l.class)
	}
	s.name.SetText(l.name + " · press T")
}
