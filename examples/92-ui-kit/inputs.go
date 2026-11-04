package main

import (
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/kit"
)

// check is a map editor's view options, one checked and one disabled.
func check(*gallery) *ui.Node {
	return kit.Column(kit.Gap(12),
		kit.Check("Show grid", ui.Checked(true)),
		kit.Check("Snap to tiles"),
		kit.Check("Solid tile", ui.Disabled()),
	)
}

// switches are a game's audio settings, one on and one disabled.
func switches(*gallery) *ui.Node {
	return kit.Column(kit.Gap(12),
		kit.Switch("Music", ui.Checked(true)),
		kit.Switch("Sound effects"),
		kit.Switch("Vibration", ui.Disabled()),
	)
}

// slider is a settings form: two volumes and a zoom on half steps.
func slider(*gallery) *ui.Node {
	return kit.Form(
		kit.Field("Music", kit.Slider(70, ui.ID("music"))),
		kit.Field("Effects", kit.Slider(40, ui.ID("effects"))),
		kit.Field("Zoom", kit.Slider(2, kit.Range(1, 4), kit.Step(0.5), ui.ID("zoom"))),
	)
}

// progress is a loading bar, experience towards a level, and health
// without its percentage.
func progress(*gallery) *ui.Node {
	return kit.Form(
		kit.Field("Loading", kit.Progress(65)),
		kit.Field("Experience", kit.Progress(340, kit.Range(0, 500))),
		kit.Field("Health", kit.Progress(80, kit.ShowValue(false))),
	)
}

// textInput is a hero's name, a password and a search field.
func textInput(*gallery) *ui.Node {
	return kit.Form(
		kit.Field("Name", kit.TextInput("", ui.Placeholder("Hero name"), kit.MaxLength(16))),
		kit.Field("Password", kit.TextInput("", kit.Password(), ui.Placeholder("Password"))),
		kit.Field("Server", kit.TextInput("eu-west.ember.gg", kit.ReadOnly(), ui.ID("server"))),
		kit.Field("Find", kit.SearchField("")),
	)
}

// textArea is a quest log six lines high.
func textArea(*gallery) *ui.Node {
	return kit.TextArea("Find the crypt key.\nBring it to the ferryman.",
		ui.ID("quest-notes"), ui.Placeholder("Quest notes"), kit.Rows(6))
}

// numberField is a tile map's size in cells and its zoom on quarter steps.
func numberField(*gallery) *ui.Node {
	return kit.Form(
		kit.Field("Width", kit.NumberField(32, kit.Range(1, 256), ui.ID("width"))),
		kit.Field("Height", kit.NumberField(24, kit.Range(1, 256), ui.ID("height"))),
		kit.Field("Zoom", kit.NumberField(1, kit.Range(0.25, 4), kit.Step(0.25), ui.ID("zoom"))),
	)
}

// choice is a level editor's layer and time of day, segmented, both
// filling the form's column, and brush colour as swatches.
func choice(*gallery) *ui.Node {
	colours := []struct{ name, hex string }{
		{"red", "#c0543c"},
		{"ochre", "#d9a441"},
		{"moss", "#5f9d58"},
		{"sea", "#4a7bb7"},
	}
	var swatches []*ui.Node
	for _, c := range colours {
		swatches = append(
			swatches,
			ui.Span(
				ui.Style("display: block; width: 18px; height: 18px; border-radius: 3px; background-color: "+c.hex),
			),
		)
	}
	brush := kit.Choice(2, swatches, ui.ID("brush"))
	for i, c := range colours {
		brush.Children()[i].SetAttr("aria-label", c.name)
	}
	return kit.Form(
		kit.Field("Layer", kit.Choice(0, choiceLabels("Tiles", "Objects", "Lights"), kit.Fill())),
		kit.Field("Time", kit.Choice(1, choiceLabels("Day", "Dusk", "Night"), kit.Segmented(), kit.Fill())),
		kit.Field("Brush", brush),
	)
}

func choiceLabels(names ...string) []*ui.Node {
	var out []*ui.Node
	for _, n := range names {
		out = append(out, kit.Label(n))
	}
	return out
}
