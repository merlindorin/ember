package control_test

import (
	"testing"

	"github.com/lantervale/ember/ui/control"

	"github.com/lantervale/ember/input"
	"github.com/stretchr/testify/assert"
)

func TestKeysAndPadButtonsAreTheIntentsUIUses(t *testing.T) {
	for _, c := range []struct {
		key  input.Key
		want control.Intent
	}{
		{input.KeyEnter, control.Activate},
		{input.KeyPadEnter, control.Activate},
		{input.KeySpace, control.Activate},
		{input.KeyEscape, control.Back},
		{input.KeyUp, control.Up},
		{input.KeyDown, control.Down},
		{input.KeyLeft, control.Left},
		{input.KeyRight, control.Right},
		{input.KeyHome, control.Home},
		{input.KeyEnd, control.End},
		{input.KeyPageUp, control.PageUp},
		{input.KeyPageDown, control.PageDown},
		{input.KeyA, control.None},
	} {
		assert.Equal(t, c.want, control.KeyIntent(c.key), c.key.String())
	}
	for _, c := range []struct {
		button input.GamepadButton
		want   control.Intent
	}{
		{input.GamepadSouth, control.Activate},
		{input.GamepadEast, control.Back},
		{input.GamepadUp, control.Up},
		{input.GamepadLeftStickUp, control.Up},
		{input.GamepadDown, control.Down},
		{input.GamepadLeftStickDown, control.Down},
		{input.GamepadLeft, control.Left},
		{input.GamepadLeftStickLeft, control.Left},
		{input.GamepadRight, control.Right},
		{input.GamepadLeftStickRight, control.Right},
		{input.GamepadNorth, control.None},
	} {
		assert.Equal(t, c.want, control.PadIntent(c.button), c.button.String())
	}
}

func TestEventsAreASetOfWhatHappened(t *testing.T) {
	e := control.Input | control.Changed
	assert.True(t, e.Has(control.Input))
	assert.True(t, e.Has(control.Changed))
	assert.False(t, e.Has(control.Clicked))
	assert.False(t, control.Events(0).Has(control.Input))
}

func TestAButtonIsClickedByActivateOrAClickUnlessDisabled(t *testing.T) {
	b := &control.Button{}
	assert.Equal(t, control.Clicked, b.Do(control.Activate))
	assert.Equal(t, control.Clicked, b.Click())
	assert.Equal(t, control.Events(0), b.Do(control.Back))
	assert.False(t, b.Handles(control.Down), "arrows move the focus")

	b.Disabled = true
	assert.Equal(t, control.Events(0), b.Do(control.Activate))
	assert.Equal(t, control.Events(0), b.Click())
}

func TestACheckboxTogglesOnActivateAndClickAndClearsIndeterminate(t *testing.T) {
	c := &control.Checkbox{Indeterminate: true}
	assert.Equal(t, control.Input|control.Changed, c.Do(control.Activate))
	assert.True(t, c.Checked)
	assert.False(t, c.Indeterminate)
	assert.Equal(t, control.Input|control.Changed, c.Click())
	assert.False(t, c.Checked)

	c.Disabled = true
	assert.Equal(t, control.Events(0), c.Click())
	assert.False(t, c.Checked)
}

func TestDetailsToggleFromTheirSummary(t *testing.T) {
	d := &control.Details{}
	assert.Equal(t, control.Toggled, d.Do(control.Activate))
	assert.True(t, d.Open)
	assert.Equal(t, control.Toggled, d.Click())
	assert.False(t, d.Open)
	assert.Equal(t, control.Events(0), d.SetOpen(false), "no change, no event")
	assert.Equal(t, control.Toggled, d.SetOpen(true))
}
