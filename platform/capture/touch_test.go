package capture_test

import (
	"testing"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/internal/inputfeed"
	"github.com/merlindorin/ember/platform/capture"
)

func TestScriptedFingersGoDownMoveAndLift(t *testing.T) {
	script, err := capture.ParseTouches("1:0.2,0.5@10;1:0.6,0.5@30;1:up@40;2:0.8,0.5@20")
	if err != nil {
		t.Fatal(err)
	}
	var in input.State
	at := map[int][]input.Touch{}
	for frame := range 50 {
		script.Apply(&in, frame)
		inputfeed.Of(&in).BeginFrame()
		at[frame] = append([]input.Touch(nil), in.Touches()...)
	}
	if len(at[9]) != 0 || len(at[10]) != 1 || at[10][0].Phase != input.TouchBegan ||
		at[10][0].Source != input.Trackpad {
		t.Fatalf("finger 1 goes down at its first point: %+v", at[10])
	}
	if f := at[20][0]; f.At.Dist(geom.V2(0.4, 0.5)) > 1e-5 || f.Phase != input.TouchMoving {
		t.Fatalf("halfway, it is halfway: %+v", f)
	}
	if len(at[20]) != 2 || at[35][0].Phase != input.TouchResting {
		t.Fatalf("finger 2 joins; finger 1 rests after its last point: %+v", at[35])
	}
	if at[40][0].Phase != input.TouchLifted || len(at[41]) != 1 || at[49][0].ID != 2 {
		t.Fatalf("finger 1 lifts, finger 2 stays: %+v %+v", at[40], at[41])
	}
}

func TestScriptedGesturesSpreadOverTheirFrames(t *testing.T) {
	script, err := capture.ParseTouches("pinch:0.5@10-20;rotate:-1@0-4;swipe:-1,0@7")
	if err != nil {
		t.Fatal(err)
	}
	var in input.State
	var pinch, turn float32
	var swiped int
	for frame := range 30 {
		script.Apply(&in, frame)
		inputfeed.Of(&in).BeginFrame()
		pinch, turn = pinch+in.Pinch(), turn+in.Rotation()
		if in.Swipe() == geom.V2(-1, 0) {
			swiped = frame
		}
	}
	if geom.Abs(pinch-0.5) > 1e-5 || geom.Abs(turn+1) > 1e-5 || swiped != 7 {
		t.Fatalf("pinch %v, rotation %v, swipe at %d", pinch, turn, swiped)
	}
}

func TestBadTouchScriptsAreRefused(t *testing.T) {
	for _, s := range []string{
		"1:0.2,0.5", "1:up@3", "1:0.2,0.5@4;1:up@4", "x:0.2,0.5@1",
		"pinch:1@5-5", "swipe:1@2", "1:0.1,0.1@5;1:0.2,0.2@3",
	} {
		if _, err := capture.ParseTouches(s); err == nil {
			t.Errorf("%q was accepted", s)
		}
	}
}
