package main

import (
	"bytes"
	"os"
	"slices"
	"testing"
)

func TestTheGamesTilesetsReadAndWriteBackByteForByte(t *testing.T) {
	for _, name := range []string{"walls.tsj", "props.tsj", "floor.tsj", "flags.tsj"} {
		b, err := os.ReadFile("testdata/crypt/tiled/" + name)
		if err != nil {
			t.Fatal(err)
		}
		n, err := parse(b)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if out := n.bytes(bytes.HasSuffix(b, []byte("\n"))); !bytes.Equal(out, b) {
			i := 0
			for i < len(out) && i < len(b) && out[i] == b[i] {
				i++
			}
			t.Fatalf(
				"%s differs at byte %d: %q / %q",
				name,
				i,
				string(out[max(i-40, 0):min(i+40, len(out))]),
				string(b[max(i-40, 0):min(i+40, len(b))]),
			)
		}
	}
}

func walls(t *testing.T) *node {
	t.Helper()
	b, err := os.ReadFile("testdata/crypt/tiled/walls.tsj")
	if err != nil {
		t.Fatal(err)
	}
	n, err := parse(b)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func TestATileReadAndWrittenBackUnchangedLeavesTheFileAsItWas(t *testing.T) {
	doc := walls(t)
	before := doc.bytes(false)
	for _, id := range tileIDs(doc) {
		writeTile(doc, readTile(doc, id))
	}
	if !bytes.Equal(doc.bytes(false), before) {
		t.Fatal("every tile written back as read changes nothing")
	}
}

func TestAShadowShapeAndPropertiesAreWrittenAsTiledReadsThem(t *testing.T) {
	doc := walls(t)
	names := func() []string {
		var out []string
		if props := tileNode(doc, 0).get("properties"); props != nil {
			for _, p := range props.items {
				out = append(out, p.get("name").str())
			}
		}
		return out
	}
	// others are the names this edit leaves alone, in the file's order.
	others := func(list []string) []string {
		return slices.DeleteFunc(
			slices.Clone(list),
			func(n string) bool { return n == "solid" || n == "shadow" || n == "occluder_group" },
		)
	}
	kept := others(names())
	tile := readTile(doc, 0)
	shapes := len(tile.Shapes)
	tile.Shapes = append(
		tile.Shapes,
		Shape{Kind: Rect, X: 0, Y: 0, W: 16, H: 8, Shadow: true},
		Shape{Kind: Ellipse, X: 2, Y: 2, W: 4, H: 4},
	)
	tile.NoShadow, tile.Group = true, 2
	writeTile(doc, tile)
	again := readTile(parseOK(t, doc.bytes(false)), 0)
	if len(again.Shapes) != shapes+2 || !again.Shapes[shapes].Shadow || again.Shapes[shapes+1].Kind != Ellipse ||
		!again.NoShadow ||
		again.Group != 2 {
		t.Fatalf("read back: %+v", again)
	}
	if got := names(); !slices.Equal(others(got), kept) || !slices.Contains(got, "shadow") ||
		!slices.Contains(got, "occluder_group") {
		t.Fatalf("the other properties stay, in order (%v), shadow and occluder_group added: %v", kept, got)
	}
	tile.Shapes, tile.NoShadow, tile.Group, tile.Solid, tile.SolidSet = nil, false, -1, false, false
	writeTile(doc, tile)
	if n := tileNode(doc, 0); n.get("objectgroup") != nil || !slices.Equal(names(), kept) {
		t.Fatalf("no shapes, no collision layer; solid gone, the rest kept: %s", n.bytes(false))
	}
}

func TestANewTileEntryIsMadeInItsPlace(t *testing.T) {
	doc := parseOK(t, []byte(`{"image": "x.png", "tilecount": 4, "tiles": [{"id": 0}, {"id": 3}]}`))
	writeTile(doc, Tile{ID: 2, Group: -1, Shapes: []Shape{{Kind: Rect, W: 4, H: 4, Shadow: true}}})
	tiles := doc.get("tiles").items
	if len(tiles) != 3 || tiles[1].get("id").num() != 2 || tiles[1].get("objectgroup") == nil {
		t.Fatalf("tile 2's entry goes between 0 and 3: %s", doc.bytes(false))
	}
}

func parseOK(t *testing.T, b []byte) *node {
	t.Helper()
	n, err := parse(b)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func TestABodyShapeIsWrittenWithItsClassAndTheToggleStepsThroughAll(t *testing.T) {
	doc := parseOK(t, []byte(`{"image": "x.png", "tilecount": 1, "tiles": [{"id": 0}]}`))
	writeTile(doc, Tile{ID: 0, Group: -1, Shapes: []Shape{{Kind: Rect, Y: 12, W: 16, H: 4, Body: true}}})
	back := readTile(parseOK(t, doc.bytes(false)), 0)
	if len(back.Shapes) != 1 || !back.Shapes[0].Body || back.Shapes[0].Shadow {
		t.Fatalf("read back as a body shape: %+v", back.Shapes)
	}
	o := tileNode(doc, 0).get("objectgroup").get("objects").items[0]
	if o.get("class").str() != "body" && o.get("type").str() != "body" {
		t.Fatalf(`written with class (or, before Tiled 1.9, type) "body": %s`, o.bytes(false))
	}

	ed := blank()
	ed.tile = Tile{ID: 0, Group: -1, Shapes: []Shape{{Kind: Rect, W: 4, H: 4}}}
	ed.selected = 0
	var steps []string
	for range 3 {
		ed.ToggleShadow()
		steps = append(steps, ed.tile.Shapes[0].class())
	}
	if !slices.Equal(steps, []string{"shadow", "body", ""}) {
		t.Fatalf("collision → light only → body only → collision: %q", steps)
	}
}
