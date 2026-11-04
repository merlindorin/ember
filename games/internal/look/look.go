// Package look is what the games share so they look alike: the window, where
// the board goes, the HUD beside it and the game-over question. The board is
// the game's own twod entity; the HUD is ui.
package look

import (
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
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
body { display: flex; align-items: center; justify-content: center; background: transparent; }
#stage { width: 640px; height: 440px; flex-shrink: 0; display: flex; align-items: center; justify-content: flex-end;
  padding-right: 40px; pointer-events: none; }
#side { width: 190px; display: flex; flex-direction: column; gap: 14px; pointer-events: auto; }
.title { font-size: 44px; font-weight: 700; color: #ff9e64; line-height: 1; }
.stats { display: flex; gap: 8px; }
.stat { flex: 1; display: flex; flex-direction: column; gap: 2px; padding: 8px 12px;
  border-radius: 8px; background: #24263a; }
.label { font-size: 11px; letter-spacing: 1px; text-transform: uppercase; color: #8b8fa8; }
.value { font-size: 24px; font-weight: 700; color: #e8e6f0; }
.new { padding: 10px; border: none; border-radius: 8px; background: #ff9e64; color: #1a1b26; font-weight: 700; }
.new:hover { background: #ffb380; }
.new:focus-visible { outline: 2px solid #ffb380; outline-offset: 2px; }
.keys { display: flex; flex-direction: column; gap: 6px; }
.key { display: flex; align-items: center; gap: 8px; font-size: 12px; color: #8b8fa8; }
dialog.over { width: 280px; padding: 0; border: none; border-radius: 14px; background: #24263a;
  color: #e8e6f0; box-shadow: 0 16px 48px rgba(0, 0, 0, 0.5); }
dialog.over::backdrop { background-color: rgba(10, 10, 18, 0.6); }
.over-body { display: flex; flex-direction: column; align-items: stretch; gap: 10px; padding: 24px; }
.over-title { margin-bottom: 6px; font-size: 28px; font-weight: 700; color: #ff9e64; text-align: center; }
.over-message { font-size: 14px; color: #8b8fa8; text-align: center; }
.item { padding: 9px; border: none; border-radius: 8px; background: #2f3148; color: #e8e6f0; }
.item:hover { background: #3a3d58; }
.item:focus-visible, input.volume:focus-visible { outline: 2px solid #ffb380; outline-offset: 2px; }
.setting { display: flex; align-items: center; justify-content: space-between; font-size: 14px; }
.setting { gap: 10px; }
.setting-name { width: 60px; }
input.volume { flex: 1; height: 18px; background-color: #2f3148; color: #ff9e64; accent-color: #ff9e64; }
.level { width: 40px; text-align: right; font-weight: 700; }
.ranks { display: flex; flex-direction: column; gap: 4px; }
.rank { display: flex; gap: 12px; font-size: 14px; }
.place { width: 20px; color: #8b8fa8; }
.points { flex: 1; font-weight: 700; }
.when { color: #8b8fa8; }
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

// Camera keeps the window's 640 by 440 points in view, centred and scaled to
// fit, so the board grows with the window, full screen too.
func Camera() *twod.Camera {
	cam := twod.NewCamera(twod.ViewSize(width, height))
	cam.SetPosition(geom.V2(width/2, height/2))
	return cam
}

// HUD is a game's document for the column beside the board.
func HUD() *ui.Document {
	return ui.NewDocument(kit.Theme(), ui.CSS(css))
}

// Side is the HUD's column: the game's name, then its parts, on a stage the
// window's size that Fit scales with the board.
func Side(name string, parts ...*ui.Node) *ui.Node {
	side := ui.Div(ui.ID("side"), ui.Div(ui.Class("title"), ui.Text(name)))
	side.Append(parts...)
	return ui.Div(ui.ID("stage"), side)
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

// GameOver is the question shown at the end, in the games' own look; again
// runs when the player asks for a new game. Append it to the body, set its
// message with SetGameOver and show it with ShowModal.
func GameOver(again func()) *ui.Node {
	button := ui.Button(ui.Class("new"), ui.Text("New game"), ui.Value("again"),
		ui.Attr("formmethod", "dialog"), ui.Attr("autofocus", ""))
	over := ui.Dialog(ui.Class("over"), ui.Attr("aria-label", "Game over"),
		ui.Div(ui.Class("over-body"),
			ui.Div(ui.Class("over-title"), ui.Text("Game over")),
			ui.P(ui.Class("over-message"), ui.ID("over-message")),
			button))
	ui.On(over, ui.EventClose, func(e ui.Event) {
		if e.Value == "again" {
			again()
		}
	})
	return over
}

// SetGameOver sets the line under the game-over title.
func SetGameOver(over *ui.Node, message string) {
	over.Children()[0].Children()[1].SetText(message)
}
