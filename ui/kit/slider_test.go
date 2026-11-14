package kit_test

import (
	"strconv"
	"testing"

	"github.com/lantervale/ember/automation"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/kit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func slider(t *testing.T, s *shown) automation.Element {
	t.Helper()
	_, all := s.auto.Snapshot("", "")
	for _, e := range all {
		if e.Role == automation.RoleSlider {
			return e
		}
	}
	t.Fatal("no slider")
	return automation.Element{}
}

func TestASliderIsASliderForAutomationAtItsValue(t *testing.T) {
	s := show(t, ui.Label(ui.Text("Volume"), kit.Slider(30)))
	e := slider(t, s)
	assert.Equal(t, "Volume", e.Label, "named by its label")
	assert.EqualValues(t, 30, e.Value)
	assert.True(t, e.Enabled)
}

func TestASliderStepsWithinItsRangeByTheArrows(t *testing.T) {
	sl := kit.Slider(5, kit.Range(0, 10), kit.Step(1))
	s := show(t, sl)
	changed := watch[ui.Changed](s)
	s.doc.Focus(sl)
	s.key(input.KeyRight)
	assert.Equal(t, "6", s.doc.Value(sl))
	require.NotEmpty(t, *changed)
	for range 10 {
		s.key(input.KeyRight)
	}
	assert.Equal(t, "10", s.doc.Value(sl), "never past its max")
}

func TestAClickPutsTheSliderThere(t *testing.T) {
	sl := kit.Slider(0, kit.Step(1))
	s := show(t, sl)
	r := sl.Box()
	s.clickAt(geom.V2(r.X+0.75*r.W, r.Y+r.H/2))
	v, err := strconv.ParseFloat(s.doc.Value(sl), 64)
	require.NoError(t, err)
	assert.InDelta(t, 75, v, 1)
}

func TestASliderIsEmberUIsSize(t *testing.T) {
	sl := kit.Slider(50)
	show(t, sl)
	assert.Equal(t, float32(160), sl.Box().W)
	assert.Equal(t, float32(20), sl.Box().H)
}

func TestDraggingASliderMovesItAndChangesItOnRelease(t *testing.T) {
	sl := kit.Slider(0, kit.Step(1))
	s := show(t, sl)
	changed := watch[ui.Changed](s)
	r := sl.Box()
	s.in.SetMouse(geom.V2(r.X+0.2*r.W, r.Y+r.H/2))
	s.frames(1)
	s.in.SetMouseButton(input.MouseLeft, true)
	s.frames(1)
	s.in.SetMouse(geom.V2(r.X+0.6*r.W, r.Y+r.H/2))
	s.frames(2)
	assert.Equal(t, "60", s.doc.Value(sl), "it follows the pointer")
	assert.Empty(t, *changed, "nothing committed while held")
	s.in.SetMouse(geom.V2(r.X+2*r.W, r.Y+3*r.H))
	s.frames(2)
	assert.Equal(t, "100", s.doc.Value(sl), "past its end, at its max")
	s.in.SetMouseButton(input.MouseLeft, false)
	s.frames(2)
	assert.Len(t, *changed, 1, "one change on release")
}
