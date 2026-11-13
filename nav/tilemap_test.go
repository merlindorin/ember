package nav_test

import (
	"image"
	"testing"

	"github.com/merlindorin/ember/collide/tiles"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/nav"
	"github.com/merlindorin/ember/twod"
)

var _ nav.Grid = (*tiles.Layer)(nil)

func TestATileMapIsAGridWhoseDoorsReplan(t *testing.T) {
	ts := twod.NewTileset(twod.NewTexture(image.NewRGBA(image.Rect(0, 0, 16, 8))), twod.Cell(8, 8), tiles.Solid(2))
	level := twod.NewTileMap(ts, twod.Cells([][]int{
		{1, 1, 1, 1, 1},
		{2, 2, 2, 2, 1},
		{1, 1, 1, 1, 1},
	}), twod.CellSize(10, 10))
	grid := tiles.Of(level)
	if !grid.Solid(0, 1) || grid.Solid(0, 0) || !grid.Solid(-1, 0) {
		t.Fatal("walls and outside are solid")
	}
	m := nav.New(grid)
	route := m.Route(level.Center(0, 0), level.Center(0, 2))
	if !route.Reaches() || len(route.Waypoints()) < 2 {
		t.Fatalf("round the wall's end: %v", route.Waypoints())
	}
	before := grid.Version()
	level.Set(0, 1, 1)
	if grid.Version() == before {
		t.Fatal("setting a cell changes the version")
	}
	if dir := route.Toward(level.Center(0, 0), 1); dir != geom.V2(0, 1) {
		t.Fatalf("the door opened: straight down now, %v via %v", dir, route.Waypoints())
	}
}

func TestARoomOfThinBarsSendsWhoIsOutsideRoundToItsDoor(t *testing.T) {
	ts := twod.NewTileset(twod.NewTexture(image.NewRGBA(image.Rect(0, 0, 24, 8))), twod.Cell(8, 8),
		tiles.Collision(2, geom.Rect{W: 2, H: 8}), tiles.Collision(3, geom.Rect{X: 6, W: 2, H: 8}), tiles.Solid(4))
	level := twod.NewTileMap(ts, twod.Cells([][]int{
		{1, 1, 1, 1, 1, 1, 1},
		{1, 2, 4, 4, 4, 3, 1},
		{1, 2, 1, 1, 1, 3, 1},
		{1, 2, 1, 1, 1, 3, 1},
		{1, 2, 4, 1, 4, 3, 1},
		{1, 1, 1, 1, 1, 1, 1},
	}), twod.CellSize(10, 10))
	grid := tiles.Of(level)
	if !grid.Solid(1, 2) || !grid.Solid(5, 2) {
		t.Fatal("a thin bar's cell blocks paths, though only its bar stops bodies")
	}
	path, ok := nav.New(grid).Find(level.Center(0, 2), level.Center(3, 2))
	if !ok {
		t.Fatal("the way in is the door")
	}
	door := false
	from := level.Center(0, 2)
	for _, p := range path {
		for k := float32(0); k <= 1; k += 0.05 {
			x, y, _ := level.Cell(from.Add(p.Sub(from).Scale(k)))
			if grid.Solid(x, y) {
				t.Fatalf("never into a bar, at cell %d,%d on %v", x, y, path)
			}
			door = door || x == 3 && y == 4
		}
		from = p
	}
	if !door {
		t.Fatalf("round to the door: %v", path)
	}
}
