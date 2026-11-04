package main

import (
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/kit"
)

// label is a character sheet's header: a caption, a name, a muted line, a
// wrapped bio and a save name cut short.
func label(*gallery) *ui.Node {
	return kit.Column(ui.Style("width: 240px"), kit.Gap(8),
		kit.Label("Character", kit.Caps()),
		kit.Label("Heron the Wanderer"),
		kit.Label("Level 12 ranger", kit.FormLabel()),
		kit.Label("Wanders the salt marshes north of the crypt, trading herbs for stories.", kit.Wrap()),
		kit.Label("Last save: Crypt of the Drowned King, second floor", kit.Truncate()),
	)
}

// richText is a quest log: bold, italic and coloured runs, one wrapped.
func richText(*gallery) *ui.Node {
	return kit.Column(ui.Style("width: 260px"), kit.Gap(8),
		kit.RichText("[b]The Drowned King[/b]"),
		kit.RichText("Bring the [i]rusty key[/i] to the ferryman.", kit.Wrap()),
		kit.RichText("Reward: [color=#ffce76]250 gold[/color], [color=tomato]Ember Blade[/color]"),
	)
}

// icons is a status line of icons and two buttons with an icon before their text.
func icons(g *gallery) *ui.Node {
	return kit.Column(kit.Gap(16),
		kit.Row(kit.AlignCenter(), kit.Gap(6),
			kit.NewIcon(g.icons, "heart", ui.Style("color: tomato; font-size: 20px")), kit.Label("100"),
			kit.NewIcon(g.icons, "shield", ui.Style("font-size: 20px")), kit.Label("35"),
			kit.NewIcon(g.icons, "coins", ui.Style("color: var(--accent-color); font-size: 20px")), kit.Label("250"),
		),
		kit.Row(kit.Gap(8),
			kit.Button("Save", kit.Icon(g.icons, "save")),
			kit.Button("Delete", kit.Destructive(), kit.Icon(g.icons, "trash-2")),
		),
	)
}

// image is a nine-patch dialog, the picture as it is, fitted and tiled,
// and a health bar of two colour rects.
func image(*gallery) *ui.Node {
	return kit.Column(ui.Style("width: 260px"), kit.Gap(12),
		kit.NinePatch("panel.png", 8, 8, 8, 8, kit.Padding(4), kit.Column(kit.Gap(2),
			kit.Label("Delete this file?", kit.AlertTitle()),
			kit.Label("This can't be undone.", kit.AlertMessage()),
		)),
		kit.Row(kit.AlignStart(), kit.Gap(12),
			kit.Image("panel.png"),
			kit.Image("panel.png", kit.KeepAspectCentered(), ui.Style("width: 96px; height: 40px")),
			kit.Column(ui.Style("width: 96px; height: 40px"), kit.TileImage("panel.png", kit.Expand())),
		),
		kit.Grid(2, ui.Style("height: 8px"), kit.Columns("7fr 3fr"), kit.Gap(0),
			kit.ColorRect("tomato"),
			kit.ColorRect("rgba(255, 255, 255, 0.15)"),
		),
	)
}

// separator is a file menu split in groups and a toolbar with a vertical one.
func separator(*gallery) *ui.Node {
	return kit.Column(ui.Style("width: 200px"), kit.Gap(12),
		kit.Panel(kit.Column(kit.Padding(4),
			kit.Label("New file"), kit.Label("Open…"),
			kit.Separator(),
			kit.Label("Save"), kit.Label("Save as…"),
			kit.Separator(),
			kit.Label("Quit"),
		)),
		kit.Toolbar(
			kit.Label("Cut"), kit.Label("Copy"),
			kit.Separator(kit.Vertical()),
			kit.Label("Paste"),
		),
	)
}

// rowAndColumn is an audio settings panel: a column of rows, one value
// pushed to the end, and a footer of buttons on the right.
func rowAndColumn(*gallery) *ui.Node {
	return kit.Panel(ui.Style("width: 320px"),
		kit.Column(kit.Body(),
			kit.Row(kit.JustifyBetween(), kit.Label("Music"), kit.Label("80%")),
			kit.Row(kit.JustifyBetween(), kit.Label("Effects"), kit.Label("65%")),
			kit.Row(kit.Label("Output"), kit.Label("Speakers", kit.Expand(), kit.Right())),
		),
		kit.Footer(kit.JustifyEnd(),
			kit.Button("Cancel"),
			kit.Button("Apply", kit.Primary()),
		),
	)
}

// grid is a tile's properties: names in a column as wide as the widest,
// values taking the rest.
func grid(*gallery) *ui.Node {
	prop := func(name, value string) []ui.Arg {
		return []ui.Arg{kit.Label(name, kit.FormLabel()), kit.Label(value)}
	}
	args := []ui.Arg{kit.Columns("max-content 1fr"), kit.Gap(8), kit.Padding(12)}
	args = append(args, prop("name", "grass")...)
	args = append(args, prop("solid", "false")...)
	args = append(args, prop("light radius", "64")...)
	args = append(args, prop("occluder group", "walls")...)
	return kit.Panel(ui.Style("width: 260px"), kit.Title("Tile 12"), kit.Grid(2, args...))
}

// panel is a titled panel, a collapsible one and two cards of a title menu.
func panel(*gallery) *ui.Node {
	return kit.Column(ui.Style("width: 280px"), kit.Gap(12),
		kit.Panel(kit.Title("Settings"), kit.Column(kit.Padding(8), kit.Label("Volume, language"))),
		kit.CollapsiblePanel("Lights", true, kit.Column(kit.Padding(8), kit.Label("Torch radius 64"))),
		kit.Row(kit.Gap(12),
			kit.Card(kit.Label("New game")),
			kit.Card(kit.Label("Continue")),
		),
	)
}

// hint is a level select's key line in each device's words, a row per
// device by its own data-input-device; touch has none and reads the
// keyboard's.
func hint(*gallery) *ui.Node {
	rows := []ui.Arg{kit.Columns("max-content 1fr"), kit.Gap(10)}
	for _, device := range []string{"keyboard", "mouse", "gamepad", "touch"} {
		rows = append(rows, ui.Div(ui.Style("display: contents"), ui.Attr("data-input-device", device),
			kit.Label(device, kit.FormLabel()),
			kit.Hint(
				kit.KeyboardHint("Arrows to look · Enter to choose"),
				kit.MouseHint("Click a level to choose"),
				kit.GamepadHint("Stick to look · South to choose"),
			),
		))
	}
	return kit.Grid(2, rows...)
}
