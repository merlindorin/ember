package capture

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
)

// Pads plays scripted gamepads: each is plugged in for a range of frames,
// holds buttons and moves its sticks and triggers, as a player would.
type Pads map[int]*Pad

// Pad is one scripted gamepad: when it is plugged in (always, if Plugged
// is nil), the buttons it holds, and the waypoints of its left and right
// sticks and of its left and right triggers, a trigger's pull in X.
type Pad struct {
	Plugged  *Span
	Buttons  []PadHold
	Sticks   [2][]Waypoint
	Triggers [2][]Waypoint
}

// Span is from frame From up to, but not including, frame To.
type Span struct{ From, To int }

// PadHold is a pad button held for a span of frames.
type PadHold struct {
	Button input.GamepadButton
	Span
}

var padButtons = func() map[string]input.GamepadButton { //nolint:gochecknoglobals // a lookup table
	m := map[string]input.GamepadButton{}
	for b := input.GamepadSouth; b <= input.GamepadTouchpad; b++ {
		m[strings.ToLower(strings.TrimPrefix(b.String(), "Gamepad"))] = b
	}
	return m
}()

// ParsePads reads a script such as "south:30-40;leftstick:0,0@0;leftstick:1,0@30;
// righttrigger:1@60;1/connect:90-180": a button, named by its position, is
// held from the first frame up to the second; a stick, x,y, and a trigger,
// its pull, rest until their first value's frame, then hold it, moving in
// straight lines to the next ones: one value leans a stick from its frame
// on, pressing that direction once; connect
// is when the pad is plugged in, every frame if it is not given. "1/" makes
// the rest of the item about pad 1 rather than pad 0.
func ParsePads(script string) (Pads, error) {
	pads := Pads{}
	for part := range strings.SplitSeq(script, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id := 0
		if n, rest, ok := strings.Cut(part, "/"); ok {
			v, err := strconv.Atoi(strings.TrimSpace(n))
			if err != nil || v < 0 || v >= input.MaxGamepads {
				return nil, fmt.Errorf("capture: %q names no pad", part)
			}
			id, part = v, strings.TrimSpace(rest)
		}
		p := pads[id]
		if p == nil {
			p = &Pad{}
			pads[id] = p
		}
		name, arg, ok := strings.Cut(part, ":")
		name = strings.ToLower(strings.TrimSpace(name))
		if !ok {
			return nil, fmt.Errorf("capture: %q is not name:value", part)
		}
		switch name {
		case "leftstick", "rightstick":
			if err := p.stick(name, arg, part); err != nil {
				return nil, err
			}
		case "lefttrigger", "righttrigger":
			v, at, timed := strings.Cut(arg, "@")
			pull, err1 := strconv.ParseFloat(strings.TrimSpace(v), 32)
			frame, err2 := strconv.Atoi(strings.TrimSpace(at))
			if !timed || err1 != nil || err2 != nil || frame < 0 {
				return nil, fmt.Errorf("capture: %q is not %s:pull@frame", part, name)
			}
			t := &p.Triggers[stickIndex(strings.Replace(name, "trigger", "stick", 1))]
			if err := addWaypoint(t, geom.V2(float32(pull), 0), frame, part); err != nil {
				return nil, err
			}
		default:
			from, to, spans := strings.Cut(arg, "-")
			f, err1 := strconv.Atoi(strings.TrimSpace(from))
			t, err2 := strconv.Atoi(strings.TrimSpace(to))
			if !spans || err1 != nil || err2 != nil || f < 0 || t <= f {
				return nil, fmt.Errorf("capture: %q is not %s:from-to", part, name)
			}
			if name == "connect" {
				p.Plugged = &Span{f, t}
				continue
			}
			b, known := padButtons[name]
			if !known {
				return nil, fmt.Errorf("capture: no pad button named %q", name)
			}
			p.Buttons = append(p.Buttons, PadHold{b, Span{f, t}})
		}
	}
	return pads, nil
}

// stick adds part's waypoint, x,y@frame in arg, to p's stick name.
func (p *Pad) stick(name, arg, part string) error {
	xy, at, timed := strings.Cut(arg, "@")
	xs, ys, paired := strings.Cut(xy, ",")
	x, err1 := strconv.ParseFloat(strings.TrimSpace(xs), 32)
	y, err2 := strconv.ParseFloat(strings.TrimSpace(ys), 32)
	frame, err3 := strconv.Atoi(strings.TrimSpace(at))
	if !timed || !paired || err1 != nil || err2 != nil || err3 != nil || frame < 0 {
		return fmt.Errorf("capture: %q is not %s:x,y@frame", part, name)
	}
	return addWaypoint(&p.Sticks[stickIndex(name)], geom.V2(float32(x), float32(y)), frame, part)
}

// held is where a stick or trigger's path has it at frame: at rest before
// its first keyframe, then held at each keyframe and moving straight
// between them, so one keyframe leans it from that frame on.
func held(path []Waypoint, frame int) (geom.Vec2, bool) {
	if len(path) == 0 || frame < path[0].Frame {
		return geom.Vec2{}, false
	}
	return Mouse{Path: path}.At(frame)
}

func stickIndex(name string) int {
	if strings.HasPrefix(name, "right") {
		return 1
	}
	return 0
}

func addWaypoint(path *[]Waypoint, at geom.Vec2, frame int, part string) error {
	if n := len(*path); n > 0 && frame <= (*path)[n-1].Frame {
		return fmt.Errorf("capture: %q comes before the point it follows", part)
	}
	*path = append(*path, Waypoint{at, frame})
	return nil
}

// Apply plugs in each scripted pad and sets all of it for frame, through
// in, after the platform has read the real pads: a real pad in the same
// slot is replaced, buttons up and sticks centred but for the script's.
func (ps Pads) Apply(in *input.State, frame int) {
	for id, p := range ps {
		if p.Plugged != nil && (frame < p.Plugged.From || frame >= p.Plugged.To) {
			in.DisconnectGamepad(id)
			continue
		}
		in.DisconnectGamepad(id)
		in.ConnectGamepad(id, fmt.Sprintf("Scripted pad %d", id))
		for _, b := range p.Buttons {
			if frame >= b.From && frame < b.To {
				in.SetGamepadButton(id, b.Button, true)
			}
		}
		for i, st := range []input.Stick{input.LeftStick, input.RightStick} {
			if at, ok := held(p.Sticks[i], frame); ok {
				in.SetGamepadStick(id, st, at)
			}
		}
		for i, tr := range []input.GamepadButton{input.GamepadLeftTrigger, input.GamepadRightTrigger} {
			if at, ok := held(p.Triggers[i], frame); ok {
				in.SetGamepadTrigger(id, tr, at.X)
			}
		}
	}
}
