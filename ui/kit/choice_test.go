package kit_test

import (
	"testing"

	"github.com/lantervale/ember/automation"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/kit"
	"github.com/stretchr/testify/assert"
)

func TestAChoiceKeepsOneSelectedByPointerKeysAndAutomation(t *testing.T) {
	c := kit.Choice(0, []*ui.Node{kit.Label("Tiles"), kit.Label("Objects"), kit.Label("Lights")})
	s := show(t, c)
	changes := watch[ui.Changed](s)
	assert.Equal(t, 0, kit.Chosen(c))

	s.click(c.Children()[2])
	assert.Equal(t, 2, kit.Chosen(c))
	assert.Len(t, *changes, 1)
	s.key(input.KeyLeft)
	assert.Equal(t, 1, kit.Chosen(c), "the arrows move the selection")
	s.key(input.KeyRight)
	s.key(input.KeyRight)
	assert.Equal(t, 0, kit.Chosen(c), "and wrap")

	tiles := s.element(t, "Tiles")
	assert.Equal(t, automation.RoleSelect, tiles.Role)
	assert.Equal(t, true, tiles.Value)
	s.auto.Queue(s.element(t, "Lights").ID, automation.ActPress, nil)
	s.frames(2)
	assert.Equal(t, 2, kit.Chosen(c))
}

func TestAChoiceMayStartWithNothingSelected(t *testing.T) {
	c := kit.Choice(-1, []*ui.Node{kit.Label("A"), kit.Label("B")}, ui.Class("kit-segmented"))
	assert.Equal(t, -1, kit.Chosen(c))
	assert.Equal(t, []string{"kit-choice", "kit-segmented"}, c.Classes())
}

func TestSetChosenSelectsAnItemAndChoiceItemsCanBeNamed(t *testing.T) {
	red := kit.ChoiceItem(ui.Span(ui.Class("swatch")), ui.ID("red"), ui.Attr("title", "Red"))
	c := kit.Choice(0, []*ui.Node{red, kit.ChoiceItem(ui.Span(), ui.ID("blue"))})
	s := show(t, c)
	kit.SetChosen(c, 1)
	s.frames(1)
	assert.Equal(t, 1, kit.Chosen(c))
	_, all := s.auto.Snapshot("", "")
	var ids []string
	for _, e := range all {
		ids = append(ids, e.ID)
	}
	assert.Contains(t, ids, "red")
	assert.Contains(t, ids, "blue")
	v, _ := red.Attr("title")
	assert.Equal(t, "Red", v)
	kit.SetChosen(c, -1)
	assert.Equal(t, -1, kit.Chosen(c))
}

func TestAFilledChoiceSharesItsWidthAmongItsItems(t *testing.T) {
	for _, segmented := range []bool{false, true} {
		args := []ui.Arg{kit.Fill()}
		if segmented {
			args = append(args, kit.Segmented())
		}
		c := kit.Choice(0, []*ui.Node{kit.Label("Off"), kit.Label("Light"), kit.Label("Occluders")}, args...)
		s := show(t, kit.Column(ui.Style("width: 300px"), c))
		box, _ := s.doc.BoxOf(c)
		items := c.Children()
		first, _ := s.doc.BoxOf(items[0])
		last, _ := s.doc.BoxOf(items[2])
		assert.InDelta(t, box.W, 300-2, 0.5, "segmented %v: the choice fills its column", segmented)
		left, right := items[0].Box(), items[2].Box()
		assert.InDelta(t, box.X, left.X, 0.5, "segmented %v: its first item starts it", segmented)
		assert.InDelta(t, box.X+box.W, right.X+right.W, 0.5, "segmented %v: its last item ends it", segmented)
		assert.InDelta(t, first.W, last.W, 0.5, "segmented %v: the items share it equally", segmented)
	}
}

func TestAChoiceInAFormKeepsItsOwnWidthUnlessFilled(t *testing.T) {
	own := kit.Choice(0, []*ui.Node{kit.Label("A"), kit.Label("B")})
	filled := kit.Choice(0, []*ui.Node{kit.Label("A"), kit.Label("B")}, kit.Fill())
	slider := kit.Slider(1, kit.Range(0, 2))
	show(t, kit.Form(kit.Field("Own", own), kit.Field("Filled", filled), kit.Field("Slider", slider)))
	column := slider.Box()
	assert.Less(t, own.Box().W, column.W/2, "a choice is as wide as its items")
	assert.InDelta(t, column.X+column.W, filled.Box().X+filled.Box().W, 0.5, "filled, it ends with the column")
	assert.InDelta(t, column.X, own.Box().X, 0.5, "both start it")
}
