// Themes: the same dialog in four looks, switched with T. The structure is
// built in Go; the look comes from CSS, written in Go or loaded from a
// file, each turned on by a class on the dialog and reaching everything in
// it. The last look is a skin: the dialog drawn as a nine-patch of game
// art.
//
//	go run ./examples/15-themes
package main

import (
	"github.com/lantervale/ember"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/kit"
)

// page centres the dialog and puts the look's name at the top left.
const page = `html, body { height: 100%; } body { display: flex; align-items: center; justify-content: center; }
#name { position: fixed; left: 16px; top: 12px; } #title { font-size: 22px; } #lantern { width: 260px; }`

func main() {
	win := platform.NewWindow(
		platform.Title("ember · themes"),
		platform.Size(560, 360),
		platform.FixedSize(),
	)
	doc := dialogPage()
	win.Add(doc)
	ember.New().
		AddEntities(win).
		AddSystems(&Switch{window: win, name: doc.ByID("name"), dialog: doc.ByID("dialog")}).
		MustRun()
}

// dialogPage is the Document: the kit's look, the three themes and the
// page's sheet, the look's name and the dialog.
func dialogPage() *ui.Document {
	doc := ui.NewDocument(kit.Theme(),
		ui.CSS(emberLook), ui.Stylesheet(files, "frost.css"), ui.Stylesheet(files, "wood.css"),
		ui.CSS(page))
	doc.Body().Append(
		kit.Label("Default · press T", ui.ID("name")),
		kit.Panel(ui.ID("dialog"), kit.Padding(16), content()),
	)
	return doc
}

// content is what the dialog says and asks, the same in every look.
func content() *ui.Node {
	button := ui.Style("width: 120px; height: 36px;")
	return kit.Column(kit.Gap(14),
		kit.Label("Leave the island?", ui.ID("title")),
		kit.Label("Your boat is ready, but the night is long."),
		kit.Check("Take the lantern", ui.Checked(true)),
		kit.Slider(0.6, kit.Range(0, 1), kit.Step(0.01), ui.ID("lantern")),
		kit.Row(kit.Gap(12), kit.Button("Stay", button), kit.Button("Sail", button)),
	)
}
