// Text shadows, sharp and blurred, light and dark, against Chrome's.
package main

import (
	"testing/fstest"

	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; }
body { padding: 8px; }
.case { padding: 6px 10px; width: 380px; height: 30px; font-size: 16px; }
.dark { background-color: #f4f1ec; color: #2b2722; }
.light { background-color: #2b2722; color: #f4ede4; }
`

// The page's two tones: shadows over a light text on dark, and over dark on light.
const (
	light = "light"
	dark  = "dark"
)

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"shadows.css": {Data: []byte(sheet)}}, "shadows.css"))
	for _, c := range []struct{ tone, shadow string }{
		{light, "1px 1.5px rgba(0,0,0,0.8)"},
		{light, "0 0 4px #e0a040"},
		{light, "2px 2px 6px rgba(0,0,0,0.9)"},
		{dark, "1px 1px #c8473d"},
		{dark, "0 2px 3px rgba(0,0,0,0.4)"},
		{dark, "-1px 0 #3a6fd8, 1px 0 #c8473d"},
	} {
		d.Body().Append(ui.Div(ui.Class("case", c.tone), ui.Style("text-shadow: "+c.shadow), ui.Text("Handgloves "+c.shadow)))
	}
	pages.Run(d, 420, 280)
}
