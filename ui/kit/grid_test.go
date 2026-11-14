package kit_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/kit"
)

func TestAGridPlacesItsChildrenInRowsOfColumns(t *testing.T) {
	a, b, c := chip(20, 10), chip(40, 10), chip(30, 16)
	g := kit.Grid(2, a, b, c)
	d := laidOut(g)
	assert.Equal(t, a.Box().Y, b.Box().Y, "a and b share the first row")
	assert.Equal(t, a.Box().X, c.Box().X, "c starts the second row")
	assert.Equal(t, a.Box().X+30+4, b.Box().X, "the first column is as wide as its widest cell, then the gap")
	assert.Equal(t, a.Box().Y+10+4, c.Box().Y)
	assert.Empty(t, d.Warnings())
}

func TestColumnsSetTheTracks(t *testing.T) {
	name, value := kit.Label("name"), ui.Div()
	g := kit.Grid(2, ui.Style("width: 200px"), kit.Columns("max-content 1fr"), name, value)
	laidOut(g)
	assert.Equal(t, g.Box().X+200, value.Box().X+value.Box().W, "the value takes the room left")
	assert.Equal(t, name.Box().X+name.Box().W+4, value.Box().X)
}

func TestAGridTakesGapAndPadding(t *testing.T) {
	a, b := chip(20, 10), chip(20, 10)
	g := kit.Grid(2, kit.Gap(10), kit.Padding(5), a, b)
	laidOut(g)
	assert.Equal(t, g.Box().X+5, a.Box().X)
	assert.Equal(t, a.Box().X+20+10, b.Box().X)
}
