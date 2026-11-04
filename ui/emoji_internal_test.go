package ui

import (
	"bytes"
	"image"
	"image/png"
	"testing"
	"testing/fstest"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/internal/emojinames"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/ui/internal/dom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fireSet(t *testing.T) *twod.EmojiSet {
	t.Helper()
	var b bytes.Buffer
	require.NoError(t, png.Encode(&b, image.NewNRGBA(image.Rect(0, 0, 4, 4))))
	set, err := twod.LoadEmoji(fstest.MapFS{"emoji/1f525.png": {Data: b.Bytes()}}, "emoji")
	require.NoError(t, err)
	return set
}

func TestAnEmojiIsOneEmWideAndDrawsAsAPicture(t *testing.T) {
	fire := emojinames.Expand(":fire:")
	require.Equal(t, "🔥", fire)
	set := fireSet(t)

	d := NewDocument(Emoji(set))
	pic := Span(Style("font-size: 20px"), Text(fire))
	word := Span(Style("font-size: 20px"), Text("a"+fire+"b"))
	d.Body().Append(Div(pic), Div(word))
	d.Layout(geom.V2(400, 300))
	assert.InDelta(t, 20, pic.Box().W, 0.01, "one em at 20px")
	a := Span(Style("font-size: 20px"), Text("ab"))
	d.Body().Append(Div(a))
	d.Layout(geom.V2(400, 300))
	assert.InDelta(t, a.Box().W+20, word.Box().W, 0.01, "text on either side keeps its own width")

	st := dom.Data(pic).Style
	assert.True(t, d.face(st.Font).Draws(fire), "paint's face draws it")

	plain := NewDocument()
	assert.False(t, plain.face(st.Font).Draws(fire), "no picture without Emoji")
}
