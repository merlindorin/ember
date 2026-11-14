package wm //nolint:testpackage // tests the unexported per-world owners

import (
	"testing"

	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/world"
)

func TestTheLastOwnerOfAWindowTakesItsInput(t *testing.T) {
	w := world.New()
	main, tool := platform.NewWindow(), platform.NewWindow()
	a, b, other := world.NewBasic(), world.NewBasic(), world.NewBasic()
	w.AddEntities(main, tool, a, b, other)
	if Owner(w, main) != nil || !Takes(w, main, other) {
		t.Fatal("an unowned window gives its input to everything")
	}
	Own(w, main, a)
	Own(w, main, b)
	if Owner(w, main) != b || Takes(w, main, a) || !Takes(w, main, b) || Takes(w, main, other) {
		t.Fatal("the last owner takes the input, the ones below wait")
	}
	if !Takes(w, tool, other) {
		t.Fatal("owning a window leaves the others free")
	}
	if !main.Input().PointerCaptured() || !main.Input().KeyboardCaptured() {
		t.Fatal("an owned window's pointer and keyboard are captured at once, so game systems stand aside")
	}
	Release(w, main, b)
	if Owner(w, main) != a || !Takes(w, main, a) {
		t.Fatal("releasing the top owner hands the input to the one below")
	}
	Own(w, main, a)
	Release(w, main, a)
	if Owner(w, main) != nil {
		t.Fatal("an owner is once in the stack")
	}
}

func TestAnOwnerLeavingTheWorldReleasesItsWindow(t *testing.T) {
	w := world.New()
	main := platform.NewWindow()
	a := world.NewBasic()
	w.AddEntities(main, a)
	w.AddPlugins(Plugin{})
	Own(w, main, a)
	w.RunFrames(1)
	if !main.Input().PointerCaptured() {
		t.Fatal("an owned window stays captured each frame")
	}
	w.RemoveEntity(a)
	w.RunFrames(1)
	if Owner(w, main) != nil {
		t.Fatal("an owner gone from the World holds nothing")
	}
}

func TestWhatIsUnderTheOwnerTakesTheInputAndItsRootHoldsIt(t *testing.T) {
	w := world.New()
	main := platform.NewWindow()
	root, owner, child, other := world.NewBasic(), world.NewBasic(), world.NewBasic(), world.NewBasic()
	root.AppendChild(owner)
	owner.AppendChild(child)
	w.AddEntities(main, root, other)
	Own(w, main, owner)
	if !Takes(w, main, child) || !Takes(w, main, owner) || Takes(w, main, root) || Takes(w, main, other) {
		t.Fatal("the owner and what is under it take the input, nothing above or beside")
	}
	if !Holds(w, main, root) || !Holds(w, main, owner) || Holds(w, main, child) || Holds(w, main, other) {
		t.Fatal("the root holding the owner is the one that holds the window")
	}
	if !Owns(w, main, owner) || Owns(w, main, child) {
		t.Fatal("only what owned is among the owners")
	}
}

type hideable struct {
	*world.BasicEntity
	hidden bool
}

func (h *hideable) Hidden() bool { return h.hidden }

func TestAClosedWindowOrAHiddenOwnerOwnsNothing(t *testing.T) {
	w := world.New().AddPlugins(Plugin{})
	main, tool := platform.NewWindow(), platform.NewWindow()
	layer := &hideable{BasicEntity: world.NewBasic()}
	a, b := world.NewBasic(), world.NewBasic()
	layer.AppendChild(a)
	w.AddEntities(main, tool, layer, b)
	Own(w, main, a)
	Own(w, tool, b)
	layer.hidden = true
	w.RemoveEntity(tool)
	w.RunFrames(1)
	if Owner(w, main) != nil {
		t.Fatal("an owner under a hidden parent owns nothing")
	}
	if len(of(w).owners) != 0 {
		t.Fatal("a closed window's owners are dropped")
	}
}
