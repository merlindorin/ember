package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/merlindorin/ember/automation"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/world"
)

// fromFixture is the fixture's tileset name, with the picture it shares, in a
// temporary folder.
func fromFixture(t *testing.T, name string) string {
	t.Helper()
	dir := t.TempDir()
	for _, n := range []string{name, "0x72_DungeonTilesetII_v1.7.png"} {
		b, err := os.ReadFile(filepath.Join("testdata", "crypt", "tiled", n))
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(dir, n), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return filepath.Join(dir, name)
}

// tiledFile is what a test reads of a Tiled tileset.
type tiledFile struct {
	Tiles []tiledTile `json:"tiles"`
}

type tiledTile struct {
	ID         int `json:"id"`
	Properties []struct {
		Name  string `json:"name"`
		Type  string `json:"type"`
		Value any    `json:"value"`
	} `json:"properties"`
	Animation []struct {
		TileID   int `json:"tileid"`
		Duration int `json:"duration"`
	} `json:"animation"`
	ObjectGroup *struct {
		Objects []tiledObject `json:"objects"`
	} `json:"objectgroup"`
}

type tiledObject struct {
	ID      int     `json:"id"`
	Type    string  `json:"type"`
	Class   string  `json:"class"`
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
	Width   float64 `json:"width"`
	Height  float64 `json:"height"`
	Ellipse bool    `json:"ellipse"`
	Polygon []struct {
		X float64 `json:"x"`
		Y float64 `json:"y"`
	} `json:"polygon"`
}

func readTiled(t *testing.T, path string) tiledFile {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var f tiledFile
	if err = json.Unmarshal(b, &f); err != nil {
		t.Fatal(err)
	}
	return f
}

func number(v float64) string { return strconv.FormatFloat(v, 'g', -1, 32) }

func TestTheInspectorMirrorsEveryDemoTileAsItsFileHasIt(t *testing.T) {
	for _, name := range []string{"walls.tsj", "props.tsj", "floor.tsj", "flags.tsj"} {
		t.Run(name, func(t *testing.T) {
			path := fromFixture(t, name)
			file := readTiled(t, path)
			w, tool, auto, _ := driven(t, path)
			in := inspect(t, w, auto)
			for _, tile := range file.Tiles {
				tool.pick(tile.ID)
				w.RunFrames(2)
				if got := in.must("", "tile/title").Label; got != fmt.Sprintf("Tile %d", tile.ID) {
					t.Fatalf("the header: %q", got)
				}
				mirrorsProperties(t, in, tile)
				mirrorsAnimation(t, in, tile)
				mirrorsObjects(t, w, tool, in, tile)
			}
		})
	}
}

// mirrorsProperties checks the inspector shows each engine property of tile
// as its file has it, or its default dimmed.
func mirrorsProperties(t *testing.T, in inspector, tile tiledTile) {
	t.Helper()
	set := map[string]any{}
	for _, p := range tile.Properties {
		set[p.Name] = p.Value
	}
	for _, d := range engineProperties() {
		v, ok := set[d.Name]
		want := fmt.Sprint(d.Default)
		if ok {
			want = fmt.Sprint(v)
		}
		e := in.prop(d.Name)
		unset := in.unset(d.Name)
		got := value(e)
		switch {
		case unset && e.Role == automation.RoleCombo:
			if e.Label != want+" (default)" {
				t.Fatalf("tile %d %s: unset shows %q, want %q dimmed", tile.ID, d.Name, e.Label, want+" (default)")
			}
			got = want
		case unset && e.Role != automation.RoleToggle:
			got = want
		}
		if got != want || unset == ok {
			t.Fatalf("tile %d %s: shows %q (unset %v), the file %q (set %v)", tile.ID, d.Name, got, unset, want, ok)
		}
	}
}

func mirrorsAnimation(t *testing.T, in inspector, tile tiledTile) {
	t.Helper()
	anim := "none"
	if len(tile.Animation) > 0 {
		var frames []string
		for _, f := range tile.Animation {
			frames = append(frames, fmt.Sprintf("%d · %d ms", f.TileID, f.Duration))
		}
		anim = strings.Join(frames, ", ")
	}
	if got := in.prop("animation").Label; got != anim {
		t.Fatalf("tile %d animation: %q, the file %q", tile.ID, got, anim)
	}
}

// mirrorsObjects selects each of tile's objects in turn and checks the
// selected one shows unfolded and the others folded, as the file has them.
func mirrorsObjects(t *testing.T, w *world.World, tool *Tool, in inspector, tile tiledTile) {
	t.Helper()
	var objects int
	if tile.ObjectGroup != nil {
		objects = len(tile.ObjectGroup.Objects)
	}
	if len(tool.ed.tile.Shapes) != objects {
		t.Fatalf("tile %d: %d objects read, %d in the file", tile.ID, len(tool.ed.tile.Shapes), objects)
	}
	for sel := range objects {
		tool.ed.selected = sel
		w.RunFrames(2)
		if _, folded := in.line(sel + 1); folded {
			t.Fatalf("tile %d: object %d, selected, is unfolded", tile.ID, sel+1)
		}
		for i, o := range tile.ObjectGroup.Objects {
			if i != sel {
				mirrorsFolded(t, in, tile.ID, i, o)
				continue
			}
			mirrorsUnfolded(t, in, tile.ID, i, o)
		}
	}
}

// objectWords are o's type, shape and polygon points as the inspector
// writes them.
func objectWords(o tiledObject) (string, string, []string) {
	typ := o.Type
	if o.Class != "" {
		typ = o.Class
	}
	if typ == "" {
		typ = "none"
	}
	kind := "rectangle"
	var pts []string
	switch {
	case o.Ellipse:
		kind = "ellipse"
	case o.Polygon != nil:
		kind = "polygon"
		for _, p := range o.Polygon {
			pts = append(pts, number(p.X)+","+number(p.Y))
		}
	}
	return typ, kind, pts
}

func mirrorsFolded(t *testing.T, in inspector, id, i int, o tiledObject) {
	t.Helper()
	typ, kind, pts := objectWords(o)
	where := fmt.Sprintf("%s,%s · %s×%s", number(o.X), number(o.Y), number(o.Width), number(o.Height))
	if kind == "polygon" {
		where = fmt.Sprintf("%s,%s · %d points", number(o.X), number(o.Y), len(pts))
	}
	line, folded := in.line(i + 1)
	if got, want := line.Label, strings.Join([]string{typ, kind, where}, " · "); !folded || got != want {
		t.Fatalf("tile %d object %d folded: %q, the file %q", id, i+1, got, want)
	}
}

func mirrorsUnfolded(t *testing.T, in inspector, id, i int, o tiledObject) {
	t.Helper()
	typ, kind, pts := objectWords(o)
	got := []string{
		in.title(),
		value(in.obj("type")),
		value(in.obj("x")),
		value(in.obj("y")),
		value(in.obj("width")),
		value(in.obj("height")),
		value(in.obj("shape")),
	}
	want := []string{
		fmt.Sprintf("▾ object %d", o.ID),
		typ,
		number(o.X),
		number(o.Y),
		number(o.Width),
		number(o.Height),
		kind,
	}
	if pe, shows := in.find("objects", "points"); shows != (kind == "polygon") ||
		shows && value(pe) != strings.Join(pts, " ") {
		t.Fatalf(
			"tile %d object %d: its points show for a polygon only, as the file has them: %v %q",
			id,
			i+1,
			shows,
			value(pe),
		)
	}
	if !slices.Equal(got, want) {
		t.Fatalf("tile %d object %d unfolded: shows %q, the file %q", id, i+1, got, want)
	}
}

// objectKey is the value key has in tile id's object i, as written.
//
//nolint:unparam // the tile is named at each call, as in the file
func objectKey(t *testing.T, path string, id, i int, key string) any {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		Tiles []struct {
			ID          int `json:"id"`
			ObjectGroup struct {
				Objects []map[string]any `json:"objects"`
			} `json:"objectgroup"`
		} `json:"tiles"`
	}
	if err = json.Unmarshal(b, &f); err != nil {
		t.Fatal(err)
	}
	for _, tile := range f.Tiles {
		if tile.ID == id {
			return tile.ObjectGroup.Objects[i][key]
		}
	}
	return nil
}

