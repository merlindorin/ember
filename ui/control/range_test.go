package control_test

import (
	"testing"

	"github.com/merlindorin/ember/ui/control"

	"github.com/stretchr/testify/assert"
)

func TestARangeDefaultsLikeHTML(t *testing.T) {
	r := control.NewRange()
	assert.Equal(t, 0.0, r.Min)
	assert.Equal(t, 100.0, r.Max)
	assert.Equal(t, 1.0, r.Step)
	assert.Equal(t, 50.0, r.Value, "the middle of its range")
}

func TestARangeStepsByItsArrowsAndJumpsByHomeEndAndPages(t *testing.T) {
	r := &control.Range{Min: 0, Max: 10, Step: 2, Value: 4}
	assert.Equal(t, control.Input|control.Changed, r.Do(control.Right))
	assert.Equal(t, 6.0, r.Value)
	assert.Equal(t, control.Input|control.Changed, r.Do(control.Left))
	assert.Equal(t, 4.0, r.Value)
	assert.False(t, r.Handles(control.Up), "up and down move the focus, as ui's slider")
	assert.False(t, r.Handles(control.Down))

	r.Do(control.End)
	assert.Equal(t, 10.0, r.Value)
	assert.Equal(t, control.Events(0), r.Do(control.Right), "at the end nothing changes")
	r.Do(control.Home)
	assert.Equal(t, 0.0, r.Value)

	r = &control.Range{Min: 0, Max: 100, Step: 1, Value: 50}
	r.Do(control.PageUp)
	assert.Equal(t, 60.0, r.Value, "a page is a tenth of the range")
	r.Do(control.PageDown)
	assert.Equal(t, 50.0, r.Value)
	r = &control.Range{Min: 0, Max: 10, Step: 5, Value: 5}
	r.Do(control.PageUp)
	assert.Equal(t, 10.0, r.Value, "a page is at least a step")
}

func TestARangeSnapsAndClampsWhatItIsGiven(t *testing.T) {
	r := &control.Range{Min: 1, Max: 9, Step: 2}
	assert.Equal(t, control.Input|control.Changed, r.SetValue(4.2))
	assert.Equal(t, 5.0, r.Value, "steps count from Min")
	r.SetValue(42)
	assert.Equal(t, 9.0, r.Value)
	r.SetValue(-3)
	assert.Equal(t, 1.0, r.Value)
	assert.Equal(t, control.Events(0), r.SetValue(1))

	r = &control.Range{Min: 0, Max: 1, Step: 0}
	r.SetValue(0.37)
	assert.InDelta(t, 0.37, r.Value, 1e-9, "step any")
}

func TestARangeIsDraggedWithInputThenChangedOnRelease(t *testing.T) {
	r := &control.Range{Min: 0, Max: 100, Step: 1, Value: 10}
	assert.Equal(t, control.Input, r.Press(0.5))
	assert.Equal(t, 50.0, r.Value)
	assert.True(t, r.Dragging())
	assert.Equal(t, control.Input, r.Drag(0.75))
	assert.Equal(t, control.Events(0), r.Drag(0.75))
	assert.Equal(t, control.Input, r.Drag(1.5), "past the track clamps")
	assert.Equal(t, 100.0, r.Value)
	assert.Equal(t, control.Changed, r.Release())
	assert.False(t, r.Dragging())

	r.Press(1)
	assert.Equal(t, control.Events(0), r.Release(), "back where it started: no change")
}

func TestARangeFractionPlacesItsThumb(t *testing.T) {
	assert.Equal(t, 0.25, (&control.Range{Min: 0, Max: 8, Value: 2}).Fraction())
	assert.Equal(t, 0.0, (&control.Range{Min: 5, Max: 5, Value: 5}).Fraction())
}

func TestADisabledRangeIgnoresEverything(t *testing.T) {
	r := &control.Range{Min: 0, Max: 10, Step: 1, Value: 5, Disabled: true}
	assert.Equal(t, control.Events(0), r.Do(control.Right))
	assert.Equal(t, control.Events(0), r.Press(0))
	assert.Equal(t, 5.0, r.Value)
}
