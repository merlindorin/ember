package kit_test

import (
	"testing"

	"github.com/merlindorin/ember/automation"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/kit"
	"github.com/stretchr/testify/assert"
)

func TestAFormLinesItsLabelsAndFieldsUp(t *testing.T) {
	name, volume := kit.TextInput("Ayla"), kit.TextInput("0.6")
	music := kit.Check("On")
	sized := kit.TextInput("", ui.Style("width: 90px"))
	rule := kit.Separator(ui.Class("kit-divider"))
	form := kit.Form(kit.Field("Name", name), kit.Field("Volume", volume), rule, kit.Field("Music", music),
		kit.Field("Seed", sized))
	form.SetStyle("width: 320px")
	s := show(t, form)

	nameLabel, musicLabel := kit.FieldLabel(name), kit.FieldLabel(music)
	assert.Equal(t, "Name", nameLabel.Text())
	assert.InDelta(t, nameLabel.Box().X+nameLabel.Box().W, musicLabel.Box().X+musicLabel.Box().W, 0.01, "one label column")
	assert.Equal(t, name.Box().X, music.Box().X, "one field column")
	assert.InDelta(t, form.Box().X+form.Box().W, name.Box().X+name.Box().W, 0.01, "fields stretch across their column")
	assert.Equal(t, name.Box().W, volume.Box().W)
	assert.Equal(t, float32(90), sized.Box().W, "a field with a width of its own keeps it")
	assert.InDelta(t, form.Box().W, rule.Box().W, 0.01, "a separator runs across the form")
	assert.Greater(t, rule.Box().Y, volume.Box().Y)

	kit.SetLabelWidth(form, 150)
	kit.SetFieldWidth(form, 120)
	s.frames(2)
	assert.GreaterOrEqual(t, name.Box().X-form.Box().X, float32(150), "the label column is at least 150 wide")
	assert.Equal(t, float32(120), name.Box().W, "fields take the form's field width")
	assert.Equal(t, float32(90), sized.Box().W)

	s.doc.Focus(name)
	s.frames(1)
	s.key(input.KeyTab)
	assert.True(t, s.doc.Focused(volume), "the focus goes row by row")
}

func TestAFormsFieldsAreNamedByTheirLabels(t *testing.T) {
	named := kit.TextInput("", ui.ID("player"))
	s := show(t,
		kit.Form(kit.Field("Name", kit.TextInput("Ayla")), kit.Field("Volume", kit.Slider(60)), kit.Field("Who", named)))
	_, all := s.auto.Snapshot("", "")
	byRole := map[string]automation.Element{}
	for _, e := range all {
		byRole[e.Role+"/"+e.Label] = e
	}
	name, ok := byRole[automation.RoleTextInput+"/Name"]
	assert.True(t, ok, "%v", all)
	assert.Contains(t, name.ID, "name")
	_, ok = byRole[automation.RoleSlider+"/Volume"]
	assert.True(t, ok, "%v", all)
	assert.Equal(t, "player", named.ID(), "a control keeps its own id")
	_, ok = byRole[automation.RoleTextInput+"/Who"]
	assert.True(t, ok, "%v", all)
}
