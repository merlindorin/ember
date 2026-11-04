//go:build chrome

package layout_test

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"testing"
	"testing/fstest"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/internal/chrome"
	"github.com/merlindorin/ember/ui/kit"
)

const sentence = "The quick brown fox jumps over the lazy dog, then naps in the warm afternoon sun."

func page(nodes ...*ui.Node) *ui.Document {
	d := ui.NewDocument()
	d.Body().Append(nodes...)
	return d
}

// numberGrid is a kit.Grid of a header and four rows of labels and a
// NumberField, rows centred in one and on their baseline in the other.
func numberGrid() *ui.Document {
	sheet := ".x, .y { column-gap: 12px; row-gap: 4px; } .x { align-items: center; }"
	d := ui.NewDocument(kit.Theme(), ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	for _, class := range []string{"x", "y"} {
		g := kit.Grid(6, ui.Class(class), kit.Columns("max-content 1fr 64px repeat(3, max-content)"))
		for _, h := range []string{"Weapon", "Stat", "Value", "Min", "Max", "Step"} {
			g.Append(kit.Label(h))
		}
		for i := range 4 {
			g.Append(kit.Label(fmt.Sprint("Bow ", i)), kit.Label("damage"),
				kit.NumberField(float64(10+i)), kit.Label("1"), kit.Label("99"), kit.Label("1"))
		}
		d.Body().Append(ui.Div(ui.Style("width: 460px; margin-bottom: 12px"), g))
	}
	return d
}

// squeezedGrid is a kit.Grid of labels and NumberFields at the top of a
// scrolling flex column whose content overflows it, as a debug panel's
// body holds it.
func squeezedGrid() *ui.Document {
	sheet := `.body { display: flex; flex-direction: column; height: 150px; overflow-y: auto; gap: 4px; width: 300px; }
.slots { column-gap: 12px; row-gap: 4px; }
.fill { width: 100%; }`
	d := ui.NewDocument(kit.Theme(), ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	g := kit.Grid(3, kit.Columns("max-content 1fr 64px"), ui.Class("slots"))
	for i := range 4 {
		g.Append(kit.Label(fmt.Sprint(i)), kit.Label("Bow"), kit.NumberField(float64(i), ui.Class("fill")))
	}
	d.Body().Append(ui.Div(ui.Class("body"), g, ui.Div(ui.Style("height: 400px; flex: none"))))
	return d
}

func TestLayoutMatchesChrome(t *testing.T) {
	cases := map[string]*ui.Document{
		"a paragraph wraps": page(
			ui.Div(ui.Style("width: 120px; padding: 4px"), ui.P(ui.Text(sentence))),
		),
		"line heights": page(
			ui.P(ui.Style("width: 150px; line-height: 1.5"), ui.Text(sentence)),
			ui.P(ui.Style("width: 150px; line-height: 30px"), ui.Text(sentence)),
		),
		"spans in a line": page(
			ui.P(ui.Style("width: 160px"), ui.Text("Some "),
				ui.Span(ui.Style("font-size: 24px"), ui.Text("large")), ui.Text(" and small words that wrap.")),
		),
		"nowrap with an ellipsis": page(
			ui.Div(ui.Style("width: 100px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis"), ui.Text(sentence)),
		),
		"a flex toolbar": page(
			ui.Div(ui.Style("display: flex; gap: 8px; align-items: center; width: 300px; padding: 6px"),
				ui.Span(ui.Text("Name")),
				ui.Div(ui.Style("flex: 1; height: 20px")),
				ui.Button(ui.Style("padding: 4px 8px; border: 1px solid #888"), ui.Text("Save")),
			),
		),
		"flex items shrink and wrap": page(
			ui.Div(ui.Style("display: flex; flex-wrap: wrap; gap: 4px 10px; width: 200px"),
				ui.Span(ui.Text("alpha")), ui.Span(ui.Text("beta gamma")), ui.Span(ui.Text("delta")),
				ui.Span(ui.Text("epsilon zeta eta")), ui.Span(ui.Text("theta")),
			),
			ui.Div(ui.Style("display: flex; width: 150px"),
				ui.Div(ui.Style("flex: 1 1 100px; min-width: 0"), ui.Text(sentence)),
				ui.Div(ui.Style("flex: 0 0 60px"), ui.Text("fixed")),
			),
		),
		"a grid of names and values": page(
			ui.Div(ui.Style("display: grid; grid-template-columns: max-content 1fr; gap: 6px 12px; width: 260px; padding: 8px"),
				ui.Span(ui.Text("name")),
				ui.Div(ui.Style("min-width: 0; white-space: nowrap; overflow: hidden; text-overflow: ellipsis"),
					ui.Text("grass_corner_north_east_long")),
				ui.Span(ui.Text("solid")), ui.Div(ui.Text("true")),
				ui.Span(ui.Text("occluder group")), ui.Div(ui.Text(sentence)),
			),
		),
		"a grid with fixed, auto and fr tracks": page(
			ui.Div(ui.Style("display: grid; grid-template-columns: 80px auto 1fr 2fr; "+
				"grid-auto-rows: minmax(30px, auto); width: 400px"),
				ui.Div(ui.Text("one")), ui.Div(ui.Text("two words")), ui.Div(ui.Text("three")), ui.Div(ui.Text(sentence)),
				ui.Div(ui.Style("grid-column: span 2"), ui.Text("spanning two columns")), ui.Div(ui.Text("six")),
			),
		),
		"inline blocks wrap": page(
			ui.Div(ui.Style("width: 150px"),
				ui.Button(ui.Style("padding: 4px 8px"), ui.Text("One")), ui.Text(" "),
				ui.Button(ui.Style("padding: 4px 8px"), ui.Text("Two")), ui.Text(" "),
				ui.Button(ui.Style("padding: 4px 8px"), ui.Text("Three")), ui.Text(" "),
				ui.Button(ui.Style("padding: 4px 8px"), ui.Text("Four")),
			),
		),
		"out-of-flow boxes inside spans": page(
			ui.Div(ui.Style("position: relative; width: 220px; padding: 6px"),
				ui.P(ui.Text("Lead words "),
					ui.Span(ui.Text("then a plain span"),
						ui.Span(ui.Style("position: absolute; width: 8px; height: 8px")), ui.Text(" goes on")),
					ui.Text(" and "),
					ui.Span(ui.Style("position: relative; padding: 0 4px"), ui.Text("a positioned span that wraps"),
						ui.Span(ui.Style("position: absolute; right: 0; bottom: 0; width: 6px; height: 6px")),
						ui.Span(ui.Style("position: absolute; top: 0; left: 0; width: 6px; height: 6px")),
						ui.Text(" across lines")),
				),
			),
		),
		"auto scrollers make room for their bars": page(
			ui.Div(ui.Style("overflow: auto; width: 160px; height: 60px"), ui.P(ui.Text(sentence))),
			ui.Div(ui.Style("overflow: auto; width: 160px; height: 60px"), ui.P(ui.Text("Fits."))),
			ui.Div(ui.Style("overflow: auto; width: 160px; height: 60px"), ui.Div(ui.Style("width: 150px; height: 100px"))),
			ui.Div(ui.Style("overflow-y: auto; overflow-x: hidden; width: 160px; height: 60px; display: flex; "+
				"flex-direction: column"),
				ui.Div(ui.Style("flex: none; height: 30px")), ui.Div(ui.Style("flex: none; height: 40px"))),
			ui.Div(ui.Style("overflow: auto; width: 160px; height: 60px; display: grid; grid-template-columns: 1fr 1fr"),
				ui.Div(ui.Style("height: 50px")), ui.Div(ui.Style("height: 50px")), ui.Div(ui.Style("height: 50px"))),
		),
		"fixed boxes as tall as their content at their clamped width": func() *ui.Document {
			box := func(style string) *ui.Node {
				return ui.Div(ui.Style("position: fixed; inset: 0; margin: auto; height: fit-content; "+
					"padding: 18px 20px 16px; display: flex; flex-direction: column; gap: 8px; border: 1px solid #000; "+style),
					ui.Div(ui.Style("height: 40px")), ui.P(ui.Style("margin: 0"), ui.Text(sentence)))
			}
			return page(box("width: fit-content; min-width: 300px; max-width: 360px"),
				box("min-width: 200px; max-width: 260px; top: 20px; bottom: auto"))
		}(),
		"tall controls align by their centred text's baseline": page(
			ui.Div(ui.Style("display: flex; align-items: baseline; gap: 8px"),
				ui.Span(ui.Style("font-size: 12px"), ui.Text("Show")),
				ui.Select(ui.Style("height: 30px; box-sizing: border-box"), ui.Option(ui.Text("Tilesets"))),
				ui.Input(ui.Style("height: 30px; box-sizing: border-box")),
				ui.Button(ui.Style("height: 30px; box-sizing: border-box"), ui.Text("Open")),
			),
		),
		"absolutely positioned boxes": page(
			ui.Div(ui.Style("position: relative; width: 200px; height: 100px; border: 2px solid #000"),
				ui.Div(ui.Style("position: absolute; right: 10px; bottom: 10px; width: 50px; height: 20px")),
				ui.Div(ui.Style("position: absolute; top: 10px; left: 10px"), ui.Text("label")),
				ui.Div(ui.Style("position: absolute; top: 40px; left: 20px; right: 20px"), ui.Text(sentence)),
			),
		),
		"block margins collapse": page(
			ui.Div(
				ui.Div(ui.Style("margin: 10px 0"), ui.Div(ui.Style("margin-top: 20px; height: 10px"))),
				ui.Div(ui.Style("margin-top: 5px; height: 10px")),
			),
		),
		"order in flex and grid": page(
			ui.Div(ui.Style("display: flex; gap: 6px; width: 300px"),
				ui.Span(ui.Style("order: 2"), ui.Text("last")),
				ui.Span(ui.Text("first")),
				ui.Span(ui.Style("order: 1"), ui.Text("middle")),
			),
			ui.Div(ui.Style("display: grid; grid-template-columns: 1fr 1fr; width: 200px"),
				ui.Div(ui.Style("order: 1"), ui.Text("b")),
				ui.Div(ui.Text("a")),
				ui.Div(ui.Style("order: -1; grid-column: 2"), ui.Text("c")),
			),
		),
		"sticky boxes": page(
			ui.Div(ui.Style("width: 200px; height: 100px; overflow: hidden"),
				ui.Div(ui.Style("height: 30px")),
				ui.Div(ui.Style("position: sticky; top: 0; height: 10px")),
				ui.Div(ui.Style("height: 150px")),
				ui.Div(ui.Style("position: sticky; bottom: 4px; height: 20px")),
			),
		),
		"an input with a value": page(
			ui.Div(ui.Text("Name "), ui.Input(ui.Value("Heron of Alexandria and a value longer than the box"))),
		),
		"an empty input with a placeholder": page(
			ui.Div(ui.Input(ui.Placeholder("Search tiles"))),
			ui.Div(ui.Input(ui.Style("font-size: 24px; width: 120px"), ui.Placeholder("Big"))),
		),
		"a select with three options": page(
			ui.Div(ui.Select(
				ui.Option(ui.Text("grass")),
				ui.Option(ui.Selected(), ui.Text("deep water")),
				ui.Option(ui.Text("sand")),
			)),
		),
		"a select's placeholder counts in its width": page(
			ui.Div(ui.Select(ui.Placeholder("+ Add an object to the tile"),
				ui.Option(ui.Text("body")), ui.Option(ui.Text("shadow")))),
			ui.Div(ui.Select(ui.Placeholder("+ Add an object to the tile"),
				ui.Option(ui.Selected(), ui.Text("body")), ui.Option(ui.Text("shadow")))),
		),
		"a textarea of three lines": page(
			ui.Div(ui.Textarea(ui.Text("First line\nSecond line\nThird line"))),
			ui.Div(ui.Textarea(ui.Attr("rows", "3"), ui.Style("width: 200px"), ui.Text(sentence))),
		),
		"controls centre their text": page(
			ui.Div(
				ui.Button(ui.Style("height: 32px; padding: 3px"), ui.Text("Delete")),
				ui.Input(ui.Style("height: 32px; padding: 3px"), ui.Value("value")),
				ui.Select(ui.Style("height: 32px; padding: 3px"), ui.Option(ui.Text("choice"))),
			),
			ui.Div(
				ui.Button(ui.Style("padding: 3px"), ui.Text("Delete")),
				ui.Input(ui.Style("padding: 3px"), ui.Value("value")),
				ui.Select(ui.Style("padding: 3px"), ui.Option(ui.Text("choice"))),
			),
		),
		"a scroll container's gutter": page(
			ui.Div(ui.Style("width: 100px; height: 50px; overflow-y: scroll"), ui.Div(ui.Style("height: 10px"))),
			ui.Div(ui.Style("width: 100px; height: 50px; overflow: scroll"), ui.Div(ui.Style("height: 10px"))),
		),
		"border-spacing": page(
			ui.Table(ui.Style("border-spacing: 4px 0"), ui.Tbody(
				ui.Tr(ui.Td(ui.Style("padding: 1px"), ui.Text("Name")), ui.Td(ui.Style("padding: 1px"), ui.Text("walls.tsj"))),
				ui.Tr(ui.Td(ui.Style("padding: 1px"), ui.Text("Size")), ui.Td(ui.Style("padding: 1px"), ui.Text("24 KB"))),
			)),
			ui.Table(ui.Style("border-spacing: 0; width: 240px"), ui.Tbody(
				ui.Tr(ui.Td(ui.Style("padding: 4px 8px"), ui.Text("crypt")),
					ui.Td(ui.Style("padding: 4px 8px"), ui.Text("folder"))),
				ui.Tr(ui.Td(ui.Style("padding: 4px 8px"), ui.Text("props.tsj")),
					ui.Td(ui.Style("padding: 4px 8px"), ui.Text("9 KB"))),
			)),
		),
		"a sized column takes no extra": page(
			ui.Table(ui.Style("width: 360px"), ui.Tbody(
				ui.Tr(ui.Td(ui.Style("padding: 4px 8px"), ui.Text("walls.tsj")),
					ui.Td(ui.Style("padding: 4px 8px; width: 70px"), ui.Text("24 KB")),
					ui.Td(ui.Style("padding: 4px 8px; width: 90px"), ui.Text("Tileset"))),
			)),
		),
		"tables": page(
			ui.Table(
				ui.Thead(ui.Tr(
					ui.Td(ui.Style("padding: 1px"), ui.Text("Name")),
					ui.Td(ui.Style("padding: 1px"), ui.Text("Value")),
				)),
				ui.Tbody(
					ui.Tr(
						ui.Td(ui.Style("padding: 1px"), ui.Text("solid")),
						ui.Td(ui.Style("padding: 1px"), ui.Text("true")),
					),
					ui.Tr(
						ui.Td(ui.Style("padding: 1px"), ui.Text("occluder group")),
						ui.Td(ui.Style("padding: 1px; height: 40px"), ui.Text("walls")),
					),
				),
			),
			ui.Div(ui.Style("width: 160px"),
				ui.Table(ui.Tbody(ui.Tr(
					ui.Td(ui.Style("padding: 1px"), ui.Text(sentence)),
					ui.Td(ui.Style("padding: 1px"), ui.Text("short")),
				))),
			),
			ui.Table(ui.Style("width: 400px; padding: 4px; border: 1px solid #000"), ui.Tbody(ui.Tr(
				ui.Td(ui.Style("padding: 1px"), ui.Text("narrow")),
				ui.Td(ui.Style("padding: 1px"), ui.Text("a wider cell")),
			))),
		),
		"cells spanning columns and rows": page(
			ui.Table(ui.Tbody(
				ui.Tr(ui.Td(ui.Attr("colspan", "2"), ui.Style("padding: 1px"), ui.Text("a heading across two columns")),
					ui.Td(ui.Style("padding: 1px"), ui.Text("x"))),
				ui.Tr(ui.Td(ui.Attr("rowspan", "2"), ui.Style("padding: 1px; height: 60px"), ui.Text("tall")),
					ui.Td(ui.Style("padding: 1px"), ui.Text("one")), ui.Td(ui.Style("padding: 1px"), ui.Text("two"))),
				ui.Tr(ui.Td(ui.Style("padding: 1px"), ui.Text("three")), ui.Td(ui.Style("padding: 1px"), ui.Text("four"))),
			)),
			ui.Div(ui.Style("width: 200px"),
				ui.Table(ui.Tbody(
					ui.Tr(ui.Td(ui.Style("padding: 1px"), ui.Text("a")),
						ui.Td(ui.Style("padding: 1px"), ui.Text("b")), ui.Td(ui.Style("padding: 1px"), ui.Text("c"))),
					ui.Tr(ui.Td(ui.Attr("colspan", "3"), ui.Style("padding: 1px"), ui.Text(sentence))),
				)),
			),
		),
		"a fixed table": page(
			ui.Table(ui.Style("table-layout: fixed; width: 300px"), ui.Tbody(
				ui.Tr(ui.Td(ui.Style("width: 80px; padding: 1px"), ui.Text("name")),
					ui.Td(ui.Style("padding: 1px"), ui.Text("value")), ui.Td(ui.Style("padding: 1px"), ui.Text("kind"))),
				ui.Tr(ui.Td(ui.Style("padding: 1px"), ui.Text("a much longer name than the column")),
					ui.Td(ui.Style("padding: 1px"), ui.Text("v")), ui.Td(ui.Style("padding: 1px"), ui.Text("k"))),
			)),
			ui.Table(ui.Style("table-layout: fixed; width: 260px; border-spacing: 4px"), ui.Tbody(
				ui.Tr(ui.Td(ui.Attr("colspan", "2"), ui.Style("width: 120px; padding: 1px"), ui.Text("wide")),
					ui.Td(ui.Style("padding: 1px"), ui.Text("rest"))),
				ui.Tr(ui.Td(ui.Style("padding: 1px"), ui.Text("a")),
					ui.Td(ui.Style("padding: 1px"), ui.Text("b")), ui.Td(ui.Style("padding: 1px"), ui.Text("c"))),
			)),
		),
		"captions above and below": page(
			ui.Table(ui.Style("border-spacing: 4px"), ui.Caption(ui.Text("Scores this week")),
				ui.Tbody(ui.Tr(ui.Td(ui.Style("padding: 1px"), ui.Text("Ada")), ui.Td(ui.Style("padding: 1px"), ui.Text("12")))),
			),
			ui.Table(ui.Style("border-spacing: 4px; caption-side: bottom"),
				ui.Caption(ui.Style("padding: 2px"), ui.Text("a caption longer than the table it sits under")),
				ui.Tbody(ui.Tr(ui.Td(ui.Style("padding: 1px"), ui.Text("Bo")), ui.Td(ui.Style("padding: 1px"), ui.Text("7")))),
			),
		),
		"a footer group written first": page(
			ui.Table(
				ui.Tfoot(ui.Tr(ui.Td(ui.Style("padding: 1px"), ui.Text("total")), ui.Td(ui.Style("padding: 1px"), ui.Text("19")))),
				ui.Thead(ui.Tr(ui.Td(ui.Style("padding: 1px"), ui.Text("name")),
					ui.Td(ui.Style("padding: 1px"), ui.Text("score")))),
				ui.Tbody(ui.Tr(ui.Td(ui.Style("padding: 1px"), ui.Text("Ada")), ui.Td(ui.Style("padding: 1px"), ui.Text("12")))),
			),
		),
		"a shrink-to-fit box whose body scrolls": page(
			ui.Div(ui.Style("position: absolute; left: 20px; top: 40px; display: flex; flex-direction: column; "+
				"min-width: 120px; max-height: 120px; border: 1px solid #000"),
				ui.Div(ui.Style("overflow-y: auto; flex: 1; min-height: 0"),
					ui.P(ui.Style("margin: 0; white-space: nowrap"), ui.Text(sentence)),
					ui.P(ui.Style("margin: 0; white-space: nowrap"), ui.Text("two")), ui.P(ui.Style("margin: 0"), ui.Text("three")),
					ui.P(ui.Style("margin: 0"), ui.Text("four")), ui.P(ui.Style("margin: 0"), ui.Text("five")),
					ui.P(ui.Style("margin: 0"), ui.Text("six")), ui.P(ui.Style("margin: 0"), ui.Text("seven")),
					ui.P(ui.Style("margin: 0"), ui.Text("eight")),
				),
			),
		),
		"a grid of labels and number fields":            numberGrid(),
		"a grid of number fields in a scrolling column": squeezedGrid(),
		"aspect-ratio items flexed": page(
			ui.Div(ui.Style("display: flex; align-items: flex-start; width: 32px; margin-bottom: 8px"),
				ui.Div(ui.Style("width: 40px; min-width: 32px; flex-shrink: 1; aspect-ratio: 1; background-color: #888"))),
			ui.Div(ui.Style("display: flex; width: 32px; height: 60px; margin-bottom: 8px"),
				ui.Div(ui.Style("width: 40px; min-width: 32px; flex-shrink: 1; aspect-ratio: 1; background-color: #888"))),
			ui.Div(ui.Style("display: flex; align-items: flex-start; width: 120px; margin-bottom: 8px"),
				ui.Div(ui.Style("width: 20px; flex-grow: 1; aspect-ratio: 2; background-color: #888"))),
			ui.Div(ui.Style("display: flex; flex-direction: column; align-items: flex-start; height: 32px; margin-bottom: 8px"),
				ui.Div(ui.Style("height: 40px; min-height: 32px; flex-shrink: 1; aspect-ratio: 1; background-color: #888"))),
			ui.Div(ui.Style("display: flex; flex-direction: column; height: 32px; width: 80px"),
				ui.Div(ui.Style("height: 40px; min-height: 32px; flex-shrink: 1; aspect-ratio: 1; background-color: #888"))),
		),
		"a column with gaps and padding": page(
			ui.Div(ui.Style("display: flex; flex-direction: column; gap: 4px; padding: 8px; width: 200px; "+
				"border: 1px solid #000"),
				ui.P(ui.Text("First paragraph.")),
				ui.P(ui.Text(sentence)),
				ui.Button(ui.Style("align-self: flex-end; padding: 2px 6px"), ui.Text("OK")),
			),
		),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			r := chrome.Match(t, d, geom.V2(800, 600))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
		})
	}
}

