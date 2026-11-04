package main

import (
	"testing"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/internal/draw"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/kit"
)

func TestTheLabelShowsTheWindowsSize(t *testing.T) {
	win := platform.NewWindow(platform.Size(640, 360), platform.Background(color.Hex("#2b2f3a")))
	size := kit.Label("")
	label := ui.NewDocument(kit.Theme(), ui.CSS(`body { padding: 16px; }`))
	label.Body().Append(size)
	win.Add(label)
	w := ember.New().AddEntities(win).AddSystems(&ShowSize{window: win, label: size})

	w.RunFrames(3)
	if size.Text() != "640 × 360" {
		t.Fatalf("label %q", size.Text())
	}
	if c := draw.FrontFor(w, win); c == nil || len(c.Unindexed()) == 0 {
		t.Fatal("the label draws in the window")
	}
}

func TestFTogglesFullscreen(t *testing.T) {
	win := platform.NewWindow()
	w := ember.New().AddEntities(win).AddSystems(&ToggleFullscreen{window: win})
	press := func() {
		win.Input().SetKey(input.KeyF, true)
		w.RunFrames(1)
		win.Input().SetKey(input.KeyF, false)
		w.RunFrames(1)
	}
	press()
	if !win.Fullscreen() {
		t.Fatal("F goes fullscreen")
	}
	press()
	if win.Fullscreen() {
		t.Fatal("F again goes back")
	}
}
