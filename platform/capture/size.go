package capture

import (
	"fmt"
	"math"

	"github.com/merlindorin/ember/geom"
)

// Sizes resizes a window along waypoints, as a player dragging its corner
// would.
type Sizes struct {
	Path []Waypoint
}

// ParseSizes reads a script such as "640,360@0;900,480@60": the window is
// that many points wide and tall at each frame, and changes in a straight
// line between them.
func ParseSizes(script string) (Sizes, error) {
	m, err := ParseMouse(script)
	if err != nil {
		return Sizes{}, err
	}
	if len(m.Buttons) > 0 {
		return Sizes{}, fmt.Errorf("capture: a size script holds no buttons: %q", script)
	}
	for _, w := range m.Path {
		if w.At.X < 1 || w.At.Y < 1 {
			return Sizes{}, fmt.Errorf("capture: a window is at least 1×1, not %v", w.At)
		}
	}
	return Sizes{m.Path}, nil
}

// At is the window's size at frame, in whole points; it is false when the
// script has none.
func (s Sizes) At(frame int) (geom.Vec2, bool) {
	at, ok := Mouse{Path: s.Path}.At(frame)
	return geom.V2(float32(math.Round(float64(at.X))), float32(math.Round(float64(at.Y)))), ok
}