func TestEditingAnObjectsFieldWritesThatKeyAlone(t *testing.T) {
	path := fromFixture(t, "walls.tsj")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	w, tool, auto, press := driven(t, path)
	press("tile-12")
	w.RunFrames(2)
	tool.ed.selected = 0
	w.RunFrames(2)
	in := inspect(t, w, auto)
	old := objectKey(t, path, 12, 0, "width")
	in.typeIn(in.obj("width"), "6")
	press("file")
	press("file/save")
	if got := objectKey(t, path, 12, 0, "width"); got != 6.0 || old == 6.0 {
		t.Fatalf("width written: %v, was %v", got, old)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	a, b := bytes.Split(before, []byte("\n")), bytes.Split(after, []byte("\n"))
	if len(a) != len(b) {
		t.Fatalf("one line changes, none added: %d lines, then %d", len(a), len(b))
	}
	changed := 0
	for i := range a {
		if !bytes.Equal(a[i], b[i]) {
			changed++
			if !bytes.Contains(b[i], []byte(`"width"`)) {
				t.Fatalf("only the width line changes: %q", b[i])
			}
		}
	}
	if changed != 1 {
		t.Fatalf("one line changes: %d", changed)
	}
}

func TestGamePropertiesStayByteForByteThroughAnEdit(t *testing.T) {
	path := fromFixture(t, "props.tsj")
	game := func() [][]byte {
		e, err := open(path)
		if err != nil {
			t.Fatal(err)
		}
		var out [][]byte
		for _, id := range e.ids {
			n := tileNode(e.doc, id)
			if n == nil || n.get("properties") == nil {
				continue
			}
			for _, p := range n.get("properties").items {
				if _, ok := definition(
					p.get("name").str(),
				); ok &&
					slices.ContainsFunc(
						engineProperties(),
						func(d twod.TileProperty) bool { return d.Name == p.get("name").str() },
					) {
					continue
				}
				out = append(out, p.bytes(false))
			}
		}
		return out
	}
	before := game()
	if len(before) == 0 {
		t.Skip("props.tsj holds no game property")
	}
	w, _, auto, press := driven(t, path)
	press("tile-2")
	in := inspect(t, w, auto)
	in.choose(in.prop("shadow"), "none")
	in.typeIn(in.prop("height"), "40")
	press("file")
	press("file/save")
	after := game()
	if !slices.EqualFunc(before, after, bytes.Equal) {
		t.Fatalf("the game's properties are written back as they were: %d before, %d after", len(before), len(after))
	}
}

func TestAnObjectsShapeAndPointsAreEditable(t *testing.T) {
	path := fromFixture(t, "walls.tsj")
	w, tool, auto, press := driven(t, path)
	press("tile-12")
	w.RunFrames(2)
	tool.ed.selected = 0
	w.RunFrames(2)
	in := inspect(t, w, auto)
	in.choose(in.obj("shape"), "polygon")
	if tool.ed.tile.Shapes[0].Kind != Polygon || len(tool.ed.tile.Shapes[0].Points) != 4 {
		t.Fatalf("a rectangle made a polygon takes its corners: %+v", tool.ed.tile.Shapes[0])
	}
	in.typeIn(in.obj("points"), "0,0 8,0 4,6")
	press("file")
	press("file/save")
	pts, ok := objectKey(t, path, 12, 0, "polygon").([]any)
	if !ok || len(pts) != 3 || objectKey(t, path, 12, 0, "width") != 0.0 {
		t.Fatalf("its three points are written as Tiled's polygon, width 0: %v", objectKey(t, path, 12, 0, "polygon"))
	}
	in.typeIn(in.obj("points"), "0,0 8,0")
	if !strings.Contains(tool.status.Text(), "three points") {
		t.Fatalf("two points are refused, saying why: %q", tool.status.Text())
	}
}

// tileProperty is the value of property name of tile id, as written.
//
//nolint:unparam // the tile is named at each call, as in the file
func tileProperty(t *testing.T, path string, id int, name string) (any, bool) {
	t.Helper()
	for _, tile := range readTiled(t, path).Tiles {
		if tile.ID != id {
			continue
		}
		for _, p := range tile.Properties {
			if p.Name == name {
				return p.Value, true
			}
		}
	}
	return nil, false
}

func TestEveryPopUpWritesItsExactKey(t *testing.T) {
	path := fromFixture(t, "walls.tsj")
	w, tool, auto, press := driven(t, path)
	press("tile-27")
	in := inspect(t, w, auto)
	for _, name := range []string{"solid", "tall"} {
		in.press(in.prop(name))
	}
	for name, item := range map[string]string{"shadow": "none", "occluder_group": "2"} {
		in.choose(in.prop(name), item)
	}
	press("file")
	press("file/save")
	for name, want := range map[string]any{"solid": true, "tall": true, "shadow": "none", "occluder_group": 2.0} {
		if got, ok := tileProperty(t, path, 27, name); !ok || got != want {
			t.Fatalf("%s written as %v (%v), want %v", name, got, ok, want)
		}
	}

	press("tile-12")
	tool.ed.selected = 0
	w.RunFrames(2)
	in.choose(in.obj("type"), "shadow")
	in.choose(in.obj("shape"), "ellipse")
	press("file")
	press("file/save")
	typ, _ := objectKey(t, path, 12, 0, "type").(string)
	class, _ := objectKey(t, path, 12, 0, "class").(string)
	if typ != "shadow" && class != "shadow" || objectKey(t, path, 12, 0, "ellipse") != true {
		t.Fatalf(
			"an object's type writes its type (or class), its shape ellipse: type %q class %q ellipse %v",
			typ,
			class,
			objectKey(t, path, 12, 0, "ellipse"),
		)
	}
}

func TestAnObjectSelectedInTheTableIsSelectedOnTheCanvasAndBack(t *testing.T) {
	w, tool, auto, press := driven(t, fromFixture(t, "walls.tsj"))
	in := inspect(t, w, auto)
	press("tile-12")
	tool.ed.selected = 0
	w.RunFrames(2)
	if len(tool.ed.tile.Shapes) < 2 {
		t.Skip("walls tile 12 has one object")
	}
	in.press(in.obj("object-2"))
	if tool.ed.selected != 1 {
		t.Fatalf("a click on object 2's line selects it, as the canvas shows it: %d", tool.ed.selected)
	}
	w.RunFrames(2)
	if _, folded := in.line(2); folded {
		t.Fatal("and unfolds it")
	}
	tool.ed.selected = 0
	w.RunFrames(2)
	if _, folded := in.line(1); folded {
		t.Fatal("selected on the canvas, object 1 unfolds in the table")
	}
	if _, folded := in.line(2); !folded {
		t.Fatal("and object 2 folds back to a line")
	}
}

func TestTheShadowsSwitchTurnsThePreviewLampsShadowsOffAndOn(t *testing.T) {
	w, tool, auto, _ := driven(t, fromFixture(t, "walls.tsj"))
	tool.previewPanel.SetCollapsed(false)
	w.RunFrames(2)
	if !tool.lamp.Shadows() {
		t.Fatal("the lamp casts shadows by default")
	}
	auto.Queue(tool.checks["shadows"].Path(), automation.ActPress, nil)
	w.RunFrames(3)
	if tool.lamp.Shadows() {
		t.Fatal("switched off, the lamp casts none")
	}
	auto.Queue(tool.checks["shadows"].Path(), automation.ActPress, nil)
	w.RunFrames(3)
	if !tool.lamp.Shadows() {
		t.Fatal("and on again")
	}
}

func TestTheInspectorNeverOverflowsSideways(t *testing.T) {
	for _, name := range []string{"walls.tsj", "props.tsj", "floor.tsj", "flags.tsj"} {
		t.Run(name, func(t *testing.T) {
			path := fromFixture(t, name)
			file := readTiled(t, path)
			w, tool, auto, _ := driven(t, path)
			for _, size := range []geom.Vec2{{X: 1100, Y: 700}, {X: 1280, Y: 800}} {
				tool.win.SetSize(size)
				for _, folded := range []bool{true, false} {
					tool.previewPanel.SetCollapsed(folded)
					for _, tile := range file.Tiles {
						tool.pick(tile.ID)
						if tile.ObjectGroup != nil {
							tool.ed.selected = 0
						}
						w.RunFrames(2)
						rootID, in := inspect(t, w, auto).root(tool)
						_, els := auto.Snapshot("", "")
						for _, e := range els {
							if !strings.HasPrefix(e.ID, rootID+"/") || e.Bounds.W <= 0 {
								continue
							}
							if right := e.Bounds.X + e.Bounds.W; right > in.X+in.W+0.5 {
								t.Fatalf(
									"at %v, preview folded %v, tile %d: %s reaches %v, past the inspector's %v",
									size,
									folded,
									tile.ID,
									e.ID,
									right,
									in.X+in.W,
								)
							}
						}
					}
				}
			}
		})
	}
}

func TestABoolStaysUnsetUntilChangedThenWritesTrueOrFalse(t *testing.T) {
	path := fromFixture(t, "walls.tsj")
	if _, ok := tileProperty(t, path, 27, "tall"); ok {
		t.Skip("walls tile 27 sets tall")
	}
	w, tool, auto, press := driven(t, path)
	press("tile-27")
	in := inspect(t, w, auto)
	save := func() {
		press("file")
		press("file/save")
	}
	in.press(in.prop("tall"))
	save()
	if v, ok := tileProperty(t, path, 27, "tall"); !ok || v != true {
		t.Fatalf("switched on, tall is written true: %v %v", v, ok)
	}
	if in.unset("tall") {
		t.Fatal("set, it no longer says (default)")
	}
	in.press(in.prop("tall"))
	save()
	if v, ok := tileProperty(t, path, 27, "tall"); !ok || v != false {
		t.Fatalf("switched off again, tall is written false, set, not taken out: %v %v", v, ok)
	}
	if _, ok := tileProperty(t, path, 27, "solid"); ok != tool.ed.tile.SolidSet {
		t.Fatal("solid, never touched, is written only as the file had it")
	}
}
