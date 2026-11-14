package kit_test

import (
	"testing"

	"github.com/lantervale/ember/automation"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/kit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestATextAreaTakesLinesAndIsATextField(t *testing.T) {
	a := kit.TextArea("", ui.Placeholder("Notes"))
	s := show(t, a)
	assert.Equal(t, automation.RoleTextInput, s.element(t, "Notes").Role)
	s.click(a)
	s.in.AddText('a')
	s.frames(1)
	s.key(input.KeyEnter)
	s.in.AddText('b')
	s.frames(2)
	assert.Equal(t, "a\nb", s.doc.Value(a), "Enter breaks the line")
}

func TestATextAreaIsFiveLinesUnlessRowsSays(t *testing.T) {
	five, two := kit.TextArea("x"), kit.TextArea("x", kit.Rows(2))
	show(t, five, two)
	assert.Equal(t, float32(5*18+12), five.Box().H, "the old ember/ui's five lines and 12")
	assert.Equal(t, float32(2*18+12), two.Box().H)
}

func TestNoWrapKeepsLinesWhole(t *testing.T) {
	long := "a line far longer than the area is wide, to see it wrap or not"
	wrapped, whole := kit.TextArea(long), kit.TextArea(long, kit.NoWrap())
	show(t, wrapped, whole)
	assert.Equal(t, wrapped.Box().W, whole.Box().W)
	assert.True(t, whole.HasClass("kit-no-wrap"))
}

func TestSetNoWrapTurnsWrappingOffAndOnAgain(t *testing.T) {
	area := kit.TextArea("a line far longer than the area is wide, to see it wrap or not")
	s := show(t, area)
	lines := func() int {
		s.frames(1)
		b, _ := dom.Data(area).Placed.(*box.Box)
		require.NotNil(t, b, "the area is laid out")
		n := 0
		var count func(b *box.Box)
		count = func(b *box.Box) {
			for _, c := range b.Children {
				if c.Kind == box.TextRun {
					n += len(c.Lines)
				}
				count(c)
			}
		}
		count(b)
		return n
	}
	wrapped := lines()
	kit.SetNoWrap(area, true)
	assert.Equal(t, 1, lines(), "kept whole")
	assert.False(t, kit.Wraps(area))
	kit.SetNoWrap(area, false)
	assert.Greater(t, wrapped, 1)
	assert.Equal(t, wrapped, lines(), "wrapped again")
	assert.True(t, kit.Wraps(area))
}
