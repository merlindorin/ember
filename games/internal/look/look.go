// Package look is what the games share so they look alike: the window, the
// page around the board, the side panel and the game-over question.
package look

import (
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/kit"
)

// The board's colours, for color.Hex.
const (
	Board = "#2b2d42"
	Empty = "#3d405b"
	Ink   = "#f4f1de"
)

const css = `html, body { height: 100%; }
body { display: flex; align-items: center; justify-content: center; gap: 28px; }
#side { width: 190px; }`

// Window is a game's window: the same size and title style for every game.
func Window(title string) *platform.Window {
	return platform.NewWindow(platform.Title("ember · "+title), platform.Size(640, 440), platform.FixedSize())
}

// Page is a game's document, with its own CSS after the shared one.
func Page(own string) *ui.Document {
	return ui.NewDocument(kit.Theme(), ui.CSS(css+"\n"+own))
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
