package main

import (
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/kit"
	"github.com/lantervale/ember/world"
)

// settings is a small settings screen: a heading, a form and two buttons.
func settings(*gallery) *ui.Node {
	return kit.Panel(ui.Style("width: 380px; padding: 20px 24px"),
		ui.H2(ui.Text("Settings"), ui.Style("margin: 0 0 12px")),
		kit.Form(
			kit.Field("Name", kit.TextInput("Ayla", ui.ID("name"))),
			kit.Field("Volume", kit.Slider(60, ui.ID("volume"))),
			kit.Field("Music", kit.Switch("", ui.ID("music"), ui.Checked(true))),
		),
		kit.Row(kit.Gap(8), kit.JustifyEnd(), ui.Style("margin-top: 12px"),
			kit.Button("Cancel"),
			kit.Button("Save", kit.Primary(), ui.ID("save")),
		),
	)
}

// styled is plain elements given their look by the style attribute.
func styled(*gallery) *ui.Node {
	return ui.Div(
		ui.Style(
			"display: grid; grid-template-columns: 48px 1fr; gap: 4px 12px; align-items: center; padding: 12px; "+
				"border: 1px solid #8a7a5a; border-radius: 8px; background-color: #2b2f3a; width: 280px",
		),
		ui.Div(ui.Style("grid-row: span 2; width: 48px; height: 48px; border-radius: 24px; background-color: #e0a040")),
		ui.Strong(ui.Text("Ember Quest"), ui.Style("color: #f2e6c9; font-size: 18px")),
		ui.Span(ui.Text("Area 2 · 1.5k to home"), ui.Style("color: #9a9384; font-size: 13px")),
	)
}

// mapCanvas is the canvas element: a map of rooms drawn by the game, and a
// marker where the pointer is, which Pointer redraws each frame.
func mapCanvas(g *gallery) *ui.Node {
	d := g.doc
	var minimap *ui.Node
	minimap = ui.Canvas(func(c *twod.Canvas) {
		c.Rect(geom.Rect{W: c.Size().X, H: c.Size().Y}, color.Hex("#1d2129"))
		rooms := []geom.Rect{
			{X: 20, Y: 20, W: 70, H: 50}, {X: 110, Y: 30, W: 60, H: 90}, {X: 30, Y: 90, W: 60, H: 40},
		}
		for _, room := range rooms {
			c.Rect(room, color.Hex("#5a4a30"))
		}
		if at, ok := d.PointerOn(minimap); ok {
			c.Circle(at, 5, color.Hex("#e0a040"))
		}
	}, ui.ID("minimap"), ui.Attr("width", "200"), ui.Attr("height", "150"),
		ui.Style("border: 1px solid #8a7a5a; border-radius: 6px"))
	return minimap
}

// Pointer has the minimap drawn again each frame, so its marker follows
// the pointer; a canvas left alone keeps its last drawing.
type Pointer struct{ doc *ui.Document }

func (p *Pointer) Update(world.Frame) {
	if minimap := p.doc.ByID("minimap"); minimap != nil {
		p.doc.Invalidate(minimap)
	}
}
