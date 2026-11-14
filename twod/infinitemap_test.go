package twod //nolint:testpackage // white-box: shares twod's test helpers, reaches unexported state

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// meadow fills chunks with floor (1), a few walls (2) and trees (3) from
// seed, and counts the chunks it generated.
func meadow(seed uint64, made *int) func(cx, cy int, cells [][]int) {
	return func(cx, cy int, cells [][]int) {
		*made++
		r := rand.New(rand.NewPCG(seed, uint64(cx)<<32|uint64(uint32(cy))))
		for y := range cells {
			for x := range cells[y] {
				cells[y][x] = []int{1, 1, 1, 1, 1, 1, 2, 3}[r.IntN(8)]
			}
		}
	}
}

func floor(_, _ int, cells [][]int) {
	for y := range cells {
		for x := range cells[y] {
			cells[y][x] = 1
		}
	}
}

func TestChunksAreTheSameForASeedAndComeBackTheSame(t *testing.T) {
	var made, again int
	a := NewInfiniteMap(tiles(), ChunkSize(8), Generator(meadow(7, &made)), KeepChunks(2))
	b := NewInfiniteMap(tiles(), ChunkSize(8), Generator(meadow(7, &again)))
	first := a.At(-3, 5)
	for x := -20; x < 20; x++ {
		assert.Equal(t, b.At(x, 5), a.At(x, 5), "same seed, same cells, cell %d", x)
	}
	assert.LessOrEqual(t, a.Chunks(), 2, "past KeepChunks, the least recently used are forgotten")
	assert.Equal(t, first, a.At(-3, 5), "a forgotten chunk comes back the same")
	assert.Greater(t, made, 5, "chunk -1 was generated again")

	other := NewInfiniteMap(tiles(), ChunkSize(8), Generator(meadow(8, &again)))
	differ := false
	for x := range 64 {
		differ = differ || other.At(x, 0) != b.At(x, 0)
	}
	assert.True(t, differ, "another seed, another meadow")
}

func TestNegativeCellsFallInTheChunksBeforeZero(t *testing.T) {
	m := NewInfiniteMap(tiles(), ChunkSize(16))
	k, x, y := m.split(-1, -16)
	assert.Equal(t, chunkKey{-1, -1}, k)
	assert.Equal(t, [2]int{15, 0}, [2]int{x, y})
	k, x, y = m.split(-17, 16)
	assert.Equal(t, chunkKey{-2, 1}, k)
	assert.Equal(t, [2]int{15, 0}, [2]int{x, y})
	m2 := NewInfiniteMap(tiles(), CellSize(32, 32), At(10, 0))
	cx, cy := m2.Cell(geom.V2(10-1, -1))
	assert.Equal(t, [2]int{-1, -1}, [2]int{cx, cy})
	assert.Equal(t, geom.V2(10-16, -16), m2.Center(-1, -1))
}

func TestChunksFollowTheCameraAndTheFarOnesAreForgotten(t *testing.T) {
	w, win, c := app()
	m := NewInfiniteMap(tiles(), ChunkSize(8), CellSize(16, 16), Generator(floor), KeepChunks(40))
	cam := NewCamera()
	win.Add(m, cam)
	w.AddEntities(m, cam)
	w.RunFrames(1)
	// An 800×600 view over chunks of 128 points spans 8 by 6 of them, 10
	// by 8 with the ring around it: more than KeepChunks, all kept.
	const needed = 10 * 8
	start := len(m.chunks)
	assert.Contains(t, m.chunks, chunkKey{0, 0})
	assert.Equal(t, needed, start, "what the view and its ring need, however few KeepChunks allows")
	cells := len(c.Unindexed()) / 6
	assert.LessOrEqual(t, cells, 51*39, "only the cells in view are drawn")
	assert.Greater(t, cells, 50*37)

	cam.SetPosition(geom.V2(20000, 0))
	w.RunFrames(1)
	assert.LessOrEqual(
		t,
		m.Chunks(),
		needed,
		"memory stays bounded: the chunks left behind are forgotten past KeepChunks",
	)
	assert.NotContains(t, m.chunks, chunkKey{0, 0}, "the chunks left behind are forgotten")
	k, _, _ := m.split(20000/16, 0)
	assert.Contains(t, m.chunks, k)
	for _, ring := range []chunkKey{{k.x + 4, 0}, {k.x - 4, 0}} {
		assert.Contains(t, m.chunks, ring, "the ring just outside the view is ready too")
	}
	assert.Equal(t, cells, len(c.Unindexed())/6, "the same view, the same cells")
}

