package ui_test

import (
	"strings"
	"testing"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/automation"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type page struct {
	doc                       *ui.Document
	win                       *platform.Window
	w                         *world.World
	auto                      *automation.Automation
	solid, name, shape, save  *ui.Node
	details, summary, caption *ui.Node
}

func newPage(_ *testing.T) *page {
	p := &page{doc: ui.NewDocument(), win: platform.NewWindow(platform.Size(640, 360))}
	p.solid = ui.Input(ui.ID("solid"), ui.Type("checkbox"))
	p.name = ui.Input(ui.ID("name"), ui.Attr("placeholder", "who"), ui.Style("width: 200px; height: 24px"))
	p.shape = ui.Select(ui.ID("shape"), ui.Style("width: 120px; height: 24px"),
		ui.Option(ui.Text("rectangle")), ui.Option(ui.Text("ellipse")), ui.Option(ui.Text("polygon")))
	p.save = ui.Button(ui.Style("width: 80px; height: 24px"), ui.Text("Save"))
	p.summary = ui.Summary(ui.ID("title"), ui.Text("object 1"))
	p.caption = ui.Span(ui.ID("caption"), ui.Text("hello"))
	p.details = ui.Details(ui.ID("object-1"), p.summary, ui.Div(ui.Text("inside")))
	p.doc.Body().Append(ui.Section(ui.ID("inspector"), p.solid, p.name, p.shape, p.save, p.details, p.caption))
	p.win.Add(p.doc)
	p.auto = automation.New(automation.Recording())
	p.w = ember.New(ember.Automation(p.auto)).AddEntities(p.win)
	p.w.RunFrames(2)
	return p
}

func (p *page) element(t *testing.T, id string) automation.Element {
	t.Helper()
	_, all := p.auto.Snapshot("", "")
	for _, e := range all {
		if e.ID == id || len(e.ID) > len(id) && e.ID[len(e.ID)-len(id)-1:] == "/"+id {
			return e
		}
	}
	t.Fatalf("no element %q among %v", id, all)
	return automation.Element{}
}

func events[T any](p *page) []T {
	var r world.Reader[T]
	var out []T
	for e := range r.Read(p.w) {
		out = append(out, e)
	}
	return out
}

func TestAutomationSeesEveryControlUnderItsAncestorsIDs(t *testing.T) {
	p := newPage(t)
	for id, role := range map[string]string{
		"inspector/solid":   automation.RoleToggle,
		"inspector/name":    automation.RoleTextInput,
		"inspector/shape":   automation.RoleCombo,
		"inspector/save":    automation.RoleButton,
		"object-1/title":    automation.RoleButton,
		"inspector/caption": automation.RoleLabel,
	} {
		e := p.element(t, id)
		assert.Equal(t, role, e.Role, id)
	}
	assert.Equal(t, "rectangle", p.element(t, "shape").Label)
	assert.Equal(t, "who", p.element(t, "name").Label)
	assert.Equal(t, p.save.Box(), p.element(t, "save").Bounds)
}

func TestActsChangeControlsAndEmitEvents(t *testing.T) {
	p := newPage(t)
	id := func(leaf string) string { return p.element(t, leaf).ID }
	p.auto.Queue(id("solid"), automation.ActPress, nil)
	p.auto.Queue(id("name"), automation.ActSet, "Heron")
	p.auto.Queue(id("shape"), automation.ActSelect, "polygon")
	p.auto.Queue(id("save"), automation.ActPress, nil)
	p.w.RunFrames(1)

	assert.True(t, p.doc.Checked(p.solid))
	assert.Equal(t, "Heron", p.doc.Value(p.name))
	assert.Equal(t, 2, p.doc.Selected(p.shape))
	var clicked []*ui.Node
	for _, e := range events[ui.Clicked](p) {
		clicked = append(clicked, e.Node)
	}
	assert.Contains(t, clicked, p.save)
	var changed []*ui.Node
	for _, e := range events[ui.Changed](p) {
		changed = append(changed, e.Node)
	}
	assert.ElementsMatch(t, []*ui.Node{p.solid, p.name, p.shape}, changed)

	p.w.RunFrames(2)
	assert.Equal(t, true, p.element(t, "solid").Value)
	assert.Equal(t, "Heron", p.element(t, "name").Value)
	assert.Equal(t, "polygon", p.element(t, "shape").Label)
}

func TestAClickOnASummaryTogglesItsDetails(t *testing.T) {
	p := newPage(t)
	in := p.win.Input()
	click := func() {
		in.SetMouse(p.summary.Box().Center())
		p.w.RunFrames(1)
		in.SetMouseButton(input.MouseLeft, true)
		p.w.RunFrames(1)
		in.SetMouseButton(input.MouseLeft, false)
		p.w.RunFrames(1)
	}
	inside := p.details.Children()[1]
	require.Zero(t, inside.Box().H, "closed")
	click()
	assert.Greater(t, inside.Box().H, float32(0), "open")
	_, open := p.details.Attr("open")
	assert.True(t, open)
	click()
	assert.Zero(t, inside.Box().H)
}

