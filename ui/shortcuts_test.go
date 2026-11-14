package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/internal/textedit"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/world"
)

func checkItem(text string, on bool, args ...ui.Arg) *ui.Node {
	v := "false"
	if on {
		v = "true"
	}
	return ui.Div(append([]ui.Arg{ui.Attr("role", "menuitemcheckbox"), ui.Attr("aria-checked", v), ui.Text(text)},
		args...)...)
}

func radioItem(text string, on bool) *ui.Node {
	n := checkItem(text, on)
	n.SetAttr("role", "menuitemradio")
	return n
}

// combo presses key with mods held, for one frame, then lets go.
func combo(l *live, key input.Key, mods ...input.Key) {
	in := l.win.Input()
	for _, m := range mods {
		in.SetKey(m, true)
	}
	l.frames(1)
	in.SetKey(key, true)
	l.frames(1)
	in.SetKey(key, false)
	for _, m := range mods {
		in.SetKey(m, false)
	}
	l.frames(2)
}

func keys(s string) ui.Arg { return ui.Attr("aria-keyshortcuts", s) }

func TestACheckboxItemTurnsOnAClickAndOnItsShortcut(t *testing.T) {
	ui.SetPlatform(t, textedit.Mac)
	grid := checkItem("Pixel grid", true, keys("Cmd+G"))
	file := ui.Span(ui.Attr("role", "menuitem"), ui.Style("position: relative; display: inline-block"),
		ui.Text("View"), menu(grid))
	l := liveDoc(t, ui.Div(ui.Attr("role", "menubar"), file))
	require.True(t, l.doc.Checked(grid))

	combo(l, input.KeyG, input.KeyLeftSuper)
	assert.False(t, l.doc.Checked(grid), "its shortcut turns it, its menu closed")
	assert.Equal(t, []*ui.Node{grid}, l.clicks, "and chooses it, as a click")

	l.clickAt(file)
	assert.Equal(t, false, recordedValue(t, l, "Pixel grid"), "automation reads its state")
	l.clickAt(grid)
	assert.True(t, l.doc.Checked(grid), "a click turns it back")
	assert.Len(t, l.clicks, 2)

	combo(l, input.KeyG, input.KeyLeftControl)
	assert.True(t, l.doc.Checked(grid), "Cmd is Command on macOS, not Control")
}

func recordedValue(t *testing.T, l *live, label string) any {
	t.Helper()
	_, all := l.auto.Snapshot("", "")
	for _, e := range all {
		if e.Label == label {
			return e.Value
		}
	}
	t.Fatalf("no element labelled %q", label)
	return nil
}

func TestARadioItemChecksItselfAndUnchecksItsGroup(t *testing.T) {
	small, large := radioItem("Small", true), radioItem("Large", false)
	other := radioItem("Elsewhere", true)
	m := menu(ui.Div(ui.Attr("role", "group"), small, large), other)
	m.SetAttr("open", "")
	l := liveDoc(t, ui.Div(ui.Style("position: relative"), m))
	l.clickAt(large)
	assert.True(t, l.doc.Checked(large))
	assert.False(t, l.doc.Checked(small), "its group's other item unchecks")
	assert.True(t, l.doc.Checked(other), "one outside the group stays")
}

func TestAShortcutStandsAsideForAFieldADisabledItemAndAModal(t *testing.T) {
	ui.SetPlatform(t, textedit.Mac)
	copyItem := item("Copy shapes", keys("Cmd+C"))
	del := item("Delete shape", keys("Backspace"))
	save := item("Save", keys("Cmd+S"), ui.Attr("aria-disabled", "true"))
	field := ui.Input(ui.Value("walls"))
	other := ui.Button(ui.Text("Other"))
	ok := ui.Button(ui.Text("OK"))
	dlg := ui.Dialog(ok)
	l := liveDoc(t,
		ui.Div(ui.Attr("role", "menubar"),
			ui.Span(ui.Attr("role", "menuitem"), ui.Text("Edit"), menu(copyItem, del, save))), field, other, dlg)

	l.doc.Focus(field)
	l.frames(1)
	combo(l, input.KeyC, input.KeyLeftSuper)
	l.key(input.KeyBackspace)
	assert.Empty(t, l.clicks, "a focused field keeps Cmd+C and Backspace")
	assert.Equal(t, "wall", l.doc.Value(field))

	l.doc.Focus(other)
	l.frames(1)
	combo(l, input.KeyS, input.KeyLeftSuper)
	assert.Empty(t, l.clicks, "a disabled item's shortcut does nothing")
	l.key(input.KeyBackspace)
	assert.Equal(t, []*ui.Node{del}, l.clicks, "with a button focused, a plain key shortcut chooses")

	l.doc.ShowModal(dlg)
	l.frames(2)
	combo(l, input.KeyC, input.KeyLeftSuper)
	assert.Equal(t, []*ui.Node{del}, l.clicks, "under a modal, the menus' shortcuts wait")
}