// tilePNG is a 40×20 picture, for the replaced-element cases.
func tilePNG() []byte {
	m := image.NewNRGBA(image.Rect(0, 0, 40, 20))
	for i := range m.Pix {
		m.Pix[i] = 0x80
	}
	var b bytes.Buffer
	if err := png.Encode(&b, m); err != nil {
		panic(err)
	}
	return b.Bytes()
}

func TestImagesMatchChrome(t *testing.T) {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"s.css": {Data: []byte("")}, "tile.png": {Data: tilePNG()}}, "s.css"))
	src := ui.Attr("src", "tile.png")
	d.Body().Append(
		ui.P(ui.Text("natural "), ui.Img(src), ui.Text(" width "), ui.Img(src, ui.Style("width: 80px")),
			ui.Text(" height "), ui.Img(src, ui.Style("height: 10px")),
			ui.Text(" both "), ui.Img(src, ui.Style("width: 30px; height: 30px")),
			ui.Text(" max "), ui.Img(src, ui.Style("max-width: 20px")),
			ui.Text(" flat "), ui.Img(src, ui.Style("width: 60px; height: 10px"))),
		ui.Div(ui.Style("display: flex; gap: 4px"), ui.Img(src),
			ui.Img(src, ui.Style("height: 30px")), ui.Img(src, ui.Style("width: 50px"))),
	)
	r := chrome.Match(t, d, geom.V2(800, 600))
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}
