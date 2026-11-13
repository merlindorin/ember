package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/twod"
)

// copyOfWalls is the fixture's walls.tsj copied, with its picture, into a
// temporary folder: tests never touch the fixture itself.
func copyOfWalls(t *testing.T) string { return copyOf(t, "walls.tsj") }

// copyOf is the fixture's tileset name copied, with the picture it shares
// with the others, into a temporary folder.
func copyOf(t *testing.T, tileset string) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{tileset, "0x72_DungeonTilesetII_v1.7.png"} {
		b, err := os.ReadFile(filepath.Join("testdata/crypt/tiled", name))
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(dir, name), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return filepath.Join(dir, tileset)
}

func TestEditsUndoRedoAndSave(t *testing.T) {
	path := copyOfWalls(t)
	e, err := open(path)
	if err != nil {
		t.Fatal(err)
	}
	e.pick(0)
	before := len(readTile(e.doc, 0).Shapes)
	shadows := len(func() []geom.Rect {
		ts, loadErr := twod.LoadTiledTileset(os.DirFS(filepath.Dir(path)), "walls.tsj")
		if loadErr != nil {
			t.Fatal(loadErr)
		}
		return shadowShapes(ts, 1)
	}())
	e.Add(Rect, true)
	i := e.selected
	if i != before {
		t.Fatalf("the new shape comes after the %d the tile had: %d", before, i)
	}
	e.Move(geom.V2(1.4, -2.6))
	moved := e.tile.Shapes[i].Bounds()
	e.Resize(geom.Rect{X: 0, Y: 0, W: 16.2, H: 0.2})
	got := e.tile.Shapes[i]
	if !got.Shadow || got.Bounds() != (geom.Rect{W: 16, H: 1}) {
		t.Fatalf("a shadow rectangle, snapped to art pixels, at least a pixel high: %+v", got)
	}
	e.Undo()
	if b := e.tile.Shapes[i].Bounds(); b != moved {
		t.Fatalf("undo takes the resize back: %v", b)
	}
	e.Redo()
	if !e.Changed() {
		t.Fatal("edits not saved yet")
	}
	if err = e.Save(); err != nil {
		t.Fatal(err)
	}
	ts, err := twod.LoadTiledTileset(os.DirFS(filepath.Dir(path)), "walls.tsj")
	if err != nil {
		t.Fatal(err)
	}
	if s := shadowShapes(ts, 1); len(s) != shadows+1 || s[len(s)-1] != (geom.Rect{W: 16, H: 1}) {
		t.Fatalf("the engine reads the saved shadow shape after the %d it had: %v", shadows, s)
	}
	if e.Changed() {
		t.Fatal("saved")
	}
	if err = os.WriteFile(path, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	e.ToggleShadow()
	if err = e.Save(); !errors.Is(err, errChangedOnDisk) {
		t.Fatalf("it refuses to save over a file changed since: %v", err)
	}
}

func TestShapesStayInsideTheTile(t *testing.T) {
	e, err := open(copyOfWalls(t))
	if err != nil {
		t.Fatal(err)
	}
	e.pick(0)
	e.Add(Rect, false)
	e.Move(geom.V2(100, -100))
	if got := e.tile.Shapes[len(e.tile.Shapes)-1].Bounds(); got.X+got.W > e.size.X || got.Y < 0 {
		t.Fatalf("moved far out, it stops at the tile's edge: %v in %v", got, e.size)
	}
	e.Resize(geom.Rect{X: -5, Y: -5, W: 400, H: 400})
	if got := e.tile.Shapes[len(e.tile.Shapes)-1].Bounds(); got != (geom.Rect{W: e.size.X, H: e.size.Y}) {
		t.Fatalf("grown past it, it fills the tile: %v", got)
	}
	e.Add(Polygon, false)
	e.Move(geom.V2(-100, 0))
	if got := e.tile.Shapes[len(e.tile.Shapes)-1].Bounds(); got.X != 0 || got.W <= 0 {
		t.Fatalf("a polygon moves back in whole, keeping its shape: %v", got)
	}
	e.MovePoint(0, geom.V2(-50, 500))
	if got := e.tile.Shapes[len(e.tile.Shapes)-1].Bounds(); got.X < 0 || got.Y+got.H > e.size.Y {
		t.Fatalf("its point dragged out stays on the edge: %v", got)
	}
}
