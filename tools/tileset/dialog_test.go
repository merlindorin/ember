package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/automation"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/ui/align"
)

func TestWithoutAFileTheToolWaitsThenOpensOneAndSavesItAsAnother(t *testing.T) {
	path := copyOfWalls(t)
	dir := filepath.Dir(path)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	copied := filepath.Join(dir, "walls-copy.tsj")
	t.Setenv("EMBER_TILESET_DIR", t.TempDir())
	auto := automation.New(automation.Recording())
	w, tool, err := build("", ember.Automation(auto))
	if err != nil {
		t.Fatal(err)
	}
	press := func(id string) {
		t.Helper()
		auto.Queue(find(t, auto, id), automation.ActPress, nil)
		w.RunFrames(3)
	}
	tool.state.Folder = dir
	w.RunFrames(3)
	if !tool.dialog.Hidden() {
		t.Fatal("no file: the tool waits, the dialog closed")
	}
	if entry(t, auto, press, "file/save").Enabled || !tool.workspace.Hidden() {
		t.Fatal("and Save and the editing controls are off")
	}
	press("file")
	press("file/open")
	if tool.dialog.Hidden() || tool.dialog.Folder() != dir || tool.dialog.Window() == nil {
		t.Fatalf("File › Open… shows the dialog in its own window, in the folder last used: %q", tool.dialog.Folder())
	}
	press("files/walls-tsj")
	if !entry(t, auto, press, "file/save").Enabled || tool.workspace.Hidden() {
		t.Fatal("a tileset loaded, they are on")
	}
	if tool.ed.path != path || !tool.dialog.Hidden() {
		t.Fatalf("a click on walls.tsj opens it: %q", tool.ed.path)
	}

	press("tile-5")
	addObject(t, w, tool, Rect)
	press("file")
	press("file/save-as")
	if tool.saver.Hidden() || !tool.saver.Saving() {
		t.Fatal("File › Save as… shows the dialog to save")
	}
	auto.Queue(find(t, auto, "file-name"), automation.ActSet, "walls-copy")
	w.RunFrames(2)
	press("ok")
	savedACopy(t, tool, path, copied, before)

	press("file")
	press("file/recent-1")
	if tool.ed.path != path {
		t.Fatalf("File › the recent one, the copy open being left out, opens walls.tsj again: %q", tool.ed.path)
	}
	file := entries(t, auto, press, "file")
	if len(file) < 3 || leafOf(file[0]) != "open" || leafOf(file[1]) != "recent-1" || leafOf(file[2]) == "recent-2" ||
		file[1].Label != filepath.Base(copied)+"  ·  "+filepath.Base(filepath.Dir(copied)) {
		t.Fatal("File lists Open…, then the recent tilesets but the one open")
	}
	if leafOf(file[len(file)-1]) != "quit" {
		t.Fatal("and Quit last")
	}
}

