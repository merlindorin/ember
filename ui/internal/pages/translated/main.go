// A page shown through a Translator: a French paragraph that wraps, a
// button, a placeholder, and a line kept by translate="no".
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
	"github.com/lantervale/ember/ui/kit"
)

const sheet = `
body { background-color: var(--window-color); color: var(--light-text); padding: 16px; width: 260px; }
p { margin: 0 0 10px; }
button { padding: 6px 12px; border: 1px solid var(--border-color); border-radius: 4px;
  background-color: var(--button-color); color: var(--light-text); }
input { display: block; margin-top: 10px; width: 200px; }
`

type french map[string]string

func (f french) Tr(m string) string {
	if s, ok := f[m]; ok {
		return s
	}
	return m
}

func main() {
	d := ui.NewDocument(kit.Theme(), ui.Stylesheet(fstest.MapFS{"t.css": {Data: []byte(sheet)}}, "t.css"),
		ui.WithTranslator(french{
			"Your changes to the level are kept until you leave.": "Vos modifications du niveau sont gardées jusqu'à votre " +
				"départ.",
			"Save":      "Enregistrer",
			"Search":    "Rechercher une salle",
			"Crypt 2.0": "La crypte",
		}))
	d.Body().Append(
		ui.P(ui.Text("Your changes to the level are kept until you leave.")),
		ui.P(ui.Attr("translate", "no"), ui.Text("Crypt 2.0")),
		ui.Button(ui.Text("Save")),
		ui.Input(ui.Type("text"), ui.Attr("placeholder", "Search")),
	)
	pages.Run(d, 300, 200)
}
