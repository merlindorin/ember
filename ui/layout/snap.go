package layout

import (
	"math"

	"github.com/lantervale/ember/ui/internal/box"
)

// unitsPerPoint is Chrome's layout unit, 1/64 of a device pixel, in points: with the
// device scale zoomed into layout, a point holds 64 × scale units.
func unitsPerPoint(fonts box.Fonts) float32 {
	if s, ok := fonts.(interface{ Scale() float32 }); ok && s.Scale() > 0 {
		return 64 * s.Scale()
	}
	return 64
}

// snapUp is w rounded up to the layout unit, Blink's LayoutUnit::FromFloatCeil.
func snapUp(w, unit float32) float32 {
	return float32(math.Ceil(float64(w*unit))) / unit
}

// pen walks a line's pieces as Chrome places them: one run's text on the line is one
// item, whose width is its shaped width rounded up to the layout unit (ShapeResult's
// SnappedWidth), so at each item's end the pen jumps to its snapped end; within it,
// positions stay as shaped, and justify's gaps come after the rounding. Edges, atoms,
// markers and anchors are items of their own.
type pen struct {
	unit  float32
	start float32
	raw   float32
	extra float32
	run   int
}

func newPen(unit, x float32) pen { return pen{unit: unit, start: x, run: -1} }

func isText(k pieceKind) bool {
	return k == word || k == gap || k == emergency || k == breakHere
}

// at is where the next piece starts.
func (p *pen) at() float32 { return p.start + p.raw + p.extra }

// close ends the open text item at its snapped end.
func (p *pen) close() {
	if p.run >= 0 {
		p.start += snapUp(p.raw, p.unit) + p.extra
		p.raw, p.extra, p.run = 0, 0, -1
	}
}

// step places q, a gap widened by perGap when justified, and returns where it starts.
func (p *pen) step(q piece, perGap float32) float32 {
	if !isText(q.kind) {
		p.close()
		at := p.start
		if q.kind != newline {
			p.start += q.width
		}
		return at
	}
	if q.run != p.run {
		p.close()
		p.run = q.run
	}
	at := p.at()
	p.raw += q.width
	if q.kind == gap {
		p.extra += perGap
	}
	return at
}

// end is where the line ends once its last item is closed.
func (p pen) end() float32 {
	p.close()
	return p.start
}
