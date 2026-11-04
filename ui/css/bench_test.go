package css //nolint:testpackage // benchmarks with the package's unexported test helpers (el, page)

import (
	"fmt"
	"strings"
	"testing"

	"github.com/merlindorin/ember/ui/internal/dom"
)

// inspector is a tool panel's worth of elements: 50 rows of a label, an
// input and a button, under sections, with a sheet of 60 rules.
func inspector(b *testing.B) (*dom.Node, []*Sheet, *dom.Node) {
	var src strings.Builder
	src.WriteString(":root { --ink: #f7efdd; --pad: 6px; }\n" +
		".props { display: grid; grid-template-columns: max-content 1fr auto; gap: 6px 12px; }\n" +
		".props > * { min-width: 0; }\n")
	for i := range 57 {
		fmt.Fprintf(&src, ".row-%d > span:first-child { color: var(--ink); padding: var(--pad) calc(var(--pad) * 2); }\n", i)
	}
	src.WriteString("button:hover:not(:disabled) { background-color: #444; }\n")
	s, warns := Parse("bench.css", src.String())
	if len(warns) > 0 {
		b.Fatal(warnings(warns))
	}
	var hover *dom.Node
	props := el(dom.Div, "", "props")
	for i := range 50 {
		btn := dom.NewElement(dom.Button)
		btn.Append(dom.NewText("set"))
		row := with(el(dom.Div, "", fmt.Sprintf("row-%d", i)),
			with(dom.NewElement(dom.Span), dom.NewText("name")),
			dom.NewElement(dom.Input),
			btn,
		)
		props.Append(row)
		if i == 25 {
			hover = btn
		}
	}
	root := page(with(el(dom.Section, "", "inspector"), props))
	return root, []*Sheet{s}, hover
}

func BenchmarkComputeAll(b *testing.B) {
	root, sheets, _ := inspector(b)
	b.ReportAllocs()
	for b.Loop() {
		root.Walk(func(n *dom.Node) bool { dom.Data(n).Style = nil; return true })
		Compute(root, sheets)
	}
}

func BenchmarkRestyleHover(b *testing.B) {
	root, sheets, btn := inspector(b)
	tr := NewTransitions()
	tr.Compute(root, sheets)
	on := false
	b.ReportAllocs()
	for b.Loop() {
		on = !on
		btn.SetState(dom.Hover, on)
		tr.Compute(root, sheets)
	}
}
