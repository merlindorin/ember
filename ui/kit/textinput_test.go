package kit_test

import (
	"testing"

	"github.com/merlindorin/ember/automation"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/kit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestATextInputIsATextFieldNamedByItsPlaceholder(t *testing.T) {
	in := kit.TextInput("", ui.Placeholder("Tile name"))
	s := show(t, in)
	e := s.element(t, "Tile name")
	assert.Equal(t, automation.RoleTextInput, e.Role)
	assert.True(t, e.Enabled)
	assert.Contains(t, e.Acts, automation.ActType)
}

func TestTypingIntoATextInputEditsItAndEnterSubmits(t *testing.T) {
	in := kit.TextInput("")
	s := show(t, in)
	edits, submits := watch[ui.Edited](s), watch[ui.Submitted](s)
	s.click(in)
	s.in.AddText('h')
	s.in.AddText('i')
	s.frames(2)
	assert.Equal(t, "hi", s.doc.Value(in))
	assert.NotEmpty(t, *edits)
	s.key(input.KeyEnter)
	require.Len(t, *submits, 1)
	assert.Same(t, in, (*submits)[0].Node)
}

func TestAPasswordShowsBullets(t *testing.T) {
	in := kit.TextInput("secret", kit.Password(), ui.ID("pw"))
	s := show(t, in)
	assert.Equal(t, "secret", s.doc.Value(in))
	_, all := s.auto.Snapshot("", "")
	for _, e := range all {
		if e.Role == automation.RoleTextInput {
			assert.Equal(t, "••••••", e.Value, "automation never reads it")
		}
	}
}

func TestASearchFieldSaysSearchAndEscapeEmptiesIt(t *testing.T) {
	in := kit.TextInput("crypt", kit.Search())
	s := show(t, in)
	assert.Equal(t, automation.RoleTextInput, s.element(t, "Search").Role)
	s.click(in)
	s.key(input.KeyEscape)
	assert.Equal(t, "", s.doc.Value(in))
}

func TestAReadOnlyInputIsNotEditable(t *testing.T) {
	in := kit.TextInput("fixed", kit.ReadOnly(), ui.Placeholder("ro"))
	s := show(t, in)
	s.click(in)
	s.in.AddText('x')
	s.frames(2)
	assert.Equal(t, "fixed", s.doc.Value(in))
	assert.False(t, s.element(t, "ro").Enabled)
}

func TestATextInputIsALineAndEmberUIsPadding(t *testing.T) {
	in, l := kit.TextInput("Ag"), kit.Label("Ag")
	show(t, in, l)
	assert.InDelta(t, l.Box().H+10, in.Box().H, 0.5, "a line and 5 above and below, as the old ember/ui")
	assert.Equal(t, float32(200), in.Box().W)
}

func TestEscapeLeavesAPlainTextInput(t *testing.T) {
	in := kit.TextInput("hello")
	s := show(t, in)
	s.click(in)
	require.True(t, s.doc.Focused(in))
	s.key(input.KeyEscape)
	assert.False(t, s.doc.Focused(in))
	assert.Equal(t, "hello", s.doc.Value(in), "kept")
}

func TestMaxLengthStopsTypingAtItsLength(t *testing.T) {
	in := kit.TextInput("He", kit.MaxLength(4))
	area := kit.TextArea("", ui.Attr("maxlength", "2"))
	s := show(t, in, area)
	s.doc.Focus(in)
	for _, r := range "ron!" {
		s.in.AddText(r)
	}
	s.frames(2)
	assert.Equal(t, "Hero", s.doc.Value(in), "four characters, no more")
	s.doc.Focus(area)
	for _, r := range "abc" {
		s.in.AddText(r)
	}
	s.frames(2)
	assert.Equal(t, "ab", s.doc.Value(area), "a textarea's too")
}

func TestASearchFieldsClearButtonEmptiesIt(t *testing.T) {
	field := kit.SearchField("crypt")
	in, clearButton := kit.SearchInput(field), field.Children()[1]
	s := show(t, field)
	changes := watch[ui.Changed](s)
	assert.Equal(t, automation.RoleButton, s.element(t, "clear").Role)
	assert.Greater(t, clearButton.Box().X, in.Box().X+in.Box().W/2, "at the field's right, inside it")
	assert.Less(t, clearButton.Box().X+clearButton.Box().W, in.Box().X+in.Box().W)

	s.click(in)
	s.click(clearButton)
	assert.Equal(t, "", s.doc.Value(in))
	require.NotEmpty(t, *changes)
	assert.Same(t, in, (*changes)[len(*changes)-1].Node)
	assert.True(t, s.doc.Focused(in), "the focus stays in the field")
	assert.Zero(t, clearButton.Box().W, "shown only while there is text")
}

func TestEachSearchFieldsClearButtonClearsItsOwnField(t *testing.T) {
	full, empty := kit.SearchField("crypt"), kit.SearchField("")
	named := kit.SearchField("walls", ui.ID("q"))
	s := show(t, full, empty, named)
	buttonOf := func(f *ui.Node) *ui.Node { return f.Children()[1] }
	_, hidden := buttonOf(empty).Attr("hidden")
	assert.True(t, hidden, "an empty field's button stays hidden")
	assert.Empty(t, kit.SearchInput(empty).ID(), "no id is made up")
	s.click(buttonOf(named))
	s.frames(1)
	assert.Empty(t, s.doc.Value(kit.SearchInput(named)))
	assert.Equal(t, "crypt", s.doc.Value(kit.SearchInput(full)))
}

func TestEachSearchFieldsClearButtonIsScopedUnderItsField(t *testing.T) {
	s := show(t, kit.SearchField("a", ui.Placeholder("Find a tile")), kit.SearchField("b", ui.ID("q")))
	_, all := s.auto.Snapshot("", "")
	var ids []string
	for _, e := range all {
		ids = append(ids, e.ID)
	}
	assert.Contains(t, ids, "find-a-tile/clear")
	assert.Contains(t, ids, "q/clear")
}
