package ui

import (
	"testing"

	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/ui/paint"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/internal/textedit"
	"github.com/lantervale/ember/ui/control"
	"github.com/lantervale/ember/ui/internal/dom"
)

// ShownBackground is what paint draws as n's background now.
func ShownBackground(n *Node) color.Color { return dom.Data(n).Shown.Background }

// ShownAccent is n's accent-color now, what paint fills a range with.
func ShownAccent(n *Node) color.Color { return dom.Data(n).Shown.AccentColor }

// ScrollSize is the size of n's scrollable overflow from the last layout.
func ScrollSize(d *Document, n *Node) geom.Vec2 { return d.tree.Of[n].Content }

// SetPlatform makes every Document follow p's key conventions until t ends.
func SetPlatform(t testing.TB, p textedit.Platform) {
	was := currentPlatform
	currentPlatform = func() textedit.Platform { return p }
	t.Cleanup(func() { currentPlatform = was })
}

// Keys runs d's keyboard handling for this frame, as interact does, without a canvas.
func Keys(d *Document, in *input.State) { d.keys(nil, in) }

// OptionTexts is what a select's list shows of its options.
func OptionTexts(d *Document, n *Node) []string {
	s, ok := d.machine(n).(*control.Select)
	if !ok {
		return nil
	}
	var out []string
	for _, o := range s.Options {
		out = append(out, o.Text())
	}
	return out
}

// HitAt is the element a click at p, in the Document's space, lands on
// after its last layout.
func HitAt(d *Document, p geom.Vec2) *Node {
	h := paint.HitTest(d.tree, p, paint.Controls(d.machine), paint.Viewport(d.viewport), paint.Scale(d.fonts.Scale()))
	for b := h; b != nil; b = b.Parent {
		if b.Node != nil {
			return b.Node
		}
	}
	return nil
}
