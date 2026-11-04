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

func files() (*ui.Node, []*ui.Node) {
	var cells []*ui.Node
	for _, n := range []string{"flags", "floor", "props", "walls", "old", "dungeon"} {
		cells = append(cells, kit.Thumbnail(n, ""))
	}
	kit.SetDimmed(cells[5], true)
	args := []ui.Arg{ui.Style("height: 260px")}
	for _, c := range cells {
		args = append(args, c)
	}
	return kit.Thumbnails(args...), cells
}

func TestThumbnailsLayTheirCellsOutInLines(t *testing.T) {
	g, cells := files()
	show(t, g)
	assert.Equal(t, float32(108), cells[0].Box().W, "118 a cell, 5 of margin each side")
	assert.Equal(t, float32(102), cells[0].Box().H)
	assert.Equal(t, cells[0].Box().Y, cells[2].Box().Y, "three to a line in 400 points")
	assert.Greater(t, cells[3].Box().Y, cells[0].Box().Y)
}

func TestThumbnailsSelectByClickAndActivateByADoubleClick(t *testing.T) {
	g, cells := files()
	s := show(t, g)
	changed, submitted := watch[ui.Changed](s), watch[ui.Submitted](s)
	s.click(cells[1])
	assert.Equal(t, 1, s.doc.Selected(g))
	require.Len(t, *changed, 1)
	assert.Same(t, g, (*changed)[0].Node)
	s.click(cells[1])
	assert.Len(t, *submitted, 1)
	s.click(cells[5])
	assert.Equal(t, 1, s.doc.Selected(g), "a dimmed cell can't be selected")
	assert.Equal(t, automation.RoleSelect, s.element(t, "props").Role)
}

func TestThumbnailsAreWalkedALineAtATime(t *testing.T) {
	g, cells := files()
	s := show(t, g)
	s.click(cells[0])
	s.key(input.KeyRight)
	assert.Equal(t, 1, s.doc.Selected(g))
	s.key(input.KeyDown)
	assert.Equal(t, 4, s.doc.Selected(g), "a line down")
	submitted := watch[ui.Submitted](s)
	s.key(input.KeyEnter)
	assert.Len(t, *submitted, 1)
}

func TestThumbnailSizeSetsTheCells(t *testing.T) {
	g := kit.Thumbnails(kit.ThumbnailSize(80, 90), kit.Thumbnail("a", ""))
	show(t, g)
	assert.Equal(t, float32(70), g.Children()[0].Box().W)
	assert.Equal(t, float32(80), g.Children()[0].Box().H)
}
