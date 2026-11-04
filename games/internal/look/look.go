// Package look is what the games share so they look alike: the window, where
// the board goes, the HUD beside it and the game-over question. The board is
// the game's own twod entity; the HUD is ui.
package look

import (
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/kit"
)

// The palette, for color.Hex; the CSS below uses the same values.
const (
	Background = "#1a1b26"
	Surface    = "#24263a"
	Cell       = "#2f3148"
	Ink        = "#e8e6f0"
	Muted      = "#8b8fa8"
	Accent     = "#ff9e64"
)

// The window is 640 by 440; the HUD takes the right from HUDLeft, the board
// the rest.
const (
	width, height = 640, 440
	HUDLeft       = 410
)

const css = `html, body { height: 100%; }
body { display: flex; align-items: center; justify-content: flex-end; padding-right: 40px; background: transparent; }
#side { width: 190px; display: flex; flex-direction: column; gap: 14px; }
.title { font-size: 44px; font-weight: 700; color: #ff9e64; line-height: 1; }
.stats { display: flex; gap: 8px; }
.stat { flex: 1; display: flex; flex-direction: column; gap: 2px; padding: 8px 12px;
  border-radius: 8px; background: #24263a; }
.label { font-size: 11px; letter-spacing: 1px; text-transform: uppercase; color: #8b8fa8; }
.value { font-size: 24px; font-weight: 700; color: #e8e6f0; }
.new { padding: 10px; border: none; border-radius: 8px; background: #ff9e64; color: #1a1b26; font-weight: 700; }
.new:hover { background: #ffb380; }
.keys { display: flex; flex-direction: column; gap: 6px; }
.key { display: flex; align-items: center; gap: 8px; font-size: 12px; color: #8b8fa8; }
kbd { width: 44px; padding: 2px 0; border-radius: 4px; background: #2f3148; color: #e8e6f0;
  font-size: 11px; text-align: center; }`

// Window is a game's window: the same size, title style and background.
func Window(title string) *platform.Window {
	return platform.NewWindow(platform.Title("ember · "+title), platform.Size(width, height), platform.FixedSize(),
		platform.Background(color.Hex(Background)))
}

// Place is where a board w by h goes: centred left of the HUD, its top-left
// at the entity's position.
func Place(w, h float32) twod.Option {
	return twod.At((HUDLeft-w)/2, (height-h)/2)
}

// HUD is a game's document for the column beside the board.
func HUD() *ui.Document {
	return ui.NewDocument(kit.Theme(), ui.CSS(css))
}

// Side is the HUD's column: the game's name, then its parts.
func Side(name string, parts ...*ui.Node) *ui.Node {
	side := ui.Div(ui.ID("side"), ui.Div(ui.Class("title"), ui.Text(name)))
	side.Append(parts...)
	return side
}

// Stat is a small card with a name and a number; set the value's text to
// change it.
func Stat(name, id string) (*ui.Node, *ui.Node) {
	value := ui.Span(ui.Class("value"), ui.ID(id), ui.Text("0"))
	return ui.Div(ui.Class("stat"), ui.Span(ui.Class("label"), ui.Text(name)), value), value
}

// Stats lines cards up side by side.
func Stats(cards ...*ui.Node) *ui.Node {
	row := ui.Div(ui.Class("stats"))
	row.Append(cards...)
	return row
}

// Card is a named card holding other content, as a Stat holds a number.
func Card(name string, content *ui.Node) *ui.Node {
	return ui.Div(ui.Class("stat"), ui.Span(ui.Class("label"), ui.Text(name)), content)
}

// Keys lists the controls, a key cap then what it does, pairs of strings.
func Keys(pairs ...string) *ui.Node {
	list := ui.Div(ui.Class("keys"))
	for i := 0; i+1 < len(pairs); i += 2 {
		list.Append(ui.Div(ui.Class("key"), ui.Kbd(ui.Text(pairs[i])), ui.Span(ui.Text(pairs[i+1]))))
	}
	return list
}

// NewGame is the button that starts again. It gives the focus back, so the
// arrows go to the game rather than move between buttons.
func NewGame(doc *ui.Document, again func()) *ui.Node {
	return ui.Button(ui.Class("new"), ui.ID("new-game"), ui.Text("New game"), ui.OnClick(func() {
		again()
		doc.Focus(nil)
	}))
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
