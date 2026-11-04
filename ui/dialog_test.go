package ui_test

import (
	"testing"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/automation"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/internal/dom"
	"github.com/merlindorin/ember/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// live is a Document in a 640×360 window, its events read after every
// frame.
type live struct {
	doc     *ui.Document
	win     *platform.Window
	w       *world.World
	closed  []ui.Closed
	clicks  []*ui.Node
	changes []*ui.Node
	submits []ui.Submitted
	sorted  []ui.Sorted
	windows []any
	auto    *automation.Automation
}

// roles is what automation sees: each recorded element's label by role.
func (l *live) roles(role string) []string {
	_, all := l.auto.Snapshot("", "")
	var out []string
	for _, e := range all {
		if e.Role == role {
			out = append(out, e.Label)
		}
	}
	return out
}

func liveDoc(t *testing.T, nodes ...*ui.Node) *live {
	t.Helper()
	return liveDocWith(t, nil, nodes...)
}

// liveDocWith is liveDoc for a Document made with opts.
func liveDocWith(t *testing.T, opts []ui.DocumentOption, nodes ...*ui.Node) *live {
	t.Helper()
	l := &live{doc: ui.NewDocument(opts...), win: platform.NewWindow(platform.Size(640, 360))}
	l.doc.Body().Append(nodes...)
	l.win.Add(l.doc)
	l.auto = automation.New(automation.Recording())
	l.w = ember.New(ember.Automation(l.auto)).AddEntities(l.win)
	l.frames(2)
	return l
}

func (l *live) frames(n int) {
	for range n {
		l.w.RunFrames(1)
		var cr world.Reader[ui.Closed]
		for e := range cr.Read(l.w) {
			l.closed = append(l.closed, e)
		}
		var kr world.Reader[ui.Clicked]
		for e := range kr.Read(l.w) {
			l.clicks = append(l.clicks, e.Node)
		}
		var gr world.Reader[ui.Changed]
		for e := range gr.Read(l.w) {
			l.changes = append(l.changes, e.Node)
		}
		var sr world.Reader[ui.Submitted]
		for e := range sr.Read(l.w) {
			l.submits = append(l.submits, e)
		}
		var so world.Reader[ui.Sorted]
		for e := range so.Read(l.w) {
			l.sorted = append(l.sorted, e)
		}
		var wm world.Reader[ui.WindowMoved]
		for e := range wm.Read(l.w) {
			l.windows = append(l.windows, e)
		}
		var wr world.Reader[ui.WindowResized]
		for e := range wr.Read(l.w) {
			l.windows = append(l.windows, e)
		}
		var wf world.Reader[ui.WindowFolded]
		for e := range wf.Read(l.w) {
			l.windows = append(l.windows, e)
		}
	}
}

func (l *live) key(k input.Key) {
	in := l.win.Input()
	in.SetKey(k, true)
	l.frames(1)
	in.SetKey(k, false)
	l.frames(2)
}

func (l *live) clickAt(n *ui.Node) {
	in := l.win.Input()
	in.SetMouse(n.Box().Center())
	l.frames(1)
	in.SetMouseButton(input.MouseLeft, true)
	l.frames(1)
	in.SetMouseButton(input.MouseLeft, false)
	l.frames(2)
}

func TestAModalDialogTrapsTheFocusAndEscapeClosesIt(t *testing.T) {
	outside := ui.Button(ui.Text("Outside"))
	ok, cancel := ui.Button(ui.Text("OK")), ui.Button(ui.Text("Cancel"))
	dlg := ui.Dialog(ui.H2(ui.Text("Delete the tileset?")), ok, cancel)
	l := liveDoc(t, outside, dlg)
	require.Zero(t, dlg.Box().W, "closed: not displayed")
	l.doc.Focus(outside)
	l.frames(1)

	l.doc.ShowModal(dlg)
	l.frames(2)
	assert.True(t, l.doc.Open(dlg))
	assert.Greater(t, dlg.Box().W, float32(0))
	assert.True(t, l.doc.Focused(ok), "the focus goes to its first control")
	assert.Equal(t, []string{"Delete the tileset?"}, l.roles(automation.RoleDialog), "named by its heading")
	l.key(input.KeyTab)
	assert.True(t, l.doc.Focused(cancel))
	l.key(input.KeyTab)
	assert.True(t, l.doc.Focused(ok), "Tab wraps inside the dialog")

	l.clickAt(outside)
	assert.NotContains(t, l.clicks, outside, "the page under a modal takes no clicks")
	assert.True(t, l.doc.Focused(ok), "nor the focus")

	l.key(input.KeyEscape)
	assert.False(t, l.doc.Open(dlg))
	require.Len(t, l.closed, 1)
	assert.Same(t, dlg, l.closed[0].Node)
	assert.Equal(t, "", l.closed[0].Value)
	assert.True(t, l.doc.Focused(outside), "the focus goes back")
	assert.Empty(t, l.roles(automation.RoleDialog))
	assert.Zero(t, dlg.Box().W)
}

func TestCloseGivesItsValueAndShowIsNotModal(t *testing.T) {
	outside := ui.Button(ui.Text("Outside"))
	dlg := ui.Dialog(ui.Button(ui.Text("OK")))
	l := liveDoc(t, outside, dlg)
	l.doc.Show(dlg)
	l.frames(2)
	assert.True(t, l.doc.Open(dlg))
	assert.False(t, l.doc.Focused(outside), "no autofocus, no focus move")
	l.clickAt(outside)
	assert.Contains(t, l.clicks, outside, "a non-modal dialog leaves the page live")
	l.key(input.KeyEscape)
	assert.True(t, l.doc.Open(dlg), "Escape closes only a modal dialog")

	l.doc.Close(dlg, "ok")
	l.frames(1)
	require.Len(t, l.closed, 1)
	assert.Equal(t, "ok", l.closed[0].Value)
	l.doc.Close(dlg, "again")
	l.frames(1)
	assert.Len(t, l.closed, 1, "closing a closed dialog does nothing")
}

func TestADialogButtonClosesItWithItsValueAndAutofocusStarts(t *testing.T) {
	save := ui.Button(ui.Text("Save"), ui.Attr("formmethod", "dialog"), ui.Value("save"), ui.Attr("autofocus", ""))
	discard := ui.Button(ui.Text("Discard"), ui.Attr("formmethod", "dialog"), ui.Value("discard"))
	dlg := ui.Dialog(discard, save)
	l := liveDoc(t, dlg)
	l.doc.ShowModal(dlg)
	l.frames(2)
	assert.True(t, l.doc.Focused(save), "autofocus takes the first focus")
	l.key(input.KeyEnter)
	require.Len(t, l.closed, 1)
	assert.Equal(t, "save", l.closed[0].Value)
	assert.Contains(t, l.clicks, save)

	l.doc.ShowModal(dlg)
	l.frames(2)
	l.clickAt(discard)
	require.Len(t, l.closed, 2)

	l.doc.Focus(discard)
	l.doc.Show(dlg)
	l.frames(2)
	assert.True(t, l.doc.Focused(save), "show also moves the focus to autofocus, as HTML's show()")
	l.clickAt(discard)
	require.Len(t, l.closed, 3)
	assert.Equal(t, "discard", l.closed[1].Value)
}

func TestAModalWithoutControlsFocusesItselfAsChromeDoes(t *testing.T) {
	outside := ui.Button(ui.Text("Outside"))
	dlg := ui.Dialog(ui.H2(ui.Text("Paused")), ui.P(ui.Text("Press Escape to resume")))
	l := liveDoc(t, outside, dlg)
	l.doc.Focus(outside)
	l.frames(1)

	l.doc.ShowModal(dlg)
	l.frames(2)
	assert.True(t, l.doc.Focused(dlg), "the dialog itself")
	assert.True(t, dlg.State().Has(dom.FocusVisible), "shown, as Chrome rings it")
	l.key(input.KeyTab)
	assert.True(t, l.doc.Focused(dlg), "Tab finds nothing else in it")

	l.clickAt(dlg)
	assert.True(t, l.doc.Focused(dlg), "a click inside keeps it")
	l.key(input.KeyEscape)
	assert.False(t, l.doc.Open(dlg))
	assert.True(t, l.doc.Focused(outside), "closed, the focus goes back")
}

func TestAClickOnADialogsEmptyPartFocusesNothingNew(t *testing.T) {
	field := ui.Input(ui.Value("Heron"))
	dlg := ui.Dialog(ui.Style("width: 200px; height: 120px"), field)
	l := liveDoc(t, dlg)
	l.doc.ShowModal(dlg)
	l.frames(2)
	require.True(t, l.doc.Focused(field))
	l.win.Input().SetMouse(geom.V2(dlg.Box().X+dlg.Box().W-10, dlg.Box().Y+dlg.Box().H-10))
	l.frames(1)
	l.win.Input().SetMouseButton(input.MouseLeft, true)
	l.frames(1)
	l.win.Input().SetMouseButton(input.MouseLeft, false)
	l.frames(2)
	assert.False(t, l.doc.Focused(dlg), "a dialog takes no focus from a click")
}

func TestShowOnADialogWithoutControlsLeavesTheFocus(t *testing.T) {
	outside := ui.Button(ui.Text("Outside"))
	tool := ui.Dialog(ui.P(ui.Text("Layers")))
	l := liveDoc(t, outside, tool)
	l.doc.Focus(outside)
	l.frames(1)
	l.doc.Show(tool)
	l.frames(2)
	assert.True(t, l.doc.Focused(outside), "Show runs no focusing steps, unlike ShowModal")
}
