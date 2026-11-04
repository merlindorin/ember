// Summaries' disclosure markers, closed and open at three sizes, and a
// display: block summary that has none, against Chrome's.
package main

import (
	"fmt"
	"testing/fstest"

	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; }
body { padding: 8px; display: grid; grid-template-columns: 1fr 1fr; gap: 6px 12px; }
summary { padding: 2px 8px; color: #2b2722; background-color: #f4f1ec; }
.light summary { color: #f4ede4; background-color: #2b2722; }
.block summary { display: block; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"markers.css": {Data: []byte(sheet)}}, "markers.css"))
	details := func(class []string, size int, open bool, title string) *ui.Node {
		args := []ui.Arg{ui.Class(class...), ui.Style(fmt.Sprintf("font-size: %dpx", size)),
			ui.Summary(ui.Text(title)), ui.Div(ui.Text("inside"))}
		if open {
			args = append(args, ui.Attr("open", ""))
		}
		return ui.Details(args...)
	}
	for _, size := range []int{13, 16, 24} {
		d.Body().Append(details(nil, size, false, "Shadows"), details([]string{"light"}, size, true, "Lights"))
	}
	d.Body().Append(details([]string{"block"}, 16, false, "Block summary"), details([]string{"block", "light"}, 16, true,
		"Block summary"))
	pages.Run(d, 400, 330)
}
