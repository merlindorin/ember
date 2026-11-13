package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/merlindorin/ember/automation"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/twod/light"
)

func TestTheSelectedShapesBoxIsTypedAndNudgedByArrows(t *testing.T) {
	w, tool, auto, press := driven(t, copyOfWalls(t))
	press("tile-5")
	addObject(t, w, tool, Rect)
	ed := tool.ed
	i := ed.selected
	ed.selected = -1
	w.RunFrames(2)
	press(fmt.Sprintf("object-%d", i+1))
	if ed.selected != i {
		t.Fatalf("a click on its block selects the object: %d", ed.selected)
	}
	w.RunFrames(2)
	ins := inspect(t, w, auto)
	if got := value(ins.obj("width")); got != "8" {
		t.Fatalf("its block shows its width: %q", got)
	}
	ins.typeIn(ins.obj("x"), "2")
	if b := ed.tile.Shapes[i].Bounds(); b.X != 2 {
		t.Fatalf("x typed and entered moves it: %v", b)
	}
	in := tool.win.Input()
	in.SetKey(input.KeyRight, true)
	w.RunFrames(1)
	in.SetKey(input.KeyRight, false)
	w.RunFrames(1)
	in.SetKey(input.KeyLeftShift, true)
	in.SetKey(input.KeyDown, true)
	w.RunFrames(1)
	in.SetKey(input.KeyDown, false)
	in.SetKey(input.KeyLeftShift, false)
	w.RunFrames(1)
	if b := ed.tile.Shapes[i].Bounds(); b.X != 3 || b.Y != min(6+4, ed.size.Y-b.H) {
		t.Fatalf("right nudges one art pixel, Shift+down four, kept in the tile: %v", b)
	}
}

func TestOnlyTheEnginesPropertiesShowAndSnappingTurnsOff(t *testing.T) {
	w, tool, auto, press := driven(t, copyOf(t, "props.tsj"))
	w.RunFrames(2)
	for _, name := range []string{"name", "reach_down", "cells", "door"} {
		if _, ok := inspect(t, w, auto).find("properties", name); ok {
			t.Fatalf("%s, a property only the game reads, shows in the inspector", name)
		}
	}
	press("view")
	press("view/snap")
	if !tool.ed.free {
		t.Fatal("View › Snap to art pixels off lets shapes off whole pixels")
	}
	addObject(t, w, tool, Rect)
	tool.ed.Move(geom.V2(0.3, 0))
	if x := tool.ed.tile.Shapes[tool.ed.selected].X; x != float32(int(x))+0.25 {
		t.Fatalf("to a quarter of a pixel: %v", x)
	}
}

func TestTheWheelZoomsTheEditorAndThePreview(t *testing.T) {
	w, tool, _, _ := driven(t, copyOfWalls(t))
	in := tool.win.Input()
	in.SetMouse(tool.editor.Rect().Center())
	in.AddWheel(geom.V2(0, 2))
	w.RunFrames(2)
	if tool.editor.zoom <= 1 {
		t.Fatalf("the wheel over the editor zooms it in: %v", tool.editor.zoom)
	}
	in.SetMouse(tool.pane.Rect().Center())
	in.AddWheel(geom.V2(0, -2))
	w.RunFrames(2)
	if z := tool.cam.Zoom(); z >= 1 {
		t.Fatalf("over the preview, the preview: %v", z)
	}
}

func TestAPolygonGainsAndLosesPointsKeepingThree(t *testing.T) {
	e, err := open(copyOfWalls(t))
	if err != nil {
		t.Fatal(err)
	}
	e.pick(5)
	e.Add(Polygon, false)
	poly := func() Shape { return e.tile.Shapes[e.selected] }
	if n := len(poly().Points); n != 3 {
		t.Fatalf("a new polygon is a triangle: %d points", n)
	}
	e.point = 0
	if err = e.DeletePoint(); err == nil || len(poly().Points) != 3 {
		t.Fatal("a triangle keeps its three points")
	}
	a, b := poly().Points[0], poly().Points[1]
	mid := a.Add(b).Scale(0.5).Add(geom.V2(poly().X, poly().Y))
	e.InsertPoint(0, mid.Add(geom.V2(0.4, 0)))
	if got := poly().Points; len(got) != 4 || e.point != 1 ||
		got[1].Add(geom.V2(poly().X, poly().Y)) != geom.V2(snap(mid.X+0.4), snap(mid.Y)) {
		t.Fatalf("a point goes in after point 0, snapped, and is selected: %v, point %d", got, e.point)
	}
	e.InsertPoint(1, geom.V2(-30, 500))
	if p := poly().Points[2].Add(geom.V2(poly().X, poly().Y)); p.X != 0 || p.Y != e.size.Y {
		t.Fatalf("one put outside the tile lands on its edge: %v", p)
	}
	e.Delete()
	if n := len(poly().Points); n != 4 || len(e.tile.Shapes) == 0 {
		t.Fatalf("Delete with a point selected removes the point, not the shape: %d points", n)
	}
	e.Undo()
	if n := len(poly().Points); n != 5 {
		t.Fatalf("undo puts it back: %d points", n)
	}
}