// savedACopy checks Save as wrote copied with tile 5's new shape, left the
// file at path as before, and made the copy the tool's file and the most
// recent.
func savedACopy(t *testing.T, tool *Tool, path, copied string, before []byte) {
	t.Helper()
	was, _ := parse(before)
	b, err := os.ReadFile(copied)
	if err != nil {
		t.Fatal(err)
	}
	now, err := parse(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(readTile(now, 5).Shapes) != len(readTile(was, 5).Shapes)+1 {
		t.Fatal("the copy holds the new shape")
	}
	if after, _ := os.ReadFile(path); string(after) != string(before) {
		t.Fatal("and the file opened is as it was")
	}
	if tool.ed.path != copied || tool.ed.Changed() {
		t.Fatalf("the tool edits the copy now, saved: %q %v", tool.ed.path, tool.ed.Changed())
	}
	var s state
	b, _ = os.ReadFile(filepath.Join(os.Getenv("EMBER_TILESET_DIR"), "state.json"))
	if err = json.Unmarshal(b, &s); err != nil || len(s.Recent) != 2 || s.Recent[0] != copied || s.Recent[1] != path {
		t.Fatalf("both are recent, the copy first: %+v", s)
	}
}

func TestOpeningAnotherFileAsksAboutUnsavedChanges(t *testing.T) {
	path := copyOfWalls(t)
	w, tool, _, press := driven(t, path)
	press("tile-5")
	addObject(t, w, tool, Rect)
	press("file")
	press("file/open")
	if !tool.dialog.Hidden() || tool.prompt.panel.Hidden() {
		t.Fatal("with changes, Open… asks first")
	}
	press(tool.prompt.cancel.Path())
	if !tool.dialog.Hidden() || !tool.ed.Changed() {
		t.Fatal("Cancel stays, the changes kept")
	}
	press("file")
	press("file/open")
	press(tool.prompt.discard.Path())
	if tool.dialog.Hidden() {
		t.Fatal("Discard goes on to the file dialog")
	}
}

func TestSaveAsElsewhereKeepsThePicturesFound(t *testing.T) {
	path := copyOfWalls(t)
	e, err := open(path)
	if err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(t.TempDir(), "maps", "walls.tsj")
	if err = os.MkdirAll(filepath.Dir(other), 0o755); err != nil {
		t.Fatal(err)
	}
	if err = e.SaveAs(other); err != nil {
		t.Fatal(err)
	}
	moved, err := open(other)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = twod.LoadTiledTileset(moved.fs()); err != nil {
		t.Fatalf("the picture's path follows the file: %v", err)
	}
}

func TestDialogsOpenInTheirOwnWindowUnlessToldAndTheToolRemembers(t *testing.T) {
	path := copyOfWalls(t)
	_, tool, auto, press := driven(t, path)
	press("file")
	press("file/open")
	if tool.dialog.Window() == nil {
		t.Fatal("by default the dialog opens in a window of its own")
	}
	press("cancel")
	if tool.dialog.Window() != nil {
		t.Fatal("and closes it")
	}
	if entry(t, auto, press, "view/own-window").Value != true {
		t.Fatal("View › Dialogs in their own window starts checked")
	}
	press("view")
	press("view/own-window")
	press("file")
	press("file/open")
	if tool.dialog.Window() != nil || tool.dialog.Hidden() {
		t.Fatal("unchecked, it shows inside the main window")
	}
	if again := loadState(); !again.Inline {
		t.Fatalf("the choice is remembered: %+v", again)
	}
	nextAuto := automation.New(automation.Recording())
	nw, next, err := build(path, ember.Automation(nextAuto))
	if err != nil {
		t.Fatal(err)
	}
	nw.RunFrames(2)
	pressNext := func(id string) {
		t.Helper()
		nextAuto.Queue(find(t, nextAuto, id), automation.ActPress, nil)
		nw.RunFrames(3)
	}
	if next.dialog.OwnWindow() || entry(t, nextAuto, pressNext, "view/own-window").Value != false {
		t.Fatal("and the next run keeps dialogs inside")
	}
}

func TestTheEmptyStartListsTheRecentTilesetsAndOneClickOpensOne(t *testing.T) {
	path := copyOfWalls(t)
	t.Setenv("EMBER_TILESET_DIR", t.TempDir())
	if err := (state{Recent: []string{path}}).save(); err != nil {
		t.Fatal(err)
	}
	auto := automation.New(automation.Recording())
	w, tool, err := build("", ember.Automation(auto))
	if err != nil {
		t.Fatal(err)
	}
	w.RunFrames(2)
	if _, ok := element(auto, "start/start-open"); !ok || !tool.workspace.Hidden() {
		t.Fatal("nothing open: the start card alone, no panels")
	}
	if row, ok := element(auto, "start/recent/recent-row-1"); !ok || row.Label != "walls.tsj" {
		t.Fatal("the card lists the recent walls.tsj")
	}
	if faults := align.Check(tool.dialogs); len(faults) > 0 {
		t.Fatalf("the start card lines up: %v", faults)
	}
	auto.Queue(find(t, auto, "recent-row-1"), automation.ActPress, nil)
	w.RunFrames(3)
	if tool.ed.path != path || tool.tmap == nil {
		t.Fatalf("one click on the recent walls.tsj opens it: %q", tool.ed.path)
	}
	if _, ok := element(auto, "start/start-open"); ok || tool.workspace.Hidden() {
		t.Fatal("open: the panels, no start card")
	}
	if faults := align.Check(tool.dialogs); len(faults) > 0 {
		t.Fatalf("the workspace lines up: %v", faults)
	}
}

func TestReopeningTheDialogLeavesNoRecentItemsBehindInTheWindow(t *testing.T) {
	path := copyOfWalls(t)
	b, err := os.ReadFile(copyOf(t, "props.tsj"))
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(filepath.Dir(path), "props.tsj"), b, 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, auto, press := driven(t, path)
	for _, next := range []string{"files/props-tsj", "files/walls-tsj", "files/props-tsj"} {
		press("file")
		press("file/open")
		press(find(t, auto, next))
	}
	file := entries(t, auto, press, "file")
	recent := 0
	for _, e := range file {
		if strings.HasPrefix(leafOf(e), "recent-") {
			recent++
			if e.Bounds.Y <= file[0].Bounds.Y || e.Bounds.H == 0 {
				t.Fatalf("the recent item is laid out in the menu, under Open…: %v", e.Bounds)
			}
		}
	}
	if recent != 1 || len(file) != 5 {
		t.Fatalf(
			"File holds Open…, the other recent tileset (not the one open), then the rest, none left behind: "+
				"%d entries, %d recent",
			len(file),
			recent,
		)
	}
}

func TestTestsNeverWriteThePlayersOwnState(t *testing.T) {
	t.Setenv("EMBER_TILESET_DIR", "")
	if home() != "" {
		t.Fatalf("under go test, without EMBER_TILESET_DIR, the tool keeps nothing: %q", home())
	}
}

func TestATilesetOpenedFromTheCommandLineShowsItsFirstTilesProperties(t *testing.T) {
	path := copyOf(t, "props.tsj")
	e, err := open(path)
	if err != nil {
		t.Fatal(err)
	}
	want := readTile(e.doc, 0)
	w, _, auto, _ := driven(t, path)
	in := inspect(t, w, auto)
	if value(in.prop("solid")) != fmt.Sprint(want.Solid) || value(in.prop("tall")) != fmt.Sprint(want.Tall) {
		t.Fatalf("props.tsj's tile 0 shows its solid %v and tall %v from the start", want.Solid, want.Tall)
	}
}

func TestTheToolsLayoutFollowsTheWindow(t *testing.T) {
	w, tool, _, _ := driven(t, copyOfWalls(t))
	tool.previewPanel.SetCollapsed(false)
	if m := tool.win.MinSize(); m.X < 1100 || m.Y < 700 {
		t.Fatalf("the window keeps at least about 1100×700: %v", m)
	}
	for _, size := range []geom.Vec2{{X: 1100, Y: 700}, {X: 1800, Y: 1000}} {
		tool.win.SetSize(size)
		w.RunFrames(3)
		for name, r := range map[string]geom.Rect{
			"tiles":   tool.grid.Rect(),
			"editor":  tool.editor.Rect(),
			"preview": tool.pane.Rect(),
			"status":  tool.status.Rect(),
		} {
			if r.X < 0 || r.Y < 0 || r.X+r.W > size.X+0.5 || r.Y+r.H > size.Y+0.5 {
				t.Fatalf("at %v, %s %v runs past the window", size, name, r)
			}
		}
		if g, e := tool.grid.Rect(), tool.status.Rect(); g.Y+g.H < size.Y-20 || e.Y+e.H < size.Y-60 {
			t.Fatalf(
				"at %v, the tiles %v and the editor's column, to its status %v, reach the window's foot",
				size,
				g,
				e,
			)
		}
		if e := tool.editor.Rect(); e.H < 300 {
			t.Fatalf("at %v, the editor keeps room: %v", size, e)
		}
		if p := tool.pane.Rect(); p.X+p.W < size.X-40 {
			t.Fatalf("at %v, the preview %v reaches the right edge, but for the inspector's scroll bar", size, p)
		}
	}
}

func TestPickingATileSetsItsPropertiesAndTheGroupIsThreeChoices(t *testing.T) {
	w, tool, auto, press := driven(t, copyOf(t, "props.tsj"))
	other := -1
	for _, id := range tool.ed.ids {
		if tile := readTile(tool.ed.doc, id); !tile.Solid && !tile.Tall {
			other = id
			break
		}
	}
	if other < 0 {
		t.Skip("props.tsj has no plain tile")
	}
	press(fmt.Sprintf("tile-%d", other))
	in := inspect(t, w, auto)
	for _, name := range []string{"solid", "tall"} {
		if value(in.prop(name)) != "false" || !in.unset(name) ||
			in.must("properties", name+"-default").Label != "(default)" {
			t.Fatalf("picking a plain tile shows %s off, its default, saying so", name)
		}
	}
	if g := in.prop(
		"occluder_group",
	); tool.ed.tile.Group < 0 &&
		(!in.unset("occluder_group") || g.Label != "0 (default)") {
		t.Fatalf("no group set: the default, 0, shows dimmed: %v %q", g.Value, g.Label)
	}
	in.choose(in.prop("occluder_group"), 2)
	if tool.ed.tile.Group != 2 || value(in.prop("occluder_group")) != "2" || in.unset("occluder_group") {
		t.Fatalf("the third choice is occluder group 2, set: %d", tool.ed.tile.Group)
	}
}

func TestNewShapesStepAsideAndDeleteSelectsTheNext(t *testing.T) {
	w, tool, auto, press := driven(t, copyOfWalls(t))
	in := inspect(t, w, auto)
	press("tile-5")
	ed := tool.ed
	n := len(ed.tile.Shapes)
	addObject(t, w, tool, Rect)
	addObject(t, w, tool, Rect)
	a, b := ed.tile.Shapes[n].Bounds(), ed.tile.Shapes[n+1].Bounds()
	if a == b || ed.selected != n+1 {
		t.Fatalf("a second shape steps aside from the first, and is selected: %v %v %d", a, b, ed.selected)
	}
	press("delete")
	if ed.selected != n {
		t.Fatalf("Delete selects the shape left in its place: %d", ed.selected)
	}
	ed.selected = -1
	press("tile-5")
	if in.must("", "delete").Enabled || entry(t, auto, press, "edit/delete-shape").Enabled {
		t.Fatal("with nothing selected, Delete is off")
	}
	for i := range tool.ed.tile.Shapes {
		if _, folded := in.line(i + 1); !folded {
			t.Fatalf("and every object is a folded line: object %d isn't", i+1)
		}
	}
}

func TestAFileChangedOnDiskOffersReloadOverwriteOrCancel(t *testing.T) {
	path := copyOfWalls(t)
	w, tool, _, press := driven(t, path)
	press("tile-5")
	addObject(t, w, tool, Rect)
	theirs := []byte("{}\n")
	if err := os.WriteFile(path, theirs, 0o644); err != nil {
		t.Fatal(err)
	}
	press("file")
	press("file/open")
	press(tool.prompt.save.Path())
	if tool.conflict.panel.Hidden() {
		t.Fatal("saving over a file changed on disk asks what to do")
	}
	press(tool.conflict.overwrite.Path())
	b, _ := os.ReadFile(path)
	if string(b) == string(theirs) || tool.ed.Changed() {
		t.Fatal("Overwrite writes the tool's over it")
	}
	if tool.dialog.Hidden() {
		t.Fatal("then goes on to what waited on the save: the open dialog")
	}
	if err := os.WriteFile(path, theirs, 0o644); err != nil {
		t.Fatal(err)
	}
	addObject(t, w, tool, Rect)
	press("file")
	press("file/save")
	press(tool.conflict.reload.Path())
	if disk, _ := os.ReadFile(path); string(disk) != string(theirs) || tool.ed.Changed() {
		t.Fatal("Reload takes the file as it is on disk, the changes dropped")
	}
}

func TestSaveAsHasItsOwnIDsAndTheEmptyPreviewShowsNothing(t *testing.T) {
	_, _, auto, press := driven(t, copyOfWalls(t))
	press("file")
	press("file/save-as")
	if id := find(t, auto, "file-name"); !strings.HasPrefix(id, "save/") {
		t.Fatalf("the Save as dialog's IDs say save: %s", id)
	}
	t.Setenv("EMBER_TILESET_DIR", t.TempDir())
	w, tool, err := build("", ember.Automation(automation.New(automation.Recording())))
	if err != nil {
		t.Fatal(err)
	}
	w.RunFrames(3)
	if !tool.stage.Hidden() || !tool.over.Hidden() {
		t.Fatal("with nothing loaded, the preview shows only its hint")
	}
}
