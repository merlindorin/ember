// The architect's inspector mockup A: a tile's properties and objects in
// name | value | type rows, for the pixel comparison with Chrome.
package main

import (
	_ "embed"
	"testing/fstest"

	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/internal/pages"
)

//go:embed mocka.css
var sheet []byte

const w, h = 520, 720

func sel(placeholder string, chosen string, items ...string) *ui.Node {
	args := []ui.Arg{}
	if placeholder != "" {
		args = append(args, ui.Placeholder(placeholder))
	}
	for _, it := range items {
		o := []ui.Arg{ui.Text(it)}
		if it == chosen {
			o = append(o, ui.Selected())
		}
		args = append(args, ui.Option(o...))
	}
	return ui.Select(args...)
}

func row(name string, value *ui.Node, typ string) *ui.Node {
	return ui.Div(ui.Class("row"),
		ui.Span(ui.Class("name"), ui.Text(name)),
		value,
		ui.Span(ui.Class("type"), ui.Text(typ)),
	)
}

func field(v, placeholder string) *ui.Node {
	if v == "" {
		return ui.Input(ui.Placeholder(placeholder))
	}
	return ui.Input(ui.Value(v))
}

func pair(a, b string) *ui.Node {
	return ui.Div(ui.Class("pair"), ui.Input(ui.Value(a)), ui.Input(ui.Value(b)))
}

func page() *ui.Document {
	doc := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"mocka.css": {Data: sheet}}, "mocka.css"))
	doc.Body().Append(ui.Section(ui.Class("panel"),
		ui.Details(ui.Open(),
			ui.Summary(ui.Text("tile 0")),
			ui.Div(ui.Class("row", "head"),
				ui.Span(ui.Text("NAME")), ui.Span(ui.Text("VALUE")), ui.Span(ui.Text("TYPE"))),
			ui.Div(ui.Class("group"), ui.Text("properties")),
			row("solid", sel("", "true", "false", "true"), "bool"),
			row("tall", sel("false (default)", "", "false", "true"), "bool"),
			row("shadow", sel("cast (default)", "", "cast", "none"), "string"),
			row("occluder_group", sel("0 (default)", "", "0", "1", "2"), "int"),
			row("height", field("64", ""), "float"),
			row("emissive", field("", "0"), "float"),
			row("animation", ui.Span(ui.Class("none"), ui.Text("none")), ""),
			ui.Hr(),
			ui.Div(ui.Class("group"), ui.Text("objectgroup")),
			ui.Div(ui.Class("object"),
				ui.Div(ui.Class("title"), ui.Text("object 1")),
				row("type", sel("", "body", "none", "shadow", "body"), ""),
				row("shape", sel("", "rectangle", "rectangle", "ellipse", "polygon"), ""),
				row("x · y", pair("0", "0"), ""),
				row("width · height", pair("16", "20"), ""),
			),
			ui.Div(ui.Class("row", "line"),
				ui.Span(ui.Class("name"), ui.Text("▸ object 2")),
				ui.Span(ui.Class("summary"), ui.Text("shadow · rectangle · 0,0 · 16×16")),
				ui.Span(),
			),
			ui.Hr(),
			ui.Div(ui.Class("actions"),
				sel("+ Add object", "", "rectangle", "ellipse", "polygon"),
				ui.Button(ui.Class("danger"), ui.Text("Delete object")),
			),
		),
	))
	return doc
}

func main() { pages.Run(page(), w, h) }
