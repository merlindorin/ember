package kit_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/automation"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/kit"
)

func TestAWindowHasItsBarThenToolbarBodyAndFooter(t *testing.T) {
	text := kit.Label("Tile 12")
	tb, ft := kit.Toolbar(kit.Button("Save")), kit.Footer(kit.Button("Close"))
	w := kit.Window("Inspector", text, ft, tb)
	s := show(t, w)
	s.doc.Show(w)
	s.frames(2)
	bar := w.Children()[0]
	assert.Same(t, tb, w.Children()[1])
	assert.True(t, w.Children()[2].HasClass("kit-body"))
	assert.Same(t, ft, w.Children()[3])
	assert.Greater(t, text.Box().Y, bar.Box().Y+bar.Box().H, "the body under the bar and toolbar")
	assert.Equal(t, automation.RoleButton, s.element(t, "close").Role)
	assert.Equal(t, automation.RoleButton, s.element(t, "zoom").Role)
}

func TestAClosedWindowShowsNothing(t *testing.T) {
	w := kit.Window("Inspector", kit.Label("Tile 12"))
	show(t, w)
	assert.Zero(t, w.Box().W)
}

func TestACompactWindowHasASmallerBar(t *testing.T) {
	full, small := kit.Window("A"), kit.Window("B", kit.Compact())
	s := show(t, full, small)
	s.doc.Show(full)
	s.doc.Show(small)
	s.frames(2)
	assert.Less(t, small.Children()[0].Box().H, full.Children()[0].Box().H)
}

func TestAKitWindowIsDraggedFromTheCentre(t *testing.T) {
	w := kit.Window("Inspector", kit.Label("Tile 12"))
	s := show(t, w)
	s.doc.Show(w)
	s.frames(2)
	before := w.Box()
	bar := w.Children()[0].Box()
	s.in.SetMouse(geom.V2(bar.X+bar.W/2, bar.Y+bar.H/2))
	s.frames(1)
	s.in.SetMouseButton(input.MouseLeft, true)
	s.frames(1)
	s.in.SetMouse(geom.V2(bar.X+bar.W/2-30, bar.Y+bar.H/2+20))
	s.frames(1)
	s.in.SetMouseButton(input.MouseLeft, false)
	s.frames(2)
	assert.InDelta(t, before.X-30, w.Box().X, 0.01)
	assert.InDelta(t, before.Y+20, w.Box().Y, 0.01)
	assert.InDelta(t, before.W, w.Box().W, 0.01, "placed, it keeps its size")
}

func TestAKitWindowZoomedFillsTheWindowEvenMinimised(t *testing.T) {
	w := kit.Window("Inspector", kit.Label("Tile 12"))
	s := show(t, w)
	s.doc.Show(w)
	s.frames(2)
	zoom := w.Children()[0].Children()[0].Children()[2]
	s.click(zoom)
	assert.Equal(t, float32(400), w.Box().W)
	assert.Equal(t, float32(300), w.Box().H)
}

func TestAUnifiedToolbarStandsForTheTitleBar(t *testing.T) {
	back := kit.Button("Back")
	w := kit.Window("", kit.UnifiedToolbar(), kit.Toolbar(back, kit.Label("Documents")), kit.Label("files"))
	s := show(t, w)
	s.doc.Show(w)
	s.frames(2)
	tb := w.Children()[0]
	assert.True(t, tb.HasClass("kit-toolbar"), "no title bar: the toolbar comes first")
	assert.Equal(t, automation.RoleButton, s.element(t, "close").Role, "the lights are in the toolbar")
	assert.Greater(t, back.Box().X, tb.Box().X+60, "after the lights")
	before := w.Box()
	s.in.SetMouse(geom.V2(tb.Box().X+tb.Box().W-10, tb.Box().Y+tb.Box().H/2))
	s.frames(1)
	s.in.SetMouseButton(input.MouseLeft, true)
	s.frames(1)
	s.in.SetMouse(geom.V2(tb.Box().X+tb.Box().W-30, tb.Box().Y+tb.Box().H/2+15))
	s.frames(1)
	s.in.SetMouseButton(input.MouseLeft, false)
	s.frames(2)
	assert.InDelta(t, before.X-20, w.Box().X, 0.01, "the toolbar drags the window")
	clicks := watch[ui.Clicked](s)
	s.click(back)
	assert.Len(t, *clicks, 1, "its buttons still click")
}

func TestAWindowIsNamedByItsTitle(t *testing.T) {
	w := kit.Window("Inspector", kit.Button("Apply"))
	s := show(t, w)
	s.doc.Show(w)
	s.frames(2)
	assert.Equal(t, "inspector", s.element(t, "Inspector").ID)
}

func TestAWindowsLightsKeepTheirColours(t *testing.T) {
	w := kit.Window("Tiles")
	s := show(t, w)
	var lights []*ui.Node
	w.Walk(func(n *ui.Node) bool {
		if n.HasClass("kit-light") {
			lights = append(lights, n)
		}
		return true
	})
	require.Len(t, lights, 3)
	for _, l := range lights {
		action, _ := l.Attr("data-window-action")
		assert.True(t, l.HasClass("kit-"+action), "%s takes its colour's class", action)
	}
	_ = s
}
