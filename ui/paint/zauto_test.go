package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"testing"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/ui/internal/box"
	"github.com/merlindorin/ember/ui/internal/style"
	"github.com/stretchr/testify/assert"
)

func positioned(c color.Color, z int, auto bool, children ...*box.Box) *box.Box {
	return el(0, 0, 50, 20, func(s *style.Computed) {
		s.Background, s.Position, s.ZIndex, s.ZAuto = c, style.Relative, z, auto
	}, children...)
}

func TestAZIndexInsideAZAutoBoxRisesAboveItsLaterSiblings(t *testing.T) {
	tip := positioned(red, 100, false)
	r := newRecorder()
	Paint(r, tree(el(0, 0, 400, 300, nil,
		positioned(white, 0, true, tip),
		positioned(blue, 0, true),
	)))
	assert.Equal(t, "white blue red", r.order(names), "the tip's z-index 100 counts in the root's context")
}

func TestZAutoBoxesAndTheirPositionedChildrenPaintInTreeOrder(t *testing.T) {
	r := newRecorder()
	Paint(r, tree(el(0, 0, 400, 300, nil,
		positioned(white, 0, true, positioned(green, 0, true)),
		positioned(blue, 0, true),
		positioned(black, -1, false),
	)))
	assert.Equal(t, "black white green blue", r.order(names), "negative first, then tree order")
}

func TestAZAutoBoxsFlowContentPaintsWithIt(t *testing.T) {
	inner := el(0, 0, 10, 10, bg(green))
	r := newRecorder()
	Paint(r, tree(el(0, 0, 400, 300, nil,
		positioned(white, 0, true, inner),
		positioned(blue, 0, true),
	)))
	assert.Equal(t, "white green blue", r.order(names), "its own flow stays with it, under the later sibling")
}
