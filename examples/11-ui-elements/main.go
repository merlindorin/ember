// UI elements: labels, buttons, rows, columns and panels are a Document's
// nodes, built by the kit with options; the Document is an entity added to
// the window. A panel in the middle of the window holds a column: a title,
// the count, and a row of two buttons whose clicks call the Counter.
//
//	go run ./examples/11-ui-elements
package main

import (
	"github.com/lantervale/ember"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/kit"
)

// page puts what the body holds in the middle of the window.
const page = `html, body { height: 100%; } body { display: flex; align-items: center; justify-content: center; }`

func main() {
	win := platform.NewWindow(
		platform.Title("ember · ui elements"),
		platform.Size(640, 360),
		platform.FixedSize(),
	)
	doc, _ := counterPage()
	win.Add(doc)
	ember.New().AddEntities(win).MustRun()
}

// counterPage is the Document: the kit's look, the page's sheet, and the
// counter's panel in its body.
func counterPage() (*ui.Document, *Counter) {
	doc := ui.NewDocument(kit.Theme(), ui.CSS(page))
	counter := &Counter{label: kit.Label("Clicks: 0")}
	button := ui.Style("width: 120px; height: 36px;")
	click := kit.Button("Click", ui.ID("click"), button, ui.OnClick(counter.Click))
	reset := kit.Button("Reset", ui.ID("reset"), button, ui.OnClick(counter.Reset))

	buttons := kit.Row(kit.Gap(8), click, reset)
	column := kit.Column(kit.Gap(12), kit.Label("A counter"), counter.label, buttons)
	doc.Body().Append(kit.Panel(kit.Padding(20), column))
	return doc, counter
}
