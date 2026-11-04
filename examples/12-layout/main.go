// Layout: a game's main menu that fills its window at any size. A Column
// padded inside the window stacks a title, a grid of levels and a footer;
// two spacers that Expand share the height left over, so the grid stays in
// the middle, and the footer Fills the width, its label Expanding to push
// the buttons apart. Resize the window to watch it follow.
//
//	go run ./examples/12-layout
package main

import (
	"fmt"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/kit"
)

// page lets the menu take the window's whole height.
const page = `html, body { height: 100%; } .menu { height: 100%; box-sizing: border-box; } #title { font-size: 32px; }`

func main() {
	win := platform.NewWindow(
		platform.Title("ember · layout"),
		platform.Size(640, 400),
		platform.MinSize(420, 320),
	)
	win.Add(newMenu())

	ember.New().
		AddEntities(win).
		MustRun()
}

// newMenu is the whole screen: a padded column of the title, the levels
// between two spacers, and the footer.
func newMenu() *ui.Document {
	doc := ui.NewDocument(kit.Theme(), ui.CSS(page))
	level := ui.Style("width: 72px; height: 56px;")
	levels := kit.Grid(4, kit.Gap(12), ui.ID("levels"))
	for n := 1; n <= 8; n++ {
		levels.Append(kit.Button(fmt.Sprint(n), level))
	}

	button := ui.Style("width: 110px; height: 36px;")
	footer := kit.Row(kit.Fill(), kit.Gap(12), kit.AlignCenter(), ui.ID("footer"),
		kit.Button("Settings", button),
		kit.Label("ember 0.1", kit.Expand(), kit.Center()),
		kit.Button("Quit", button),
	)

	doc.Body().Append(kit.Column(ui.Class("menu"), kit.Padding(24), kit.Gap(16), kit.AlignCenter(),
		kit.Label("Ember Quest", ui.ID("title")),
		ui.Div(kit.Expand()),
		levels,
		ui.Div(kit.Expand()),
		footer,
	))
	return doc
}
