package light //nolint:testpackage // white-box: shares light's test scene, reaches unexported state

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/merlindorin/ember/collide"
	"github.com/merlindorin/ember/collide/tiles"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/twod"
)

// BenchmarkCPUMirror is what light probes were weighed against (decision
// 0042, performance.md): working out the light at points on the CPU, each
// point testing each light in reach for a shadow with collide.Space.Cast,
// through 10,000 solid cells.
func BenchmarkCPUMirror(b *testing.B) {
	const side = 200
	cells := make([][]int, side)
	r := rand.New(rand.NewPCG(1, 2))
	solid := 0
	for y := range cells {
		cells[y] = make([]int, side)
		for x := range cells[y] {
			if r.IntN(4) == 0 {
				cells[y][x], solid = 1, solid+1
			}
		}
	}
	m := twod.NewTileMap(
		twod.NewTileset(picture(16, 16), twod.Cell(16, 16), tiles.Solid(1)),
		twod.Cells(cells),
		twod.CellSize(16, 16),
	)
	space := collide.NewSpace()
	space.AddSolid(tiles.Of(m))
	lights := make([]geom.Vec2, 64)
	for i := range lights {
		lights[i] = geom.V2(1200+r.Float32()*800, 1200+r.Float32()*450)
	}
	points := make([]geom.Vec2, 100)
	for i := range points {
		points[i] = geom.V2(1200+r.Float32()*800, 1200+r.Float32()*450)
	}
	const radius = 250
	b.ReportMetric(float64(solid), "solids")
	b.ResetTimer()
	for b.Loop() {
		for _, p := range points {
			sum := float32(0)
			for _, l := range lights {
				d := p.Sub(l).Len()
				if d >= radius {
					continue
				}
				if _, hit := space.Cast(l, p); hit {
					continue
				}
				sum += float32(1 - math.Min(1, float64(d/radius)))
			}
			_ = sum
		}
	}
}
