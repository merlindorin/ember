package kit_test

import (
	"testing"

	"github.com/merlindorin/ember/automation"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/kit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestACheckIsAToggleNamedByItsText(t *testing.T) {
	c := kit.Check("Solid", ui.Checked(true))
	s := show(t, c)
	e := s.element(t, "Solid")
	assert.Equal(t, automation.RoleToggle, e.Role)
	assert.Equal(t, true, e.Value)
	assert.ElementsMatch(t, []string{automation.ActPress, automation.ActSet}, e.Acts)
}

func TestClickingTheBoxOrItsTextTogglesIt(t *testing.T) {
	c := kit.Check("Solid")
	s := show(t, c)
	changes := watch[ui.Changed](s)
	box := kit.CheckBox(c)
	require.NotNil(t, box)
	s.click(box)
	assert.True(t, s.doc.Checked(box))
	s.clickAt(geom.V2(c.Box().X+c.Box().W-4, c.Box().Center().Y))
	assert.False(t, s.doc.Checked(box), "the text is its label")
	assert.Len(t, *changes, 2)
	assert.Same(t, box, (*changes)[0].Node)
}

func TestSpaceTogglesAFocusedCheck(t *testing.T) {
	c := kit.Check("Solid")
	s := show(t, c)
	s.doc.Focus(kit.CheckBox(c))
	s.frames(1)
	s.key(input.KeySpace)
	assert.True(t, s.doc.Checked(kit.CheckBox(c)))
}

func TestADisabledCheckNeverToggles(t *testing.T) {
	c := kit.Check("Solid", ui.Disabled())
	s := show(t, c)
	s.click(c)
	assert.False(t, s.doc.Checked(kit.CheckBox(c)))
	assert.False(t, s.element(t, "Solid").Enabled)
}

func TestACheckIsEmberUIsSixteenPointBoxAndItsText(t *testing.T) {
	c := kit.Check("Solid")
	l := kit.Label("Solid")
	show(t, c, l)
	box := kit.CheckBox(c).Box()
	assert.Equal(t, float32(16), box.W)
	assert.Equal(t, float32(16), box.H)
	assert.InDelta(t, 16+8+l.Box().W, c.Box().W, 0.5, "the box, 8 points, then the text")
}
