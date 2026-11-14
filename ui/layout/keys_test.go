package layout //nolint:testpackage // tests the layout tree and its unexported helpers

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func allKeys(of map[*dom.Node]*box.Box) map[*dom.Node]bool {
	out := map[*dom.Node]bool{}
	for k := range of {
		out[k] = true
	}
	return out
}

// keysOf are a tree's Of keys, in no order, but for the text under a display: none
// element: a subtree kept from before keeps its text's boxes there, where a fresh layout
// writes none, as kept layouts always have.
func keysOf(of map[*dom.Node]*box.Box) map[*dom.Node]bool {
	out := map[*dom.Node]bool{}
	for k := range of {
		if !k.IsText() || !underHidden(k) {
			out[k] = true
		}
	}
	return out
}

func underHidden(n *dom.Node) bool {
	for p := n.Parent(); p != nil; p = p.Parent() {
		if styleOf(p).Display == style.DisplayNone {
			return true
		}
	}
	return false
}

// keysPage is a column holding rows, a paragraph of text and spans, an inline-block, an
// absolute box in the text, a span beside a block, and a display: contents wrapper of two
// rows.
func keysPage() (*dom.Node, map[string]*dom.Node) {
	parts := map[string]*dom.Node{}
	at := func(name string, n *dom.Node) *dom.Node { parts[name] = n; return n }
	contents := func(st *style.Computed) { st.Display = style.Contents }
	outOfFlow := func(st *style.Computed) { st.Position = style.Absolute }
	root := el(dom.Div, block,
		at("rows", el(dom.Div, block,
			at("row0", el(dom.Div, block, text("first"))),
			at("row1", el(dom.Div, block, at("nested", el(dom.Div, block, text("deep"))))),
			at("row2", el(dom.Div, block, text("third"))))),
		at("p", el(dom.P, block,
			at("words", text("some words ")),
			at("span", el(dom.Span, nil, at("spanText", text("in a span")))),
			at("chip", el(dom.Span, inlineBlock, text("chip"))),
			at("abs", el(dom.Span, outOfFlow, text("pinned"))))),
		at("mixed", el(dom.Div, block,
			at("loose", el(dom.Span, nil, text("loose text"), at("spanAbs", el(dom.Span, outOfFlow, text("in it"))))),
			at("looseAbs", el(dom.Span, outOfFlow, text("pinned too"))),
			el(dom.Div, block, text("a block beside it")))),
		at("wrap", el(dom.Div, contents,
			at("w0", el(dom.Div, block, text("wrapped"))),
			at("w1", el(dom.Div, block, text("too"))))),
	)
	return root, parts
}

func TestAKeptLayoutsOfHoldsTheKeysOfAFreshOneAndDropsTheRest(t *testing.T) {
	steps := []struct {
		name   string
		change func(p map[string]*dom.Node)
	}{
		{"nothing", func(map[string]*dom.Node) {}},
		{"a row removed", func(p map[string]*dom.Node) { p["rows"].Remove(p["row2"]) }},
		{"a row and its subtree removed", func(p map[string]*dom.Node) { p["rows"].Remove(p["row1"]) }},
		{"the removed row back", func(p map[string]*dom.Node) { p["rows"].Append(p["row1"]) }},
		{"a row moved", func(p map[string]*dom.Node) { p["rows"].Append(p["row0"]) }},
		{"a row emptied while detached", func(p map[string]*dom.Node) {
			p["rows"].Remove(p["row1"])
			p["row1"].Remove(p["nested"])
		}},
		{"a text to white space", func(p map[string]*dom.Node) { p["words"].SetText("   ") }},
		{"the text back", func(p map[string]*dom.Node) { p["words"].SetText("words again ") }},
		{"a span's text removed", func(p map[string]*dom.Node) { p["span"].Remove(p["spanText"]) }},
		{"an inline-block made a span", func(p map[string]*dom.Node) {
			restyle(p["chip"], func(st *style.Computed) { st.Display = style.Inline })
		}},
		{"a span made an inline-block", func(p map[string]*dom.Node) {
			restyle(p["span"], func(st *style.Computed) { st.Display = style.InlineBlock })
		}},
		{"the absolute box removed", func(p map[string]*dom.Node) { p["p"].Remove(p["abs"]) }},
		{"a span beside a block hidden", func(p map[string]*dom.Node) {
			restyle(p["loose"], func(st *style.Computed) { st.Display = style.DisplayNone })
		}},
		{"the span beside a block shown", func(p map[string]*dom.Node) {
			restyle(p["loose"], func(st *style.Computed) { st.Display = style.Inline })
		}},
		{"an absolute box out of a span beside a block", func(p map[string]*dom.Node) { p["loose"].Remove(p["spanAbs"]) }},
		{"an absolute box beside a block removed", func(p map[string]*dom.Node) { p["mixed"].Remove(p["looseAbs"]) }},
		{"the span beside a block emptied", func(p map[string]*dom.Node) { p["loose"].Children()[0].SetText(" ") }},
		{"a row hidden", func(p map[string]*dom.Node) {
			restyle(p["row0"], func(st *style.Computed) { st.Display = style.DisplayNone })
		}},
		{"the paragraph hidden", func(p map[string]*dom.Node) {
			restyle(p["p"], func(st *style.Computed) { st.Display = style.DisplayNone })
		}},
		{"the paragraph shown", func(p map[string]*dom.Node) {
			restyle(p["p"], func(st *style.Computed) { st.Display = style.Block })
		}},
		{"a row out of the contents wrapper", func(p map[string]*dom.Node) { p["wrap"].Remove(p["w1"]) }},
		{"the contents wrapper removed", func(p map[string]*dom.Node) { p["wrap"].Parent().Remove(p["wrap"]) }},
	}
	root, parts := keysPage()
	var r Retained
	before := allKeys(layKept(&r, root, 300).Of)
	for _, s := range steps {
		s.change(parts)
		tr := layKept(&r, root, 300)
		kept := keysOf(tr.Of)
		fresh := keysOf(lay(root, 300).Of)
		require.Equal(t, fresh, kept, "after %s, the kept Of holds a fresh one's keys", s.name)
		all := allKeys(tr.Of)
		var gone []*dom.Node
		for k := range before {
			if !all[k] {
				gone = append(gone, k)
			}
		}
		assert.ElementsMatch(t, gone, tr.Dropped, "after %s, Dropped is what left", s.name)
		before = all
	}
}

func TestAReattachedElementGetsAFreshEntry(t *testing.T) {
	root, parts := keysPage()
	var r Retained
	old := layKept(&r, root, 300).Of[parts["row1"]]
	parts["rows"].Remove(parts["row1"])
	tr := layKept(&r, root, 300)
	_, in := tr.Of[parts["row1"]]
	assert.False(t, in, "a removed row is gone from Of")
	assert.True(t, slices.Contains(tr.Dropped, parts["nested"]), "with its subtree")
	parts["rows"].Append(parts["row1"])
	tr = layKept(&r, root, 300)
	require.NotNil(t, tr.Of[parts["row1"]])
	assert.NotSame(t, old, tr.Of[parts["row1"]], "back, it has a box of this layout")
	assert.Equal(t, boxes(lay(root, 300)), boxes(tr))
}
