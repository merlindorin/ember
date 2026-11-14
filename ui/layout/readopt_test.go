package layout //nolint:testpackage // tests the layout tree and its unexported helpers

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// bigTable is a table of n rows of a name and a size, in a block.
func bigTable(n int) (*dom.Node, *dom.Node, *dom.Node, []*dom.Node) {
	var tb, body *dom.Node
	var rows []*dom.Node
	body = el(dom.Tbody, nil)
	for i := range n {
		r := row(cell(fmt.Sprintf("tile_%04d", i)), cell(fmt.Sprint(i%97, " KB")))
		rows = append(rows, r)
		body.Append(r)
	}
	tb = table(el(dom.Thead, nil, row(cell("Name"), cell("Size"))), body)
	return el(dom.Div, block, tb), tb, body, rows
}

// keptNodes are the kept node of each element.
func keptNodes(r *Retained, els []*dom.Node) []*node {
	out := make([]*node, len(els))
	for i, e := range els {
		out[i] = r.nodes[e]
	}
	return out
}

func TestAKeptTableMatchesAFreshOneAndKeepsItsRowsAfterEachChange(t *testing.T) {
	changes := map[string]func(tb, body *dom.Node, rows []*dom.Node) (changed int){
		"a cell's text": func(_, _ *dom.Node, rows []*dom.Node) int {
			rows[4].Children()[1].SetText("1234 KB")
			return 4
		},
		"a cell's padding": func(_, _ *dom.Node, rows []*dom.Node) int {
			restyle(rows[2].Children()[0], func(st *style.Computed) { st.Padding = style.All(style.Px(6)) })
			return 2
		},
		"a row appended": func(_, body *dom.Node, _ []*dom.Node) int {
			body.Append(row(cell("tile_new"), cell("HHHHHHHHHH")))
			return -1
		},
		"the table restyled": func(tb, _ *dom.Node, _ []*dom.Node) int {
			restyle(tb, func(st *style.Computed) { st.BorderSpacing = [2]float32{5, 3} })
			return -1
		},
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			root, tb, body, rows := bigTable(12)
			var r Retained
			layKept(&r, root, 300)
			before := keptNodes(&r, rows)
			changed := change(tb, body, rows)
			kept := boxes(layKept(&r, root, 300))
			assert.Equal(t, boxes(lay(root, 300)), kept)
			for i, n := range keptNodes(&r, rows) {
				require.NotNil(t, n)
				assert.Same(t, before[i], n, "row %d keeps its node", i)
				if i != changed {
					assert.Same(t, before[i].box, r.of[rows[i]], "row %d keeps its box", i)
				}
			}
		})
	}
}

func TestTheAncestorsOfAChangeKeepTheirNodesAndBoxes(t *testing.T) {
	root, rows := page()
	var r Retained
	layKept(&r, root, 300)
	path := []*dom.Node{root, rows[7].Parent(), rows[7]}
	nodes := keptNodes(&r, path)
	var boxesBefore []any
	for _, n := range nodes {
		boxesBefore = append(boxesBefore, n.box)
	}
	rows[7].Children()[0].SetText("a much longer label")
	kept := boxes(layKept(&r, root, 300))
	assert.Equal(t, boxes(lay(root, 300)), kept)
	for i, n := range keptNodes(&r, path) {
		assert.Same(t, nodes[i], n, "ancestor %d keeps its node", i)
		assert.Same(t, boxesBefore[i], r.of[path[i]], "and its box")
		assert.Equal(t, r.gen, n.performed, "and is laid out again")
	}
}

func TestARestyledAncestorAdoptsItsUnchangedChildren(t *testing.T) {
	root, rows := page()
	var r Retained
	layKept(&r, root, 300)
	col := rows[0].Parent()
	before := keptNodes(&r, rows)
	old := r.nodes[col]
	restyle(col, func(st *style.Computed) { st.Gap.Height = style.Px(9) })
	kept := boxes(layKept(&r, root, 300))
	assert.Equal(t, boxes(lay(root, 300)), kept)
	assert.NotSame(t, old, r.nodes[col], "the restyled column is built anew")
	for i, n := range keptNodes(&r, rows) {
		assert.Same(t, before[i], n, "row %d is adopted under it", i)
		assert.Same(t, before[i].box, n.box)
	}
}

func TestACellChangeInABigTableAllocatesNothingPerRow(t *testing.T) {
	const n = 2000
	root, _, _, rows := bigTable(n)
	var r Retained
	layKept(&r, root, 300)
	c, i := rows[3].Children()[1], 0
	allocs := testing.AllocsPerRun(20, func() {
		i++
		c.SetText(fmt.Sprint(i%50, " KB"))
		layKept(&r, root, 300)
	})
	assert.Less(t, allocs, float64(n)/10, "a layout's allocations don't grow with the rows kept")
}

// BenchmarkACellChangeInABigTable lays a 5,000-row table out again, kept, after one cell's
// text changed.
func BenchmarkACellChangeInABigTable(b *testing.B) {
	root, _, _, rows := bigTable(5000)
	var r Retained
	layKept(&r, root, 300)
	c, i := rows[3].Children()[1], 0
	b.ReportAllocs()
	for b.Loop() {
		i++
		c.SetText(fmt.Sprint(i%50, " KB"))
		layKept(&r, root, 300)
	}
}

func countLookups(n *int) Option { return func(t *tree) { t.lookups = n } }

func TestACellChangeInABigTableLooksUpOnlyItsRow(t *testing.T) {
	const n = 2000
	root, _, _, rows := bigTable(n)
	var r Retained
	layKept(&r, root, 300)
	rows[3].Children()[1].SetText("7 KB")
	lookups := 0
	tr := Layout(root, geom.V2(300, 600), ahemFonts{}, Retain(&r), countLookups(&lookups))
	assert.Less(t, lookups, n/20, "a one-cell edit measures its row's path, not every row")
	assert.Equal(t, boxes(lay(root, 300)), boxes(tr))
}
