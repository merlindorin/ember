package kit

import (
	"math"
	"strconv"

	"github.com/merlindorin/ember/ui"
)

// Progress is a block of class "kit-progress" holding a
// progress element of class "kit-progress-bar" at value, between 0 and 100 unless Range bounds it,
// and the value as a percentage over it unless ShowValue(false). args
// go to the progress element.
func Progress(value float64, args ...ui.Arg) *ui.Node {
	bar := ui.Progress(append([]ui.Arg{ui.Class("kit-progress-bar")}, args...)...)
	lo, hi := attr(bar, "min", 0), attr(bar, "max", 100)
	bar.RemoveAttr("min")
	t := 0.0
	if hi > lo {
		t = min(max((value-lo)/(hi-lo), 0), 1)
	}
	bar.SetAttr("value", format(value-lo))
	bar.SetAttr("max", format(hi-lo))
	p := ui.Div(ui.Class("kit-progress"), ui.Attr("data-min", format(lo)), bar)
	if _, hidden := bar.Attr("data-hide-value"); hidden {
		bar.RemoveAttr("data-hide-value")
		return p
	}
	p.Append(ui.Span(ui.Class("kit-progress-value"), ui.Text(percent(t))))
	return p
}

// SetProgress moves the Progress p to value, in the units it was made
// with, its percentage with it.
func SetProgress(p *ui.Node, value float64) {
	bar := ProgressBar(p)
	if bar == nil {
		return
	}
	lo, span := attr(p, "data-min", 0), attr(bar, "max", 100)
	t := 0.0
	if span > 0 {
		t = min(max((value-lo)/span, 0), 1)
	}
	bar.SetAttr("value", format(value-lo))
	for _, c := range p.Children() {
		if c.HasClass("kit-progress-value") {
			c.Children()[0].SetText(percent(t))
		}
	}
}

func percent(t float64) string { return strconv.Itoa(int(math.Round(t*100))) + "%" }

// ShowValue(false) leaves a Progress's percentage off.
func ShowValue(on bool) ui.Arg {
	if on {
		return args{}
	}
	return ui.Attr("data-hide-value", "")
}

// ProgressBar is the progress element of a Progress.
func ProgressBar(p *ui.Node) *ui.Node {
	for _, c := range p.Children() {
		if c.HasClass("kit-progress-bar") {
			return c
		}
	}
	return nil
}

func attr(n *ui.Node, name string, def float64) float64 {
	if s, ok := n.Attr(name); ok {
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			return v
		}
	}
	return def
}
