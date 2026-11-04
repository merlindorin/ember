// Controls: a settings window with each kind of control, a text input, a
// slider, checks, a progress bar and a scrolling list in a section that
// folds. Each control tells the game what changed through its callback,
// one per kind, and the status line at the bottom shows the last one. The keyboard and a pad
// work too: the volume slider has focus from the start, Tab and the arrows
// or the pad's cross move it, Enter, Space or south presses, and Escape or
// east closes the settings, which Enter opens again. Their title bar drags
// them anywhere in the window, and their edges resize them.
//
//	go run ./examples/13-controls
package main

import (
	"fmt"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/kit"
)

// page places the settings under the window's top, at most 560 points
// wide and 375 high, so resizing them never covers the status line at the
// bottom left, and the slots' list 100 points high.
const page = `
#settings { top: 30px; bottom: auto; width: 380px; max-width: 560px; max-height: 375px; }
#slots { height: 100px; padding: 8px; }
#status { position: fixed; left: 20px; bottom: 12px; }
`

func main() {
	win := platform.NewWindow(
		platform.Title("ember · controls"),
		platform.Size(640, 440),
		platform.FixedSize(),
	)
	s := newSettings()
	win.Add(s.doc)

	ember.New().
		AddEntities(win).
		AddSystems(&Loading{doc: s.doc, bar: s.loading}, &Reopen{settings: s, in: win.Input()}).
		MustRun()
}

// settings is the page: the settings window and its controls, each
// reporting on the status line under it through its callbacks.
type settings struct {
	doc               *ui.Document
	window, slots     *ui.Node
	name, volume      *ui.Node
	music, fullscreen *ui.Node
	loading, status   *ui.Node
}

func newSettings() *settings {
	s := &settings{doc: ui.NewDocument(kit.Theme(), ui.CSS(page))}
	s.name = kit.TextInput("Ada", ui.Placeholder("Your name"), ui.OnInput(s.typing), ui.OnSubmit(s.greet))
	s.volume = kit.Slider(0.7, kit.Range(0, 1), kit.Step(0.05), ui.Attr("autofocus", ""), ui.OnInput(s.level))
	s.music = kit.Check("Music", ui.Checked(true), ui.OnChange(s.checked))
	s.fullscreen = kit.Check("Fullscreen", ui.OnChange(s.checked))
	s.loading = kit.Progress(0, kit.Range(0, 1))
	s.status = kit.Label("Change something", ui.ID("status"))

	list := kit.Column(kit.Gap(6))
	for n := 1; n <= 12; n++ {
		list.Append(kit.Label(fmt.Sprintf("Save slot %d", n)))
	}
	s.slots = kit.CollapsiblePanel("Save slots", false, kit.Scroll(ui.ID("slots"), list), ui.OnToggle(s.folded))

	s.window = kit.Window(
		"Settings",
		kit.Resizable(),
		ui.ID("settings"),
		ui.OnMove(s.moved),
		ui.OnResize(s.resized),
		ui.OnClose(s.closed),
		ui.OnBack(s.back),
		kit.Column(kit.Gap(12), kit.Padding(20),
			kit.Form(
				kit.Field("Name", s.name),
				kit.Field("Volume", s.volume),
				kit.Field("Loading", s.loading),
			),
			kit.Row(kit.Gap(24), s.music, s.fullscreen),
			s.slots,
		),
	)
	s.doc.Body().Append(s.window, s.status)
	s.doc.Show(s.window)
	return s
}
