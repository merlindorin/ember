package layout //nolint:testpackage // tests the layout tree and its unexported helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/merlindorin/ember/ui/internal/box"
	"github.com/merlindorin/ember/ui/internal/style"
)

func TestARunsTextOnALineIsRoundedUpToTheLayoutUnit(t *testing.T) {
	p := newPen(64, 0)
	p.step(piece{kind: word, run: 0, width: 10.1}, 0)
	p.step(piece{kind: gap, run: 0, width: 3.3}, 0)
	assert.InDelta(t, 13.4, p.at(), 1e-5, "inside a run positions stay as shaped")
	p.step(piece{kind: word, run: 0, width: 20}, 0)
	assert.Equal(t, snapUp(33.4, 64), p.end(), "one rounding for the whole run, not per word")
	at := p.step(piece{kind: word, run: 1, width: 5}, 0)
	assert.Equal(t, snapUp(33.4, 64), at, "the next run starts at the snapped end")
}

func TestEdgesAndAtomsAreItemsOfTheirOwn(t *testing.T) {
	p := newPen(64, 0)
	p.step(piece{kind: word, run: 0, width: 1.01}, 0)
	at := p.step(piece{kind: edge, run: 0, width: 4}, 0)
	assert.Equal(t, float32(65)/64, at)
	p.step(piece{kind: word, run: 0, width: 1.01}, 0)
	assert.Equal(t, float32(65)/64+4+float32(65)/64, p.end(), "the same run on both sides of an edge is two items")
}

func TestJustifyGapsComeAfterTheRounding(t *testing.T) {
	p := newPen(64, 0)
	p.step(piece{kind: word, run: 0, width: 1.01}, 3)
	p.step(piece{kind: gap, run: 0, width: 1}, 3)
	p.step(piece{kind: word, run: 0, width: 1}, 3)
	assert.Equal(t, snapUp(3.01, 64)+3, p.end())
}

func TestTheUnitFollowsTheDeviceScale(t *testing.T) {
	assert.Equal(t, float32(1.0078125), snapUp(1.001, 128), "1/128 pt at scale 2")
	assert.Equal(t, float32(128), unitsPerPoint(scaledFonts{2}))
	assert.Equal(t, float32(64), unitsPerPoint(ahemFonts{}))
}

type scaledFonts struct{ s float32 }

func (f scaledFonts) Scale() float32                   { return f.s }
func (scaledFonts) Metrics(style.Font) box.Metrics     { return box.Metrics{} }
func (scaledFonts) Advance(style.Font, string) float32 { return 0 }
