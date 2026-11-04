package twod_test

import (
	"image"
	"testing"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/twod"
)

func TestRoomIsWhereABodyFitsInACell(t *testing.T) {
	// Tile 1 is a bar 2 pixels wide on its cell's left, tile 2 solid whole,
	// tile 3 a strip reaching 2 pixels past its cell's right into the next.
	tiles := twod.NewTileset(twod.NewTexture(image.NewRGBA(image.Rect(0, 0, 24, 8))), twod.Cell(8, 8),
		twod.Collision(1, geom.Rect{W: 2, H: 8}), twod.Solid(2), twod.Collision(3, geom.Rect{X: 4, W: 6, H: 8}))
	m := twod.NewInfiniteMap(tiles, twod.ChunkSize(8), twod.CellSize(16, 16),
		twod.Generator(func(cx, cy int, cells [][]int) {
			if cx == 0 && cy == 0 {
				cells[1][1], cells[1][3], cells[1][5] = 1, 2, 3
			}
		}))
	a := m.Area(0, 0, 8, 8)
	body := geom.V2(8, 6)
	if at, ok := a.Room(0, 0, body); !ok || at != a.Center(0, 0) {
		t.Fatalf("an empty cell: its middle, %v %v", at, ok)
	}
	at, ok := a.Room(1, 1, body)
	if !ok || at.X-body.X/2 < 16+4-0.01 {
		t.Fatalf("beside the bar, clear of its 4 points: %v %v", at, ok)
	}
	if _, fits := a.Room(3, 1, body); fits {
		t.Fatal("a solid tile leaves no room")
	}
	if moved, fits := a.Room(6, 1, body); !fits || moved.X-body.X/2 < 6*16+4-0.01 {
		t.Fatalf("a neighbour's shape reaching in counts: %v %v", moved, ok)
	}
}

func TestAShapeReachingPastItsCellIsFoundFromTheNextCell(t *testing.T) {
	// Tile 1's shape reaches 4 pixels below its cell, tile 2's 3 past its
	// left side.
	tiles := twod.NewTileset(twod.NewTexture(image.NewRGBA(image.Rect(0, 0, 16, 8))), twod.Cell(8, 8),
		twod.Collision(1, geom.Rect{W: 8, H: 12}), twod.Collision(2, geom.Rect{X: -3, W: 4, H: 8}))
	m := twod.NewTileMap(tiles, twod.Cells([][]int{{0, 1, 0, 0}, {0, 0, 0, 2}, {0, 0, 0, 0}}), twod.CellSize(16, 16))
	below := m.SolidIn(geom.Rect{X: 16, Y: 16, W: 16, H: 16})
	if len(below) != 1 || below[0].Y+below[0].H != 24 {
		t.Fatalf("the shape reaching 8 points into the cell below: %v", below)
	}
	left := m.SolidIn(geom.Rect{X: 32, Y: 16, W: 16, H: 16})
	if len(left) != 1 || left[0].X != 42 {
		t.Fatalf("the shape reaching 6 points into the cell on its left: %v", left)
	}
}