func TestClickingAPolygonEdgesPlusPutsAPointThere(t *testing.T) {
	w, tool, _, press := driven(t, copyOfWalls(t))
	press("tile-5")
	addObject(t, w, tool, Polygon)
	ed := tool.ed
	s := ed.tile.Shapes[ed.selected]
	pts := handles(s)
	mid := pts[0].Add(pts[1]).Scale(0.5)
	r := tool.editor.Rect()
	origin, zoom := tool.editor.frame(geom.V2(r.W, r.H), ed.extent())
	at := geom.V2(r.X, r.Y).Add(origin).Add(mid.Scale(zoom))
	in := tool.win.Input()
	in.SetMouse(at)
	w.RunFrames(1)
	in.SetMouseButton(input.MouseLeft, true)
	w.RunFrames(1)
	in.SetMouseButton(input.MouseLeft, false)
	w.RunFrames(2)
	if n := len(ed.tile.Shapes[ed.selected].Points); n != 4 {
		t.Fatalf("a click on the edge's + handle adds a point: %d points", n)
	}
	in.SetKey(input.KeyLeftAlt, true)
	in.SetMouseButton(input.MouseLeft, true)
	w.RunFrames(1)
	in.SetMouseButton(input.MouseLeft, false)
	in.SetKey(input.KeyLeftAlt, false)
	w.RunFrames(2)
	if n := len(ed.tile.Shapes[ed.selected].Points); n != 3 {
		t.Fatalf("Alt-click on it takes it away again: %d points", n)
	}
}

func TestABoxReachingPastItsTileKeepsItsReachWhenEdited(t *testing.T) {
	e, err := open(copyOfWalls(t))
	if err != nil {
		t.Fatal(err)
	}
	e.pick(0)
	e.selected = 0
	before := e.tile.Shapes[0].Bounds()
	if before.Y+before.H <= e.size.Y {
		t.Skip("walls.tsj tile 0 no longer reaches under its picture")
	}
	e.Move(geom.V2(1, 0))
	e.Move(geom.V2(-1, 0))
	if got := e.tile.Shapes[0].Bounds(); got != before {
		t.Fatalf("moved and back, the box reaching %v under its tile is as it was: %v", before.Y+before.H-e.size.Y, got)
	}
	if ext := e.extent(); ext.H != before.H {
		t.Fatalf("the editor shows all of it: %v", ext)
	}
}

func TestTilesPickedTogetherTakePropertiesAndShapesTogether(t *testing.T) {
	w, tool, auto, press := driven(t, copyOfWalls(t))
	ed := tool.ed
	press("tile-5")
	in := tool.win.Input()
	in.SetKey(input.KeyLeftShift, true)
	press("tile-6")
	press("tile-7")
	in.SetKey(input.KeyLeftShift, false)
	w.RunFrames(1)
	if ed.tile.ID != 5 || !ed.marked[6] || !ed.marked[7] {
		t.Fatalf("Shift-click marks tiles beside the one picked: picked %d, marked %v", ed.tile.ID, ed.marked)
	}
	inspect(t, w, auto).act(inspect(t, w, auto).prop("tall"), automation.ActSet, true)
	for _, id := range []int{5, 6, 7} {
		if !readTile(ed.doc, id).Tall {
			t.Fatalf("Tall reaches tile %d too", id)
		}
	}
	ed.Undo()
	for _, id := range []int{5, 6, 7} {
		if readTile(ed.doc, id).Tall {
			t.Fatalf("one undo takes it back from tile %d", id)
		}
	}

	addObject(t, w, tool, Rect)
	press("edit")
	press("edit/apply")
	want := len(ed.tile.Shapes)
	for _, id := range []int{6, 7} {
		if n := len(readTile(ed.doc, id).Shapes); n != want {
			t.Fatalf("Apply shapes gives tile %d the picked tile's %d shapes: %d", id, want, n)
		}
	}

	press("edit")
	press("edit/copy")
	press("tile-8")
	if len(ed.marked) != 0 {
		t.Fatal("a plain click picks one tile again")
	}
	before := len(ed.tile.Shapes)
	press("edit")
	press("edit/paste")
	if n := len(ed.tile.Shapes); n != before+1 || ed.selected != n-1 {
		t.Fatalf("Paste adds the shape copied to tile 8 and selects it: %d → %d", before, n)
	}
}

