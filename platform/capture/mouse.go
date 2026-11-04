package capture

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
)

// Mouse moves the cursor along waypoints, holds buttons and turns the
// wheel, as a player would.
type Mouse struct {
	Path    []Waypoint
	Buttons []Press
	Wheel   []Turn
}

// Waypoint is where the cursor is at a frame, in window points.
type Waypoint struct {
	At    geom.Vec2
	Frame int
}

// Press is a button held from frame From up to, but not including, To.
type Press struct {
	Button   input.MouseButton
	From, To int
}

// Turn is the wheel turned by Lines at a frame, up negative.
type Turn struct {
	Lines float32
	Frame int
}

var buttons = map[string]input.MouseButton{ //nolint:gochecknoglobals // a lookup table
	"left": input.MouseLeft, "right": input.MouseRight, "middle": input.MouseMiddle,
	"back": input.MouseBack, "forward": input.MouseForward,
}

// ParseMouse reads a script such as "100,80@0;400,200@40;left:30-40;wheel:-1@50":
// the cursor is at each point at its frame, moving in a straight line
// between them, a button is held from the first frame up to the second, and
// the wheel turns by its lines at its frame.
func ParseMouse(script string) (Mouse, error) {
	var m Mouse
	for part := range strings.SplitSeq(script, ";") {
		part = strings.TrimSpace(part)
		switch {
		case part == "":
		case strings.HasPrefix(strings.ToLower(part), "wheel:"):
			lines, at, ok := strings.Cut(part[len("wheel:"):], "@")
			n, err1 := strconv.ParseFloat(strings.TrimSpace(lines), 32)
			frame, err2 := strconv.Atoi(strings.TrimSpace(at))
			if !ok || err1 != nil || err2 != nil || frame < 0 {
				return Mouse{}, fmt.Errorf("capture: %q is not wheel:lines@frame", part)
			}
			m.Wheel = append(m.Wheel, Turn{float32(n), frame})
		case strings.Contains(part, "@"):
			xy, at, _ := strings.Cut(part, "@")
			xs, ys, ok := strings.Cut(xy, ",")
			x, err1 := strconv.ParseFloat(strings.TrimSpace(xs), 32)
			y, err2 := strconv.ParseFloat(strings.TrimSpace(ys), 32)
			frame, err3 := strconv.Atoi(strings.TrimSpace(at))
			if !ok || err1 != nil || err2 != nil || err3 != nil || frame < 0 {
				return Mouse{}, fmt.Errorf("capture: %q is not x,y@frame", part)
			}
			if n := len(m.Path); n > 0 && frame <= m.Path[n-1].Frame {
				return Mouse{}, fmt.Errorf("capture: %q comes before the point it follows", part)
			}
			m.Path = append(m.Path, Waypoint{geom.V2(float32(x), float32(y)), frame})
		default:
			name, frames, ok := strings.Cut(part, ":")
			from, to, ok2 := strings.Cut(frames, "-")
			b, known := buttons[strings.ToLower(strings.TrimSpace(name))]
			f, err1 := strconv.Atoi(strings.TrimSpace(from))
			t, err2 := strconv.Atoi(strings.TrimSpace(to))
			if !ok || !ok2 || !known || err1 != nil || err2 != nil || f < 0 || t <= f {
				return Mouse{}, fmt.Errorf(
					"capture: %q is not x,y@frame or left|right|middle|back|forward:from-to",
					part,
				)
			}
			m.Buttons = append(m.Buttons, Press{b, f, t})
		}
	}
	return m, nil
}

// At is where the cursor is at frame: on a waypoint, between two of them
// in a straight line, or at the first or last one outside them. It is
// false when the script has no path.
func (m Mouse) At(frame int) (geom.Vec2, bool) {
	if len(m.Path) == 0 {
		return geom.Vec2{}, false
	}
	i, _ := slices.BinarySearchFunc(m.Path, frame, func(w Waypoint, f int) int { return w.Frame - f })
	switch {
	case i == 0:
		return m.Path[0].At, true
	case i == len(m.Path):
		return m.Path[i-1].At, true
	}
	a, b := m.Path[i-1], m.Path[i]
	t := float32(frame-a.Frame) / float32(b.Frame-a.Frame)
	return a.At.Add(b.At.Sub(a.At).Scale(t)), true
}

// Apply puts the cursor where the path says, holds the scripted buttons and
// turns the wheel, through in, as the window's own events would.
func (m Mouse) Apply(in *input.State, frame int) {
	if at, ok := m.At(frame); ok {
		in.SetMouse(at)
	}
	down := make(map[input.MouseButton]bool)
	for _, p := range m.Buttons {
		down[p.Button] = down[p.Button] || frame >= p.From && frame < p.To
	}
	for b, d := range down {
		in.SetMouseButton(b, d)
	}
	for _, t := range m.Wheel {
		if t.Frame == frame {
			in.AddWheel(geom.V2(0, t.Lines))
		}
	}
}
