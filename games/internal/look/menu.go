package look

import (
	"fmt"
	"strconv"

	"github.com/merlindorin/ember/games/internal/save"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/world"
)

// Hooks are what the Escape menu does in its game.
type Hooks struct {
	NewGame func()
	// Data is the game's saved data, read for the leaderboard and the sound.
	Data func() *save.Data
	// SoundChanged runs after the player changed the sound; the new values
	// are in Data.
	SoundChanged func()
}

// Menu is the Escape menu: New game, the leaderboard, music and sound, and
// Quit. While it is open the game waits.
type Menu struct {
	doc                    *ui.Document
	main, board, options   *ui.Node
	ranks                  *ui.Node
	music, effects, toggle *ui.Node
	hooks                  Hooks
}

// NewMenu adds the menu to the document; Escape opens it.
func NewMenu(doc *ui.Document, hooks Hooks) *Menu {
	m := &Menu{doc: doc, hooks: hooks}
	m.main = dialog("Paused",
		item("Resume", "resume"), item("New game", "new"), item("Leaderboard", "board"),
		item("Music & sound", "sound"), item("Quit", "quit"))
	ui.On(m.main, ui.EventClose, func(w *world.World, e ui.Event) {
		switch e.Value {
		case "new":
			m.hooks.NewGame()
		case "board":
			m.showBoard()
		case "sound":
			m.showOptions()
		case "quit":
			w.Quit()
		}
	})

	m.ranks = ui.Div(ui.Class("ranks"))
	m.board = dialog("Leaderboard", m.ranks, item("Back", "back"))
	ui.On(m.board, ui.EventClose, m.Show)

	m.music, m.effects = ui.Span(ui.Class("level")), ui.Span(ui.Class("level"))
	m.toggle = ui.Button(ui.Class("item"), ui.ID("music-toggle"), ui.OnClick(func() {
		d := m.hooks.Data()
		d.MusicOff = !d.MusicOff
		m.changed()
	}))
	m.options = dialog("Music & sound",
		m.stepper("Music", "music", m.music, func(d *save.Data, by float32) { d.Music = step(d.Music, by) }),
		m.stepper("Effects", "effects", m.effects, func(d *save.Data, by float32) { d.Effects = step(d.Effects, by) }),
		m.toggle, item("Back", "back"))
	ui.On(m.options, ui.EventClose, m.Show)

	doc.Body().Append(m.main, m.board, m.options)
	ui.On(doc.Body(), ui.EventBack, m.Show)
	return m
}

// Show opens the menu.
func (m *Menu) Show() {
	if !m.Open() {
		m.doc.ShowModal(m.main)
	}
}

// Open reports whether the menu or one of its pages is showing.
func (m *Menu) Open() bool {
	for _, d := range []*ui.Node{m.main, m.board, m.options} {
		if _, open := d.Attr("open"); open {
			return true
		}
	}
	return false
}

func (m *Menu) showBoard() {
	for _, row := range m.ranks.Children() {
		m.ranks.Remove(row)
	}
	scores := m.hooks.Data().Scores
	if len(scores) == 0 {
		m.ranks.Append(ui.P(ui.Class("over-message"), ui.Text("No score yet.")))
	}
	for i, e := range scores {
		m.ranks.Append(ui.Div(ui.Class("rank"),
			ui.Span(ui.Class("place"), ui.Text(strconv.Itoa(i+1))),
			ui.Span(ui.Class("points"), ui.Text(strconv.Itoa(e.Score))),
			ui.Span(ui.Class("when"), ui.Text(e.When.Local().Format("2 Jan")))))
	}
	m.doc.ShowModal(m.board)
}

func (m *Menu) showOptions() {
	m.show()
	m.doc.ShowModal(m.options)
}

func (m *Menu) changed() {
	m.show()
	m.hooks.SoundChanged()
}

// show writes the sound settings on the options page.
func (m *Menu) show() {
	d := m.hooks.Data()
	m.music.SetText(fmt.Sprintf("%.0f%%", d.Music*100))
	m.effects.SetText(fmt.Sprintf("%.0f%%", d.Effects*100))
	m.toggle.SetText("Music: on")
	if d.MusicOff {
		m.toggle.SetText("Music: off")
	}
}

// stepper is a setting's row: its name, − and + buttons, its level.
func (m *Menu) stepper(name, id string, level *ui.Node, change func(d *save.Data, by float32)) *ui.Node {
	button := func(text string, by float32) *ui.Node {
		return ui.Button(ui.Class("step"), ui.ID(id+"-"+text), ui.Attr("aria-label", name+" "+text), ui.Text(text),
			ui.OnClick(func() {
				change(m.hooks.Data(), by)
				m.changed()
			}))
	}
	return ui.Div(ui.Class("setting"), ui.Span(ui.Text(name)),
		ui.Div(ui.Class("stepper"), button("−", -0.1), level, button("+", 0.1)))
}

func step(v, by float32) float32 { return min(max(float32(int((v+by)*10+0.5))/10, 0), 1) }

// dialog is one of the games' modals: a title over its parts.
func dialog(title string, parts ...*ui.Node) *ui.Node {
	body := ui.Div(ui.Class("over-body"), ui.Div(ui.Class("over-title"), ui.Text(title)))
	body.Append(parts...)
	return ui.Dialog(ui.Class("over"), ui.Attr("aria-label", title), body)
}

// item is a menu button that closes its dialog with value.
func item(text, value string) *ui.Node {
	return ui.Button(ui.Class("item"), ui.Text(text), ui.Value(value), ui.Attr("formmethod", "dialog"))
}
