package main

import (
	"strings"
	"testing"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/kit"
	"github.com/merlindorin/ember/world"
)

// app is the notes app as main builds it, headless.
func app(t *testing.T) (*world.World, *ui.Document, *input.State) {
	t.Helper()
	win := platform.NewWindow(platform.Size(640, 440))
	doc := notes()
	win.Add(doc)
	w := ember.New().AddEntities(win)
	w.RunFrames(2)
	return w, doc, win.Input()
}

func TestTheCountSaysTheNotesWordsAndCharacters(t *testing.T) {
	if got := countOf("one two three 🔥"); got != "4 words · 15 characters" {
		t.Fatalf("the count: %q", got)
	}
}

func TestTypingInTheNoteUpdatesTheCountAndNotTheTitle(t *testing.T) {
	w, doc, in := app(t)
	count := doc.ByID("count")
	before := count.Text()
	if !doc.Focused(doc.ByID("note")) {
		t.Fatal("the note has the focus from the start")
	}
	for _, r := range " ok" {
		in.AddText(r)
		w.RunFrames(1)
	}
	w.RunFrames(2)
	if got := doc.Value(doc.ByID("note")); !strings.HasSuffix(got, " ok") {
		t.Fatalf("typed at the end: %q", got[len(got)-10:])
	}
	if count.Text() != countOf(doc.Value(doc.ByID("note"))) || count.Text() == before {
		t.Fatalf("the count follows the note: %q", count.Text())
	}
	doc.Focus(doc.ByID("title"))
	w.RunFrames(2)
	in.AddText('!')
	w.RunFrames(3)
	if count.Text() != countOf(doc.Value(doc.ByID("note"))) {
		t.Fatal("only the note's changes count")
	}
}

func TestUncheckingWrapKeepsLinesWhole(t *testing.T) {
	w, doc, in := app(t)
	note := doc.ByID("note")
	box := doc.ByID("wrap")
	r, _ := doc.BoxOf(box)
	in.SetMouse(r.Center())
	w.RunFrames(1)
	in.SetMouseButton(input.MouseLeft, true)
	w.RunFrames(1)
	in.SetMouseButton(input.MouseLeft, false)
	w.RunFrames(2)
	if doc.Checked(box) || kit.Wraps(note) {
		t.Fatal("unchecked, the note keeps its lines whole")
	}
}

func TestTheNoteDrawsInTheEnginesFace(t *testing.T) {
	if !twod.Regular().Draws(note) {
		t.Fatal("the note draws in the engine's face, no ? anywhere")
	}
}
