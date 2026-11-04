package ui_test

import (
	"testing"

	"github.com/merlindorin/ember/automation"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/ui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// french says what it knows, and counts what it was asked.
type french struct {
	words map[string]string
	asked map[string]int
}

func newFrench(words map[string]string) *french {
	return &french{words: words, asked: map[string]int{}}
}

func (f *french) Tr(m string) string {
	f.asked[m]++
	if s, ok := f.words[m]; ok {
		return s
	}
	return m
}

func saveWords() map[string]string {
	return map[string]string{"Save": "Enregistrer", "Search": "Rechercher", "Red": "Rouge", "Pick": "Choisir"}
}

func element(t *testing.T, l *live, role string) automation.Element {
	t.Helper()
	_, all := l.auto.Snapshot("", "")
	for _, e := range all {
		if e.Role == role {
			return e
		}
	}
	t.Fatalf("no %s among %v", role, all)
	return automation.Element{}
}

func TestATranslatorShowsTheLabelAndAutomationKeepsItsID(t *testing.T) {
	plain := ui.Span(ui.Style("display: inline-block"), ui.Text("Save"))
	shown := ui.Span(ui.Style("display: inline-block"), ui.Text("Save"))
	width := liveDoc(t, plain).doc.Body().Children()[0].Box().W

	tl := liveDocWith(t, []ui.DocumentOption{ui.WithTranslator(newFrench(saveWords()))}, shown, ui.Button(ui.Text("Save")))
	assert.Greater(t, shown.Box().W, width, "laid out as Enregistrer")
	assert.Equal(t, "Save", shown.Text(), "the node keeps its own text")
	b := element(t, tl, automation.RoleButton)
	assert.Equal(t, "Enregistrer", b.Label)
	assert.Contains(t, b.ID, "save")
}

func TestTranslateNoOnAnAncestorKeepsTheText(t *testing.T) {
	plain := ui.Span(ui.Style("display: inline-block"), ui.Text("Save"))
	kept := ui.Span(ui.Style("display: inline-block"), ui.Text("Save"))
	again := ui.Span(ui.Style("display: inline-block"), ui.Attr("translate", "yes"), ui.Text("Save"))
	liveDocWith(t, []ui.DocumentOption{ui.WithTranslator(newFrench(saveWords()))},
		plain, ui.Div(ui.Attr("translate", "no"), ui.P(kept), ui.P(again)))
	assert.Less(t, kept.Box().W, plain.Box().W)
	assert.Equal(t, plain.Box().W, again.Box().W, "the nearest translate attribute decides")
}

func TestThePlaceholderAndASelectsLabelAreTranslatedButNotAFieldsValue(t *testing.T) {
	f := newFrench(saveWords())
	search := ui.Input(ui.Type("text"), ui.Attr("placeholder", "Search"))
	typed := ui.Input(ui.Type("text"), ui.Value("Save"))
	sel := ui.Select(ui.Attr("placeholder", "Pick"), ui.Option(ui.Text("Red")))
	l := liveDocWith(t, []ui.DocumentOption{ui.WithTranslator(f)}, search, typed, sel)
	assert.Contains(t, l.roles(automation.RoleTextInput), "Rechercher")
	assert.Contains(t, l.roles(automation.RoleCombo), "Choisir")
	assert.Zero(t, f.asked["Save"], "what a field holds is no message")
	assert.Equal(t, "Save", l.doc.Value(typed))

	l.doc.SetSelected(sel, 0)
	l.frames(2)
	assert.Contains(t, l.roles(automation.RoleCombo), "Rouge")
}

func TestSetTranslatorLaysALongerTranslationOutAgain(t *testing.T) {
	p := ui.P(ui.Style("width: 80px"), ui.Text("Save"))
	l := liveDoc(t, p)
	one := p.Box().H
	f := newFrench(map[string]string{"Save": "Enregistrer les modifications du niveau"})
	l.doc.SetTranslator(f)
	l.frames(1)
	assert.Equal(t, f, l.doc.Translator())
	assert.Greater(t, p.Box().H, 2*one, "the translation wraps")

	l.doc.SetTranslator(nil)
	l.frames(1)
	assert.Equal(t, one, p.Box().H)
}

func TestAMessageIsAskedOnceAcrossFrames(t *testing.T) {
	f := newFrench(saveWords())
	b := ui.Button(ui.Text("Save"))
	l := liveDocWith(t, []ui.DocumentOption{ui.WithTranslator(f)}, b, ui.P(ui.Text("Search")))
	l.frames(10)
	l.win.Input().SetMouse(b.Box().Center())
	l.frames(5)
	require.Equal(t, 1, f.asked["Save"])
	assert.Equal(t, 1, f.asked["Search"])
}

func TestAnOpenSelectListsItsOptionsTranslated(t *testing.T) {
	sel := ui.Select(ui.Option(ui.Text("Red")), ui.Option(ui.Attr("translate", "no"), ui.Text("Pick")))
	l := liveDocWith(t, []ui.DocumentOption{ui.WithTranslator(newFrench(saveWords()))}, sel)
	l.clickAt(sel)
	l.frames(1)
	assert.Equal(t, []string{"Rouge", "Pick"}, ui.OptionTexts(l.doc, sel))
}

func TestControlsShowChromesCursors(t *testing.T) {
	b := ui.Button(ui.Text("Save"))
	in := ui.Input(ui.Type("text"), ui.Value("x"))
	l := liveDoc(t, b, in)
	for n, want := range map[*ui.Node]platform.Cursor{b: platform.CursorArrow, in: platform.CursorText} {
		l.win.Input().SetMouse(n.Box().Center())
		l.frames(2)
		assert.Equal(t, want, l.win.Cursor(), n.Tag().String())
	}
}
