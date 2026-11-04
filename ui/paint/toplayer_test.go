package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"testing"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/ui/control"
	"github.com/merlindorin/ember/ui/internal/dom"
	"github.com/merlindorin/ember/ui/internal/style"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModalDialogsStackAboveEveryZIndexInShowModalOrder(t *testing.T) {
	first, second := &control.Dialog{}, &control.Dialog{}
	a := ctl(dom.Dialog, 100, 60)
	a.Style.Background, a.Style.Position, a.Style.ZAuto, a.Style.Backdrop = red, style.FixedPosition, true, backdrop
	b := ctl(dom.Dialog, 100, 60)
	b.Style.Background, b.Style.Position, b.Style.ZAuto, b.Style.Backdrop = blue, style.FixedPosition, true, backdrop
	high := el(0, 0, 50, 50, func(s *style.Computed) {
		s.Background, s.Position, s.ZAuto, s.ZIndex = green, style.Relative, false, 10
	})
	inA := el(0, 0, 40, 40, func(s *style.Computed) {
		s.Background, s.Position, s.ZAuto, s.ZIndex = white, style.Relative, false, 5
	})
	a.Append(inA)
	second.ShowModal()
	first.ShowModal()
	find := Controls(func(n *dom.Node) any {
		switch n {
		case a.Node:
			return second
		case b.Node:
			return first
		}
		return nil
	})
	r := newRecorder()
	Paint(r, tree(el(0, 0, 400, 300, nil, b, a, high)), find)
	var order []color.Color
	for _, o := range r.of("rect") {
		order = append(order, o.Color)
	}
	require.Len(t, order, 6)
	assert.Equal(t, []color.Color{green, backdrop, red, white, backdrop, blue}, order,
		"the page with its z-index 10 first; then a, opened modal first, its backdrop and its own stacking; then b over it")
	assert.Equal(t, geom.Rect{W: 400, H: 300}, r.of("rect")[1].Rect)
}

func TestAModalShownAgainPaintsOnTop(t *testing.T) {
	first, second := &control.Dialog{}, &control.Dialog{}
	a := ctl(dom.Dialog, 100, 60)
	a.Style.Background, a.Style.Position, a.Style.ZAuto, a.Style.Backdrop = red, style.FixedPosition, true, backdrop
	b := ctl(dom.Dialog, 100, 60)
	b.Style.Background, b.Style.Position, b.Style.ZAuto, b.Style.Backdrop = blue, style.FixedPosition, true, backdrop
	first.ShowModal()
	second.ShowModal()
	first.Close("")
	first.ShowModal()
	find := Controls(func(n *dom.Node) any {
		switch n {
		case a.Node:
			return first
		case b.Node:
			return second
		}
		return nil
	})
	r := newRecorder()
	Paint(r, tree(el(0, 0, 400, 300, nil, a, b)), find)
	var order []color.Color
	for _, o := range r.of("rect") {
		order = append(order, o.Color)
	}
	assert.Equal(t, []color.Color{backdrop, blue, backdrop, red}, order, "closed and shown modal again, the highest")
}

func TestTheTopLayerTakesThePointerOverAnyZIndex(t *testing.T) {
	dlg := ctl(dom.Dialog, 100, 60)
	dlg.Style.Background, dlg.Style.Position = white, style.FixedPosition
	dlg.Style.ZAuto, dlg.Style.Backdrop = true, backdrop
	m := &control.Dialog{}
	m.ShowModal()
	high := el(0, 0, 50, 50, func(s *style.Computed) {
		s.Background, s.Position, s.ZAuto, s.ZIndex = green, style.Relative, false, 10
	})
	find := Controls(func(n *dom.Node) any {
		if n == dlg.Node {
			return m
		}
		return nil
	})
	root := el(0, 0, 400, 300, nil, dlg, high)
	assert.Same(t, dlg, HitTest(tree(root), geom.V2(20, 20), find), "the dialog, over the z-index 10 box")
	assert.Same(t, dlg, HitTest(tree(root), geom.V2(5, 5), find),
		"and its backdrop, over the corner of the box it leaves bare")
}
