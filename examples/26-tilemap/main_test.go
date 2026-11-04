package main

import (
	"testing"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/internal/draw"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
)

func TestTheMeadowIsDrawnFromItsPicture(t *testing.T) {
	sheet, err := twod.LoadTexture(files, "tiles.png", twod.Pixelated())
	if err != nil {
		t.Fatal(err)
	}
	tiles := twod.NewTileset(sheet, twod.Cell(16, 16), twod.TileAnimation(water, 2, water, ripple))
	meadow := twod.NewTileMap(tiles, twod.ASCII(level, legend()), twod.CellSize(32, 32))
	if columns, rows := meadow.Columns(), meadow.Rows(); columns != 20 || rows != 12 {
		t.Fatalf("the picture is 20×12 cells: %d×%d", columns, rows)
	}
	if meadow.At(2, 1) != water || meadow.At(14, 3) != floor || meadow.At(13, 2) != wall {
		t.Fatal("each character is its tile")
	}
	win := platform.NewWindow(platform.Size(640, 384))
	win.AppendChild(meadow)
	w := ember.New().AddEntities(win)
	w.RunFrames(1)
	if n := len(draw.FrontFor(w, win).Unindexed()); n != 20*12*6 {
		t.Fatalf("every cell in view is a quad: %d vertices", n)
	}
}
