package input_test

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/internal/inputfeed"
)

func phases(s *input.State) map[int]input.TouchPhase {
	out := map[int]input.TouchPhase{}
	for _, t := range s.Touches() {
		out[t.ID] = t.Phase
	}
	return out
}

func TestATouchBeginsMovesRestsAndLifts(t *testing.T) {
	var s input.State
	s.AddTouch(7, input.Trackpad, geom.V2(0.2, 0.5))
	inputfeed.Of(&s).BeginFrame()
	touches := s.Touches()
	want := input.Touch{ID: 7, Source: input.Trackpad, At: geom.V2(0.2, 0.5), Phase: input.TouchBegan}
	if len(touches) != 1 || touches[0] != want {
		t.Fatalf("a new touch: %+v", touches)
	}
	s.MoveTouch(7, geom.V2(0.3, 0.5))
	inputfeed.Of(&s).BeginFrame()
	if phases(&s)[7] != input.TouchMoving || s.Touches()[0].At != geom.V2(0.3, 0.5) {
		t.Fatalf("a moved touch: %+v", s.Touches())
	}
	inputfeed.Of(&s).BeginFrame()
	if phases(&s)[7] != input.TouchResting {
		t.Fatal("a still touch rests")
	}
	s.EndTouch(7)
	inputfeed.Of(&s).BeginFrame()
	if phases(&s)[7] != input.TouchLifted {
		t.Fatal("a lifted touch shows for the frame it lifts")
	}
	inputfeed.Of(&s).BeginFrame()
	if len(s.Touches()) != 0 {
		t.Fatal("then it is gone")
	}
	if input.Trackpad.String() != "trackpad" || input.TouchResting.String() != "resting" {
		t.Fatal("sources and phases print")
	}
}

func TestATapBetweenTwoFramesStillBegins(t *testing.T) {
	var s input.State
	s.AddTouch(1, input.Touchscreen, geom.V2(40, 30))
	s.EndTouch(1)
	inputfeed.Of(&s).BeginFrame()
	if phases(&s)[1] != input.TouchBegan {
		t.Fatal("a tap shows as began first")
	}
	inputfeed.Of(&s).BeginFrame()
	if phases(&s)[1] != input.TouchLifted {
		t.Fatal("then as lifted")
	}
}

func TestTwoTouchesMovingTogetherPan(t *testing.T) {
	var s input.State
	s.AddTouch(1, input.Trackpad, geom.V2(0.4, 0.5))
	s.AddTouch(2, input.Trackpad, geom.V2(0.6, 0.5))
	inputfeed.Of(&s).BeginFrame()
	s.MoveTouch(1, geom.V2(0.4, 0.4))
	s.MoveTouch(2, geom.V2(0.6, 0.3))
	inputfeed.Of(&s).BeginFrame()
	if p := s.Pan(); geom.Abs(p.Y+0.15) > 1e-5 || p.X != 0 {
		t.Fatalf("their mean movement: %v", p)
	}
	s.EndTouch(2)
	s.MoveTouch(1, geom.V2(0.5, 0.4))
	inputfeed.Of(&s).BeginFrame()
	if s.Pan() != (geom.Vec2{}) {
		t.Fatal("one finger does not pan")
	}
}

func TestGesturesLastOneFrame(t *testing.T) {
	var s input.State
	s.AddPinch(0.1)
	s.AddPinch(0.05)
	s.AddRotation(-0.2)
	s.AddSwipe(geom.V2(-1, 0))
	inputfeed.Of(&s).BeginFrame()
	if geom.Abs(s.Pinch()-0.15) > 1e-6 || s.Rotation() != -0.2 || s.Swipe() != geom.V2(-1, 0) {
		t.Fatalf("pinch %v rotation %v swipe %v", s.Pinch(), s.Rotation(), s.Swipe())
	}
	inputfeed.Of(&s).BeginFrame()
	if s.Pinch() != 0 || s.Rotation() != 0 || s.Swipe() != (geom.Vec2{}) {
		t.Fatal("gestures reset each frame")
	}
}
