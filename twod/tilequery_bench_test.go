package twod_test

import (
	"testing"

	"github.com/merlindorin/ember/collide/tiles"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/twod"
)

// The questions collision and paths ask a tile map every frame, on a
// temple-sized map: 40×109 cells of 32 points, a third of them walls, half
// of those with a collision shape, asked of its collide/tiles layer; light's
// are twod/light's BenchmarkTileMapShadows, on the same map. They are the
// gate of moving tile properties out of twod (design/tiled-maps.md §6, decision
// 0071), against BenchmarkTileMap before the move, on one machine at a fixed
// bench time,
//
//	go test -run xxx -bench TileMap -benchtime 2s -count 5 ./twod ./twod/light

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
	return twod.NewTileset(nil, twod.Cell(16, 16), tiles.Solid(wall, shaped),
		tiles.Collision(shaped, geom.Rect{X: 0, Y: 8, W: 16, H: 8}))
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
	m.Fill(0, 0, benchColumns, benchRows, make([]int, benchColumns*benchRows))
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
	sinkRects []geom.Rect
	sinkSolid bool
)

// benchLayer asks collision's and paths' questions of the map's layer in
// collide/tiles, which moves them out of twod: SolidIn as a collide.Solid
// lists, AppendSolidIn as a collide.Space asks, into a slice it keeps, and
// Solid as nav does.
func benchLayer(b *testing.B, l *tiles.Layer) {
	b.Run("SolidIn-body", func(b *testing.B) {
		around := areas(64, 48)
		b.ReportAllocs()
		i := 0
		for b.Loop() {
			sinkRects = l.SolidIn(around[i%len(around)])
			i++
		}
	})
	b.Run("AppendSolidIn-body", func(b *testing.B) {
		around := areas(64, 48)
		dst := make([]geom.Rect, 0, 64)
		b.ReportAllocs()
		i := 0
		for b.Loop() {
			dst = l.AppendSolidIn(dst[:0], around[i%len(around)])
			i++
		}
		sinkRects = dst
	})
	b.Run("Solid-cell", func(b *testing.B) {
		b.ReportAllocs()
		i := 0
		for b.Loop() {
			sinkSolid = l.Solid(i%benchColumns, i*7%benchRows)
			i++
		}
	})
}

func BenchmarkTileMapLayer(b *testing.B) { benchLayer(b, tiles.Of(benchTileMap())) }

func BenchmarkTileMapInfiniteLayer(b *testing.B) { benchLayer(b, tiles.Of(benchInfiniteMap())) }
