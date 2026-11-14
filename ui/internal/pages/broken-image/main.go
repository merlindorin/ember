// Images that don't load, as Chrome draws them: the icon and alt text,
// cut to a sized box; a silver frame round the icon without alt text;
// nothing for an empty alt; unsized ones as wide as their icon and text.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; }
body { padding: 12px; }
.row { display: flex; gap: 16px; margin-bottom: 12px; align-items: flex-start; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"broken-image.css": {Data: []byte(sheet)}}, "broken-image.css"))
	img := func(style string, attrs ...string) *ui.Node {
		args := []ui.Arg{ui.Attr("src", "missing.png"), ui.Style(style)}
		for i := 0; i+1 < len(attrs); i += 2 {
			args = append(args, ui.Attr(attrs[i], attrs[i+1]))
		}
		return ui.Img(args...)
	}
	d.Body().Append(
		ui.Div(ui.Class("row"),
			img("width: 120px; height: 60px", "alt", "A crypt door"),
			img("width: 40px; height: 20px", "alt", "Short"),
			img("width: 60px; height: 40px"),
			img("", "alt", ""),
		),
		ui.Div(ui.Class("row"),
			img("", "alt", "no size given"),
			img(""),
		),
	)
	pages.Run(d, 420, 160)
}
