package ui_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/merlindorin/ember/automation"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/ui"
)

// pictures is a listbox laid out three options a line, 60 high, in a box 100 high that
// scrolls: walls, floor, props / flags, old (dimmed), dungeon / crypt.
func pictures() (*ui.Node, []*ui.Node) {
	var opts []*ui.Node
	lb := ui.Div(ui.Attr("role", "listbox"),
		ui.Style("display: grid; grid-template-columns: repeat(3, 80px); grid-auto-rows: 60px; "+
			"width: 260px; height: 100px; overflow-y: auto"))
	for _, name := range []string{"walls", "floor", "props", "flags", "old", "dungeon", "crypt"} {
		o := ui.Div(ui.Attr("role", "option"), ui.Span(ui.Text(name)))
		if name == "old" {
			o.SetAttr("aria-disabled", "true")
		}
		lb.Append(o)
		opts = append(opts, o)
	}
	return lb, opts
}

func TestAListboxSelectsByClickAndActivatesByADoubleClick(t *testing.T) {
	lb, opts := pictures()
	l := liveDoc(t, lb)
	assert.Equal(t, -1, l.doc.Selected(lb))
	l.clickAt(opts[1].Children()[0])
	assert.Equal(t, "true", attr(opts[1], "aria-selected"), "a click inside an option selects it")
	assert.Equal(t, 1, l.doc.Selected(lb))
	assert.True(t, slices.Equal([]*ui.Node{lb}, l.changes))
	assert.True(t, l.doc.Focused(lb), "the listbox takes the focus")

	l.clickAt(opts[1])
	require.Len(t, l.submits, 1, "a second click soon after activates it")
	assert.Same(t, lb, l.submits[0].Node)

	l.clickAt(opts[4])
	assert.Equal(t, 1, l.doc.Selected(lb), "a dimmed option is never selected")
}

func TestAListboxIsWalkedALineAtATime(t *testing.T) {
	lb, opts := pictures()
	l := liveDoc(t, lb)
	l.clickAt(opts[0])
	l.key(input.KeyRight)
	assert.Equal(t, 1, l.doc.Selected(lb))
	l.key(input.KeyDown)
	assert.Equal(t, 3, l.doc.Selected(lb), "below floor is old, dimmed: of the two nearest that can be, the first")
	l.key(input.KeyRight)
	assert.Equal(t, 5, l.doc.Selected(lb), "Right skips the dimmed one")
	l.key(input.KeyDown)
	assert.Equal(t, 6, l.doc.Selected(lb), "the only one on the line below")
	l.key(input.KeyUp)
	assert.Equal(t, 3, l.doc.Selected(lb), "up to the nearest on the line above")
	l.key(input.KeyEnd)
	assert.Equal(t, 6, l.doc.Selected(lb))
	box, sel := lb.Box(), opts[6].Box()
	assert.LessOrEqual(t, sel.Y+sel.H, box.Y+box.H+0.5, "the selection scrolled into view")
	l.key(input.KeyHome)
	assert.Equal(t, 0, l.doc.Selected(lb))
	l.key(input.KeyEnter)
	assert.Len(t, l.submits, 1, "Enter activates the selection")
}

func TestAListboxsOptionsAreSelectablesForAutomation(t *testing.T) {
	lb, opts := pictures()
	l := liveDoc(t, lb)
	assert.Contains(t, l.roles(automation.RoleSelect), "props")
	l.doc.SetSelected(lb, 2)
	assert.Equal(t, "true", attr(opts[2], "aria-selected"))
	assert.Empty(t, l.changes, "SetSelected emits nothing")
}
