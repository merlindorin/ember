package main

import (
	"fmt"
	"strconv"

	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/kit"
)

// Settings is what the game keeps once the player saves.
type Settings struct {
	Name   string
	Volume float32
	Music  bool
}

// page centres the form and sizes its fields and button.
const page = `html, body { height: 100%; } body { display: flex; align-items: center; justify-content: center; }
#name, #volume { width: 240px; } #save { width: 120px; height: 34px; }`

// form is the settings form and the controls the game reads.
type form struct {
	doc                       *ui.Document
	name, volume, music, save *ui.Node
	status                    *ui.Node
	settings                  *Settings
}

// newForm gives the controls a test or a tool drives IDs, so their
// automation IDs, such as "settings/name", don't change with their text.
// Save copies the form into settings when it is clicked.
func newForm(settings *Settings) *form {
	f := &form{doc: ui.NewDocument(kit.Theme(), ui.CSS(page)), settings: settings}
	f.name = kit.TextInput("", ui.ID("name"), ui.Placeholder("Your name"))
	f.volume = kit.Slider(0.7, ui.ID("volume"), kit.Range(0, 1), kit.Step(0.05))
	f.music = kit.Check("Music", ui.ID("music"), ui.Checked(true))
	f.save = kit.Button("Save", ui.ID("save"), ui.OnClick(f.saved))
	f.status = kit.Label("Not saved yet", ui.ID("status"))
	f.doc.Body().Append(kit.Panel(ui.ID("settings"), kit.Padding(20),
		kit.Column(kit.Gap(12),
			kit.Form(kit.Field("Name", f.name), kit.Field("Volume", f.volume), kit.Field("", f.music)),
			kit.Row(kit.Gap(16), kit.AlignCenter(), f.save, f.status),
		)))
	return f
}

func (f *form) saved() {
	volume, _ := strconv.ParseFloat(f.doc.Value(f.volume), 32)
	*f.settings = Settings{
		Name:   f.doc.Value(f.name),
		Volume: float32(volume),
		Music:  f.doc.Checked(f.doc.ByID("music")),
	}
	f.status.SetText(fmt.Sprintf("Saved %s, %.0f%%", f.settings.Name, f.settings.Volume*100))
}
