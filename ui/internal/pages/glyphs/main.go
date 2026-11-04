// Text at 13, 16 and 24 px, dark on light and light on dark: the glyphs'
// weight, colour and placement against Chrome's.
package main

import (
	"fmt"
	"testing/fstest"

	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; }
body { padding: 8px; }
.case { padding: 6px 10px; width: 300px; height: 48px; }
.dark { background-color: #fff; color: #000; }
.light { background-color: #1e1e1e; color: #fff; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"glyphs.css": {Data: []byte(sheet)}}, "glyphs.css"))
	for _, tone := range []string{"dark", "light"} {
		for _, size := range []int{13, 16, 24} {
			d.Body().Append(ui.Div(ui.Class("case", tone), ui.Style(fmt.Sprintf("font-size: %dpx", size)),
				ui.Text("Handgloves")))
		}
	}
	pages.Run(d, 400, 320)
}
