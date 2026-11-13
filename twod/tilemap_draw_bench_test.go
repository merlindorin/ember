package twod //nolint:testpackage // white-box: draws into the canvas's own list

import (
	"image"
	stdcolor "image/color"
	"testing"
)

// drawBenchMap is a 40×109 map of 32-point cells of a 16-pixel water and
// stone tileset, three in four cells water.
func drawBenchMap() *TileMap {
	img := image.NewNRGBA(image.Rect(0, 0, 32, 16))
	for y := range 16 {
		for x := range 32 {
			c := stdcolor.NRGBA{0x68, 0x6f, 0x99, 0xff}
			if x >= 16 {
				c = stdcolor.NRGBA{0x80, 0x80, 0x80, 0xff}
			}
			img.SetNRGBA(x, y, c)
		}
	}
	grid := make([][]int, 109)
	for y := range grid {
		grid[y] = make([]int, 40)
		for x := range grid[y] {
			grid[y][x] = 1 + int(min(uint(x*73856093^y*19349663)%4, 1))
		}
	}
	return NewTileMap(NewTileset(NewTexture(img), Cell(16, 16)), Cells(grid), CellSize(32, 32))
}

// BenchmarkTileMapDraw is a frame's draw of a map in an 800×600 view: what
// a frame costs a level without variants. Compare it with an older tree at
// one bench time, interleaved:
//
//	go test -run xxx -bench TileMapDraw -benchtime 2s -count 5 ./twod
func BenchmarkTileMapDraw(b *testing.B) {
	w, _, _ := app()
	c := CanvasFor(w, nil)
	m := drawBenchMap()
	b.ReportAllocs()
	for b.Loop() {
		c.list.Reset(0)
		m.Draw(c)
	}
}
