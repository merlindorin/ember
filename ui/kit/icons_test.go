package kit_test

import (
	"os"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/kit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testIcons(t *testing.T) *kit.Icons {
	t.Helper()
	set, err := kit.LoadIcons(os.DirFS("testdata/icons"), "icons.ttf", "names.json")
	require.NoError(t, err)
	return set
}

func TestIconsLoadTheirNames(t *testing.T) {
	set := testIcons(t)
	assert.True(t, set.Has("square"))
	assert.False(t, set.Has("circle"))
	assert.Equal(t, "", kit.NewIcon(set, "triangle").Text())
	assert.Equal(t, "", kit.NewIcon(set, "circle").Text(), "a name the set lacks shows nothing")
	_, err := kit.LoadIcons(os.DirFS("testdata/icons"), "icons.ttf", "missing.json")
	assert.Error(t, err)
}

func TestAnIconDrawsInItsFontAtItsElementsSize(t *testing.T) {
	set := testIcons(t)
	d := ui.NewDocument(kit.Theme(), set.Sheet())
	small := kit.NewIcon(set, "square")
	big := kit.NewIcon(set, "square", ui.Style("font-size: 30px"))
	label := kit.Label("Save", kit.Icon(set, "triangle"))
	d.Body().Append(ui.Div(small), ui.Div(big), ui.Div(label))
	d.Layout(geom.V2(400, 300))
	assert.Empty(t, d.Warnings())
	assert.InDelta(t, 15, small.Box().W, 0.01, "the test font's glyphs are one em wide, at the theme's 15px")
	assert.InDelta(t, 30, big.Box().W, 0.01)
	icon := label.Children()[0]
	assert.Equal(t, "", icon.Text())
	assert.Equal(t, "Save", label.Text(), "the icon, then the text")
	assert.Greater(t, label.Box().W, icon.Box().W+15, "room for the gap and Save")
}

func TestAnUnknownIconAddsNothingAndWarns(t *testing.T) {
	set := testIcons(t)
	b := kit.Button("Save", kit.Icon(set, "circle"))
	require.Len(t, b.Children(), 1)
	assert.Equal(t, "Save", b.Text())
	require.Len(t, set.Warnings(), 1)
	assert.Contains(t, set.Warnings()[0].Error(), `"circle"`)
}

func TestSetTextKeepsTheIcon(t *testing.T) {
	set := testIcons(t)
	b := kit.Button("Save", kit.Icon(set, "square"))
	kit.SetText(b, "Saved")
	require.Len(t, b.Children(), 2)
	assert.True(t, b.Children()[0].HasClass("kit-icon"))
	assert.Equal(t, "Saved", b.Children()[1].Text())
}
