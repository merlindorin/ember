// HTML elements: each ui element alone in a window, in its own look
// before any theme, with -element and its tag, and the events it sends
// printed as they come.
//
//	go run ./examples/93-html-elements -element p
package main

import (
	"flag"
	"log"
	"slices"
	"strings"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/ui"
)

func main() {
	only := flag.String("element", "p", "the element to show, by its tag: "+strings.Join(names(), ", "))
	flag.Parse()
	e, ok := elements()[*only]
	if !ok {
		log.Fatalf("no element %q; try one of %s", *only, strings.Join(names(), ", "))
	}

	doc := page(e)
	w, h := e.size(doc)
	win := platform.NewWindow(
		platform.Title("ember · "+*only),
		platform.Size(w, h),
		platform.FixedSize(),
		platform.Background(color.Hex("#ffffff")),
	)
	win.Add(doc)

	ember.New().AddEntities(win).AddSystems(&Pointer{doc: doc}).MustRun()
}

// page is a Document showing e on its own, 16 points from the window's
// edges with body's 8.
func page(e element) *ui.Document {
	doc := ui.NewDocument(ui.Images(pictures()))
	shown := e.build(doc)
	logEvents(shown)
	doc.Body().Append(ui.Div(ui.Style("padding: 8px"), shown))
	return doc
}

// logEvents prints what the player does with each element under root,
// through its callbacks.
func logEvents(root *ui.Node) {
	root.Walk(func(n *ui.Node) bool {
		ui.On(n, ui.EventClick, func(e ui.Event) { log.Printf("clicked %s %q", e.Node.Tag(), e.Node.Text()) })
		ui.On(n, ui.EventInput, func(e ui.Event) { log.Printf("edited %s", e.Node.Tag()) })
		ui.On(n, ui.EventChange, func(e ui.Event) { log.Printf("changed %s", e.Node.Tag()) })
		ui.On(n, ui.EventToggle, func(e ui.Event) { log.Printf("toggled %s", e.Node.Tag()) })
		ui.On(n, ui.EventClose, func(e ui.Event) { log.Printf("closed %s %q", e.Node.Tag(), e.Value) })
		ui.On(
			n,
			ui.EventSort,
			func(e ui.Event) { log.Printf("sorted column %d, descending %v", e.Column, e.Descending) },
		)
		return true
	})
}

func names() []string {
	var all []string
	for name := range elements() {
		all = append(all, name)
	}
	slices.Sort(all)
	return all
}