func TestAViewNeedingMoreThanKeepChunksKeepsThemAll(t *testing.T) {
	w, win, c := app()
	m := NewInfiniteMap(tiles(), ChunkSize(4), CellSize(16, 16), Generator(floor), KeepChunks(1))
	win.Add(m)
	w.AddEntities(m)
	w.RunFrames(2)
	assert.Greater(t, len(c.Unindexed())/6, 50*37, "the view drew whole")
	kept := m.Chunks()
	w.RunFrames(3)
	assert.Equal(t, kept, m.Chunks(), "the view's chunks, kept frame to frame, and no more")
}

func TestEditedChunksAreKept(t *testing.T) {
	var made int
	m := NewInfiniteMap(tiles(), ChunkSize(4), Generator(meadow(3, &made)), KeepChunks(1))
	m.Set(1, 1, 2)
	v := m.Version()
	for x := range 40 {
		m.At(x*4, 100)
	}
	assert.Equal(t, 2, m.At(1, 1), "a Set cell survives any number of other chunks")
	assert.Equal(t, v, m.Version())
	assert.Contains(t, m.chunks, chunkKey{0, 0})
}

func TestTallTilesStandAndLeaveWithTheirChunk(t *testing.T) {
	sorted := NewLayer(0, SortByY())
	trees := NewInfiniteMap(tiles(), ChunkSize(2), CellSize(32, 32), At(100, 0), TallIn(sorted), KeepChunks(1),
		Generator(func(cx, _ int, cells [][]int) {
			if cx == 0 {
				cells[1][0] = 3
			}
		}))
	trees.At(0, 0)
	require.Len(t, trees.TallSprites(), 1)
	s := trees.TallSprites()[0]
	assert.Equal(t, geom.V2(100+16, 2*32), s.Position(), "bottom centre of its cell, in TallIn's space")
	assert.Equal(t, sorted.GetSceneNode(), s.GetSceneNode().ParentNode())
	trees.At(10, 0)
	assert.Empty(t, trees.TallSprites())
	assert.Nil(t, s.GetSceneNode().ParentNode(), "a forgotten chunk takes its trees")
	trees.Set(4, 1, 3)
	assert.Len(t, trees.TallSprites(), 1, "Set stands a tree too")
}

func TestFarFromTheOriginCellsStillLandOnWholePoints(t *testing.T) {
	w, win, c := app()
	m := NewInfiniteMap(tiles(), CellSize(16, 16), Generator(floor))
	// 30 minutes walking straight at 150 points a second.
	cam := NewCamera(At(270000, 270000))
	win.Add(m, cam)
	w.AddEntities(m, cam)
	w.RunFrames(1)
	require.NotEmpty(t, c.Unindexed())
	worst := 0.0
	for _, v := range c.Unindexed() {
		for _, f := range []float32{v.Pos.X, v.Pos.Y} {
			worst = max(worst, math.Abs(float64(f)-math.Round(float64(f))))
		}
	}
	assert.Less(t, worst, 1.0/16, "cells within a sixteenth of a point")
}

func TestChunksAreReadyBeforeTheyComeIntoViewEvenWhenTheCameraDoesNotDrawTheMap(t *testing.T) {
	w, win, _ := app()
	floor, crowd := NewLayer(0), NewLayer(2, SortByY())
	things := NewInfiniteMap(
		tiles(),
		ChunkSize(4),
		CellSize(32, 32),
		TallIn(crowd),
		Generator(func(_, _ int, cells [][]int) {
			cells[1][1] = 3
		}),
	)
	cam := NewCamera(Layers(floor, crowd))
	win.Add(floor, crowd, things, cam)
	w.AddEntities(floor, crowd, things)
	w.RunFrames(1)
	before := map[chunkKey]bool{}
	for k := range things.chunks {
		before[k] = true
	}
	for step := range 120 {
		cam.SetPosition(geom.V2(float32(step)*9, float32(step)*4))
		w.RunFrames(1)
		view := geom.Rect{X: cam.Center().X - 400, Y: cam.Center().Y - 300, W: 800, H: 600}
		x0, y0, x1, y1 := things.span(view)
		k0, _, _ := things.split(x0, y0)
		k1, _, _ := things.split(x1, y1)
		for cy := k0.y; cy <= k1.y; cy++ {
			for cx := k0.x; cx <= k1.x; cx++ {
				if !before[chunkKey{cx, cy}] {
					t.Fatalf(
						"step %d: chunk %d, %d is in view but wasn't made the frame before: it would pop in",
						step,
						cx,
						cy,
					)
				}
			}
		}
		clear(before)
		for k := range things.chunks {
			before[k] = true
		}
	}
	assert.NotEmpty(t, things.TallSprites(), "and their tall tiles stand in the crowd")
}

func TestAMinimapCameraHasItsChunksMadeToo(t *testing.T) {
	w, win, _ := app()
	m := NewInfiniteMap(tiles(), ChunkSize(4), CellSize(32, 32), Generator(floor))
	minimap := NewRenderTexture(100, 100)
	win.Add(m, NewCamera(Layers(NewLayer(0))), NewCamera(Into(minimap), At(5000, 0), Zoom(0.25)))
	w.AddEntities(m)
	w.RunFrames(1)
	k, _, _ := m.split(5000/32, 0)
	assert.Contains(t, m.chunks, k, "around what the minimap shows, though no camera draws the map")
}