func TestAMatchedShortcutIsCapturedFromTheGame(t *testing.T) {
	ui.SetPlatform(t, textedit.Mac)
	save := item("Save", keys("Cmd+S"))
	l := liveDoc(t, ui.Div(ui.Attr("role", "menu"), save))
	var seen []bool
	l.w.AddSystems(&gameKey{key: input.KeyS, seen: &seen, win: l.win})
	combo(l, input.KeyS, input.KeyLeftSuper)
	assert.Equal(t, []*ui.Node{save}, l.clicks, "Cmd+S chooses Save")
	assert.NotContains(t, seen, true, "and the game doesn't see it too")
}

type gameKey struct {
	key  input.Key
	seen *[]bool
	win  *platform.Window
}

func (g *gameKey) Update(world.Frame) { *g.seen = append(*g.seen, g.win.Input().Down(g.key)) }

func TestOfTwoDocumentsWithOneShortcutTheLastDrawnChooses(t *testing.T) {
	ui.SetPlatform(t, textedit.Mac)
	under, over := ui.NewDocument(), ui.NewDocument()
	a, b := item("Save A", keys("Cmd+S")), item("Save B", keys("Cmd+S"))
	under.Body().Append(ui.Div(ui.Attr("role", "menu"), a))
	over.Body().Append(ui.Div(ui.Attr("role", "menu"), b))
	win := platform.NewWindow(platform.Size(320, 200))
	win.Add(under, over)
	w := ember.New().AddEntities(win)
	w.RunFrames(2)
	var clicked []*ui.Node
	run := func(n int) {
		for range n {
			w.RunFrames(1)
			var r world.Reader[ui.Clicked]
			for e := range r.Read(w) {
				clicked = append(clicked, e.Node)
			}
		}
	}
	win.Input().SetKey(input.KeyLeftSuper, true)
	run(1)
	win.Input().SetKey(input.KeyS, true)
	run(1)
	win.Input().SetKey(input.KeyS, false)
	win.Input().SetKey(input.KeyLeftSuper, false)
	run(3)
	assert.Equal(t, []*ui.Node{b}, clicked)
}

func TestAFieldOfAnotherDocumentKeepsItsEditingKeys(t *testing.T) {
	ui.SetPlatform(t, textedit.Mac)
	menus, inspector := ui.NewDocument(), ui.NewDocument()
	del := item("Delete shape", keys("Backspace"))
	menus.Body().Append(ui.Div(ui.Attr("role", "menu"), del))
	field := ui.Input(ui.Value("64"))
	inspector.Body().Append(field)
	win := platform.NewWindow(platform.Size(320, 200))
	win.Add(menus, inspector)
	w := ember.New().AddEntities(win)
	w.RunFrames(2)
	inspector.Focus(field)
	w.RunFrames(2)
	var clicked int
	win.Input().SetKey(input.KeyBackspace, true)
	w.RunFrames(1)
	win.Input().SetKey(input.KeyBackspace, false)
	for range 3 {
		w.RunFrames(1)
		var r world.Reader[ui.Clicked]
		clicked += r.Count(w)
	}
	assert.Zero(t, clicked, "Backspace edits the other document's field, not the menu")
	assert.Equal(t, "6", inspector.Value(field))
}
