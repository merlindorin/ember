package ui

import (
	"slices"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/internal/inputfeed"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/ui/internal/box"
	"github.com/merlindorin/ember/ui/paint"
)

// replay is what a Document keeps to draw a frame again when nothing in
// it changed: the last frame's drawing, and what tells a changed frame.
type replay struct {
	// in is the input the last frame read: the window's, or an idle one
	// while another owns it.
	in       *input.State
	last     twod.Recording
	recorded bool
	// repaint is set by anything that changed what paint would draw:
	// a change css saw, a layout that moved boxes, a scroll.
	repaint bool
	// invalidated is set by Invalidate until the next frame: a canvas drawn
	// in its place, covered, then repaints the frame.
	invalidated bool
	// live is what the last paint drew live (decision 0052), and inPlace
	// whether it was drawn in its place, something painted after a part
	// covering it; animating is whether a transition ran in the last frame.
	live               paint.LiveSet
	inPlace, animating bool
	// opts are the paint options a frame of live parts takes, made once
	// for viewport and scale, so such a frame allocates nothing.
	opts []paint.Option
	// frame is what each paint takes from this frame, through frameOpts:
	// opts and Each(&frame).
	frame     paint.Frame
	frameOpts []paint.Option
	optsAt    geom.Rect
	optsScale float32
	size      geom.Vec2
	scale     float32
	// replayedLast is whether the last frame replayed, and quiet whether
	// this one, so far, changed nothing: interact then reuses its last
	// hit tests.
	replayedLast, quiet bool
	// hit and pick are the last hit tests' answers.
	hit  *box.Box
	pick paint.Pick
}

// Why a frame repaints, as the F3 overlay counts them.
const (
	replayed          = "ui replays"
	replayedLive      = "ui replays + live"
	repaintFirst      = "ui repaints: first"
	repaintChanged    = "ui repaints: changed"
	repaintTransition = "ui repaints: transition"
	repaintLive       = "ui repaints: live"
	repaintInput      = "ui repaints: input"
	repaintSize       = "ui repaints: size"
	repaintCanvas     = "ui repaints: canvas"
	repaintRefused    = "ui repaints: state"
)

// repaintReason is why this frame must be painted, "" when only the clock
// moved: nothing css or layout saw changed, no transition ran in it or
// the last, no input arrived and the canvas is the same size and scale.
// The last frame is then replayed, its live parts drawn again on top.
func (d *Document) repaintReason(c *twod.Canvas, size geom.Vec2, animating bool) string {
	switch {
	case !d.replay.recorded:
		return repaintFirst
	case d.replay.repaint:
		return repaintChanged
	case animating || d.replay.animating:
		return repaintTransition
	case c.Input() != nil && inputfeed.Of(c.Input()).Changed():
		return repaintInput
	case size != d.replay.size || d.fonts.Scale() != d.replay.scale:
		return repaintSize
	}
	return ""
}

// quietNow is whether nothing changed this frame before interact: no
// input, nothing css or layout saw, no transition, and the last frame
// replayed, so what lies under the pointer is what lay there.
func (d *Document) quietNow(c *twod.Canvas, animating bool) bool {
	in := c.Input()
	switched := in != d.replay.in
	d.replay.in = in
	return !switched && d.replay.replayedLast && !d.replay.repaint && !animating &&
		(in == nil || !inputfeed.Of(in).Changed())
}

// options are the paint options of d's frames but the clock's, kept for
// the viewport and scale they were made for.
func (d *Document) options() []paint.Option {
	if d.replay.opts == nil || d.replay.optsAt != d.viewport || d.replay.optsScale != d.fonts.Scale() {
		d.replay.opts = []paint.Option{
			paint.Faces(d.face), paint.Fonts(d.fonts), paint.Controls(d.machine),
			paint.Viewport(d.viewport), paint.Images(d.image), paint.Scale(d.fonts.Scale()),
		}
		d.replay.frameOpts = slices.Concat(d.replay.opts, []paint.Option{paint.Each(&d.replay.frame)})
		d.replay.optsAt, d.replay.optsScale = d.viewport, d.fonts.Scale()
	}
	return d.replay.opts
}

// frameOptions are options() and this frame's paint.Frame.
func (d *Document) frameOptions() []paint.Option {
	d.options()
	return d.replay.frameOpts
}

// count adds one to the World's counter name, for the F3 overlay.
func count(c *twod.Canvas, name string) {
	if w := c.World(); w != nil {
		w.Stats().Count(name)
	}
}
