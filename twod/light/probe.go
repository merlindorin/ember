package light

import (
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/internal/draw"
	"github.com/lantervale/ember/twod"
)

// Probe asks the light where it stands: a 2D entity, a child of what the
// game asks about, such as an enemy whose eyes show in the dark. The GPU
// reads the light picture there each frame it is in its window's view and
// hands the answer back 1 to 3 frames later, so it is exactly the light
// drawn, shadows, shapes and glows included. A rule with a threshold,
// such as eyes showing below 25%, should turn on and off at two levels a
// little apart (25% and 27%), or a flickering light makes it blink.
type Probe struct {
	twod.Item
	result draw.ProbeResult
}

// NewProbe returns a probe, placed by opts in its parent's space.
func NewProbe(opts ...twod.Option) *Probe {
	return twod.Build(&Probe{Item: twod.Base()}, "light-probe", opts...)
}

// Light is the light at the probe, the colour a white lit pixel there
// takes, as of a few frames ago: before the first answer, black; in a
// window nothing lights, white.
func (p *Probe) Light() color.Color { return p.result.Light }

// Ready reports whether Light has been answered at least once.
func (p *Probe) Ready() bool { return p.result.Ready }

// Stale reports whether the probe stands out of its window's view (or
// past the 1,024 probes a window answers): it isn't asked, and Light
// keeps what it found last.
func (p *Probe) Stale() bool { return p.result.Stale }
