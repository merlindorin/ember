package twod //nolint:testpackage // white-box: shares twod's test helpers, reaches unexported state

import (
	"bytes"
	"image"
	"image/png"
	"testing"
	"testing/fstest"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/internal/draw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/font/gofont/gomono"
)

func pngBytes(t *testing.T) []byte {
	t.Helper()
	var b bytes.Buffer
	require.NoError(t, png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 4, 4))))
	return b.Bytes()
}

func TestEmojiLoadByTheirCodepointNames(t *testing.T) {
	data := pngBytes(t)
	fsys := fstest.MapFS{
		"emoji/1f468-200d-1f469-200d-1f467.png": {Data: data},
		"emoji/emoji_u1f1eb_1f1f7.png":          {Data: data},
		"emoji/2764-fe0f.png":                   {Data: data},
		"emoji/31-20e3.png":                     {Data: data},
		"emoji/README.md":                       {Data: []byte("not a picture")},
		"emoji/notes.png":                       {Data: data},
	}
	set, err := LoadEmoji(fsys, "emoji")
	require.NoError(t, err)
	assert.Equal(t, 4, set.Len(), "named PNGs only")
	assert.True(t, set.Has("👨‍👩‍👧"), "Twemoji's names")
	assert.True(t, set.Has("🇫🇷"), "Noto's names")
	assert.True(t, set.Has("❤️") && set.Has("❤"), "FE0F ignored on both sides")
	assert.True(t, set.Has("1️⃣"), "a keycap")
	assert.False(t, set.Has("😀"))
	_, err = LoadEmoji(fsys, "missing")
	assert.Error(t, err)
}

func TestAFaceTakesFallbacksAndEmoji(t *testing.T) {
	monoFace, err := NewFace(gomono.TTF)
	require.NoError(t, err)
	set, err := LoadEmoji(fstest.MapFS{"e/1f525.png": {Data: pngBytes(t)}}, "e")
	require.NoError(t, err)
	face := Regular().With(Fallback(monoFace), Emoji(set))
	assert.Equal(t, []*Face{monoFace}, face.Fallbacks())
	assert.Same(t, set, face.Emoji())
	assert.Empty(t, Regular().Fallbacks(), "the engine face is unchanged")
	assert.Equal(t, float32(24), face.Measure("🔥", 24).X, "a picture is as wide as the text is tall")
	loaded, err := LoadFace(fstest.MapFS{"m.ttf": {Data: gomono.TTF}}, "m.ttf", Fallback(Regular()))
	require.NoError(t, err)
	assert.Equal(t, []*Face{Regular()}, loaded.Fallbacks())
}

func TestEmojiDrawInlineUntinted(t *testing.T) {
	w, win, list := app()
	set, err := LoadEmoji(fstest.MapFS{"e/1f525.png": {Data: pngBytes(t)}}, "e")
	require.NoError(t, err)
	win.Add(
		NewText("hot 🔥", Typeface(Regular().With(Emoji(set))), FontSize(20), Color(color.RGB(1, 0, 0)), At(100, 100)),
	)
	w.RunFrames(1)
	var picture []geom.Vec4
	for _, v := range list.Unindexed() {
		if v.Shape.X == draw.KindImage {
			picture = append(picture, geom.Vec4{X: v.Color.R, Y: v.Color.G, Z: v.Color.B, W: v.Color.A})
		}
	}
	require.Len(t, picture, 6, "the fire is one picture")
	assert.Equal(
		t,
		geom.Vec4{X: 1, Y: 1, Z: 1, W: 1},
		picture[0],
		"white: a colour emoji is not tinted by the red text",
	)
}

func TestShortcodesExpandInTextsThatAskForIt(t *testing.T) {
	assert.Equal(t, "gg 🔥 :nope:", Shortcodes("gg :fire: :nope:"))
	plain := NewText(":fire:")
	expanded := NewText(":fire: :+1:", ExpandShortcodes())
	assert.Equal(t, []string{":fire:"}, plain.Lines(), "only when asked")
	assert.Equal(t, []string{"🔥 👍"}, expanded.Lines())
	assert.Equal(t, ":fire: :+1:", expanded.Text(), "the text keeps what was written")
	assert.True(t, expanded.ExpandShortcodes())
	expanded.SetText(":cry:")
	assert.Equal(t, []string{"😢"}, expanded.Lines(), "a new text is expanded again")
}
