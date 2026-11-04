package css //nolint:testpackage // uses the package's unexported test helpers (el, page, sheet, styleOf)

import (
	"testing"

	"github.com/merlindorin/ember/ui/internal/dom"
)

func TestPointerEventsNoneInheritsUntilAutoTakesItBack(t *testing.T) {
	inner, back := el(dom.Span, "inner"), el(dom.Span, "back")
	outer := el(dom.Div, "outer")
	outer.Append(inner, back)
	plain := el(dom.Div, "plain")
	root := page(outer, plain)
	Compute(root, []*Sheet{sheet(t, `#outer { pointer-events: none } #back { pointer-events: auto }`)})
	if !styleOf(outer).PointerNone || !styleOf(inner).PointerNone {
		t.Error("none, and its children with it")
	}
	if styleOf(back).PointerNone || styleOf(plain).PointerNone {
		t.Error("auto takes the pointer again, and is the initial value")
	}
}

func TestPointerEventsChangedOnAParentRestylesItsChildren(t *testing.T) {
	inner := el(dom.Span, "inner")
	outer := el(dom.Div, "outer")
	outer.Append(inner)
	root := page(outer)
	outer.SetStyle("pointer-events: none")
	sheets := []*Sheet{sheet(t, ``)}
	Compute(root, sheets)
	if !styleOf(inner).PointerNone {
		t.Fatal("none inherited")
	}
	outer.SetStyle("pointer-events: auto")
	Compute(root, sheets)
	if styleOf(inner).PointerNone {
		t.Error("auto on the parent reaches the child on the next compute")
	}
}
