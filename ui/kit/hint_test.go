package kit_test

import (
	"testing"

	"github.com/merlindorin/ember/automation"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/kit"
	"github.com/stretchr/testify/assert"
)

func pickHint() *ui.Node {
	return kit.Hint(kit.KeyboardHint("Enter to choose"), kit.MouseHint("Click a card"), kit.GamepadHint("South to choose"))
}

// shownHint is the text of hint's span that shows.
func shownHint(t *testing.T, hint *ui.Node) string {
	t.Helper()
	shown := ""
	for _, c := range hint.Children() {
		if !c.IsText() && c.Box().W > 0 {
			assert.Empty(t, shown, "one text shows")
			shown = c.Text()
		}
	}
	return shown
}

func hintDoc(hint ...*ui.Node) *ui.Document {
	d := ui.NewDocument(kit.Theme())
	d.Body().Append(hint...)
	d.Layout(geom.V2(400, 300))
	return d
}

func TestAHintReadsForTheDeviceUsedLast(t *testing.T) {
	h := pickHint()
	d := hintDoc(h)
	assert.Equal(t, "Enter to choose", shownHint(t, h), "the keyboard's before any input")
	for dev, want := range map[string]string{"keyboard": "Enter to choose", "mouse": "Click a card",
		"gamepad": "South to choose", "touch": "Enter to choose"} {
		d.HTML().SetAttr("data-input-device", dev)
		d.Layout(geom.V2(400, 300))
		assert.Equal(t, want, shownHint(t, h), dev)
	}
}

func TestAPanelsOwnDeviceWinsOverTheWindows(t *testing.T) {
	inner, outer := pickHint(), pickHint()
	d := hintDoc(outer, ui.Div(ui.Attr("data-input-device", "gamepad"), inner))
	d.HTML().SetAttr("data-input-device", "mouse")
	d.Layout(geom.V2(400, 300))
	assert.Equal(t, "Click a card", shownHint(t, outer))
	assert.Equal(t, "South to choose", shownHint(t, inner))

	keys := pickHint()
	d.Body().Append(ui.Div(ui.Attr("data-input-device", "keyboard"), keys))
	d.HTML().SetAttr("data-input-device", "gamepad")
	d.Layout(geom.V2(400, 300))
	assert.Equal(t, "Enter to choose", shownHint(t, keys), "a keyboard panel inside a gamepad window")
}

func TestSetHintChangesAndAddsATextAndHintTextReadsIt(t *testing.T) {
	h := pickHint()
	kit.SetHint(h, input.GamepadDevice, "South to choose  ·  North to reroll")
	kit.SetHint(h, input.TouchDevice, "Tap a card")
	assert.Equal(t, "South to choose  ·  North to reroll", kit.HintText(h, input.GamepadDevice))
	assert.Equal(t, "Tap a card", kit.HintText(h, input.TouchDevice))
	assert.Len(t, h.Children(), 4)
	d := hintDoc(h)
	d.HTML().SetAttr("data-input-device", "touch")
	d.Layout(geom.V2(400, 300))
	assert.Equal(t, "Tap a card", shownHint(t, h))
}

func TestAutomationReadsOnlyTheShownHint(t *testing.T) {
	h := pickHint()
	h.SetID("hint")
	s := show(t, h)
	s.in.SetMouse(geom.V2(390, 290))
	s.frames(2)
	_, all := s.auto.Snapshot("", "")
	var label string
	for _, e := range all {
		if e.ID == "hint" {
			label = e.Label
		}
	}
	assert.Equal(t, "Click a card", label)
	_ = automation.RoleLabel
}
