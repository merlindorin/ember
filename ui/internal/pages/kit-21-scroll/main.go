// 21 Scroll: kit.Scroll overflowing down, both ways, fitting, with its bars always and
// never, on ember's window colour.
package main

import (
	"fmt"
	"testing/fstest"

	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/internal/pages"
	"github.com/merlindorin/ember/ui/kit"
)

const sheet = `
body { background-color: var(--window-color); padding: 16px; display: grid; grid-template-columns: max-content 200px;
  gap: 14px 16px; align-items: baseline; }
.kit-scroll { height: 72px; border: 1px dashed #5a524a; }
.wide { white-space: nowrap; }
`

func lines(n int, text string, args ...ui.Arg) []ui.Arg {
	out := append([]ui.Arg{}, args...)
	for i := range n {
		out = append(out, kit.Label(fmt.Sprintf("%s %d", text, i+1), ui.Style("display: block")))
	}
	return out
}

func main() {
	d := ui.NewDocument(kit.Theme(), ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	row := func(name string, n *ui.Node) {
		d.Body().Append(kit.Label(name, ui.Class("kit-form-label")), n)
	}
	row("overflows down", kit.Scroll(lines(8, "Torch")...))
	row("both ways", kit.Scroll(lines(8, "A line much wider than the scroll box", ui.Class("wide"))...))
	row("fits", kit.Scroll(lines(2, "Torch")...))
	row("VerticalBar(BarAlways)", kit.Scroll(lines(2, "Torch", kit.VerticalBar(kit.BarAlways))...))
	row("VerticalBar(BarNever)", kit.Scroll(lines(8, "Torch", kit.VerticalBar(kit.BarNever))...))
	pages.Run(d, 420, 520)
}
