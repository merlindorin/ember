package input

import "github.com/merlindorin/ember/geom"

// TouchSource is what a finger touches.
type TouchSource int

const (
	// Trackpad touches are where the finger is on the pad, from (0, 0) at
	// its top-left to (1, 1) at its bottom-right, not window points.
	Trackpad TouchSource = iota
	// Touchscreen touches are where the finger is on the window, in points.
	Touchscreen
)

func (s TouchSource) String() string {
	if s == Touchscreen {
		return "touchscreen"
	}
	return "trackpad"
}

// TouchPhase is where a touch is in its life this frame.
type TouchPhase int

const (
	TouchBegan TouchPhase = iota
	TouchMoving
	TouchResting
	TouchLifted
)

func (p TouchPhase) String() string {
	return [...]string{"began", "moving", "resting", "lifted"}[p]
}

// Touch is one finger this frame.
type Touch struct {
	// ID stays the same while the finger stays down.
	ID     int
	Source TouchSource
	// At is on the pad, 0 to 1, for a Trackpad; in window points for a
	// Touchscreen.
	At    geom.Vec2
	Phase TouchPhase
}

// TouchStarted, TouchMoved and TouchEnded are emitted as the frame a
// touch begins, moves or lifts begins.
type (
	TouchStarted struct{ Touch Touch }
	TouchMoved   struct{ Touch Touch }
	TouchEnded   struct{ Touch Touch }
)

type liveTouch struct {
	Touch
	fresh, lifted bool
}

// Touches is this frame's touches, a lifted one for the frame it lifts. A
// touch that begins and lifts between two frames shows as began, then
// lifted the frame after.
func (s *State) Touches() []Touch { return s.touches }

// Pinch is this frame's change of scale, 0 when none: multiply a zoom by
// 1 + Pinch().
func (s *State) Pinch() float32 { return s.pinch }

// Rotation is this frame's change of angle in radians, turning the way
// twod rotations do.
func (s *State) Rotation() float32 { return s.rotation }

// Pan is how far two or more touches moved together this frame, their
// mean movement, in their source's units.
func (s *State) Pan() geom.Vec2 { return s.pan }

// Swipe is a quick swipe's direction this frame, -1, 0 or 1 on each axis,
// Y down.
func (s *State) Swipe() geom.Vec2 { return s.swipe }

// AddTouch puts finger id down at at on source, seen next frame.
func (s *State) AddTouch(id int, source TouchSource, at geom.Vec2) {
	t := liveTouch{Touch: Touch{ID: id, Source: source, At: at}, fresh: true}
	if l := s.liveTouch(id); l != nil {
		*l = t
		return
	}
	s.liveTouches = append(s.liveTouches, t)
}

// MoveTouch moves finger id to at.
func (s *State) MoveTouch(id int, at geom.Vec2) {
	if l := s.liveTouch(id); l != nil {
		l.At = at
	}
}

// EndTouch lifts finger id.
func (s *State) EndTouch(id int) {
	if l := s.liveTouch(id); l != nil {
		l.lifted = true
	}
}

// AddPinch, AddRotation and AddSwipe make the gestures, seen next frame.
func (s *State) AddPinch(d float32)     { s.pendingPinch += d }
func (s *State) AddRotation(d float32)  { s.pendingRotation += d }
func (s *State) AddSwipe(dir geom.Vec2) { s.pendingSwipe = dir }

func (s *State) liveTouch(id int) *liveTouch {
	for i := range s.liveTouches {
		if s.liveTouches[i].ID == id {
			return &s.liveTouches[i]
		}
	}
	return nil
}

// beginTouches publishes the touches and gestures since the last frame.
func (s *State) beginTouches() {
	before := map[int]geom.Vec2{}
	for _, t := range s.touches {
		if t.Phase != TouchLifted {
			before[t.ID] = t.At
		}
	}
	s.touches = s.touches[:0]
	var moved geom.Vec2
	together := 0
	kept := s.liveTouches[:0]
	for _, l := range s.liveTouches {
		t := l.Touch
		was, seen := before[t.ID]
		switch {
		case l.fresh:
			t.Phase = TouchBegan
		case l.lifted:
			t.Phase = TouchLifted
		case t.At != was:
			t.Phase = TouchMoving
		default:
			t.Phase = TouchResting
		}
		if seen && !l.fresh && !l.lifted {
			moved, together = moved.Add(t.At.Sub(was)), together+1
		}
		s.touches = append(s.touches, t)
		if t.Phase != TouchLifted {
			l.fresh = false
			kept = append(kept, l)
		}
	}
	s.liveTouches = kept
	s.pan = geom.Vec2{}
	if together >= 2 {
		s.pan = moved.Scale(1 / float32(together))
	}
	s.pinch, s.pendingPinch = s.pendingPinch, 0
	s.rotation, s.pendingRotation = s.pendingRotation, 0
	s.swipe, s.pendingSwipe = s.pendingSwipe, geom.Vec2{}
}
