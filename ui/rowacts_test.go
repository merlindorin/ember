package ui_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/automation"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/kit"
)

// idEnding is the automation ID ending in suffix.
func (l *live) idEnding(t *testing.T, suffix string) string {
	t.Helper()
	_, all := l.auto.Snapshot("", "")
	for _, e := range all {
		if e.ID == suffix || strings.HasSuffix(e.ID, "/"+suffix) {
			return e.ID
		}
	}
	t.Fatalf("no automation ID ends in %q", suffix)
	return ""
}

func (l *live) act(t *testing.T, suffix, kind string) {
	t.Helper()
	l.auto.Queue(l.idEnding(t, suffix), kind, nil)
	l.frames(2)
}

func TestAutomationPressActivatesAGridRowAndSelectOnlySelects(t *testing.T) {
	grid, _, _, rows := files()
	grid.SetAttr("data-name", "files")
	l := liveDoc(t, grid)
	l.act(t, "files/props-tsj", automation.ActSelect)
	assert.Equal(t, "true", attr(rows[2], "aria-selected"))
	assert.Equal(t, []*ui.Node{grid}, l.changes)
	assert.Empty(t, l.submits, "select only selects")
	l.act(t, "files/walls-tsj", automation.ActPress)
	assert.Equal(t, "true", attr(rows[1], "aria-selected"))
	require.Len(t, l.submits, 1, "press activates")
	assert.Same(t, grid, l.submits[0].Node)
	l.act(t, "files/crypt", automation.ActPress)
	l.act(t, "files/crypt", automation.ActSelect)
	assert.Equal(t, "true", attr(rows[1], "aria-selected"), "a dimmed row takes neither")
	assert.Len(t, l.submits, 1)
}

func TestAutomationPressActivatesAListboxOption(t *testing.T) {
	lb, opts := pictures()
	l := liveDoc(t, lb)
	l.act(t, "props", automation.ActSelect)
	assert.Equal(t, 2, l.doc.Selected(lb))
	assert.Empty(t, l.submits)
	l.act(t, "floor", automation.ActPress)
	assert.Equal(t, 1, l.doc.Selected(lb))
	assert.Len(t, l.submits, 1)
	l.act(t, "old", automation.ActPress)
	assert.Equal(t, "", attr(opts[4], "aria-selected"), "a disabled option takes neither")
	assert.Len(t, l.submits, 1)
}

func TestThumbnailsArePressedByAutomation(t *testing.T) {
	cells := []*ui.Node{kit.Thumbnail("walls", ""), kit.Thumbnail("floor", "")}
	g := kit.Thumbnails(ui.Attr("data-name", "pictures"), cells[0], cells[1])
	l := liveDoc(t, g)
	l.act(t, "pictures/floor", automation.ActPress)
	assert.Equal(t, 1, l.doc.Selected(g))
	require.Len(t, l.submits, 1)
	assert.Same(t, g, l.submits[0].Node)
}

func TestADataNameIsTheAutomationLeaf(t *testing.T) {
	b := ui.Button(ui.Text("Save"), ui.Attr("data-name", "ok"))
	l := liveDoc(t, ui.Div(ui.Attr("data-name", "save"), b))
	l.act(t, "save/ok", automation.ActPress)
	assert.Contains(t, l.clicks, b)
}
