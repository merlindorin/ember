package main

import (
	"runtime"
	"strings"
	"testing"

	"github.com/merlindorin/ember/automation"
	"github.com/merlindorin/ember/input"
)

// element is the element whose automation ID ends in id, ok false when
// none shows.
func element(auto *automation.Automation, id string) (automation.Element, bool) {
	_, els := auto.Snapshot("", "")
	for _, e := range els {
		if e.ID == id || strings.HasSuffix(e.ID, "/"+id) {
			return e, true
		}
	}
	return automation.Element{}, false
}

// entries are the entries of the menu named menu, in order, read through
// automation with the menu opened by press, then closed again.
func entries(t *testing.T, auto *automation.Automation, press func(string), menu string) []automation.Element {
	t.Helper()
	press(menu)
	defer press(menu)
	_, els := auto.Snapshot("", "")
	var out []automation.Element
	for _, e := range els {
		if i := strings.LastIndex(
			e.ID,
			"/"+menu+"/",
		); i >= 0 && e.Role == automation.RoleMenuItem &&
			!strings.Contains(e.ID[i+len(menu)+2:], "/") {
			out = append(out, e)
		}
	}
	return out
}

// entry is the menu entry at path, menu/leaf, read as entries does.
func entry(t *testing.T, auto *automation.Automation, press func(string), path string) automation.Element {
	t.Helper()
	menu, leaf, _ := strings.Cut(path, "/")
	for _, e := range entries(t, auto, press, menu) {
		if leafOf(e) == leaf {
			return e
		}
	}
	t.Fatalf("no menu entry %s", path)
	return automation.Element{}
}

func leafOf(e automation.Element) string { return e.ID[strings.LastIndex(e.ID, "/")+1:] }

func TestTheMenusShortcutsChooseTheirItems(t *testing.T) {
	w, tool, _, press := driven(t, copyOfWalls(t))
	press("tile-5")
	n := len(tool.ed.tile.Shapes)
	addObject(t, w, tool, Rect)
	cmd := input.KeyLeftControl
	if runtime.GOOS == "darwin" {
		cmd = input.KeyLeftSuper
	}
	in := tool.win.Input()
	hit := func(key input.Key, mods ...input.Key) {
		for _, m := range mods {
			in.SetKey(m, true)
		}
		w.RunFrames(1)
		in.SetKey(key, true)
		w.RunFrames(1)
		in.SetKey(key, false)
		for _, m := range mods {
			in.SetKey(m, false)
		}
		w.RunFrames(3)
	}
	hit(input.KeyBackspace)
	if len(tool.ed.tile.Shapes) != n {
		t.Fatalf("Backspace is Edit › Delete shape: %d shapes", len(tool.ed.tile.Shapes))
	}
	hit(input.KeyZ, cmd)
	if len(tool.ed.tile.Shapes) != n+1 {
		t.Fatalf("Cmd+Z is Edit › Undo: %d shapes", len(tool.ed.tile.Shapes))
	}
}

func TestViewShadowsAndThePreviewSwitchAreOneSetting(t *testing.T) {
	w, tool, auto, press := driven(t, fromFixture(t, "walls.tsj"))
	tool.previewPanel.SetCollapsed(false)
	w.RunFrames(2)
	if entry(t, auto, press, "view/shadows").Value != true {
		t.Fatal("View › Shadows starts checked, as the lamp casts shadows")
	}
	press("view")
	press("view/shadows")
	if tool.lamp.Shadows() || tool.checks["shadows"].Checked() {
		t.Fatal("unchecked, the lamp casts none and the preview's switch follows")
	}
	auto.Queue(tool.checks["shadows"].Path(), automation.ActPress, nil)
	w.RunFrames(3)
	if !tool.lamp.Shadows() || entry(t, auto, press, "view/shadows").Value != true {
		t.Fatal("the switch on again checks View › Shadows")
	}
}
