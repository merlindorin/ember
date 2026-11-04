package main

import (
	"testing"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/platform"
)

func TestTheThemesParseWithoutWarnings(t *testing.T) {
	if w := dialogPage().Warnings(); len(w) > 0 {
		t.Fatalf("every sheet reads cleanly: %v", w)
	}
}

func TestTPassesTheDialogFromLookToLook(t *testing.T) {
	win := platform.NewWindow(platform.Size(560, 360))
	doc := dialogPage()
	win.Add(doc)
	dialog, name := doc.ByID("dialog"), doc.ByID("name")
	w := ember.New().AddEntities(win).AddSystems(&Switch{window: win, name: name, dialog: dialog})
	w.RunFrames(2)
	press := func() {
		win.Input().SetKey(input.KeyT, true)
		w.RunFrames(1)
		win.Input().SetKey(input.KeyT, false)
		w.RunFrames(1)
	}
	for _, want := range []struct{ class, name string }{
		{"ember", "Ember, from Go · press T"},
		{"frost", "Frost, from a file · press T"},
		{"wood", "Wood, a skin · press T"},
	} {
		press()
		if !dialog.HasClass(want.class) || name.Text() != want.name {
			t.Fatalf("the dialog takes %s: %q", want.class, name.Text())
		}
		for _, l := range looks() {
			if l.class != "" && l.class != want.class && dialog.HasClass(l.class) {
				t.Fatalf("one look at a time: %s still on", l.class)
			}
		}
	}
	press()
	if dialog.HasClass("wood") || name.Text() != "Default · press T" {
		t.Fatalf("the fourth press comes back to the default look: %q", name.Text())
	}
}
