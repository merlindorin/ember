package main

import "github.com/merlindorin/ember/ui"

func controls() map[string]element {
	return map[string]element{
		"button":   {button, 240},
		"input":    {input, 296},
		"select":   {selectElement, 288},
		"option":   {option, 200},
		"textarea": {textarea, 300},
		"progress": {progress, 296},
		"meter":    {meter, 272},
	}
}

// button is a pair of a game's dialog buttons, the second disabled.
func button(*ui.Document) *ui.Node {
	return ui.Div(
		ui.Button(ui.ID("save"), ui.Text("Save")),
		ui.Text(" "),
		ui.Button(ui.Text("Delete"), ui.Disabled()),
	)
}

// form lays labels and controls out in two columns, the controls one
// width; a label sits on its control's text, or centred beside a control
// with none.
func form(cells ...ui.Arg) *ui.Node {
	return ui.Div(
		append(
			[]ui.Arg{
				ui.Style(
					"display: grid; width: max-content; grid-template-columns: max-content 180px; " +
						"gap: 6px 10px; align-items: baseline",
				),
			},
			cells...)...)
}

const (
	field   = "width: 100%"
	beside  = "align-self: center"
	ownSize = "align-self: center; justify-self: start"
)

// input is a settings form with one input of each type.
func input(*ui.Document) *ui.Node {
	return form(
		ui.Label(ui.For("name"), ui.Text("Name")),
		ui.Input(ui.ID("name"), ui.Style(field), ui.Value("Ayla")),
		ui.Label(ui.For("gold"), ui.Text("Gold")),
		ui.Input(ui.ID("gold"), ui.Style(field), ui.Type("number"), ui.Value("250"), ui.Attr("min", "0")),
		ui.Label(ui.For("volume"), ui.Text("Volume"), ui.Style(beside)),
		ui.Input(ui.ID("volume"), ui.Style(ownSize+"; width: 100%"), ui.Type("range"), ui.Value("70")),
		ui.Label(ui.For("subtitles"), ui.Text("Subtitles"), ui.Style(beside)),
		ui.Input(ui.ID("subtitles"), ui.Style(ownSize), ui.Type("checkbox"), ui.Checked(true)),
		ui.Label(ui.For("music"), ui.Text("Music"), ui.Style(beside)),
		ui.Input(ui.ID("music"), ui.Style(ownSize), ui.Type("checkbox"), ui.Attr("switch", ""), ui.Checked(true)),
		ui.Label(ui.Text("Camera"), ui.Style(beside)),
		ui.Div(
			ui.Style(beside),
			ui.Label(
				ui.Input(ui.Type("radio"), ui.Attr("name", "camera"), ui.Value("follow"), ui.Checked(true)),
				ui.Text(" Follow "),
			),
			ui.Label(ui.Input(ui.Type("radio"), ui.Attr("name", "camera"), ui.Value("free")), ui.Text(" Free")),
		),
		ui.Label(ui.For("search"), ui.Text("Search")),
		ui.Input(ui.ID("search"), ui.Style(field), ui.Type("search"), ui.Placeholder("Item name")),
		ui.Label(ui.For("password"), ui.Text("Password")),
		ui.Input(ui.ID("password"), ui.Style(field), ui.Type("password"), ui.Value("dragon")),
	)
}

// selectElement is a game's difficulty, chosen, and its class, not yet.
func selectElement(*ui.Document) *ui.Node {
	return form(
		ui.Label(ui.For("difficulty"), ui.Text("Difficulty")),
		ui.Select(ui.ID("difficulty"), ui.Style(field),
			ui.Option(ui.Text("Easy")),
			ui.Option(ui.Text("Normal"), ui.Selected()),
			ui.Option(ui.Text("Hard")),
		),
		ui.Label(ui.For("class"), ui.Text("Class")),
		ui.Select(ui.ID("class"), ui.Style(field), ui.Placeholder("Pick one"),
			ui.Option(ui.Value("warrior"), ui.Text("Warrior")),
			ui.Option(ui.Value("mage"), ui.Text("Mage")),
			ui.Option(ui.Value("rogue"), ui.Text("Rogue")),
		),
	)
}

// option is a select's options: one chosen, one not yet unlocked.
func option(*ui.Document) *ui.Node {
	return ui.Label(ui.Text("Difficulty "), ui.Select(
		ui.Option(ui.Value("easy"), ui.Text("Easy")),
		ui.Option(ui.Value("normal"), ui.Text("Normal"), ui.Selected()),
		ui.Option(ui.Value("nightmare"), ui.Text("Nightmare"), ui.Disabled()),
	))
}

// textarea is a quest log's notes, four rows of thirty characters.
func textarea(*ui.Document) *ui.Node {
	return ui.Textarea(ui.Attr("rows", "4"), ui.Attr("cols", "30"),
		ui.Text("Find the old well north of the village. The miller said the key is under the bridge."))
}

// progress is a download at 70% and a wait of unknown length.
func progress(*ui.Document) *ui.Node {
	return form(
		ui.Label(ui.For("download"), ui.Text("Download"), ui.Style(beside)),
		ui.Progress(ui.ID("download"), ui.Style(ownSize), ui.Value("70"), ui.Attr("max", "100")),
		ui.Label(ui.For("waiting"), ui.Text("Waiting"), ui.Style(beside)),
		ui.Progress(ui.ID("waiting"), ui.Style(ownSize)),
	)
}

// meter is a hero's needs, each in the colour of its region.
func meter(*ui.Document) *ui.Node {
	var cells []ui.Arg
	needs := []struct{ id, name, value string }{
		{"health", "Health", "0.9"}, {"food", "Food", "0.5"}, {"water", "Water", "0.2"},
	}
	for _, need := range needs {
		cells = append(
			cells,
			ui.Label(ui.For(need.id), ui.Text(need.name), ui.Style(beside)),
			ui.Meter(
				ui.ID(need.id),
				ui.Style(ownSize),
				ui.Value(need.value),
				ui.Attr("low", "0.3"),
				ui.Attr("high", "0.7"),
				ui.Attr("optimum", "1"),
			),
		)
	}
	return form(cells...)
}
