package kit_test

import (
	"testing"

	"github.com/lantervale/ember/automation"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/kit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestANumberFieldStepsAndKeepsItsRange(t *testing.T) {
	f := kit.NumberField(12, kit.Range(0, 64), kit.Step(0.5), ui.Placeholder("Height"))
	s := show(t, f)
	changes := watch[ui.Changed](s)
	assert.Equal(t, "12", s.doc.Value(f))
	assert.Equal(t, automation.RoleTextInput, s.element(t, "Height").Role)

	s.click(f)
	s.key(input.KeyUp)
	assert.Equal(t, "12.5", s.doc.Value(f), "Up steps by its step")
	s.in.SetKey(input.KeyLeftShift, true)
	s.key(input.KeyDown)
	s.in.SetKey(input.KeyLeftShift, false)
	assert.Equal(t, "7.5", s.doc.Value(f), "Shift steps ten steps")

	for range 3 {
		s.key(input.KeyBackspace)
	}
	for _, r := range "99" {
		s.in.AddText(r)
	}
	s.frames(2)
	s.key(input.KeyEnter)
	assert.Equal(t, "64", s.doc.Value(f), "Enter takes it within its range")
	require.Len(t, *changes, 1)
	assert.Same(t, f, (*changes)[0].Node)
}