func TestWalkingFarLeavesNoForgottenTreesInTheWorld(t *testing.T) {
	w, win, _ := app()
	crowd := NewLayer(2, SortByY())
	m := NewInfiniteMap(
		tiles(),
		ChunkSize(4),
		CellSize(32, 32),
		TallIn(crowd),
		KeepChunks(16),
		Generator(func(_, _ int, cells [][]int) {
			cells[1][1], cells[2][3] = 3, 3
		}),
	)
	cam := NewCamera(Layers(crowd))
	win.Add(crowd, m, cam)
	w.AddEntities(crowd, m)
	w.RunFrames(1)
	most := 0
	for step := range 400 {
		cam.SetPosition(geom.V2(float32(step)*16, 0))
		w.RunFrames(1)
		most = max(most, w.Len())
	}
	assert.LessOrEqual(t, w.Len(), most, "50 chunks out")
	held := m.Chunks() * 2
	assert.LessOrEqual(
		t,
		w.Len(),
		held+20,
		"the World holds the trees of the chunks kept, not of every chunk visited: %d entities for %d chunks",
		w.Len(),
		m.Chunks(),
	)
}

func TestReplacingATallTileTakesItsSpriteOutOfTheWorld(t *testing.T) {
	w := world.New()
	m := NewTileMap(tiles(), ASCII(room, legend), CellSize(32, 32))
	w.AddEntities(m)
	tree := m.TallSprites()[0]
	_, in := w.Entity(tree.ID())
	require.True(t, in)
	m.Set(2, 1, 1)
	_, in = w.Entity(tree.ID())
	assert.False(t, in, "gone from the World, not only from the map")
	assert.Nil(t, world.Of(tree))
}

func TestACellTintColoursEachCellOnceWhenItsChunkIsMade(t *testing.T) {
	w, win, c := app()
	asked := 0
	red := color.RGB(1, 0.5, 0.5)
	m := NewInfiniteMap(tiles(), ChunkSize(4), CellSize(16, 16), Generator(floor), CellTint(func(x, _ int) color.Color {
		asked++
		if x < 0 {
			return red
		}
		return color.White
	}), At(400, 300))
	win.Add(m)
	w.AddEntities(m)
	w.RunFrames(1)
	made := asked
	assert.Equal(t, 16*m.Chunks(), made, "each cell of each chunk, once")
	var left, right int
	for _, v := range c.Unindexed() {
		switch {
		case v.Pos.X < 400 && v.Color == red:
			left++
		case v.Pos.X > 400 && v.Color == color.White:
			right++
		case v.Pos.X != 400:
			t.Fatalf("a cell drawn in another's tint at %v: %v", v.Pos, v.Color)
		}
	}
	assert.Positive(t, left)
	assert.Positive(t, right)
	w.RunFrames(2)
	assert.Equal(t, made, asked, "drawing again asks nothing")
}

func TestAResetMapRegeneratesFromWhatItsGeneratorReadsNow(t *testing.T) {
	sorted := NewLayer(0, SortByY())
	id := 1
	m := NewInfiniteMap(tiles(), ChunkSize(4), CellSize(32, 32), TallIn(sorted),
		Generator(func(_, _ int, cells [][]int) {
			for y := range cells {
				for x := range cells[y] {
					cells[y][x] = id
				}
			}
		}),
		CellTint(func(_, _ int) color.Color { return color.Gray(float32(id) / 4) }))
	assert.Equal(t, 1, m.At(0, 0))
	m.Set(9, 9, 2)
	id = 3
	assert.Equal(t, 1, m.At(0, 0), "a chunk made stays as it was")
	v := m.Version()
	m.Reset()
	assert.Greater(t, m.Version(), v, "a reset changes the Version, as a Set does")
	assert.Equal(t, 0, m.Chunks(), "it forgets every chunk")
	assert.Equal(t, 3, m.At(0, 0), "then makes them again from the generator as it now reads")
	assert.Equal(t, 3, m.At(9, 9), "edited ones too")
	assert.Equal(t, color.Gray(0.75), m.chunk(chunkKey{0, 0}).tints[0], "and their tint")
	assert.NotEmpty(t, m.TallSprites())
	for _, s := range m.TallSprites() {
		assert.Equal(t, sorted.GetSceneNode(), s.GetSceneNode().ParentNode())
	}
	m.Reset()
	assert.Empty(t, m.TallSprites(), "and takes their trees")
	m.At(0, 0)
	assert.Len(t, m.TallSprites(), 4*4, "standing a chunk's new ones when it's made again")
}
