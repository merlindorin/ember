// Package look is what the games share so they look alike: the window, where
// the board goes, the HUD beside it and the game-over question. The board is
// the game's own twod entity; the HUD is ui.
package look

import (
	"embed"
	"io/fs"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/kit"
)

// The palette, for color.Hex: the UI pack's wood and parchment on a dark
// green.
const (
	Background = "#1f2a22"
	Surface    = "#5e3a1a"
	Cell       = "#7a4d24"
	Ink        = "#fff1d2"
	Muted      = "#d8c4a0"
	Accent     = "#c58747"
)

// The window is 640 by 440; the HUD takes the right from HUDLeft, the board
// the rest.
const (
	width, height = 640, 440
	HUDLeft       = 410
)

// css lays the HUD out; adventure/adventure.css gives it its look.
const css = `html, body { height: 100%; }
body { display: flex; align-items: center; justify-content: center; background: transparent; }
#stage { width: 640px; height: 440px; flex-shrink: 0; display: flex; align-items: center; justify-content: flex-end;
  padding-right: 16px; pointer-events: none; }
#side { width: 214px; display: flex; flex-direction: column; gap: 10px; pointer-events: auto; }
.title { height: 64px; display: flex; align-items: center; justify-content: center; }
.stats { display: flex; gap: 8px; }
.stat { flex: 1; display: flex; flex-direction: column; gap: 2px; }
.keys { display: grid; grid-template-columns: 1fr 1fr; gap: 4px 6px; }
.key { display: flex; align-items: center; gap: 6px; }
.key.wide { grid-column: 1 / -1; }
.prompts { display: flex; }
.prompts img { width: 32px; height: 32px; }
.pad { display: none; }
body.gamepad .pad { display: flex; }
body.gamepad .kb { display: none; }
dialog.over { width: 300px; padding: 0; }
.over-body { display: flex; flex-direction: column; align-items: stretch; gap: 8px; }
.over-title, .over-message { text-align: center; }
.setting { display: flex; align-items: center; justify-content: space-between; gap: 10px; }
.setting-name { width: 60px; }
input.volume { flex: 1; height: 18px; }
.level { width: 40px; text-align: right; }
.ranks { display: flex; flex-direction: column; gap: 4px; }
.rank { display: flex; gap: 12px; }
.place { width: 20px; }
.points { flex: 1; }`

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

//go:embed adventure
var skin embed.FS

// HUD is a game's document for the column beside the board, in Kenney's UI
// Pack - Pixel Adventure (adventure/CREDITS.md) over the layout.
func HUD() *ui.Document {
	adventure, err := fs.Sub(skin, "adventure")
	if err != nil {
		panic(err) // an embedded folder that is always there
	}
	return ui.NewDocument(kit.Theme(), ui.CSS(css), ui.Stylesheet(adventure, "adventure.css"), ui.Images(adventure))
}

// Side is the HUD's column: the game's name, then its parts, on a stage the
// window's size that Fit scales with the board.
func Side(name string, parts ...*ui.Node) *ui.Node {
	title := ui.Div(ui.Class("title"),
		ui.Span(ui.Class("banner-left")), ui.Span(ui.Class("banner-right")), ui.Span(ui.Class("name"), ui.Text(name)))
	side := ui.Div(ui.ID("side"), title)
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

// AgainButtons start a new game in every game: N or the pad's View button.
func AgainButtons() []input.Button { return []input.Button{input.KeyN, input.GamepadBack} }

// MenuButtons open the menu: the pad's Menu button (Escape opens it through
// the HUD).
func MenuButtons() []input.Button { return []input.Button{input.GamepadStart} }

// AgainControl is the controls line for AgainButtons.
func AgainControl() Control {
	return Control{Keys: []string{"n"}, Pad: []string{"view"}, Does: "new game"}
}

// MenuControl is the controls line for MenuButtons and Escape.
func MenuControl() Control {
	return Control{Keys: []string{"esc"}, Pad: []string{"menu"}, Does: "menu"}
}

// A Control is a line of the controls list: the prompts of its keys and of
// its gamepad buttons, named after the pictures in adventure/prompts
// without their "key-" or "pad-", then what they do.
type Control struct {
	Keys, Pad []string
	Does      string
}

// Controls lists the controls, two a line (one with more than two prompts
// takes a whole line), with the keyboard's prompts or,
// once a gamepad was used, the pad's (Fit switches them).
func Controls(controls ...Control) *ui.Node {
	list := ui.Div(ui.Class("keys"))
	for _, c := range controls {
		line := ui.Div(ui.Class("key"), prompts("kb", "key-", c.Keys), prompts("pad", "pad-", c.Pad),
			ui.Span(ui.Text(c.Does)))
		if max(len(c.Keys), len(c.Pad)) > 2 {
			line.AddClass("wide")
		}
		list.Append(line)
	}
	return list
}

func prompts(class, prefix string, names []string) *ui.Node {
	row := ui.Span(ui.Class("prompts", class))
	for _, n := range names {
		row.Append(ui.Img(ui.Attr("src", "prompts/"+prefix+n+".png"), ui.Attr("alt", n)))
	}
	return row
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
