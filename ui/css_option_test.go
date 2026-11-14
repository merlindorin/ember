package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
)

func TestASheetWrittenInTheCodeStylesTheDocument(t *testing.T) {
	box := ui.Div(ui.ID("box"))
	d := ui.NewDocument(ui.CSS(`#box { width: 120px; height: 30px; }`))
	d.Body().Append(box)
	d.Layout(geom.V2(300, 200))
	assert.Equal(t, float32(120), box.Box().W)
	assert.Equal(t, float32(30), box.Box().H)
	assert.Empty(t, d.Warnings())
}

func TestByIDFindsAnElementByItsID(t *testing.T) {
	save := ui.Button(ui.ID("save"), ui.Text("Save"))
	d := ui.NewDocument()
	d.Body().Append(ui.Div(ui.Div(save)))
	assert.Same(t, save, d.ByID("save"))
	assert.Nil(t, d.ByID("missing"))

	other := ui.Button(ui.ID("other"))
	d.Body().Append(other)
	require.Same(t, other, d.ByID("other"), "an element appended later")
	d.Body().Remove(other)
	assert.Nil(t, d.ByID("other"), "an element removed")
}