func TestTypingGoesToTheFocusedFieldAndTabMovesOn(t *testing.T) {
	p := newPage(t)
	in := p.win.Input()
	in.SetMouse(p.name.Box().Center())
	p.w.RunFrames(1)
	in.SetMouseButton(input.MouseLeft, true)
	p.w.RunFrames(1)
	in.SetMouseButton(input.MouseLeft, false)
	p.w.RunFrames(1)
	require.True(t, p.doc.Focused(p.name))
	assert.True(t, in.KeyboardCaptured())

	in.AddText('R')
	in.AddText('o')
	p.w.RunFrames(1)
	assert.Equal(t, "Ro", p.doc.Value(p.name))
	assert.Len(t, events[ui.Edited](p), 2)

	in.SetKey(input.KeyTab, true)
	p.w.RunFrames(1)
	assert.True(t, p.doc.Focused(p.shape))
	assert.Len(t, events[ui.Changed](p), 1, "leaving the field commits it")
	in.SetKey(input.KeyTab, false)
	p.w.RunFrames(1)
}

func TestTheGameSetsValuesWithoutEvents(t *testing.T) {
	p := newPage(t)
	p.doc.SetChecked(p.solid, true)
	p.doc.SetValue(p.name, "Heron")
	p.doc.SetSelected(p.shape, 1)
	p.w.RunFrames(2)
	assert.Empty(t, events[ui.Changed](p))
	assert.Equal(t, "ellipse", p.element(t, "shape").Label)
	assert.Equal(t, "Heron", p.element(t, "name").Value)
}

func TestAClickPlacesTheCaretAndADragSelects(t *testing.T) {
	p := newPage(t)
	p.doc.SetValue(p.name, "Heron")
	p.w.RunFrames(1)
	in := p.win.Input()
	left := p.name.Box()
	in.SetMouse(geom.V2(left.X+left.W-2, left.Y+left.H/2))
	p.w.RunFrames(1)
	in.SetMouseButton(input.MouseLeft, true)
	p.w.RunFrames(1)
	in.AddText('!')
	p.w.RunFrames(1)
	assert.Equal(t, "Heron!", p.doc.Value(p.name), "a click past the end puts the caret there")

	in.SetMouse(geom.V2(left.X+1, left.Y+left.H/2))
	p.w.RunFrames(1)
	in.SetMouseButton(input.MouseLeft, false)
	p.w.RunFrames(1)
	in.AddText('X')
	p.w.RunFrames(1)
	assert.Equal(t, "X", p.doc.Value(p.name), "the drag selected it all, typing replaces it")
}

func TestTheKeysOpenASelectAndChooseAnOption(t *testing.T) {
	p := newPage(t)
	p.doc.Focus(p.shape)
	tap := func(k input.Key) {
		in := p.win.Input()
		in.SetKey(k, true)
		p.w.RunFrames(1)
		in.SetKey(k, false)
		p.w.RunFrames(1)
	}
	tap(input.KeyEnter)
	tap(input.KeyDown)
	tap(input.KeyDown)
	tap(input.KeyEnter)
	assert.Equal(t, 2, p.doc.Selected(p.shape))
	assert.True(t, p.doc.Focused(p.shape))
}

func TestAutomationDoesNotSeeWhatIsNotDisplayed(t *testing.T) {
	p := newPage(t)
	p.caption.SetStyle("display: none")
	p.w.RunFrames(2)
	_, all := p.auto.Snapshot("", "")
	for _, e := range all {
		assert.NotContains(t, e.ID, "caption")
	}
}

func TestADisabledAttributeSetLaterStopsTheControl(t *testing.T) {
	p := newPage(t)
	p.save.SetAttr("disabled", "")
	p.w.RunFrames(2)
	p.auto.Queue(p.element(t, "save").ID, automation.ActPress, nil)
	p.w.RunFrames(1)
	assert.Empty(t, events[ui.Clicked](p))
	p.save.RemoveAttr("disabled")
	p.w.RunFrames(2)
	p.auto.Queue(p.element(t, "save").ID, automation.ActPress, nil)
	p.w.RunFrames(1)
	assert.Len(t, events[ui.Clicked](p), 1)
}

func TestNamesSkipTooltipsAndHiddenText(t *testing.T) {
	doc := ui.NewDocument()
	save := ui.Button(ui.Text("Save"),
		ui.Span(ui.Attr("role", "tooltip"), ui.Text("Writes the tileset")),
		ui.Span(ui.Style("display: none"), ui.Text("hidden")))
	doc.Body().Append(ui.Section(ui.ID("bar"), save))
	win := platform.NewWindow(platform.Size(320, 200))
	win.Add(doc)
	auto := automation.New(automation.Recording())
	w := ember.New(ember.Automation(auto)).AddEntities(win)
	w.RunFrames(3)
	_, all := auto.Snapshot("", "")
	var found bool
	for _, e := range all {
		if e.Role == automation.RoleButton {
			found = true
			assert.Equal(t, "Save", e.Label)
			assert.True(t, strings.HasSuffix(e.ID, "bar/save"), e.ID)
		}
	}
	assert.True(t, found)
}
