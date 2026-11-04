package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/merlindorin/ember/automation"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/twod/light"
)

func TestTheTilePanelEditsTheDeclaredPropertiesAndTheEngineReadsThem(t *testing.T) {
	path := copyOfWalls(t)
	w, tool, auto, press := driven(t, path)
	press("tile-5")
	in := inspect(t, w, auto)
	in.typeIn(in.prop("height"), "24")
	in.choose(in.prop("shadow"), "none")
	in.typeIn(in.prop("emissive"), "bright")
	if !strings.Contains(tool.status.Text(), "not a number") {
		t.Fatalf("a word for a float property is refused, saying why: %q", tool.status.Text())
	}
	press("file")
	press("file/save")
	ts, err := twod.LoadTiledTileset(os.DirFS(filepath.Dir(path)), filepath.Base(path))
	if err != nil {
		t.Fatal(err)
	}
	if ts.Height(6) != 24 || ts.CastShadow(6) {
		t.Fatalf("Tiled tile 5 reads height 24 and shadow none: %v %v", ts.Height(6), ts.CastShadow(6))
	}
	if got := value(in.prop("height")); got != "24" {
		t.Fatalf("the field shows it: %q", got)
	}
	in.typeIn(in.prop("height"), "0")
	if _, ok := tool.ed.tile.Values["height"]; ok {
		t.Fatal("its default takes it out of the file")
	}
}

func TestAMisspelledPropertyIsFlaggedInThePanel(t *testing.T) {
	path := copyOfWalls(t)
	e, err := open(path)
	if err != nil {
		t.Fatal(err)
	}
	n := tileNode(e.doc, 0)
	p := &node{object: true}
	p.set("name", scalar("occluder_grup"))
	p.set("type", scalar("int"))
	p.set("value", num(1))
	n.get("properties").items = append(n.get("properties").items, p)
	if err = os.WriteFile(path, e.doc.bytes(e.newline), 0o644); err != nil {
		t.Fatal(err)
	}
	w, _, auto, _ := driven(t, path)
	w.RunFrames(2)
	if e, shows := inspect(
		t,
		w,
		auto,
	).find("properties", "near-misses"); !shows ||
		!strings.Contains(e.Label, "occluder_grup = 1  (occluder_group?)") {
		t.Fatalf("the unknown property shows, read only, flagged: %q", e.Label)
	}
}

func TestAGamesPropertiesComeFromItsPropsFile(t *testing.T) {
	file := filepath.Join(t.TempDir(), "game.json")
	if err := os.WriteFile(
		file,
		[]byte(`[{"name": "loot", "kind": "enum", "values": ["none", "gold"], "doc": "what it drops"},
		{"name": "hp", "kind": "int", "default": 3, "min": 1, "max": 9}]`),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	defs, err := readProperties(file)
	if err != nil {
		t.Fatal(err)
	}
	if len(defs) != 2 || defs[0].Kind != twod.KindEnum || defs[0].Default != "none" || defs[1].Default != 3 ||
		defs[1].Max != 9 {
		t.Fatalf("read as declared, an enum defaulting to its first value: %+v", defs)
	}
	if err = os.WriteFile(file, []byte(`[{"name": "x", "kind": "vector"}]`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err = readProperties(file); err == nil || !strings.Contains(err.Error(), "vector") {
		t.Fatalf("an unknown kind is an error naming it: %v", err)
	}
	if got := string(twod.TiledProject(defs...)); !strings.Contains(got, `"name": "loot"`) {
		t.Fatal("the Tiled project carries the game's properties")
	}
}

func TestTheLampsElevationSizeAndFadeAndTheAutomaticHeightShow(t *testing.T) {
	w, tool, auto, press := driven(t, copyOf(t, "props.tsj"))
	tool.previewPanel.SetCollapsed(false)
	in := inspect(t, w, auto)
	w.RunFrames(1)
	if got := in.prop("height").Label; got != "48 (picture)" {
		t.Fatalf("a tall column's blank height reads what the engine uses: %q", got)
	}
	auto.Queue(tool.sliders["elevation"].Path(), automation.ActSet, 120)
	auto.Queue(tool.sliders["size"].Path(), automation.ActSet, 6)
	auto.Queue(part{tool.dialogs, tool.fades[2]}.Path(), automation.ActPress, nil)
	w.RunFrames(3)
	if tool.lamp.Elevation() != 120 || tool.lamp.Size() != 6 || tool.lamp.ShadowFade() != light.FadeSmooth {
		t.Fatalf(
			"the preview lamp takes them: %v %v %v",
			tool.lamp.Elevation(),
			tool.lamp.Size(),
			tool.lamp.ShadowFade(),
		)
	}
	for _, id := range tool.ed.ids {
		if tile := readTile(tool.ed.doc, id); !tile.Tall && !tile.Solid && len(tile.Shapes) == 0 {
			press(fmt.Sprintf("tile-%d", id))
			if got := in.prop("height").Label; got != "0 (flat)" {
				t.Fatalf("a flat tile's reads 0: %q", got)
			}
			return
		}
	}
}

func TestATilesStoredHeightShowsInItsField(t *testing.T) {
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
	 "tiles": [{"id": 0, "properties": [{"name": "solid", "type": "bool", "value": true},
	    {"name": "height", "type": "float", "value": 64}]}]}`
	path := filepath.Join(dir, "set.tsj")
	if err = os.WriteFile(path, []byte(tsj), 0o644); err != nil {
		t.Fatal(err)
	}
	w, _, auto, press := driven(t, path)
	press("tile-0")
	w.RunFrames(2)
	if e := inspect(t, w, auto).prop("height"); value(e) != "64" {
		t.Fatalf("a height of 64 in the file shows 64, not the placeholder: %q (placeholder %q)", value(e), e.Label)
	}
}
