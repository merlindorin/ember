package desktop //nolint:testpackage // tests the unexported trackpad bridge

import (
	"math"
	"testing"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/internal/inputfeed"
)

func TestTrackpadTouchesGetSmallIDsAndTheirPhases(t *testing.T) {
	var in input.State
	f := newFingers(&in)
	f.touch(0xBEEF, 0.2, 0.3, touchBegan)
	f.touch(0xCAFE, 0.7, 0.3, touchBegan)
	inputfeed.Of(&in).BeginFrame()
	got := in.Touches()
	if len(got) != 2 || got[0].ID != 1 || got[1].ID != 2 || got[0].Source != input.Trackpad ||
		got[0].At != geom.V2(0.2, 0.3) {
		t.Fatalf("two fingers down: %+v", got)
	}
	f.touch(0xBEEF, 0.25, 0.3, touchMoved)
	f.touch(0xCAFE, 0.7, 0.3, touchStationary)
	inputfeed.Of(&in).BeginFrame()
	if p := in.Touches(); p[0].Phase != input.TouchMoving || p[1].Phase != input.TouchResting {
		t.Fatalf("one moved, one rested: %+v", p)
	}
	f.touch(0xBEEF, 0.25, 0.3, touchEnded)
	f.touch(0xCAFE, 0.7, 0.3, touchCancelled)
	f.touch(0xF00D, 0.5, 0.5, touchMoved)
	inputfeed.Of(&in).BeginFrame()
	p := in.Touches()
	if len(p) != 3 || p[0].Phase != input.TouchLifted || p[1].Phase != input.TouchLifted || p[2].ID != 3 ||
		p[2].Phase != input.TouchBegan {
		t.Fatalf("ended and cancelled lift; an unknown finger begins: %+v", p)
	}
	if len(f.ids) != 1 {
		t.Fatal("lifted fingers are forgotten")
	}
}

func TestGesturesTurnTheWayTwodDoes(t *testing.T) {
	var in input.State
	f := newFingers(&in)
	f.gesture(gestureMagnify, 0.1, 0)
	f.gesture(gestureRotate, 90, 0)
	f.gesture(gestureSwipe, 1, 0)
	inputfeed.Of(&in).BeginFrame()
	if geom.Abs(in.Pinch()-0.1) > 1e-6 || geom.Abs(in.Rotation()+math.Pi/2) > 1e-5 || in.Swipe() != geom.V2(-1, 0) {
		t.Fatalf("pinch %v rotation %v swipe %v", in.Pinch(), in.Rotation(), in.Swipe())
	}
}
