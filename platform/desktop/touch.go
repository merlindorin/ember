package desktop

import (
	"math"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
)

// The phases and gestures the macOS backend reports, as AppKit numbers
// them.
const (
	touchBegan      = 1
	touchMoved      = 2
	touchStationary = 4
	touchEnded      = 8
	touchCancelled  = 16

	gestureMagnify = 0
	gestureRotate  = 1
	gestureSwipe   = 2
)

// fingers turns a window's raw trackpad touches, keyed by the system's
// identity, into its input's touches with small IDs.
type fingers struct {
	in   *input.State
	ids  map[uint64]int
	next int
}

func newFingers(in *input.State) *fingers { return &fingers{in: in, ids: map[uint64]int{}} }

// touch feeds one touch, at x, y on the pad from its top-left.
func (f *fingers) touch(key uint64, x, y float64, phase int) {
	at := geom.V2(float32(x), float32(y))
	id, known := f.ids[key]
	switch phase {
	case touchEnded, touchCancelled:
		if known {
			f.in.MoveTouch(id, at)
			f.in.EndTouch(id)
			delete(f.ids, key)
		}
	default:
		if !known {
			f.next++
			id = f.next
			f.ids[key] = id
			f.in.AddTouch(id, input.Trackpad, at)
			return
		}
		f.in.MoveTouch(id, at)
	}
}

// gesture feeds a recognised gesture: a magnification, a rotation in
// degrees counterclockwise, or a swipe's deltas, positive left and up.
func (f *fingers) gesture(kind int, a, b float64) {
	switch kind {
	case gestureMagnify:
		f.in.AddPinch(float32(a))
	case gestureRotate:
		f.in.AddRotation(float32(-a * math.Pi / 180))
	case gestureSwipe:
		f.in.AddSwipe(geom.V2(float32(-sign(a)), float32(-sign(b))))
	}
}

func sign(v float64) float64 {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	}
	return 0
}
