package ui

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/dom"
)

func TestEachLayoutPointsEveryNodeAtItsBox(t *testing.T) {
	d, list, items, _ := scrollingList(20)
	d.Layout(geom.V2(300, 200))
	removed := items[5]
	for _, step := range []struct {
		name   string
		change func()
	}{
		{"a row moved", func() { list.Append(items[2]) }},
		{"a row's text", func() { items[7].SetText("seventh") }},
		{"a row removed", func() { list.Remove(removed) }},
		{"the row back", func() { list.Append(removed) }},
		{"a row hidden", func() { items[9].SetAttr("style", "display: none") }},
		{"a row removed for good", func() { list.Remove(items[11]) }},
	} {
		step.change()
		d.Layout(geom.V2(300, 200))
		for n, b := range d.tree.Of {
			require.Same(t, b, dom.Data(n).Placed, "after %s, %s points at its box", step.name, n.Tag())
		}
		if step.name == "a row removed" {
			assert.Nil(t, dom.Data(removed).Placed, "a removed row points at none")
		}
	}
	assert.Nil(t, dom.Data(items[11]).Placed, "a removed row points at none")
}

func TestACellChangeInABigTablePlacesItsPathAlone(t *testing.T) {
	body := Tbody()
	var rows []*Node
	for i := range 2000 {
		r := Tr(Td(Text("tile")), Td(Text(fmt.Sprint(i, " KB"))))
		rows = append(rows, r)
		body.Append(r)
	}
	d := NewDocument()
	d.Body().Append(Table(body))
	d.Layout(geom.V2(400, 300))
	rows[3].Children()[1].SetText("7 KB")
	d.Layout(geom.V2(400, 300))
	assert.Less(t, len(d.tree.Written), 20, "a one-cell edit points the nodes along its path alone")
	for n, b := range d.tree.Of {
		require.Same(t, b, dom.Data(n).Placed)
	}
}
