package kit_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/kit"
)

func chip(w, h int) *ui.Node {
	return ui.Div(ui.Style("width: " + itoa(w) + "px; height: " + itoa(h) + "px"))
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	s := ""
	for ; v > 0; v /= 10 {
		s = string(rune('0'+v%10)) + s
	}
	return s
}

func laidOut(nodes ...*ui.Node) *ui.Document {
	d := ui.NewDocument(kit.Theme())
	d.Body().Append(nodes...)
	d.Layout(geom.V2(400, 300))
	return d
}

func TestARowLinesItsChildrenUpFourPointsApart(t *testing.T) {
	a, b := chip(20, 10), chip(30, 20)
	row := kit.Row(a, b)
	d := laidOut(row)
	assert.Equal(t, a.Box().X+20+4, b.Box().X)
	assert.Equal(t, a.Box().Y, b.Box().Y)
	assert.Equal(t, float32(20), row.Box().H)
	assert.Empty(t, d.Warnings())
}

func TestAColumnStacksItsChildren(t *testing.T) {
	a, b := chip(20, 10), chip(30, 20)
	col := kit.Column(a, b)
	laidOut(col)
	assert.Equal(t, a.Box().Y+10+4, b.Box().Y)
	assert.Equal(t, a.Box().X, b.Box().X)
}

func TestExpandTakesTheRoomLeft(t *testing.T) {
	a, grow := chip(20, 10), ui.Div(ui.Class("kit-expand"))
	row := kit.Row(ui.Style("width: 200px"), kit.Gap(0), a, grow)
	laidOut(row)
	assert.Equal(t, float32(180), grow.Box().W)
}

func TestJustifyAndAlignPlaceTheChildren(t *testing.T) {
	a, b := chip(20, 10), chip(20, 30)
	row := kit.Row(ui.Class("kit-justify-between", "kit-align-center"), ui.Style("width: 200px"), a, b)
	laidOut(row)
	assert.Equal(t, row.Box().X+180, b.Box().X)
	assert.Equal(t, row.Box().Y+10, a.Box().Y, "centred across the 30-point row")
}

func TestGapAndPaddingAddUp(t *testing.T) {
	a, b := chip(20, 10), chip(20, 10)
	row := kit.Row(kit.Gap(12), kit.Padding(6), a, b)
	laidOut(row)
	assert.Equal(t, row.Box().X+6, a.Box().X)
	assert.Equal(t, a.Box().X+20+12, b.Box().X)
	assert.Equal(t, float32(22), row.Box().H)
}

func TestToolbarBodyAndFooterPad(t *testing.T) {
	for class, pad := range map[string]float32{"kit-toolbar": 8, "kit-body": 16, "kit-footer": 12} {
		a := chip(20, 10)
		row := kit.Row(ui.Class(class), a)
		laidOut(row)
		assert.Equal(t, row.Box().X+pad, a.Box().X, class)
	}
}
