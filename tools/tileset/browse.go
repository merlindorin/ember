package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	_ "image/png"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/merlindorin/ember/twod"
)

// tilesetFile is what the file dialog reads of a .tsj: its tiles' size
// and count, and where their pictures are.
type tilesetFile struct {
	TileCount  int    `json:"tilecount"`
	TileWidth  int    `json:"tilewidth"`
	TileHeight int    `json:"tileheight"`
	Columns    int    `json:"columns"`
	Image      string `json:"image"`
	Tiles      []struct {
		Image  string `json:"image"`
		X      int    `json:"x"`
		Y      int    `json:"y"`
		Width  int    `json:"width"`
		Height int    `json:"height"`
	} `json:"tiles"`
}

func readTileset(path string) (tilesetFile, bool) {
	var t tilesetFile
	if !strings.EqualFold(filepath.Ext(path), ".tsj") {
		return t, false
	}
	b, err := os.ReadFile(path)
	return t, err == nil && json.Unmarshal(b, &t) == nil
}

// describeTileset is the file dialog's line about a tileset: "Tileset ·
// 42 tiles · 16×16 · uses walls.png"; "" for another file.
func describeTileset(path string) string {
	t, ok := readTileset(path)
	if !ok {
		return ""
	}
	line := fmt.Sprintf("Tileset · %d tiles · %d×%d", t.TileCount, t.TileWidth, t.TileHeight)
	if pics := t.pictures(); len(pics) > 0 {
		line += " · uses " + strings.Join(pics, ", ")
	}
	return line
}

// pictures are the pictures the tiles are cut from, in order.
func (t tilesetFile) pictures() []string {
	var out []string
	add := func(p string) {
		if p != "" && !slices.Contains(out, p) {
			out = append(out, p)
		}
	}
	add(t.Image)
	for _, tile := range t.Tiles {
		add(tile.Image)
	}
	return out
}

// tilesetPicture is the file dialog's picture of a tileset: its first six
// tiles, three by two; nil for another file or one whose picture can't be
// read.
func tilesetPicture(path string) *twod.Texture {
	if img := tilesetImage(path); img != nil {
		return twod.NewTexture(img)
	}
	return nil
}

// tilesetImage is tilesetPicture's image.
func tilesetImage(path string) image.Image {
	t, ok := readTileset(path)
	if !ok || t.TileWidth <= 0 || t.TileHeight <= 0 {
		return nil
	}
	type cut struct {
		image string
		r     image.Rectangle
	}
	var cuts []cut
	switch {
	case t.Image != "" && t.Columns > 0:
		for i := range min(6, t.TileCount) {
			at := image.Pt(i%t.Columns*t.TileWidth, i/t.Columns*t.TileHeight)
			cuts = append(
				cuts,
				cut{t.Image, image.Rectangle{Min: at, Max: at.Add(image.Pt(t.TileWidth, t.TileHeight))}},
			)
		}
	default:
		for _, tile := range t.Tiles[:min(6, len(t.Tiles))] {
			w, h := tile.Width, tile.Height
			if w == 0 || h == 0 {
				w, h = t.TileWidth, t.TileHeight
			}
			cuts = append(cuts, cut{tile.Image, image.Rect(tile.X, tile.Y, tile.X+w, tile.Y+h)})
		}
	}
	if len(cuts) == 0 {
		return nil
	}
	sources := map[string]image.Image{}
	out := image.NewNRGBA(image.Rect(0, 0, 3*t.TileWidth, 2*t.TileHeight))
	for i, c := range cuts {
		src, cached := sources[c.image]
		if !cached {
			src = decode(filepath.Join(filepath.Dir(path), c.image))
			sources[c.image] = src
		}
		if src == nil {
			return nil
		}
		at := image.Pt(i%3*t.TileWidth, i/3*t.TileHeight)
		draw.Draw(out, image.Rectangle{Min: at, Max: at.Add(c.r.Size())}, src, c.r.Min, draw.Over)
	}
	return out
}

func decode(path string) image.Image {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer func() { _ = f.Close() }()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil
	}
	return img
}
