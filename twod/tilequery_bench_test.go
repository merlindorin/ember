package twod_test

import (
	"testing"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/twod"
)

// The questions collision, paths and light ask a tile map every frame, on
// a temple-sized map: 40×109 cells of 32 points, a third of them walls,
// half of those with a collision shape. They are the gate of moving tile
// properties out of twod (design/tiled-maps.md §6): compare before and
// after on one machine at a fixed bench time,
//
//	go test -run xxx -bench TileMap -benchtime 2s -count 5 ./twod

const (
	benchColumns, benchRows = 40, 109
	benchCell               = 32
	wall, shaped            = 2, 3
)

// benchTile is cell x, y's tile: a third walls, half of them shaped.
func benchTile(x, y int) int {
	switch uint(x*73856093^y*19349663) % 6 {
	case 0:
		return wall
	case 1:
		return shaped
	}
	return 1
}

func benchTileset() *twod.Tileset {
	return twod.NewTileset(nil, twod.Cell(16, 16), twod.Solid(wall, shaped),
		twod.Collision(shaped, geom.Rect{X: 0, Y: 8, W: 16, H: 8}))
}

func benchTileMap() *twod.TileMap {
	grid := make([][]int, benchRows)
	for y := range grid {
		grid[y] = make([]int, benchColumns)
		for x := range grid[y] {
			grid[y][x] = benchTile(x, y)
		}
	}
	return twod.NewTileMap(benchTileset(), twod.Cells(grid), twod.CellSize(benchCell, benchCell))
}

// benchInfiniteMap is the same cells, endless, every chunk the benches
// read already made.
func benchInfiniteMap() *twod.InfiniteMap {
	m := twod.NewInfiniteMap(benchTileset(), twod.ChunkSize(16), twod.KeepChunks(64),
		twod.CellSize(benchCell, benchCell), twod.Generator(func(cx, cy int, cells [][]int) {
			for y := range cells {
				for x := range cells[y] {
					cells[y][x] = benchTile(cx*16+x, cy*16+y)
				}
			}
		}))
	m.SolidIn(geom.Rect{W: benchColumns * benchCell, H: benchRows * benchCell})
	return m
}

// areas are n squares of side points, spread over the map.
func areas(n int, side float32) []geom.Rect {
	out := make([]geom.Rect, n)
	w, h := float32(benchColumns*benchCell)-side, float32(benchRows*benchCell)-side
	for i := range out {
		f := float32(i) / float32(n)
		out[i] = geom.Rect{X: w * f, Y: h * (1 - f), W: side, H: side}
	}
	return out
}

//nolint:gochecknoglobals // keeps the compiler from dropping the work
var (
	sinkRects   []geom.Rect
	sinkShadows []twod.ShadowRect
	sinkSolid   bool
)

type tileQueries interface {
	SolidIn(area geom.Rect) []geom.Rect
	ShadowsIn(area geom.Rect) []twod.ShadowRect
	Solid(x, y int) bool
}

func benchQueries(b *testing.B, m tileQueries) {
	b.Run("SolidIn-body", func(b *testing.B) {
		around := areas(64, 48)
		b.ReportAllocs()
		i := 0
		for b.Loop() {
			sinkRects = m.SolidIn(around[i%len(around)])
			i++
		}
	})
	b.Run("ShadowsIn-light", func(b *testing.B) {
		reach := areas(16, 600)
		b.ReportAllocs()
		i := 0
		for b.Loop() {
			sinkShadows = m.ShadowsIn(reach[i%len(reach)])
			i++
		}
	})
	b.Run("Solid-cell", func(b *testing.B) {
		b.ReportAllocs()
		i := 0
		for b.Loop() {
			sinkSolid = m.Solid(i%benchColumns, i*7%benchRows)
			i++
		}
	})
}

func BenchmarkTileMap(b *testing.B) { benchQueries(b, benchTileMap()) }

func BenchmarkTileMapInfinite(b *testing.B) { benchQueries(b, benchInfiniteMap()) }
