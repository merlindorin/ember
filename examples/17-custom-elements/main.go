// Custom elements: Orb is drawn by the game on a canvas, and sits in a row
// next to the kit's labels and buttons. Health changes it when a button
// is clicked.
//
//	go run ./examples/17-custom-elements
package main

import (
	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/kit"
)

// page centres the panel and gives the orb and the buttons their size.
const page = `html, body { height: 100%; } body { display: flex; align-items: center; justify-content: center; }
#orb { width: 96px; height: 96px; } #hurt, #heal { width: 100px; height: 32px; }`

func main() {
	win := platform.NewWindow(
		platform.Title("ember · custom elements"),
		platform.Size(640, 360),
		platform.FixedSize(),
	)
	doc, _ := orbPage()
	win.Add(doc)
	ember.New().AddEntities(win).MustRun()
}

// orbPage is the Document: the orb beside the label and two buttons whose
// clicks call Health.
func orbPage() (*ui.Document, *Health) {
	doc := ui.NewDocument(kit.Theme(), ui.CSS(page))
	health := &Health{orb: NewOrb(doc, 100, ui.ID("orb")), label: kit.Label("Health: 100", ui.ID("label"))}
	hurt := kit.Button("Hurt", ui.ID("hurt"), ui.OnClick(health.Hurt))
	heal := kit.Button("Heal", ui.ID("heal"), ui.OnClick(health.Heal))

	side := kit.Column(kit.Gap(12), health.label, kit.Row(kit.Gap(8), hurt, heal))
	doc.Body().Append(kit.Panel(kit.Padding(20), kit.Row(kit.Gap(20), health.orb.node, side)))
	return doc, health
}
