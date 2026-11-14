package control_test

import (
	"testing"

	"github.com/lantervale/ember/ui/control"

	"github.com/stretchr/testify/assert"
)

func fruits() *control.Select {
	return control.NewSelect(
		control.Option{Label: "Apple", Value: "a"},
		control.Option{Label: "Banana", Value: "b", Disabled: true},
		control.Option{Label: "Cherry", Value: "c"},
		control.Option{Label: "Coconut", Value: "co"},
	)
}

func TestASelectStartsOnItsFirstEnabledOption(t *testing.T) {
	s := fruits()
	assert.Equal(t, 0, s.Selected)
	assert.Equal(t, "a", s.Value())
	s = control.NewSelect(control.Option{Label: "x", Disabled: true}, control.Option{Label: "y"})
	assert.Equal(t, 1, s.Selected)
	assert.Equal(t, "y", s.Value(), "no Value: the label")
	assert.Equal(t, -1, control.NewSelect().Selected)
}

func TestAClosedSelectOpensOnActivateAndLeavesArrowsToTheFocus(t *testing.T) {
	s := fruits()
	assert.False(t, s.Handles(control.Down))
	assert.Equal(t, control.Events(0), s.Do(control.Activate))
	assert.True(t, s.Open)
	assert.Equal(t, 0, s.Highlight)
	assert.True(t, s.Handles(control.Down), "open, it takes the arrows")
}

func TestAnOpenSelectWalksEnabledOptionsAndChoosesOnActivate(t *testing.T) {
	s := fruits()
	s.Do(control.Activate)
	s.Do(control.Down)
	assert.Equal(t, 2, s.Highlight, "skips the disabled Banana")
	s.Do(control.Down)
	s.Do(control.Down)
	assert.Equal(t, 3, s.Highlight, "stops at the last")
	s.Do(control.Home)
	assert.Equal(t, 0, s.Highlight)
	s.Do(control.End)
	assert.Equal(t, 3, s.Highlight)
	s.Do(control.Up)
	assert.Equal(t, control.Input|control.Changed, s.Do(control.Activate))
	assert.Equal(t, 2, s.Selected)
	assert.False(t, s.Open)

	s.Do(control.Activate)
	assert.Equal(t, 2, s.Highlight, "reopens on the chosen one")
	assert.Equal(t, control.Events(0), s.Do(control.Activate), "the same one again changes nothing")
}

func TestBackClosesASelectWithoutChoosing(t *testing.T) {
	s := fruits()
	s.Do(control.Activate)
	s.Do(control.Down)
	assert.True(t, s.Handles(control.Back))
	assert.Equal(t, control.Events(0), s.Do(control.Back))
	assert.False(t, s.Open)
	assert.Equal(t, 0, s.Selected)
	assert.False(t, s.Handles(control.Back), "closed, Back goes up")
}

func TestClickingAnOptionChoosesItButNotADisabledOne(t *testing.T) {
	s := fruits()
	s.Click()
	assert.True(t, s.Open)
	assert.Equal(t, control.Events(0), s.ClickOption(1))
	assert.True(t, s.Open)
	assert.Equal(t, control.Input|control.Changed, s.ClickOption(3))
	assert.Equal(t, "co", s.Value())
	assert.False(t, s.Open)
	s.Click()
	s.ClickOutside()
	assert.False(t, s.Open)
}

func TestTypingJumpsToTheNextOptionStartingSo(t *testing.T) {
	s := fruits()
	s.Do(control.Activate)
	s.TypeAhead("c")
	assert.Equal(t, 2, s.Highlight)
	s.TypeAhead("c")
	assert.Equal(t, 3, s.Highlight, "the same letter goes on to the next")
	s.TypeAhead("C")
	assert.Equal(t, 2, s.Highlight, "and wraps, any case")
	s.TypeAhead("b")
	assert.Equal(t, 2, s.Highlight, "disabled options are never highlighted")

	closed := fruits()
	assert.Equal(t, control.Input|control.Changed, closed.TypeAhead("c"), "closed, typing chooses")
	assert.Equal(t, 2, closed.Selected)
}

func TestADisabledSelectNeverOpens(t *testing.T) {
	s := fruits()
	s.Disabled = true
	s.Do(control.Activate)
	s.Click()
	assert.False(t, s.Open)
}
