// Text editing: a notes app. A title input and a text area edit like any
// desktop's: select with Shift or the mouse, jump by word, undo, cut,
// copy and paste, double-click a word, scroll a long note. The word count
// follows as you type, and Wrap turns soft wrapping off for a horizontal
// scrollbar.
//
//	go run ./examples/14-text-editing
package main

import (
	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/kit"
)

// page centres the 600 × 400 notes panel, sets the title's larger text,
// and lets the title and the note take the panel's width.
const page = `html, body { height: 100%; } body { display: flex; align-items: center; justify-content: center; }
#notes { width: 600px; height: 400px; box-sizing: border-box; } #title { font-size: 20px; }
#title, #note { width: 100%; box-sizing: border-box; }`

// note is what the app opens with.
//
//nolint:lll // its paragraphs are single lines the text area wraps
const note = `Ember notes

Select with Shift and the arrows, or drag with the mouse. Alt or Control with the arrows jumps a word, Home and End go to the line's ends.

Cut, copy and paste with Command or Control and X, C, V; undo with Z. Double-click a word, triple-click a line.

Scroll a long note with the wheel or the bar on the right; the caret stays in view as you type.`

func main() {
	win := platform.NewWindow(
		platform.Title("ember · text editing"),
		platform.Size(640, 440),
		platform.FixedSize(),
	)
	win.Add(notes())
	ember.New().AddEntities(win).MustRun()
}

// notes is the app: the title, the note, and under it the count and the
// Wrap check, which call back as the note changes and the check flips:
// unchecked, the note's lines stay whole, scrolling sideways.
func notes() *ui.Document {
	doc := ui.NewDocument(kit.Theme(), ui.CSS(page))
	title := kit.TextInput("Ideas", ui.ID("title"), ui.Placeholder("Title"))
	count := kit.Label(countOf(note), ui.ID("count"), kit.Expand())
	body := kit.TextArea(
		note,
		ui.ID("note"),
		ui.Placeholder("Write something…"),
		kit.Rows(12),
		ui.Attr("autofocus", ""),
		ui.OnInput(func(e ui.Event) { count.SetText(countOf(doc.Value(e.Node))) }),
	)
	wrap := kit.Check("Wrap", ui.ID("wrap"), ui.Checked(true),
		ui.OnChange(func(e ui.Event) { kit.SetNoWrap(doc.ByID("note"), !doc.Checked(e.Node)) }))

	doc.Body().Append(kit.Panel(ui.ID("notes"), kit.Padding(16),
		kit.Column(kit.Gap(10), kit.Fill(), title, body, kit.Row(kit.Gap(16), count, wrap)),
	))
	return doc
}
