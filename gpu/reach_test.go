package gpu //nolint:testpackage // tests the unexported occluder reach helpers

import (
	"testing"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/internal/draw"
)

func TestOnlyTheLightsAnExcludedOccluderReachesAreRedrawn(t *testing.T) {
	var occ draw.List
	occ.Reset(0)
	occ.Rect(geom.Rect{X: 100, Y: 100, W: 10, H: 4}, draw.OccluderColor(0b100, 0))
	occ.Rect(geom.Rect{X: 400, Y: 100, W: 10, H: 40}, draw.OccluderColor(0b001, 0))
	excluded := excludedBy(&occ, 0b100, nil)
	if len(excluded) != 2 || excluded[0].X != 100 || excluded[1].X != 100 {
		t.Fatalf("both triangles of the group 2 box, not the wall: %v", excluded)
	}
	lights := []lightReach{
		{bounds: geom.Rect{X: 50, Y: 50, W: 100, H: 100}, groups: 0b111, rows: 1},
		{bounds: geom.Rect{X: 300, Y: 50, W: 200, H: 100}, groups: 0b111, rows: 1},
		{bounds: geom.Rect{X: 50, Y: 50, W: 100, H: 100}, groups: 0b011, rows: 1},
		{bounds: geom.Rect{X: 50, Y: 50, W: 100, H: 100}, groups: 0b111},
	}
	if got := reached(lights, excluded, 0b100, nil); len(got) != 1 || got[0] != 0 {
		t.Fatalf("only the light whose reach holds the box and which group 2 blocks, with shadows: %v", got)
	}
	if groupsOf(color.Color{R: 1, B: 1, A: 0.3}) != 0b101 {
		t.Fatal("an occluder's groups are its colour's channels, whatever its height")
	}
}
