package main

import (
	"fmt"

	"github.com/merlindorin/ember/ui"
)

// The settings' callbacks: each writes on the status line what its
// control or the window reported, and Escape or east on anything in the
// settings, a back reaching the window, closes them.

func (s *settings) say(text string) { s.status.SetText(text) }

func (s *settings) checked(e ui.Event) {
	s.say(fmt.Sprintf("%s is %s", e.Node.Parent().Text(), onOff(s.doc.Checked(e.Node))))
}

func (s *settings) level(e ui.Event) {
	var v float64
	if _, err := fmt.Sscan(s.doc.Value(e.Node), &v); err != nil {
		return
	}
	s.say(fmt.Sprintf("Volume %.0f%%", v*100))
}

func (s *settings) typing(e ui.Event) { s.say("Typing " + s.doc.Value(e.Node)) }
func (s *settings) greet(e ui.Event)  { s.say("Hello, " + s.doc.Value(e.Node)) }

func (s *settings) folded(e ui.Event) { s.say("Save slots " + foldedOpen(!s.doc.Open(e.Node))) }

func (s *settings) moved(e ui.Event) {
	s.say(fmt.Sprintf("Settings moved to %.0f, %.0f", e.At.X, e.At.Y))
}

func (s *settings) resized(e ui.Event) {
	s.say(fmt.Sprintf("Settings now %.0f × %.0f", e.Size.X, e.Size.Y))
}

func (s *settings) back() {
	if s.doc.Open(s.window) {
		s.doc.Close(s.window, "")
	}
}

func (s *settings) closed() { s.say("Settings closed: Enter opens them again") }

func onOff(on bool) string {
	if on {
		return "on"
	}
	return "off"
}

func foldedOpen(folded bool) string {
	if folded {
		return "folded"
	}
	return "open"
}
