package layout //nolint:testpackage // tests the layout tree and its unexported helpers

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
)

func TestRefreshFollowsANewScrollOffset(t *testing.T) {
	header := blockOf(10, stickyTop(5))
	body := blockOf(500, nil)
	sc := scrollBox(0, blockOf(20, nil), header, body)
	root := el(dom.Div, block, sc)
	tr := lay(root, 400)
	assert.Equal(t, float32(20), tr.Of[header].Pos.Y)

	dom.Data(sc).Scroll = geom.V2(0, 50)
	Refresh(tr, root)
	assert.Equal(t, geom.V2(0, 50), tr.Of[sc].Scroll)
	assert.Equal(t, float32(55), tr.Of[header].Pos.Y, "stuck at the new offset")
	assert.Equal(t, float32(30), tr.Of[body].Pos.Y, "in-flow boxes keep their places")

	dom.Data(sc).Scroll = geom.V2(0, 9999)
	Refresh(tr, root)
	assert.Equal(t, geom.V2(0, 430), tr.Of[sc].Scroll, "clamped to the overflow")
	assert.Equal(t, geom.V2(0, 430), dom.Data(sc).Scroll)
	assert.Equal(t, float32(435), tr.Of[header].Pos.Y,
		"a scroll container's own sticky child stays stuck down its whole overflow")

	dom.Data(sc).Scroll = geom.V2(0, 0)
	Refresh(tr, root)
	assert.Equal(t, float32(20), tr.Of[header].Pos.Y, "back in the flow")
}

func TestRefreshMatchesALayout(t *testing.T) {
	build := func(scroll float32) (*dom.Node, *dom.Node, *dom.Node) {
		header := blockOf(10, stickyTop(0))
		sc := scrollBox(scroll, blockOf(30, nil), header, blockOf(300, nil))
		return el(dom.Div, block, sc), sc, header
	}
	root, sc, header := build(0)
	tr := lay(root, 400)
	dom.Data(sc).Scroll = geom.V2(0, 120)
	Refresh(tr, root)

	fresh, _, freshHeader := build(120)
	want := lay(fresh, 400)
	assert.Equal(t, want.Of[freshHeader].Rect(), tr.Of[header].Rect())
}

func TestRefreshReportsWhetherAnythingMoved(t *testing.T) {
	header := blockOf(10, stickyTop(5))
	sc := scrollBox(0, blockOf(20, nil), header, blockOf(500, nil))
	root := el(dom.Div, block, sc)
	tr := lay(root, 400)
	assert.False(t, Refresh(tr, root), "nothing scrolled")
	dom.Data(sc).Scroll = geom.V2(0, 50)
	assert.True(t, Refresh(tr, root))
	assert.False(t, Refresh(tr, root), "the same offset again")
}

func TestRefreshVisitsOnlyScrollersAndStickyBoxes(t *testing.T) {
	header := blockOf(10, stickyTop(5))
	sc := scrollBox(0, blockOf(20, nil), header, blockOf(500, nil))
	root := el(dom.Div, block, blockOf(30, nil), sc)
	tr := lay(root, 400)
	assert.True(t, slices.Equal([]*dom.Node{sc, header}, nodesOf(tr.Live)),
		"parents first, so a sticky box sees its scroller's new offset")
}

func nodesOf(bs []*box.Box) []*dom.Node {
	var out []*dom.Node
	for _, b := range bs {
		out = append(out, b.Node)
	}
	return out
}

func TestRefreshSeesAnOffsetWrittenStraightIntoTheBox(t *testing.T) {
	sc := scrollBox(0, blockOf(500, nil))
	root := el(dom.Div, block, sc)
	tr := lay(root, 400)
	dom.Data(sc).Scroll = geom.V2(0, 40)
	tr.Of[sc].Scroll = geom.V2(0, 40)
	assert.True(t, Refresh(tr, root), "moved since the last refresh, whoever wrote the box")
	assert.False(t, Refresh(tr, root))
}