// roomTree is the fixture's crypt room and its tilesets copied into a
// temporary folder, laid out as the game lays them: the path of its
// walls.tsj.
func roomTree(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	tiled := filepath.Join(dir, "crypt", "tiled")
	rooms := filepath.Join(dir, "dungeon", "rooms")
	for _, d := range []string{tiled, rooms} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	copyFile := func(from, to string) {
		b, err := os.ReadFile(from)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(to, b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, n := range []string{
		"walls.tsj", "floor.tsj", "props.tsj", "flags.tsj", "0x72_DungeonTilesetII_v1.7.png", "flags.png",
	} {
		copyFile(filepath.Join("testdata/crypt/tiled", n), filepath.Join(tiled, n))
	}
	copyFile("testdata/dungeon/rooms/crypt.tmj", filepath.Join(rooms, "crypt.tmj"))
	return filepath.Join(tiled, "walls.tsj")
}

func TestThePreviewShowsARoomWithTheTilesetAsEdited(t *testing.T) {
	path := roomTree(t)
	w, tool, auto, press := driven(t, path)
	press("view")
	press("view/room")
	if tool.rooms.Hidden() || filepath.Base(tool.rooms.Folder()) != "rooms" {
		t.Fatalf("View › Preview room… opens on the rooms beside the tilesets: %q", tool.rooms.Folder())
	}
	press(find(t, auto, "files/crypt-tmj"))
	if len(tool.shown.maps) < 2 || len(tool.shown.SolidIn(geom.Rect{X: -5000, Y: -5000, W: 10000, H: 10000})) == 0 {
		t.Fatalf("the room's layers stand in the preview, their walls solid: %d layers", len(tool.shown.maps))
	}
	walls := func() *twod.Tileset {
		for _, m := range tool.shown.maps {
			if filepath.Base(m.Tileset().Source()) == "walls.tsj" {
				return m.Tileset()
			}
		}
		t.Fatal("the room draws walls.tsj")
		return nil
	}
	if !casts(walls(), 1) {
		t.Fatal("walls.tsj's tile 0 casts, as saved")
	}
	auto.Queue(inspect(t, w, auto).prop("shadow").ID, automation.ActSelect, 1)
	press("tile-0")
	if casts(walls(), 1) {
		t.Fatal("walls.tsj's tile 0 made No shadow, the room's tileset reads it at once, so light casts none from it")
	}
	press("view")
	press("view/tile-alone")
	if len(tool.shown.maps) != 1 {
		t.Fatalf("back to the tile alone: %d maps", len(tool.shown.maps))
	}
}

func TestAnEditOfSeveralTilesIsSavedAndReadBack(t *testing.T) {
	path := copyOfWalls(t)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	was, err := parse(before)
	if err != nil {
		t.Fatal(err)
	}
	w, tool, auto, press := driven(t, path)
	press("tile-5")
	in := tool.win.Input()
	in.SetKey(input.KeyLeftShift, true)
	press("tile-6")
	press("tile-7")
	in.SetKey(input.KeyLeftShift, false)
	w.RunFrames(1)
	inspect(t, w, auto).act(inspect(t, w, auto).prop("tall"), automation.ActSet, true)
	inspect(t, w, auto).choose(inspect(t, w, auto).prop("shadow"), "none")
	inspect(t, w, auto).choose(inspect(t, w, auto).prop("occluder_group"), 1)
	addObject(t, w, tool, Rect)
	press("edit")
	press("edit/apply")
	press("file")
	press("file/save")
	if tool.ed.Changed() {
		t.Fatalf("saved: %s", tool.status.Text())
	}

	ts, err := twod.LoadTiledTileset(os.DirFS(filepath.Dir(path)), filepath.Base(path))
	if err != nil {
		t.Fatal(err)
	}
	shapes := len(readTile(tool.ed.doc, 5).Shapes)
	for _, id := range []int{5, 6, 7} {
		n := id + 1
		if g, _ := ts.Value(n, "occluder_group"); !ts.Tall(n) || casts(ts, n) || g.Int() != 1 {
			t.Fatalf("the engine reads Tiled tile %d (its %d) tall, casting no shadow, in group 1", id, n)
		}
		if got := len(readTile(tool.ed.doc, id).Shapes); got != shapes {
			t.Fatalf("tile %d holds the picked tile's %d shapes: %d", id, shapes, got)
		}
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	now, err := parse(b)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range tileIDs(was) {
		if id >= 5 && id <= 7 {
			continue
		}
		if a, b := strip(readTile(was, id)), strip(readTile(now, id)); !reflect.DeepEqual(a, b) {
			t.Fatalf("tile %d is as it was: %+v → %+v", id, a, b)
		}
	}
	again, err := open(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []int{5, 6, 7} {
		if tile := readTile(
			again.doc,
			id,
		); !tile.Tall || !tile.NoShadow || tile.Group != 1 ||
			len(tile.Shapes) != shapes {
			t.Fatalf("a fresh tool reads tile %d back as saved: %+v", id, tile)
		}
	}
}

func TestTheViewMenusLightDebugViewsShowOneAtATimeOverThePreview(t *testing.T) {
	w, tool, auto, press := driven(t, copyOf(t, "props.tsj"))
	w.RunFrames(2)
	press("view")
	press("view/debug-occluders")
	if light.DebugOf(tool.win) != light.DebugOccluders || light.DebugAreaOf(tool.win) != tool.pane.Rect() {
		t.Fatalf(
			"the occluder picture over the preview pane: %v in %v, pane %v",
			light.DebugOf(tool.win),
			light.DebugAreaOf(tool.win),
			tool.pane.Rect(),
		)
	}
	press("view")
	press("view/debug-shadows")
	if occluders := entry(
		t,
		auto,
		press,
		"view/debug-occluders",
	).Value; light.DebugOf(tool.win) != light.DebugShadows ||
		occluders != false {
		t.Fatalf("the shadow maps in its place: %v, occluders still checked %v", light.DebugOf(tool.win), occluders)
	}
	press("view")
	press("view/debug-outlines")
	press("view")
	press("view/debug-shadows")
	if light.DebugOf(tool.win) != light.DebugOff || !light.OutlinesOf(tool.win) {
		t.Fatalf("unchecked, no view, the outlines kept: %v %v", light.DebugOf(tool.win), light.OutlinesOf(tool.win))
	}
}

func TestTheFirstClickOnAnItemOfAMenuOpenedForTheFirstTimeChoosesIt(t *testing.T) {
	w, tool, auto, _ := driven(t, copyOf(t, "props.tsj"))
	in := tool.win.Input()
	click := func(at geom.Vec2) {
		in.SetMouse(at)
		w.RunFrames(1)
		in.SetMouseButton(input.MouseLeft, true)
		w.RunFrames(1)
		in.SetMouseButton(input.MouseLeft, false)
		w.RunFrames(2)
	}
	view := find(t, auto, "view")
	_, els := auto.Snapshot("", "")
	for _, e := range els {
		if e.ID == view {
			click(e.Bounds.Center())
		}
	}
	item, ok := element(auto, "view/debug-occluders")
	if !ok || item.Bounds.H == 0 {
		t.Fatal("a click on View opens its menu")
	}
	click(item.Bounds.Center())
	if light.DebugOf(tool.win) != light.DebugOccluders {
		t.Fatalf("one click on an item far down the menu chooses it: %v", light.DebugOf(tool.win))
	}
}

func TestTheViewMenuBakesThePreviewLampAsAStaticLight(t *testing.T) {
	w, tool, _, press := driven(t, copyOf(t, "props.tsj"))
	w.RunFrames(2)
	press("view")
	press("view/static")
	if !tool.lamp.Static() {
		t.Fatal("View › Static light makes the lamp static")
	}
	press("view")
	press("view/static")
	if tool.lamp.Static() {
		t.Fatal("and again live")
	}
}

func TestThePreviewRoomActionShowsARoomWithoutTheDialog(t *testing.T) {
	path := roomTree(t)
	w, tool, auto, _ := driven(t, path)
	tool.define(auto)
	room := filepath.Join(filepath.Dir(path), "..", "..", "dungeon", "rooms", "crypt.tmj")
	if _, err := auto.Invoke("preview-room", map[string]any{"path": room}); err != nil {
		t.Fatal(err)
	}
	if !tool.rooms.Hidden() || filepath.Base(tool.room) != "crypt.tmj" {
		t.Fatalf("the room previewed, no dialog shown: %q", tool.room)
	}
	w.RunFrames(2)
	if len(tool.shown.maps) < 2 {
		t.Fatalf("its layers stand in the preview: %d", len(tool.shown.maps))
	}
	if _, err := auto.Invoke(
		"preview-room",
		map[string]any{"path": filepath.Join(t.TempDir(), "none.tmj")},
	); err == nil {
		t.Fatal("a room that isn't there is an error")
	}
	if _, err := auto.Invoke("preview-room", nil); err == nil {
		t.Fatal("so is no path")
	}
}

func TestMarkedTilesShowTheirCountAndADashWhereTheyDiffer(t *testing.T) {
	dir := t.TempDir()
	png, err := os.ReadFile("testdata/crypt/tiled/0x72_DungeonTilesetII_v1.7.png")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "0x72_DungeonTilesetII_v1.7.png"), png, 0o644); err != nil {
		t.Fatal(err)
	}
	tsj := `{"type": "tileset", "image": "0x72_DungeonTilesetII_v1.7.png", "imagewidth": 512, "imageheight": 512,
	 "tilewidth": 16, "tileheight": 16, "columns": 32, "tilecount": 1024, "margin": 0, "spacing": 0,
	 "tiles": [
	  {"id": 0, "properties": [{"name": "solid", "type": "bool", "value": true},
	     {"name": "name", "type": "string", "value": "wall"}, {"name": "height", "type": "float", "value": 64}]},
	  {"id": 1, "properties": [{"name": "solid", "type": "bool", "value": true},
	     {"name": "name", "type": "string", "value": "bar"}]}]}`
	path := filepath.Join(dir, "set.tsj")
	if err = os.WriteFile(path, []byte(tsj), 0o644); err != nil {
		t.Fatal(err)
	}
	w, tool, auto, press := driven(t, path)
	press("tile-0")
	in := tool.win.Input()
	in.SetKey(input.KeyLeftShift, true)
	press("tile-1")
	in.SetKey(input.KeyLeftShift, false)
	w.RunFrames(2)
	ins := inspect(t, w, auto)
	if got := ins.must("", "tile/title").Label; got != "2 tiles" {
		t.Fatalf("the header counts the tiles: %q", got)
	}
	if n := ins.prop("height"); value(n) != "" || n.Label != "—" {
		t.Fatalf("heights differ: a dash, %q / %q", value(n), n.Label)
	}
	if value(ins.prop("solid")) != "true" || ins.unset("solid") {
		t.Fatal("both solid: true")
	}
	press("tile-0")
	w.RunFrames(2)
	if got := ins.must("", "tile/title").Label; got != "Tile 0" || value(ins.prop("height")) != "64" {
		t.Fatalf("one tile again: %q, height %q", got, value(ins.prop("height")))
	}
}

func TestTheShapeClassChoiceShowsAndSetsTheSelectedShapesClass(t *testing.T) {
	w, tool, auto, press := driven(t, copyOfWalls(t))
	press("tile-5")
	addObject(t, w, tool, Rect)
	w.RunFrames(2)
	w.RunFrames(2)
	ins := inspect(t, w, auto)
	if got := value(ins.obj("type")); got != "none" {
		t.Fatalf("a new object: its type none: %q", got)
	}
	ins.choose(ins.obj("type"), "body")
	s := tool.ed.tile.Shapes[tool.ed.selected]
	if !s.Body || s.Shadow {
		t.Fatalf("body chosen: %+v", s)
	}
	ins.choose(ins.obj("type"), "shadow")
	if shape := tool.ed.tile.Shapes[tool.ed.selected]; !shape.Shadow || shape.Body {
		t.Fatalf("light chosen: %+v", shape)
	}
}
