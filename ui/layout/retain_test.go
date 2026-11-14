package layout //nolint:testpackage // tests the layout tree and its unexported helpers

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// page is a column of rows, each a label and a padded button, as a list in a game menu.
func page() (*dom.Node, []*dom.Node) {
	col := el(dom.Div, func(st *style.Computed) {
		st.Display = style.FlexDisplay
		st.Direction = style.Column
		st.Gap.Height = style.Px(4)
	})
	var rows []*dom.Node
	for i := range 20 {
		r := el(dom.Div, func(st *style.Computed) { st.Display = style.FlexDisplay; st.Gap.Width = style.Px(8) },
			el(dom.Span, nil, text(fmt.Sprint("tile ", i))),
			el(dom.Button, func(st *style.Computed) { st.Display = style.InlineBlock; st.Padding = style.All(style.Px(4)) },
				text("Edit")))
		rows = append(rows, r)
		col.Append(r)
	}
	return el(dom.Div, block, el(dom.P, block, text("HH HH HHH")), col), rows
}

// restyle changes e's style as css does: a new value, e flagged DirtyLayout and its
// ancestors DirtyDescendant.
func restyle(e *dom.Node, edit func(*style.Computed)) {
	st := *dom.Data(e).Style
	edit(&st)
	dom.Data(e).Style = &st
	dom.Data(e).Dirty |= dom.DirtyLayout
	for p := e.Parent(); p != nil; p = p.Parent() {
		dom.Data(p).Dirty |= dom.DirtyDescendant
	}
}

// boxes lists every box of tr with its node, in order, for comparing two layouts.
func boxes(tr *box.Tree) []string {
	var out []string
	var walk func(b *box.Box, depth int)
	walk = func(b *box.Box, depth int) {
		var lines []box.Fragment
		lines = append(lines, b.Lines...)
		out = append(out,
			fmt.Sprintf("%d %v %v %v %v %v %v", depth, b.Kind, b.Rect(), b.Content, b.Scrollbar, lines, b.Fragments))
		for _, c := range b.Children {
			walk(c, depth+1)
		}
	}
	walk(tr.Root, 0)
	return out
}

func layKept(r *Retained, root *dom.Node, w float32) *box.Tree {
	return Layout(root, geom.V2(w, 600), ahemFonts{}, Retain(r))
}

func TestAKeptLayoutMatchesAFreshOneAfterEachChange(t *testing.T) {
	changes := map[string]func(root *dom.Node, rows []*dom.Node){
		"nothing":        func(*dom.Node, []*dom.Node) {},
		"a label's text": func(_ *dom.Node, rows []*dom.Node) { rows[7].Children()[0].SetText("a much longer label") },
		"a button's padding": func(_ *dom.Node, rows []*dom.Node) {
			restyle(rows[3].Children()[1], func(st *style.Computed) { st.Padding = style.All(style.Px(9)) })
		},
		"a row added":   func(_ *dom.Node, rows []*dom.Node) { rows[0].Parent().Append(el(dom.Div, block, text("new row"))) },
		"a row removed": func(_ *dom.Node, rows []*dom.Node) { rows[5].Parent().Remove(rows[5]) },
		"a row moved":   func(_ *dom.Node, rows []*dom.Node) { rows[0].Parent().Append(rows[2]) },
		"the paragraph": func(root *dom.Node, _ []*dom.Node) { root.Children()[0].Children()[0].SetText("HHHHHHHH HH") },
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			root, rows := page()
			var r Retained
			layKept(&r, root, 300)
			change(root, rows)
			kept := boxes(layKept(&r, root, 300))
			assert.Equal(t, boxes(lay(root, 300)), kept)
		})
	}
}

func TestAKeptLayoutFollowsTheViewport(t *testing.T) {
	root, _ := page()
	var r Retained
	layKept(&r, root, 300)
	kept := boxes(layKept(&r, root, 120))
	assert.Equal(t, boxes(lay(root, 120)), kept)
	kept = boxes(layKept(&r, root, 300))
	assert.Equal(t, boxes(lay(root, 300)), kept)
}

