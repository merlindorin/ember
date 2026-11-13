package light //nolint:testpackage // white-box: times light's own map cache

import (
	"testing"

	"github.com/merlindorin/ember/collide/tiles"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/twod"
)

// BenchmarkTileMapShadows is the shadows of one light's reach, 600×600
// points, as light reads them, on the map twod's BenchmarkTileMap
// times ShadowsIn on: 40×109 cells of 32 points, a third walls, half of
// them shaped. Compare them at one bench time,
//
//	go test -run xxx -bench 'TileMap' -benchtime 2s -count 5 ./twod ./twod/light
func BenchmarkTileMapShadows(b *testing.B) {
	const columns, rows, wall, shaped = 40, 109, 2, 3
	tile := func(x, y int) int {
		switch uint(x*73856093^y*19349663) % 6 {
		case 0:
			return wall
		case 1:
			return shaped
		}
		return 1
	}
	ts := twod.NewTileset(nil, twod.Cell(16, 16), tiles.Solid(wall, shaped),
		tiles.Collision(shaped, geom.Rect{X: 0, Y: 8, W: 16, H: 8}))
	grid := make([][]int, rows)
	for y := range grid {
		grid[y] = make([]int, columns)
		for x := range grid[y] {
			grid[y][x] = tile(x, y)
		}
	}
	level := twod.NewTileMap(ts, twod.Cells(grid), twod.CellSize(32, 32))
	endless := twod.NewInfiniteMap(ts, twod.ChunkSize(16), twod.CellSize(32, 32),
		twod.Generator(func(cx, cy int, cells [][]int) {
			for y := range cells {
				for x := range cells[y] {
					cells[y][x] = tile(cx*16+x, cy*16+y)
				}
			}
		}))
	reach := make([]geom.Rect, 16)
	w, h := float32(columns*32-600), float32(rows*32-600)
	for i := range reach {
		f := float32(i) / float32(len(reach))
		reach[i] = geom.Rect{X: w * f, Y: h * (1 - f), W: 600, H: 600}
	}
	b.Run("Set-alone", func(b *testing.B) {
		b.ReportAllocs()
		i := 0
		for b.Loop() {
			level.Set(i%columns, i*7%rows, tile(i, i))
			i++
		}
	})
	b.Run("Set-then-TileMap", func(b *testing.B) {
		c := &mapCache{}
		var dst []shadow
		b.ReportAllocs()
		i := 0
		for b.Loop() {
			level.Set(i%columns, i*7%rows, tile(i, i))
			dst = c.shadowsIn(dst[:0], level, reach[i%len(reach)])
			i++
		}
	})
	for _, m := range []struct {
		name string
		m    tileSource
	}{{"TileMap", level}, {"InfiniteMap", endless}} {
		b.Run(m.name, func(b *testing.B) {
			c := &mapCache{}
			var dst []shadow
			for _, r := range reach {
				dst = c.shadowsIn(dst[:0], m.m, r)
			}
			b.ReportAllocs()
			i := 0
			for b.Loop() {
				dst = c.shadowsIn(dst[:0], m.m, reach[i%len(reach)])
				i++
			}
		})
	}
}
