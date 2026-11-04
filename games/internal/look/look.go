// Package look is what the games share so they look alike: the window, where
// the board goes, the HUD beside it and the game-over question. The board is
// the game's own twod entity; the HUD is ui.
package look

import (
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/kit"
)

// The board's colours, for color.Hex.
const (
	Board = "#2b2d42"
	Empty = "#3d405b"
	Ink   = "#f4f1de"
)

// The window is 640 by 440; the HUD's panel takes the right from HUDLeft,
// the board the rest.
const (
	width, height = 640, 440
	HUDLeft       = 410
)

const css = `html, body { height: 100%; }
body { display: flex; align-items: center; justify-content: flex-end; padding-right: 40px; }
#side { width: 190px; }`

// Window is a game's window: the same size and title style for every game.
func Window(title string) *platform.Window {
	return platform.NewWindow(platform.Title("ember · "+title), platform.Size(width, height), platform.FixedSize())
}

// Place is where a board w by h goes: centred left of the HUD, its top-left
// at the entity's position.
func Place(w, h float32) twod.Option {
	return twod.At((HUDLeft-w)/2, (height-h)/2)
}

// HUD is a game's document for the panel beside the board.
func HUD() *ui.Document {
	return ui.NewDocument(kit.Theme(), ui.CSS(css))
}

// Side is the panel beside the board.
func Side(title string, rows ...*ui.Node) *ui.Node {
	col := kit.Column(kit.Gap(10))
	col.Append(rows...)
	return kit.Panel(kit.Title(title), ui.ID("side"), kit.Padding(16), col)
}

// Stat is a named number on the side panel; set the value's text to change it.
func Stat(name, id string) (*ui.Node, *ui.Node) {
	value := kit.Label("0", ui.ID(id), kit.Right())
	return kit.Row(kit.JustifyBetween(), kit.Label(name), value), value
}

// Keys lists the controls, one per line.
func Keys(lines ...string) *ui.Node {
	col := kit.Column(kit.Gap(2))
	for _, l := range lines {
		col.Append(kit.Label(l, kit.Caps()))
	}
	return col
}

// GameOver is the question shown at the end; again runs when the player
// asks for a new game. Append it to the body and show it with ShowModal.
func GameOver(again func()) *ui.Node {
	alert := kit.Alert("Game over", "", kit.Button("New game", kit.Primary(), ui.Value("again")))
	ui.On(alert, ui.EventClose, func(e ui.Event) {
		if e.Value == "again" {
			again()
		}
	})
	return alert
}
