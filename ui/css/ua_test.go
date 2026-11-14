package css //nolint:testpackage // uses the package's unexported test helpers (el, page, sheet, styleOf)

import (
	"testing"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func TestInsetAndOutsetBordersAndADialogsBackdrop(t *testing.T) {
	in, out := el(dom.Div, "in"), el(dom.Div, "out")
	d := el(dom.Dialog, "d")
	root := page(in, out, d)
	Compute(root, []*Sheet{sheet(t, `#in { border: 1px inset gray } #out { border-style: outset } `+
		`#d::backdrop { background-color: rgba(0, 0, 0, 0.5) }`)})
	if got := styleOf(in).BorderStyle.Top; got != style.BorderInset {
		t.Errorf("inset: %v", got)
	}
	if got := styleOf(out).BorderStyle.Left; got != style.BorderOutset {
		t.Errorf("outset: %v", got)
	}
	if got := styleOf(d).Backdrop; got != color.RGBA(0, 0, 0, 0.5) {
		t.Errorf("backdrop: %v", got)
	}
	if got := styleOf(in).Backdrop; got != (color.Color{}) {
		t.Errorf("only a dialog has a backdrop: %v", got)
	}
}

func TestAClosedDetailsShowsOnlyItsSummaryWhateverTheirDisplay(t *testing.T) {
	sum, sum2 := el(dom.Summary, "s"), el(dom.Summary, "s2")
	body := el(dom.Div, "", "body")
	d := el(dom.Details, "d")
	d.Append(sum, body, sum2)
	root := page(d)
	sheets := []*Sheet{sheet(t, `.body { display: flex }`)}
	Compute(root, sheets)
	if styleOf(sum).Display == style.DisplayNone {
		t.Error("the first summary shows")
	}
	if styleOf(body).Display != style.DisplayNone || styleOf(sum2).Display != style.DisplayNone {
		t.Errorf("closed: body %v, second summary %v", styleOf(body).Display, styleOf(sum2).Display)
	}
	d.SetAttr("open", "")
	Compute(root, sheets)
	if styleOf(body).Display != style.FlexDisplay {
		t.Errorf("open, the body takes its display: %v", styleOf(body).Display)
	}
}
