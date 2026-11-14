package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/automation"
	"github.com/lantervale/ember/collide/tiles"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/world"
)

// driven is the tool built on path with a recording automation, its
// remembered state in a temporary folder, and press, which presses an
// element by the end of its automation path and runs the frames it takes.
func driven(t *testing.T, path string) (*world.World, *Tool, *automation.Automation, func(id string)) {
	t.Helper()
	t.Setenv("EMBER_TILESET_DIR", t.TempDir())
	auto := automation.New(automation.Recording())
	w, tool, err := build(path, ember.Automation(auto))
	if err != nil {
		t.Fatal(err)
	}
	w.RunFrames(2)
	press := func(id string) {
		t.Helper()
		auto.Queue(find(t, auto, id), automation.ActPress, nil)
		w.RunFrames(3)
	}
	return w, tool, auto, press
}

// find is the automation ID ending in id among the elements shown.
func find(t *testing.T, auto *automation.Automation, id string) string {
	t.Helper()
	_, els := auto.Snapshot("", "")
	for _, e := range els {
		if e.ID == id || strings.HasSuffix(e.ID, "/"+id) {
			return e.ID
		}
	}
	t.Fatalf("no element %s shown", id)
	return ""
}

func TestAutomationAddsAShadowShapeAndSavesOnlyThatTile(t *testing.T) {
	path := copyOfWalls(t)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	w, tool, auto, press := driven(t, path)
	const id = 5
	press("tile-5")
	w.RunFrames(3)
	addObject(t, w, tool, Rect)
	inspect(t, w, auto).choose(inspect(t, w, auto).obj("type"), "shadow")
	w.RunFrames(3)
	press("file")
	press("file/save")

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	was, err := parse(before)
	if err != nil {
		t.Fatal(err)
	}
	now, err := parse(after)
	if err != nil {
		t.Fatal(err)
	}
	for _, other := range tileIDs(was) {
		a, b := strip(readTile(was, other)), strip(readTile(now, other))
		if other != id {
			if !reflect.DeepEqual(a, b) {
				t.Fatalf("tile %d changed: %+v → %+v", other, a, b)
			}
			continue
		}
		if len(b.Shapes) != len(a.Shapes)+1 || !b.Shapes[len(a.Shapes)].Shadow {
			t.Fatalf("tile %d: %+v → %+v, want one light-only shape more", id, a, b)
		}
	}

	ts, err := twod.LoadTiledTileset(os.DirFS(filepath.Dir(path)), filepath.Base(path))
	if err != nil {
		t.Fatal(err)
	}
	if got := shadowShapes(ts, id+1); len(got) != 1 || got[0].W != 8 || got[0].H != 8 ||
		got[0].X < 0 || got[0].X+8 > 16 || got[0].Y < 0 || got[0].Y+8 > 16 {
		t.Fatalf("the engine reads tile %d's shadow shapes as %v", id, got)
	}
	if tool.ed.Changed() {
		t.Fatal("saved, yet still changed")
	}
}

// strip is t without what ties its shapes to the document read.
func strip(t Tile) Tile {
	for i := range t.Shapes {
		t.Shapes[i].from = nil
	}
	return t
}

func TestTheWalkerStopsAtTheTilesCollisionShape(t *testing.T) {
	w, tool, _, _ := driven(t, copyOfWalls(t))
	tool.previewPanel.SetCollapsed(false)
	solids := tiles.Of(tool.tmap).SolidIn(geom.Rect{X: -500, Y: -500, W: 1000, H: 1000})
	if len(solids) == 0 {
		t.Fatal("tile 0 has no collision shape")
	}
	wall := solids[0]
	tool.walker.SetPosition(geom.V2(wall.X-60, wall.Y+wall.H/2))
	tool.walker.Snap()
	in := tool.win.Input()
	in.SetKey(input.KeyD, true)
	w.RunFrames(30)
	if tool.walker.Position().X != wall.X-60 {
		t.Fatal("WASD waits for the preview to have the focus")
	}
	tool.pane.Focus()
	w.RunFrames(120)
	b := tool.body.Bounds()
	if b.X+b.W > wall.X+0.01 || b.X+b.W < wall.X-1 {
		t.Fatalf("the walker's body %v, the wall %v: want it stopped against it", b, wall)
	}
}

// addObject adds an object of kind to the picked tile, as the inspector
// asks the Tool to.
func addObject(t *testing.T, w *world.World, tool *Tool, kind Kind) {
	t.Helper()
	tool.AddObject(kind)
	w.RunFrames(3)
}
