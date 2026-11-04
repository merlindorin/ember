package kit_test

import (
	"testing"
	"testing/fstest"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/align"
	"github.com/merlindorin/ember/ui/internal/dom"
	"github.com/merlindorin/ember/ui/internal/pages"
	"github.com/merlindorin/ember/ui/kit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tipIn lays out a label with a tooltip, its pseudo-classes held at state.
func tipIn(t *testing.T, state dom.State, tipClass ...string) (*ui.Node, *ui.Node) {
	t.Helper()
	d := ui.NewDocument(kit.Theme())
	tip := kit.Label("Saves the tileset", ui.Class(tipClass...))
	target := kit.Label("Save", kit.Tooltip(tip))
	d.Body().Append(ui.Div(ui.Style("padding: 40px 150px"), target))
	if state != 0 {
		pages.Pin(d, target, state)
	}
	d.Layout(geom.V2(400, 300))
	d.Layout(geom.V2(400, 300))
	assert.Empty(t, d.Warnings())
	return target, tip
}

func TestATooltipShowsBesideItsElementOnHoverOrKeyboardFocus(t *testing.T) {
	_, tip := tipIn(t, 0)
	assert.Equal(t, float32(0), tip.Box().W, "hidden until hovered")

	target, tip := tipIn(t, dom.Hover)
	r, tr := target.Box(), tip.Box()
	assert.Equal(t, r.X+r.W+8, tr.X, "8 points right of its element")
	assert.Equal(t, r.Y, tr.Y, "level with its top")
	assert.Greater(t, tr.W, float32(100))

	_, tip = tipIn(t, dom.Focus|dom.FocusVisible)
	assert.Greater(t, tip.Box().W, float32(0), "keyboard focus shows it")

	_, tip = tipIn(t, dom.Focus)
	assert.Equal(t, float32(0), tip.Box().W, "a click's focus doesn't")

	target, tip = tipIn(t, dom.Hover, "kit-left")
	assert.Equal(t, target.Box().X-8, tip.Box().X+tip.Box().W, "class left: 8 points left of it")

	target, tip = tipIn(t, dom.Hover, "kit-below")
	assert.Equal(t, target.Box().Y+target.Box().H+8, tip.Box().Y, "class below: 8 points under it")
	assert.Equal(t, target.Box().X, tip.Box().X)
}

func TestATooltipKeepsItsOwnLook(t *testing.T) {
	d := ui.NewDocument(kit.Theme())
	tip := kit.Label("plain words")
	b := ui.Button(ui.Style("font-weight: bold; text-align: center; color: #ff0000"), ui.Text("Save"), kit.Tooltip(tip))
	d.Body().Append(b)
	pages.Pin(d, b, dom.Hover)
	d.Layout(geom.V2(400, 300))
	d.Layout(geom.V2(400, 300))
	assert.Equal(t, "tooltip", tip.Attrs()[len(tip.Attrs())-1].Value)
	assert.Greater(t, tip.Box().W, float32(0))
}

func TestATipAtTheWindowsEdgeFlipsAndStaysWhileThePointerMoves(t *testing.T) {
	save := kit.Label("Save", kit.Tooltip(kit.Label("Saves the tileset to its file")),
		ui.Style("position: absolute; right: 8px; top: 40px"))
	s := showIn(t, 300, 200, save)
	tip := save.Children()[1]
	at := save.Box()
	s.in.SetMouse(geom.V2(at.X+2, at.Y+at.H/2))
	s.frames(2)
	side, _ := tip.Attr("data-side")
	assert.Equal(t, "left", side)
	r := tip.Box()
	assert.True(t, r.X >= 0 && r.X+r.W <= 300, "%v inside the window", r)
	for x := at.X + 2; x < at.X+at.W-1; x += 4 {
		s.in.SetMouse(geom.V2(x, at.Y+at.H/2))
		s.frames(1)
		side, _ = tip.Attr("data-side")
		assert.Equal(t, "left", side, "no flip-flop at %.0f", x)
	}
	s.in.SetMouse(geom.V2(10, 150))
	s.frames(2)
	_, placed := tip.Attr("data-side")
	assert.False(t, placed, "a hidden tip is placed afresh")
}

func TestAGamesOwnDisplayDoesntOpenWhatKitKeepsClosed(t *testing.T) {
	sheet := fstest.MapFS{"game.css": {Data: []byte(".x { display: flex; }")}}
	d := ui.NewDocument(kit.Theme(), ui.Stylesheet(sheet, "game.css"))
	tip := kit.Label("Saves the file", ui.Class("x"))
	save := kit.Label("Save", kit.Tooltip(tip))
	menu := kit.Menu(kit.MenuItem("Copy"))
	menu.AddClass("x")
	hint := kit.Hint(kit.KeyboardHint("Enter"), kit.GamepadHint("South"))
	for _, c := range hint.Children() {
		c.AddClass("x")
	}
	d.Body().Append(save, ui.Div(menu), hint)
	d.Layout(geom.V2(400, 300))
	assert.Zero(t, tip.Box().W, "a closed tip")
	assert.Zero(t, menu.Box().W, "a closed menu")
	assert.Zero(t, hint.Children()[1].Box().W, "another device's hint")

	pages.Pin(d, save, dom.Hover)
	d.Layout(geom.V2(400, 300))
	assert.Positive(t, tip.Box().W, "hovered, the tip shows")
}

func TestATooltipHangsPastAFixedSizeElementAsMeant(t *testing.T) {
	tip := kit.Label("a tip wider than its slot")
	slot := ui.Button(ui.Style("width: 30px; height: 30px"), ui.Text("·"), kit.Tooltip(tip))
	s := show(t, slot)
	s.in.SetMouse(slot.Box().Center())
	s.frames(2)
	require.NotZero(t, tip.Box().W, "the tip is open")
	_, marked := tip.Attr(align.OverflowOK)
	assert.True(t, marked, "a tooltip carries align's mark")
	for _, f := range align.Check(s.doc) {
		assert.NotEqual(t, "spill", f.Rule, "%v", f)
	}
}
