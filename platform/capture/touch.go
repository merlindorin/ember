package capture

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
)

// Touches plays scripted fingers on the trackpad and gestures, as a player
// would.
type Touches struct {
	Fingers map[int]*Finger
	Pinches []Spread
	Turns   []Spread
	Swipes  []Waypoint
}

// Finger is one scripted finger: down at its first waypoint, moving in
// straight lines between them, resting after the last, lifted at Up (never
// when negative).
type Finger struct {
	Path []Waypoint
	Up   int
}

// Spread is Total of a gesture spread evenly over frames From up to To.
type Spread struct {
	Total    float32
	From, To int
}

// ParseTouches reads a script such as "1:0.3,0.5@0;1:0.6,0.5@30;1:up@40;
// pinch:0.5@10-40;rotate:1.57@10-40;swipe:-1,0@50": finger 1 goes down on
// the trackpad, 0 to 1 from its top-left, at its first point and moves
// between them until it lifts; a pinch or a rotation, in radians, is
// spread over its frames; a swipe happens at its frame.
func ParseTouches(script string) (Touches, error) {
	t := Touches{Fingers: map[int]*Finger{}}
	for part := range strings.SplitSeq(script, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		name, arg, ok := strings.Cut(part, ":")
		if !ok {
			return Touches{}, fmt.Errorf("capture: %q is not name:value", part)
		}
		name = strings.ToLower(strings.TrimSpace(name))
		value, at, ok := strings.Cut(arg, "@")
		if !ok {
			return Touches{}, fmt.Errorf("capture: %q has no @frame", part)
		}
		switch name {
		case "pinch", "rotate":
			total, err1 := strconv.ParseFloat(strings.TrimSpace(value), 32)
			from, to, spans := strings.Cut(at, "-")
			f, err2 := strconv.Atoi(strings.TrimSpace(from))
			e, err3 := strconv.Atoi(strings.TrimSpace(to))
			if !spans || err1 != nil || err2 != nil || err3 != nil || f < 0 || e <= f {
				return Touches{}, fmt.Errorf("capture: %q is not %s:amount@from-to", part, name)
			}
			s := Spread{float32(total), f, e}
			if name == "pinch" {
				t.Pinches = append(t.Pinches, s)
			} else {
				t.Turns = append(t.Turns, s)
			}
		case "swipe":
			p, frame, err := point(value, at)
			if err != nil {
				return Touches{}, fmt.Errorf("capture: %q is not swipe:x,y@frame", part)
			}
			t.Swipes = append(t.Swipes, Waypoint{p, frame})
		default:
			id, err := strconv.Atoi(name)
			if err != nil {
				return Touches{}, fmt.Errorf("capture: %q names no finger or gesture", part)
			}
			f := t.Fingers[id]
			if f == nil {
				f = &Finger{Up: -1}
				t.Fingers[id] = f
			}
			if strings.TrimSpace(value) == "up" {
				frame, bad := strconv.Atoi(strings.TrimSpace(at))
				if bad != nil || len(f.Path) == 0 || frame <= f.Path[len(f.Path)-1].Frame {
					return Touches{}, fmt.Errorf("capture: %q lifts a finger before it is down", part)
				}
				f.Up = frame
				continue
			}
			p, frame, err := point(value, at)
			if err != nil {
				return Touches{}, fmt.Errorf("capture: %q is not id:x,y@frame", part)
			}
			if err = addWaypoint(&f.Path, p, frame, part); err != nil {
				return Touches{}, err
			}
		}
	}
	return t, nil
}

func point(xy, at string) (geom.Vec2, int, error) {
	xs, ys, ok := strings.Cut(xy, ",")
	x, err1 := strconv.ParseFloat(strings.TrimSpace(xs), 32)
	y, err2 := strconv.ParseFloat(strings.TrimSpace(ys), 32)
	frame, err3 := strconv.Atoi(strings.TrimSpace(at))
	if !ok || err1 != nil || err2 != nil || err3 != nil || frame < 0 {
		return geom.Vec2{}, 0, fmt.Errorf("bad point")
	}
	return geom.V2(float32(x), float32(y)), frame, nil
}

// Apply puts the fingers down, moves and lifts them and makes the gestures
// due at frame, through in.
func (t Touches) Apply(in *input.State, frame int) {
	for id, f := range t.Fingers {
		first := f.Path[0].Frame
		switch {
		case frame == f.Up:
			in.EndTouch(id)
		case frame == first:
			in.AddTouch(id, input.Trackpad, f.Path[0].At)
		case frame > first && (f.Up < 0 || frame < f.Up):
			at, _ := Mouse{Path: f.Path}.At(frame)
			in.MoveTouch(id, at)
		}
	}
	for _, p := range t.Pinches {
		if frame >= p.From && frame < p.To {
			in.AddPinch(p.Total / float32(p.To-p.From))
		}
	}
	for _, r := range t.Turns {
		if frame >= r.From && frame < r.To {
			in.AddRotation(r.Total / float32(r.To-r.From))
		}
	}
	for _, s := range t.Swipes {
		if frame == s.Frame {
			in.AddSwipe(s.At)
		}
	}
}
