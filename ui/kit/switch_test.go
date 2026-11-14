package kit_test

import (
	"testing"

	"github.com/lantervale/ember/automation"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/kit"
	"github.com/stretchr/testify/assert"
)

func TestASwitchIsAToggleNamedByItsText(t *testing.T) {
	sw := kit.Switch("Music", ui.Checked(true))
	s := show(t, sw)
	e := s.element(t, "Music")
	assert.Equal(t, automation.RoleToggle, e.Role)
	assert.Equal(t, true, e.Value)
}

func TestClickingASwitchOrItsTextFlipsIt(t *testing.T) {
	sw := kit.Switch("Music")
	s := show(t, sw)
	changes := watch[ui.Changed](s)
	track := kit.SwitchTrack(sw)
	s.click(track)
	assert.True(t, s.doc.Checked(track))
	s.clickAt(sw.Box().Center().Add(sw.Box().Center().Sub(track.Box().Center()).Scale(0.5)))
	assert.False(t, s.doc.Checked(track))
	assert.Len(t, *changes, 2)
	s.doc.Focus(track)
	s.frames(1)
	s.key(input.KeySpace)
	assert.True(t, s.doc.Checked(track))
}

func TestASwitchIsEmberUIsTrack(t *testing.T) {
	sw := kit.Switch("Music")
	l := kit.Label("Music")
	show(t, sw, l)
	track := kit.SwitchTrack(sw).Box()
	assert.Equal(t, float32(34), track.W)
	assert.Equal(t, float32(18), track.H)
	assert.InDelta(t, 34+8+l.Box().W, sw.Box().W, 0.5)
}

func TestADisabledSwitchNeverFlips(t *testing.T) {
	sw := kit.Switch("Music", ui.Disabled())
	s := show(t, sw)
	s.click(sw)
	assert.False(t, s.doc.Checked(kit.SwitchTrack(sw)))
	assert.False(t, s.element(t, "Music").Enabled)
}
