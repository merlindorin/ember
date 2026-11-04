package main

import (
	"testing"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/platform"
)

func TestClickingTheButtonsChangesTheCount(t *testing.T) {
	win := platform.NewWindow(platform.Size(640, 360))
	doc, counter := counterPage()
	win.Add(doc)
	w := ember.New().AddEntities(win)
	w.RunFrames(2)

	in := win.Input()
	press := func(id string) {
		r, _ := doc.BoxOf(doc.ByID(id))
		in.SetMouse(r.Center())
		w.RunFrames(1)
		in.SetMouseButton(input.MouseLeft, true)
		w.RunFrames(1)
		in.SetMouseButton(input.MouseLeft, false)
		w.RunFrames(2)
	}
	press("click")
	press("click")
	press("click")
	if got := counter.label.Text(); got != "Clicks: 3" {
		t.Fatalf("after three clicks: %q", got)
	}
	press("reset")
	if got := counter.label.Text(); got != "Clicks: 0" {
		t.Fatalf("after reset: %q", got)
	}
}
