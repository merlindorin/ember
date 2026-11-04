package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTheFileDialogDescribesAndPicturesATileset(t *testing.T) {
	path := copyOfWalls(t)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		TileCount  int `json:"tilecount"`
		TileWidth  int `json:"tilewidth"`
		TileHeight int `json:"tileheight"`
	}
	if err = json.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	want := fmt.Sprintf("Tileset · %d tiles · %d×%d · uses ", doc.TileCount, doc.TileWidth, doc.TileHeight)
	if line := describeTileset(
		path,
	); !strings.HasPrefix(line, want) ||
		!strings.Contains(line, "0x72_DungeonTilesetII_v1.7.png") {
		t.Fatalf("the line reads its count, size and picture: %q", line)
	}
	pic := tilesetPicture(path)
	if pic == nil {
		t.Fatal("a tileset has a picture")
	}
	if size := pic.Size(); int(size.X) != 3*doc.TileWidth || int(size.Y) != 2*doc.TileHeight {
		t.Fatalf("six tiles, three by two: %v", size)
	}

	png := filepath.Join(filepath.Dir(path), "0x72_DungeonTilesetII_v1.7.png")
	if describeTileset(png) != "" || tilesetPicture(png) != nil {
		t.Fatal("another file gets the dialog's own line and no picture")
	}
	broken := filepath.Join(t.TempDir(), "broken.tsj")
	if err = os.WriteFile(
		broken,
		[]byte(`{"tilecount": 1, "tilewidth": 16, "tileheight": 16, "tiles": [{"image": "missing.png"}]}`),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	if tilesetPicture(broken) != nil {
		t.Fatal("a picture that can't be read gives none")
	}
}
