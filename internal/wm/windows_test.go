package wm //nolint:testpackage // tests the unexported per-world owners

import (
	"testing"

	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/world"
)

func TestTheFirstWindowAddedIsThePrimary(t *testing.T) {
	w := world.New()
	if Primary(w) != nil || Target(w) != nil || Presenting(w) {
		t.Fatal("no window yet")
	}
	main, tool := platform.NewWindow(platform.Title("main")), platform.NewWindow(platform.Title("tool"))
	w.AddEntities(main, tool)
	if Primary(w) != main || Resolve(w, nil) != main || Resolve(w, tool) != tool {
		t.Fatal("the first window added is the primary")
	}
	if Target(w) != main {
		t.Fatal("without a presented window the target is the primary")
	}
	SetCurrent(w, tool)
	if Target(w) != tool || !Presenting(w) {
		t.Fatal("the presented window is the target")
	}
	SetCurrent(w, nil)
	w.RemoveEntity(main)
	if Primary(w) != tool {
		t.Fatal("closing the primary makes the next window primary")
	}
	if Input(w) != tool.Input() {
		t.Fatal("input is the primary window's")
	}
}

func TestAnEntityBelongsToTheWindowItIsUnder(t *testing.T) {
	w := world.New()
	main, tool := platform.NewWindow(), platform.NewWindow()
	group, inside, outside := world.NewBasic(), world.NewBasic(), world.NewBasic()
	tool.AppendChild(group)
	group.AppendChild(inside)
	w.AddEntities(main, tool, outside)
	if Of(w, inside) != tool || Of(w, outside) != nil {
		t.Fatal("the nearest window above it, through other entities")
	}
	if Home(w, inside, nil) != tool || Home(w, outside, nil) != main || Home(w, inside, main) != main {
		t.Fatal("an explicit window wins, then the one it is under, then the primary")
	}
}
