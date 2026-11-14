// 33 Hint: kit.Hint for each device, a row each set to its device by its
// own data-input-device; touch has no text of its own and reads the
// keyboard's.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
	"github.com/lantervale/ember/ui/kit"
)

const sheet = `
body { background-color: var(--window-color); padding: 16px; display: grid; grid-template-columns: max-content 1fr;
  gap: 12px 16px; align-items: baseline; }
.kit-row { display: contents; }
.kit-hint { color: var(--muted-color); }
`

func main() {
	d := ui.NewDocument(kit.Theme(), ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	for _, device := range []string{"keyboard", "mouse", "gamepad", "touch"} {
		d.Body().Append(ui.Div(ui.Class("kit-row"), ui.Attr("data-input-device", device),
			kit.Label(device, ui.Class("kit-form-label")),
			kit.Hint(
				kit.KeyboardHint("Arrows to look · Enter to choose"),
				kit.MouseHint("Click a card to choose"),
				kit.GamepadHint("Cross or stick to look · South to choose"),
			)))
	}
	pages.Run(d, 420, 180)
}
