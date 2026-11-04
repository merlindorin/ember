package input

import (
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/internal/inputfeed"
)

// MaxRumbleSeconds is the longest a pad rumbles from one call.
const MaxRumbleSeconds = 30

// Rumble is what a pad's motors play: Low drives the heavy motor in the
// left grip and High the light one in the right, each from 0 to 1, for
// Seconds. The zero Rumble is a stop.
type Rumble struct{ Low, High, Seconds float32 }

// Rumble shakes the pad, low and high from 0 to 1, for seconds up to
// MaxRumbleSeconds, replacing what it was playing. It does nothing while
// rumble is off, or for a slot with no pad. Only macOS pads rumble for
// now; elsewhere the call is recorded and nothing shakes.
func (g *Gamepad) Rumble(low, high, seconds float32) {
	r := Rumble{geom.Clamp01(low), geom.Clamp01(high), geom.Clamp(seconds, 0, MaxRumbleSeconds)}
	if r.Seconds == 0 || r.Low == 0 && r.High == 0 {
		r = Rumble{}
	}
	g.rumble(r)
}

// StopRumble stops the pad's motors.
func (g *Gamepad) StopRumble() { g.rumble(Rumble{}) }

func (g *Gamepad) rumble(r Rumble) {
	s := g.state
	if s == nil || s.rumbleOff || !g.connected {
		return
	}
	s.sendRumble(g.id, r)
	if s.rumbles != nil {
		s.rumbles.last[g.id], s.rumbles.played[g.id] = r, true
	}
}

// RumbleEnabled reports whether pads rumble, true unless
// SetRumbleEnabled turned it off.
func (s *State) RumbleEnabled() bool { return !s.rumbleOff }

// SetRumbleEnabled turns rumble on or off for every pad, as a game's
// vibration setting does. Turning it off stops the pads, and every Rumble
// after does nothing until it is on again.
func (s *State) SetRumbleEnabled(on bool) {
	if !on && !s.rumbleOff {
		for i := range s.pads {
			if s.pads[i].connected {
				s.sendRumble(i, Rumble{})
			}
		}
	}
	s.rumbleOff = !on
}

func (s *State) sendRumble(id int, r Rumble) {
	s.outRumbles[id], s.outSent[id] = r, true
}

// RumbleRecorder keeps the last rumble played on each pad, for tests.
type RumbleRecorder struct {
	last   [MaxGamepads]Rumble
	played [MaxGamepads]bool
}

// NewRumbleRecorder records the rumbles played on s from now on.
func NewRumbleRecorder(s *State) *RumbleRecorder {
	s.rumbles = &RumbleRecorder{}
	return s.rumbles
}

// Last is the last rumble played on pad id, the zero Rumble for a stop,
// and whether any was.
func (r *RumbleRecorder) Last(id int) (Rumble, bool) {
	if id < 0 || id >= MaxGamepads {
		return Rumble{}, false
	}
	return r.last[id], r.played[id]
}

// Reset forgets what was played.
func (r *RumbleRecorder) Reset() { *r = RumbleRecorder{} }

func (f feed) Rumbles() []inputfeed.Rumble {
	s := f.s
	var out []inputfeed.Rumble
	for i, sent := range s.outSent {
		if sent {
			r := s.outRumbles[i]
			out = append(out, inputfeed.Rumble{Pad: i, Low: r.Low, High: r.High, Seconds: r.Seconds})
		}
	}
	s.outSent = [MaxGamepads]bool{}
	return out
}
