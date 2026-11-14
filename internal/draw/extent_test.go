package draw_test

import (
	"testing"

	"github.com/lantervale/ember/internal/draw"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
)

func TestAListsExtentAndShiftMoveItsVerticesAndClips(t *testing.T) {
	var l draw.List
	l.Reset(0)
	if _, ok := l.Extent(); ok {
		t.Fatal("nothing drawn, no extent")
	}
	l.PushClip(geom.Rect{X: 0, Y: 0, W: 100, H: 100})
	l.Rect(geom.Rect{X: 10, Y: 20, W: 5, H: 5}, color.White)
	l.PopClip()
	l.Rect(geom.Rect{X: 40, Y: 8, W: 2, H: 2}, color.White)
	if e, _ := l.Extent(); e != (geom.Rect{X: 10, Y: 8, W: 32, H: 17}) {
		t.Fatalf("around both rectangles: %v", e)
	}
	l.Shift(geom.V2(-10, -8))
	if e, _ := l.Extent(); e != (geom.Rect{W: 32, H: 17}) {
		t.Fatalf("moved to the origin: %v", e)
	}
	if l.Batches[0].Clip != (geom.Rect{X: -10, Y: -8, W: 100, H: 100}) {
		t.Fatalf("the clip moves with it: %v", l.Batches[0].Clip)
	}
}
