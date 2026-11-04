package kit_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/merlindorin/ember/automation"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/kit"
)

func TestATitledPanelPutsItsTitleAboveItsChildren(t *testing.T) {
	body := chip(100, 40)
	p := kit.Panel(kit.Title("Settings"), body)
	show(t, p)
	title := p.Children()[0]
	assert.Equal(t, "Settings", title.Text())
	assert.Equal(t, title.Box().Y+title.Box().H, body.Box().Y, "the children start under the bar")
}

func TestACollapsiblePanelFoldsToItsTitle(t *testing.T) {
	body := chip(100, 40)
	p := kit.CollapsiblePanel("Lights", true, body)
	s := show(t, p)
	toggled := watch[ui.Toggled](s)
	assert.Equal(t, automation.RoleButton, s.element(t, "Lights").Role, "its bar presses")
	open := p.Box().H
	s.click(p.Children()[0])
	assert.Len(t, *toggled, 1)
	assert.Less(t, p.Box().H, open, "folded to its bar")
	assert.Zero(t, body.Box().H, "its children are hidden")
	s.click(p.Children()[0])
	assert.Len(t, *toggled, 2)
	assert.Equal(t, open, p.Box().H)
}

func TestACardIsPressedByThePointerAndByKeys(t *testing.T) {
	c := kit.Card(kit.Label("New game"))
	s := show(t, c)
	clicks := watch[ui.Clicked](s)
	s.click(c)
	require.Len(t, *clicks, 1)
	assert.Same(t, c, (*clicks)[0].Node)
	s.doc.Focus(c)
	s.frames(1)
	s.key(input.KeyEnter)
	assert.Len(t, *clicks, 2)
	assert.Equal(t, automation.RoleButton, s.element(t, "New game").Role)
}

func TestACardIsPaddedAsEmberUIs(t *testing.T) {
	l := kit.Label("New game")
	c := kit.Card(l)
	show(t, c)
	assert.InDelta(t, c.Box().X+24+1, l.Box().X, 0.5, "24 of padding inside a 1 px edge")
	assert.InDelta(t, c.Box().Y+20+1, l.Box().Y, 0.5)
}

func TestGlowFollowsEmberUIsFalloff(t *testing.T) {
	p := kit.Panel(kit.Glow("rgba(240, 192, 96, 0.6)", 18))
	assert.Equal(t, "box-shadow: 0 0 12px 4px rgba(240, 192, 96, 0.6)", p.Style(), "full at the edge, gone 18 out")
	show(t, p)
}

func TestACollapsiblePanelOpenedFromCodeStaysOpen(t *testing.T) {
	body := chip(100, 40)
	p := kit.CollapsiblePanel("Lights", false, body)
	s := show(t, p)
	toggled := watch[ui.Toggled](s)
	s.doc.SetOpen(p, true)
	s.frames(3)
	assert.True(t, s.doc.Open(p))
	assert.Equal(t, float32(40), body.Box().H, "its children shown, frames later")
	assert.Empty(t, *toggled, "without events, as SetChecked")
	s.doc.SetOpen(p, false)
	s.frames(2)
	assert.False(t, s.doc.Open(p))
	assert.Zero(t, body.Box().H)
}
