package tiles_test

import (
	"slices"
	"testing"

	"github.com/lantervale/ember/collide/tiles"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/twod"
)

func TestATilesetSaysWhichTilesStopBodiesAsALayerDoes(t *testing.T) {
	ts := newTileset()
	stops := map[int]bool{0: false, floor: false, wall: true, shaped: true, body: true, spilling: true, 6: false}
	for id, want := range stops {
		if got := tiles.Stops(ts, id); got != want {
			t.Fatalf("tile %d stops bodies %v, want %v", id, got, want)
		}
	}
	m := twod.NewTileMap(ts, twod.Cells([][]int{{0, floor, wall, shaped, body, spilling, 6}}), twod.CellSize(16, 16))
	l := tiles.Of(m)
	for x := range m.Columns() {
		if l.Solid(x, 0) != tiles.Stops(ts, m.At(x, 0)) {
			t.Fatalf("tile %d: the layer and Stops differ", m.At(x, 0))
		}
	}
	if got := tiles.Rects(ts, body); !slices.Equal(got, []geom.Rect{{X: 4, Y: 4, W: 8, H: 8}}) {
		t.Fatalf("a body-only tile's rects are its body shapes: %v", got)
	}
	if got := tiles.Rects(ts, wall); len(got) != 0 {
		t.Fatalf("a tile solid whole has no rects, its cell stops: %v", got)
	}
	both := twod.NewTileset(nil, twod.Cell(16, 16), tiles.BodyShapes(1, geom.Rect{W: 1, H: 1}),
		tiles.Collision(1, geom.Rect{X: 2, W: 3, H: 4}))
	if got := tiles.Rects(both, 1); !slices.Equal(got, []geom.Rect{{X: 2, W: 3, H: 4}, {W: 1, H: 1}}) {
		t.Fatalf("collision shapes come before body shapes, as a layer places them: %v", got)
	}
}