func TestAChangeInOneRowLaysOutOnlyItsPath(t *testing.T) {
	root, rows := page()
	var r Retained
	layKept(&r, root, 300)
	before := map[*dom.Node]*node{}
	for _, row := range rows {
		before[row] = r.nodes[row]
	}
	restyle(rows[3].Children()[1], func(st *style.Computed) { st.Padding = style.All(style.Px(5)) })
	layKept(&r, root, 300)
	for i, row := range rows {
		n := r.nodes[row]
		require.NotNil(t, n)
		if i == 3 {
			assert.Same(t, before[row], n, "the row of the change is built again in its own node")
			assert.Equal(t, r.gen, n.performed, "and laid out again")
			continue
		}
		assert.Same(t, before[row], n, "row %d keeps its node", i)
		assert.NotEqual(t, r.gen, n.performed, "row %d is not laid out again", i)
	}
}

func TestAKeptTableMatchesAFreshOneAfterItsRowsMove(t *testing.T) {
	var rows []*dom.Node
	body := el(dom.Tbody, nil)
	for _, name := range []string{"walls", "floor", "props"} {
		r := row(cell(name), cell("HH"))
		rows = append(rows, r)
		body.Append(r)
	}
	root := el(dom.Div, block, table(body))
	var r Retained
	layKept(&r, root, 300)
	body.Append(rows[0])
	kept := boxes(layKept(&r, root, 300))
	assert.Equal(t, boxes(lay(root, 300)), kept)
}

func scroller20() (*dom.Node, *dom.Node, []*dom.Node) {
	_, rows := page()
	col := rows[0].Parent()
	sc := el(dom.Div, autoBox(200, 100), col)
	return el(dom.Div, block, sc), sc, rows
}

func TestAKeptScrollerKeepsItsBarWhileItOverflows(t *testing.T) {
	root, sc, rows := scroller20()
	var r Retained
	layKept(&r, root, 300)
	require.Equal(t, geom.Vec2{X: 15}, r.of[sc].Scrollbar)
	restyle(rows[4].Children()[1], func(st *style.Computed) { st.Padding = style.All(style.Px(6)) })
	kept := boxes(layKept(&r, root, 300))
	assert.Equal(t, boxes(lay(root, 300)), kept)
	assert.NotEqual(t, r.gen, r.nodes[rows[9]].performed, "the other rows keep their layout under the same bar")
}

func TestAKeptScrollerDropsItsBarOnceItFits(t *testing.T) {
	root, sc, rows := scroller20()
	var r Retained
	layKept(&r, root, 300)
	for _, row := range rows[2:] {
		row.Parent().Remove(row)
	}
	kept := layKept(&r, root, 300)
	assert.Zero(t, kept.Of[sc].Scrollbar)
	assert.Equal(t, boxes(lay(root, 300)), boxes(kept))
}

func TestAChangedCellOfABaselineRowAloneIsLaidOutAgain(t *testing.T) {
	var cells []*dom.Node
	grid := el(dom.Div, func(st *style.Computed) {
		st.Display = style.GridDisplay
		st.AlignItems = style.AlignBaseline
		st.Columns = []style.TrackRepeat{{Count: 3, Tracks: []style.Track{{Min: style.Px(80), Max: style.Px(80)}}}}
	})
	for i := range 9 {
		c := el(dom.Div, func(st *style.Computed) { st.Padding.Top = style.Px(float32(i % 3 * 3)) },
			text(fmt.Sprint("cell ", i)))
		cells = append(cells, c)
		grid.Append(c)
	}
	root := el(dom.Div, block, grid)
	var r Retained
	layKept(&r, root, 300)
	layKept(&r, root, 300)
	restyle(cells[4], func(st *style.Computed) { st.Padding.Top = style.Px(9) })
	kept := boxes(layKept(&r, root, 300))
	for i, c := range cells {
		if i == 4 {
			continue
		}
		assert.NotEqual(t, r.gen, r.nodes[c].performed, "cell %d keeps its probe and its layout", i)
	}
	assert.Equal(t, boxes(lay(root, 300)), kept, "and the grid lays out as a fresh one, the row on its new baseline")
}
