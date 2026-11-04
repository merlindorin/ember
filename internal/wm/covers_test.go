package wm //nolint:testpackage // tests the unexported per-world owners

import (
	"testing"

	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/world"
)

func TestACoverHoldsThePointerForOneFrameAndLeavesTheKeys(t *testing.T) {
	w := world.New().AddPlugins(Plugin{})
	main, tool := platform.NewWindow(), platform.NewWindow()
	menu, item, widget := world.NewBasic(), world.NewBasic(), world.NewBasic()
	menu.AppendChild(item)
	w.AddEntities(main, tool, menu, widget)
	w.RunFrames(1)
	Cover(w, main, menu)
	if !TakesPointer(w, main, widget) {
		t.Fatal("a cover set this frame holds from the next")
	}
	w.RunFrames(1)
	if TakesPointer(w, main, widget) || !TakesPointer(w, main, menu) || !TakesPointer(w, main, item) {
		t.Fatal("the next frame, only the cover and what is under it take the pointer")
	}
	if !Takes(w, main, widget) || main.Input().KeyboardCaptured() {
		t.Fatal("the keys stay everyone's")
	}
	if !TakesPointer(w, tool, widget) {
		t.Fatal("another window is free")
	}
	w.RunFrames(1)
	if !TakesPointer(w, main, widget) || Covering(w, main) != nil {
		t.Fatal("not covered again, it lapses after one frame")
	}
}

func TestAnOwnerWinsOverACover(t *testing.T) {
	w := world.New().AddPlugins(Plugin{})
	main := platform.NewWindow()
	menu, modal := world.NewBasic(), world.NewBasic()
	w.AddEntities(main, menu, modal)
	Own(w, main, modal)
	Cover(w, main, menu)
	w.RunFrames(1)
	if TakesPointer(w, main, menu) || !TakesPointer(w, main, modal) {
		t.Fatal("under an owner, a cover takes nothing; the owner keeps the pointer")
	}
}

func TestAClosedWindowsCoverIsDropped(t *testing.T) {
	w := world.New().AddPlugins(Plugin{})
	main, tool := platform.NewWindow(), platform.NewWindow()
	menu := world.NewBasic()
	w.AddEntities(main, tool, menu)
	Cover(w, tool, menu)
	w.RemoveEntity(tool)
	w.RunFrames(1)
	if _, ok := of(w).covers[tool]; ok {
		t.Fatal("a closed window keeps no cover")
	}
}
