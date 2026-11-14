package tiles_test

import (
	"testing"

	"github.com/lantervale/ember/collide/tiles"
	"github.com/lantervale/ember/geom"
)

// BenchmarkAreaFillSolid reads a 128×128 area's solid cells after a change
// to the map, as a flow field does each time it plans again. Beside twod's
// own TileArea, before it went (Apple M1 Max, -count 10, median): twod
// 1.02 ms, tiles 0.97 ms.
func BenchmarkAreaFillSolid(b *testing.B) {
	m := endless(newTileset())
	a := tiles.Of(m).AreaAround(geom.V2(0, 0), 128, 128)
	dst := make([]bool, 128*128)
	a.FillSolid(dst)
	b.ReportAllocs()
	i := 0
	for b.Loop() {
		m.Set(i%64, 0, i%6)
		a.FillSolid(dst)
		i++
	}
}
