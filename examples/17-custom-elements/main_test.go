package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/automation"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/platform"
)

func TestTheOrbShowsTheHealthTheButtonsChange(t *testing.T) {
	win := platform.NewWindow(platform.Size(640, 360))
	doc, health := orbPage()
	win.Add(doc)
	auto := automation.New(automation.Recording())
	w := ember.New(ember.Automation(auto)).AddEntities(win)
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
	orb := func() (float32, string) { return health.orb.Health(), health.label.Text() }
	o, _ := doc.BoxOf(doc.ByID("orb"))
	l, _ := doc.BoxOf(doc.ByID("label"))
	if o.W != 96 || o.H != 96 || o.X >= l.X {
		t.Fatalf("the layout places the orb like any element, left of the label: %v", o)
	}
	press("hurt")
	press("hurt")
	press("heal")
	if h, l := orb(); h != 80 || l != "Health: 80" {
		t.Fatalf("health %v, label %q", h, l)
	}
	for range 5 {
		press("heal")
	}
	if h, _ := orb(); h != 100 {
		t.Fatalf("healing stops at the orb's maximum: %v", h)
	}

	press("hurt")
	w.RunFrames(2)
	_, elements := auto.Snapshot("", "")
	for _, e := range elements {
		if strings.HasSuffix(e.ID, "orb") {
			if fmt.Sprint(e.Value) != "85" || e.Role != automation.RoleMeter {
				t.Fatalf("automation reads the orb's health: %+v", e)
			}
			return
		}
	}
	t.Fatal("the orb records itself for automation")
}
