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

// shapes are a pop-up's items.
func shapes() []string { return []string{"Rectangle", "Ellipse", "Polygon"} }

func TestAPopUpIsAComboShowingItsItem(t *testing.T) {
	s := show(t, kit.PopUp(1, shapes(), ui.Attr("aria-label", "Shape")))
	e := s.element(t, "Ellipse")
	assert.Equal(t, automation.RoleCombo, e.Role)
	assert.EqualValues(t, 1, e.Value)
}

func TestAPopUpWithoutAnItemShowsItsPlaceholder(t *testing.T) {
	p := kit.PopUp(-1, shapes(), ui.Placeholder("Shape"))
	s := show(t, p)
	assert.Equal(t, -1, s.doc.Selected(p))
	assert.Equal(t, automation.RoleCombo, s.element(t, "Shape").Role)
}

func TestChoosingAnItemByKeysChangesThePopUp(t *testing.T) {
	p := kit.PopUp(0, shapes())
	s := show(t, p)
	changed := watch[ui.Changed](s)
	s.doc.Focus(p)
	s.key(input.KeyEnter)
	s.key(input.KeyDown)
	s.key(input.KeyDown)
	s.key(input.KeyEnter)
	assert.Equal(t, 2, s.doc.Selected(p))
	require.Len(t, *changed, 1)
	assert.Same(t, p, (*changed)[0].Node)
}

func TestAPopUpIsEmberUIsButtonWithItsChevron(t *testing.T) {
	p := kit.PopUp(0, shapes())
	show(t, p)
	assert.InDelta(t, 25, p.Box().H, 0.5, "a 13-point line, 4 padding, a border")
	wide := kit.PopUp(0, []string{"Rectangle with a long name"})
	show(t, wide)
	assert.Greater(t, wide.Box().W, p.Box().W, "as wide as its widest item and the chevron's room")
}
