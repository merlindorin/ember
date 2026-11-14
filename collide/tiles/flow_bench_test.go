package tiles_test

import (
	"image"
	"math/rand/v2"
	"testing"

	"github.com/lantervale/ember/collide/tiles"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/nav"
	"github.com/lantervale/ember/twod"
)

// meadow fills chunks with floor (1), a few walls (2) and trees (3) from
// seed.
func meadow(seed uint64) func(cx, cy int, cells [][]int) {
	return func(cx, cy int, cells [][]int) {
		r := rand.New(rand.NewPCG(seed, uint64(cx)<<32|uint64(uint32(cy))))
		for y := range cells {
			for x := range cells[y] {
				cells[y][x] = []int{1, 1, 1, 1, 1, 1, 2, 3}[r.IntN(8)]
			}
		}
	}
}

// BenchmarkFlowFieldOverAnInfiniteMap is a horde's field over a meadow, its
// target jumping between two places 5,000 points apart each time, so each
// SetTarget cuts a new 128×128 window and plans it whole. It was twod's
// own, over InfiniteMap.AreaAround, until twod's collision moved here.
func BenchmarkFlowFieldOverAnInfiniteMap(b *testing.B) {
	ts := twod.NewTileset(twod.NewTexture(image.NewRGBA(image.Rect(0, 0, 32, 16))), twod.Cell(8, 8),
		tiles.Solid(2), twod.Tall(3))
	m := twod.NewInfiniteMap(ts, twod.ChunkSize(16), twod.CellSize(32, 32), twod.Generator(meadow(1)),
		twod.KeepChunks(256))
	field := nav.NewFlowField(nil, nav.Around(tiles.Of(m).AreaAround, 128, 128))
	b.ReportAllocs()
	i := 0
	for b.Loop() {
		field.SetTarget(geom.V2(float32(i%2)*5000, 0))
		i++
	}
}
