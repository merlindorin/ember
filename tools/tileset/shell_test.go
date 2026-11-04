package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/align"
)

func TestAFoldedSectionShowsItsTitleAlone(t *testing.T) {
	w, tool, _, _ := driven(t, copyOfWalls(t))
	tool.previewPanel.SetCollapsed(true)
	w.RunFrames(2)
	if r := tool.sliders["radius"].Rect(); r.W > 0 || r.H > 0 {
		t.Fatalf("a folded preview shows no lamp settings: %v", r)
	}
	if !tool.pane.Hidden() {
		t.Fatal("nor the preview pane")
	}
	tool.previewPanel.SetCollapsed(false)
	w.RunFrames(2)
	if r := tool.sliders["radius"].Rect(); r.W <= 0 || tool.pane.Hidden() {
		t.Fatalf("open, both show: %v %v", r, tool.pane.Hidden())
	}
}

func TestTheInspectorWindowLinesUpAtEverySize(t *testing.T) {
	w, tool, _, _ := driven(t, copyOfWalls(t))
	tool.previewPanel.SetCollapsed(false)
	tool.SelectObject(0)
	for _, size := range []geom.Vec2{{X: 1100, Y: 700}, {X: 1280, Y: 800}} {
		tool.win.SetSize(size)
		w.RunFrames(3)
		if faults := align.Check(tool.dialogs); len(faults) > 0 {
			t.Fatalf("at %v, the inspector lines up with an object open: %v", size, faults)
		}
		if r := tool.inspectorWin; !tool.dialogs.Open(r) {
			t.Fatal("the inspector's window shows while a tileset is open")
		}
		if r, _ := tool.dialogs.BoxOf(tool.inspectorWin); r.X+r.W > size.X || r.Y+r.H > size.Y {
			t.Fatalf("at %v, the inspector %v stays in the window", size, r)
		}
	}
}

func TestTheFooterNamesTheFileItsTilesAndItsChanges(t *testing.T) {
	path := copyOfWalls(t)
	w, tool, _, _ := driven(t, path)
	tiles := len(readTiled(t, path).Tiles)
	if got, want := tool.fileStatus.Text(), fmt.Sprintf(
		"walls.tsj · %d tiles",
		int(tool.ed.doc.get("tilecount").num()),
	); got != want ||
		tiles == 0 {
		t.Fatalf("the footer: %q, want %q", got, want)
	}
	if !tool.editedDot.HasClass("clean") {
		t.Fatal("no dot before a change")
	}
	tool.SetBool("tall", true)
	w.RunFrames(2)
	if got := tool.fileStatus.Text(); !strings.HasSuffix(got, " · edited") || tool.editedDot.HasClass("clean") {
		t.Fatalf("a change says edited, with the dot: %q", got)
	}
}

func TestATilesClassTitlesItsSection(t *testing.T) {
	path := copyOfWalls(t)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var f map[string]any
	if err = json.Unmarshal(b, &f); err != nil {
		t.Fatal(err)
	}
	tiles, _ := f["tiles"].([]any)
	for _, tile := range tiles {
		if m, isMap := tile.(map[string]any); isMap && m["id"] == 5.0 {
			m["class"] = "wall_left"
		}
	}
	if b, err = json.Marshal(f); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
	w, tool, auto, _ := driven(t, path)
	tool.pick(5)
	w.RunFrames(2)
	if e, ok := element(auto, "tile/title"); !ok || e.Label != "Tile 5 · wall_left" {
		t.Fatalf("the title: %+v", e)
	}
}

func TestOpeningThePreviewScrollsItIntoView(t *testing.T) {
	w, tool, _, press := driven(t, copyOfWalls(t))
	tool.win.SetSize(geom.V2(1100, 700))
	tool.SelectObject(0)
	w.RunFrames(3)
	press("preview-section/title")
	w.RunFrames(3)
	view, _ := tool.dialogs.BoxOf(tool.inspector.n)
	if p := tool.paneFull; tool.pane.Hidden() || p.Y < view.Y || p.Y+p.H > view.Y+view.H+0.5 {
		t.Fatalf("the opened preview %v shows whole in the inspector %v", p, view)
	}
}

func TestTheInspectorsFieldsReachItsRightPadding(t *testing.T) {
	w, tool, auto, _ := driven(t, copyOfWalls(t))
	tool.SelectObject(0)
	for _, size := range []geom.Vec2{{X: 1280, Y: 800}, {X: 1100, Y: 700}} {
		tool.win.SetSize(size)
		w.RunFrames(3)
		body, _ := tool.dialogs.BoxOf(tool.tilePanel.n.Children()[1])
		edge := body.X + body.W
		root, _ := inspect(t, w, auto).root(tool)
		_, els := auto.Snapshot("", "")
		fields := 0
		for _, e := range els {
			leaf := e.ID[strings.LastIndex(e.ID, "/")+1:]
			if !strings.HasPrefix(e.ID, root+"/") || e.Bounds.W <= 0 || leaf == "add-object" || leaf == "x" ||
				leaf == "width" ||
				e.Role == "toggle" ||
				e.Role == "label" {
				continue
			}
			if e.Role != "combo" && e.Role != "text-input" && leaf != "delete" {
				continue
			}
			fields++
			if end := e.Bounds.X + e.Bounds.W; math.Abs(float64(end-edge)) > 0.5 {
				t.Fatalf("at %v, %s ends at %v, the inspector's padding at %v", size, e.ID, end, edge)
			}
		}
		if fields < 8 {
			t.Fatalf("at %v, the properties, the open object's fields and Delete: %d", size, fields)
		}
		var band *ui.Node
		tool.tilePanel.n.Walk(func(n *ui.Node) bool {
			if n.HasClass("band") {
				band = n
			}
			return band == nil
		})
		if r, _ := tool.dialogs.BoxOf(band); math.Abs(float64(r.X+r.W-edge)) > 0.5 {
			t.Fatalf("at %v, the object's band %v reaches the padding at %v", size, r, edge)
		}
	}
}
