package net

import (
	"slices"
	"time"
)

// Remote mirrors are shown a little in the past, interpolated between the
// two snapshots around that moment, so their motion is as even as the
// server's whatever the snapshot rate and the network jitter.
const (
	// minDelay and maxDelay bound the interpolation delay.
	minDelay = 60 * time.Millisecond
	maxDelay = 300 * time.Millisecond
	// teleport is the jump between two samples that is shown at once.
	teleport   = 6
	maxSamples = 32
)

type sample struct {
	tick uint64
	pose Pose
}

// timeline maps server ticks to the local clock and sizes the delay from
// the snapshot spacing and how late snapshots arrive.
type timeline struct {
	has      bool
	epoch    time.Time
	offset   float64
	interval float64
	// late is a decaying peak of how much later than the earliest ones
	// snapshots arrive: the jitter the delay must absorb.
	late     float64
	lastTick uint64
	// shown is the tick being shown; it follows the target by speeding up
	// or slowing down time a little, so a clock correction never jumps.
	shown   float64
	showing bool
}

func (tl *timeline) seconds(at time.Time) float64 { return at.Sub(tl.epoch).Seconds() }

// observe records a snapshot of tick arriving at local time now. The offset
// follows the least delayed arrivals, drifting up slowly.
func (tl *timeline) observe(tick uint64, now time.Time, step float32) {
	if !tl.has {
		tl.has, tl.epoch = true, now
		tl.offset = -float64(tick) * float64(step)
		tl.interval, tl.lastTick = 0.05, tick
		return
	}
	if tick <= tl.lastTick {
		return
	}
	local := tl.seconds(now)
	server := float64(tick) * float64(step)
	if sample := local - server; sample < tl.offset {
		tl.offset = sample
	} else {
		tl.offset += (sample - tl.offset) * 0.01
	}
	spacing := min(float64(tick-tl.lastTick)*float64(step), 0.25)
	tl.interval += (spacing - tl.interval) / 8
	tl.late = max(tl.late*0.98, min(local-server-tl.offset, maxDelay.Seconds()))
	tl.lastTick = tick
}

// delay keeps the shown moment behind the newest snapshot even when it is
// late: one snapshot spacing plus the lateness seen, with a margin.
func (tl *timeline) delay() float64 {
	d := tl.interval + tl.late + 0.01
	return max(minDelay.Seconds(), min(d, maxDelay.Seconds()))
}

// renderTick is the server tick, fractional, to show remote mirrors at.
func (tl *timeline) renderTick(now time.Time, step float32) float64 {
	return (tl.seconds(now) - tl.offset - tl.delay()) / float64(step)
}

// advance moves the shown tick on by dt, at up to 10% off real time toward
// the render tick, or straight to it when it is more than half a second
// away.
func (tl *timeline) advance(now time.Time, dt, step float32) float64 {
	target := tl.renderTick(now, step)
	ticks := float64(dt / step)
	switch gap := target - tl.shown; {
	case !tl.showing || gap > 0.5/float64(step) || gap < -0.5/float64(step):
		tl.shown, tl.showing = target, true
	default:
		tl.shown += ticks * (1 + max(-0.1, min(gap/max(ticks, 1e-6)*0.05, 0.1)))
	}
	return tl.shown
}

// record adds a sample in tick order, as samples may arrive out of order,
// starting over on a teleport so it is shown at once. A sample older than
// all those kept is too late to matter.
func (m *mirror) record(tick uint64, p Pose) {
	s := m.samples
	i := len(s)
	for i > 0 && s[i-1].tick > tick {
		i--
	}
	switch {
	case i > 0 && s[i-1].tick == tick:
		s[i-1].pose = p
		return
	case i == 0 && len(s) > 0:
		return
	case i == len(s) && i > 0 && s[i-1].pose.Position.Dist(p.Position) > teleport:
		m.samples = append(m.samples[:0], sample{tick, p})
		return
	}
	s = slices.Insert(s, i, sample{tick, p})
	if len(s) > maxSamples {
		s = s[1:]
	}
	m.samples = s
}

// at is the pose to show at a fractional tick. Past the newest sample it is
// held, never extrapolated: the server sends nothing for an entity that
// stopped, and goes quiet when nothing changes, so a dry buffer most often
// means a stop, and carrying on would overshoot and snap back.
func (m *mirror) at(tick float64) (Pose, bool) {
	s := m.samples
	if len(s) == 0 {
		return Pose{}, false
	}
	for len(s) > 2 && float64(s[1].tick) <= tick {
		s = s[1:]
	}
	m.samples = append(m.samples[:0], s...)
	s = m.samples
	first, last := s[0], s[len(s)-1]
	switch {
	case tick <= float64(first.tick) || len(s) == 1:
		return first.pose, true
	case tick >= float64(last.tick):
		return last.pose, true
	}
	for i := 1; i < len(s); i++ {
		if float64(s[i].tick) >= tick {
			a, b := s[i-1], s[i]
			return blend(a.pose, b.pose, (tick-float64(a.tick))/float64(b.tick-a.tick)), true
		}
	}
	return last.pose, true
}

func blend(a, b Pose, k float64) Pose {
	t := float32(k)
	return Pose{
		Position: a.Position.Lerp(b.Position, t),
		Rotation: a.Rotation.Slerp(b.Rotation, t),
		Scale:    a.Scale.Lerp(b.Scale, t),
	}
}
