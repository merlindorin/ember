package ui_test

import (
	"bytes"
	"image"
	"image/png"
	"testing"
	"testing/fstest"

	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/ui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// emojiSet draws 🔥 and 🐟 as pictures.
func emojiSet(t *testing.T) *twod.EmojiSet {
	t.Helper()
	var b bytes.Buffer
	require.NoError(t, png.Encode(&b, image.NewNRGBA(image.Rect(0, 0, 4, 4))))
	set, err := twod.LoadEmoji(fstest.MapFS{"emoji/1f525.png": {Data: b.Bytes()},
		"emoji/1f41f.png": {Data: b.Bytes()}}, "emoji")
	require.NoError(t, err)
	return set
}

// shortcoded is a text field taking shortcodes and the listbox of its
// suggestions.
func shortcoded() (*ui.Node, *ui.Node) {
	field := ui.Input(ui.Type("text"), ui.Attr("data-shortcodes", ""), ui.Attr("role", "combobox"),
		ui.Attr("aria-controls", "suggestions"))
	list := ui.Div(ui.ID("suggestions"), ui.Attr("role", "listbox"), ui.Attr("hidden", ""))
	return field, list
}

func optionTexts(list *ui.Node) []string {
	var out []string
	for _, o := range list.Children() {
		out = append(out, o.Text())
	}
	return out
}

func TestAClosingColonTurnsAShortcodeIntoItsEmoji(t *testing.T) {
	field, list := shortcoded()
	plain := ui.Input(ui.Type("text"))
	l := liveDocWith(t, []ui.DocumentOption{ui.Emoji(emojiSet(t))}, field, list, plain)
	l.doc.Focus(field)
	l.frames(1)
	l.typed("hot :fire:")
	assert.Equal(t, "hot 🔥", l.doc.Value(field))

	l.doc.Focus(plain)
	l.frames(1)
	l.typed(":fire:")
	assert.Equal(t, ":fire:", l.doc.Value(plain), "only a field taking shortcodes")
}

func TestShortcodeSuggestionsAreWalkedChosenAndClosed(t *testing.T) {
	field, list := shortcoded()
	after := ui.Button(ui.Text("After"))
	l := liveDocWith(t, []ui.DocumentOption{ui.Emoji(emojiSet(t))}, field, list, after)
	l.doc.Focus(field)
	l.frames(1)
	l.typed(":fi")
	require.Equal(t, []string{"🔥 :fire:", "🐟 :fish:"}, optionTexts(list), "matches the face draws, at most six")
	assert.Greater(t, list.Box().H, float32(0), "the list shows")
	assert.Equal(t, "true", attr(field, "aria-expanded"))
	assert.Equal(t, "true", attr(list.Children()[0], "aria-selected"))

	l.key(input.KeyDown)
	assert.Equal(t, "true", attr(list.Children()[1], "aria-selected"), "Down walks the list")
	assert.True(t, l.doc.Focused(field), "the focus stays in the field")
	l.key(input.KeyTab)
	assert.Equal(t, "🐟", l.doc.Value(field), "Tab chooses rather than moving the focus")
	assert.True(t, l.doc.Focused(field))
	assert.Zero(t, list.Box().H, "and the list closes")
	assert.Equal(t, "false", attr(field, "aria-expanded"))

	l.typed(" :fi")
	require.Len(t, list.Children(), 2)
	l.key(input.KeyEscape)
	assert.Zero(t, list.Box().H, "Escape closes it")
	assert.True(t, l.doc.Focused(field))
	l.typed("r")
	assert.Equal(t, []string{"🔥 :fire:"}, optionTexts(list), "a new name opens it again")

	l.clickAt(list.Children()[0])
	assert.Equal(t, "🐟 🔥", l.doc.Value(field), "a click chooses")
	assert.True(t, l.doc.Focused(field), "and leaves the focus in the field")
}
