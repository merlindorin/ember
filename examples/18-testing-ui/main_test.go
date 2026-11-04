package main

import (
	"testing"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/automation"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/platform"
)

// game is the form in a window that is never shown, and a frame function
// that runs one frame of it with whatever input the test set.
func game(t *testing.T, auto *automation.Automation) (*form, *Settings, *platform.Window, func()) {
	t.Helper()
	win := platform.NewWindow(platform.Size(520, 320))
	settings := &Settings{}
	f := newForm(settings)
	win.Add(f.doc)
	w := ember.New(ember.Automation(auto)).AddEntities(win)
	frame := func() { w.RunFrames(1) }
	frame()
	return f, settings, win, frame
}

func TestAPlayerTypesTheirNameAndSaves(t *testing.T) {
	f, settings, win, frame := game(t, automation.New())
	in := win.Input()
	click := func(id string) {
		r, _ := f.doc.BoxOf(f.doc.ByID(id))
		in.SetMouse(r.Center())
		frame()
		in.SetMouseButton(input.MouseLeft, true)
		frame()
		in.SetMouseButton(input.MouseLeft, false)
		frame()
	}

	click("name")
	for _, r := range "Rowan" {
		in.AddText(r)
	}
	frame()
	click("music")
	click("save")
	frame()

	if *settings != (Settings{Name: "Rowan", Volume: 0.7, Music: false}) {
		t.Fatalf("saved %+v", *settings)
	}
	if f.status.Text() != "Saved Rowan, 70%" {
		t.Fatalf("status %q", f.status.Text())
	}
}

func TestAutomationFillsTheFormByID(t *testing.T) {
	auto := automation.New(automation.Recording())
	_, settings, _, frame := game(t, auto)

	auto.Queue("settings/name", automation.ActType, "Rowan")
	auto.Queue("settings/volume", automation.ActSet, 0.3)
	auto.Queue("settings/music", automation.ActSet, false)
	frame()
	auto.Queue("settings/save", automation.ActPress, nil)
	for range 3 {
		frame()
	}

	if *settings != (Settings{Name: "Rowan", Volume: 0.3, Music: false}) {
		t.Fatalf("saved %+v", *settings)
	}
	_, elements := auto.Snapshot("", "settings")
	for _, e := range elements {
		if e.ID == "settings/status" {
			if e.Label != "Saved Rowan, 30%" {
				t.Fatalf("automation reads the status: %+v", e)
			}
			return
		}
	}
	t.Fatalf("no settings/status among %v", elements)
}
