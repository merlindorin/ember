package main

import "github.com/lantervale/ember/ui"

func texts() map[string]element {
	return map[string]element{
		"p":      {p, 360},
		"h1":     {h1, 260},
		"h2":     {h2, 240},
		"h3":     {h3, 280},
		"h4":     {h4, 190},
		"span":   {span, 120},
		"strong": {strong, 190},
		"em":     {em, 220},
		"code":   {code, 310},
		"label":  {label, 260},
		"kbd":    {kbd, 260},
		"samp":   {samp, 300},
		"pre":    {pre, 160},
	}
}

// p is two paragraphs of a quest log, wrapping and stacked without a gap.
func p(*ui.Document) *ui.Node {
	return ui.Div(
		ui.P(ui.Text("The miller's daughter has not come home. Her lantern was found by the old bridge.")),
		ui.P(ui.Text("Search the riverbank before nightfall.")),
	)
}

// h1 is a screen's title over a line of body text.
func h1(*ui.Document) *ui.Node {
	return ui.Div(
		ui.H1(ui.Text("Inventory")),
		ui.P(ui.Text("Twelve slots, four of them free.")),
	)
}

// h2 is a section's title over a line of body text.
func h2(*ui.Document) *ui.Node {
	return ui.Div(
		ui.H2(ui.Text("Settings")),
		ui.P(ui.Text("Sound, controls and display.")),
	)
}

// h3 is a group's title over a line of body text.
func h3(*ui.Document) *ui.Node {
	return ui.Div(
		ui.H3(ui.Text("Controls")),
		ui.P(ui.Text("Keyboard and gamepad bindings.")),
	)
}

// h4 is a small title over a line of body text, the same size.
func h4(*ui.Document) *ui.Node {
	return ui.Div(
		ui.H4(ui.Text("Gamepad")),
		ui.P(ui.Text("Hold South to sprint.")),
	)
}

// span is a score whose number a span colours.
func span(*ui.Document) *ui.Node {
	return ui.P(
		ui.Text("Gold: "),
		ui.Span(ui.Style("color: #b8860b"), ui.Text("1,250")),
	)
}

// strong is a hint whose key is bold.
func strong(*ui.Document) *ui.Node {
	return ui.P(
		ui.Text("Press "),
		ui.Strong(ui.Text("Space")),
		ui.Text(" to jump."),
	)
}

// em is a message whose state is italic.
func em(*ui.Document) *ui.Node {
	return ui.P(
		ui.Text("The cellar door is "),
		ui.Em(ui.Text("locked")),
		ui.Text("."),
	)
}

// code is a chat hint whose command is monospace.
func code(*ui.Document) *ui.Node {
	return ui.P(
		ui.Text("Type "),
		ui.Code(ui.Text("/invite name")),
		ui.Text(" to add a player."),
	)
}

// label is a player's name field, its label naming the input by For.
func label(*ui.Document) *ui.Node {
	return ui.Div(
		ui.Label(ui.For("name"), ui.Text("Name")),
		ui.Text(" "),
		ui.Input(ui.ID("name"), ui.Value("Ayla")),
	)
}

// kbd is a key hint: the keys the player presses.
func kbd(*ui.Document) *ui.Node {
	return ui.P(
		ui.Text("Press "),
		ui.Kbd(ui.Text("Ctrl")),
		ui.Text("+"),
		ui.Kbd(ui.Text("S")),
		ui.Text(" to save."),
	)
}

// samp is what the game answered to a command.
func samp(*ui.Document) *ui.Node {
	return ui.P(
		ui.Text("The console says "),
		ui.Samp(ui.Text("player not found")),
		ui.Text("."),
	)
}

// pre is a save file's lines, kept as typed.
func pre(*ui.Document) *ui.Node {
	return ui.Pre(ui.Text("level:  3\nhero:   Ayla\ngold:   1250"))
}
