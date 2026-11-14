package spatial_test

import (
	"math/rand"
	"slices"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/scene"
	"github.com/lantervale/ember/spatial"
	"github.com/lantervale/ember/world"
)

func TestQueryMatchesBruteForce(t *testing.T) {
	g := spatial.NewGrid(4)
	rng := rand.New(rand.NewSource(1))
	pos := map[uint64]geom.Vec3{}
	for i := uint64(1); i <= 500; i++ {
		p := geom.V3(rng.Float32()*100-50, rng.Float32()*5, rng.Float32()*100-50)
		pos[i] = p
		g.Insert(i, p)
	}
	for i := uint64(1); i <= 100; i++ {
		p := geom.V3(rng.Float32()*100-50, 0, rng.Float32()*100-50)
		pos[i] = p
		g.Move(i, p)
	}
	center, radius := geom.V3(3, 0, -7), float32(12)
	var want []uint64
	for id, p := range pos {
		if p.XZ().Dist(center.XZ()) <= radius {
			want = append(want, id)
		}
	}
	got := g.Collect(nil, center, radius)
	slices.Sort(want)
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Fatalf("got %d ids, want %d", len(got), len(want))
	}
}

func TestEarlyStopAndRemove(t *testing.T) {
	g := spatial.NewGrid(2)
	for i := uint64(1); i <= 10; i++ {
		g.Insert(i, geom.V3(float32(i)*0.1, 0, 0))
	}
	n := 0
	g.Query(geom.Zero3, 5, func(uint64, geom.Vec3) bool { n++; return n < 3 })
	if n != 3 {
		t.Fatalf("visited %d", n)
	}
	g.Remove(5)
	g.Remove(99)
	if g.Len() != 9 {
		t.Fatalf("len %d", g.Len())
	}
	if _, ok := g.Position(5); ok {
		t.Fatal("removed id still indexed")
	}
}

func TestNearestWithFilter(t *testing.T) {
	g := spatial.NewGrid(4)
	g.Insert(1, geom.V3(1, 0, 0))
	g.Insert(2, geom.V3(2, 0, 0))
	g.Insert(3, geom.V3(-50, 0, 0))
	if id, ok := g.Nearest(geom.Zero3, 10, nil); !ok || id != 1 {
		t.Fatalf("nearest %v %v", id, ok)
	}
	if id, ok := g.Nearest(geom.Zero3, 10, func(id uint64) bool { return id != 1 }); !ok || id != 2 {
		t.Fatalf("filtered nearest %v", id)
	}
	if _, ok := g.Nearest(geom.V3(100, 0, 100), 5, nil); ok {
		t.Fatal("nothing should be found")
	}
}

func TestNegativeCoordinates(t *testing.T) {
	g := spatial.NewGrid(4)
	g.Insert(1, geom.V3(-0.5, 0, -0.5))
	if got := g.Collect(nil, geom.V3(0.5, 0, 0.5), 2); len(got) != 1 {
		t.Fatalf("got %v", got)
	}
}

type thing struct {
	*world.BasicEntity
	scene.Transform
	spatial.Spatial
}

func TestIndexerTracksEntities(t *testing.T) {
	w := world.NewWorld()
	idx := spatial.NewIndex(4)
	w.AddSystem(spatial.NewIndexer(idx))
	a := &thing{world.NewBasic(), scene.At(1, 0, 1), spatial.Spatial{}}
	w.AddEntity(a)
	if idx.Len() != 1 {
		t.Fatal("entity not indexed on add")
	}
	a.Position = geom.V3(30, 0, 30)
	w.Update(0)
	if p, _ := idx.Position(a.ID()); p.X != 30 {
		t.Fatalf("position not synced: %v", p)
	}
	w.RemoveEntity(a)
	if idx.Len() != 0 {
		t.Fatal("entity not removed from index")
	}
}

func TestFixedIndexerFollowsLocalPositionsEachStep(t *testing.T) {
	a := world.New()
	idx := spatial.NewIndex(4)
	a.AddSystems(&scene.TransformSystem{}, spatial.NewIndexer(idx, spatial.Fixed()))
	e := &thing{world.NewBasic(), scene.At(1, 0, 1), spatial.Spatial{}}
	a.AddEntities(e)
	a.RunFrames(1)
	e.Position = geom.V3(30, 0, 30)
	step := a.Time().FixedDelta
	a.Frame(step / 4)
	if p, _ := idx.Position(e.ID()); p.X != 1 {
		t.Fatalf("indexed %v before a fixed step", p)
	}
	a.Frame(step)
	if p, _ := idx.Position(e.ID()); p.X != 30 {
		t.Fatalf("indexed %v, want the local position after a fixed step", p)
	}
}

func BenchmarkQuery(b *testing.B) {
	g := spatial.NewGrid(8)
	rng := rand.New(rand.NewSource(1))
	for i := uint64(0); i < 20000; i++ {
		g.Insert(i, geom.V3(rng.Float32()*1000, 0, rng.Float32()*1000))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.Query(geom.V3(500, 0, 500), 20, func(uint64, geom.Vec3) bool { return true })
	}
}
