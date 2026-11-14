package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/automation"
	"github.com/lantervale/ember/ui"
)

func TestADataNamedTextIsRecordedAsALabelButANamedBoxIsNot(t *testing.T) {
	stock := ui.Span(ui.Attr("data-name", "stock"), ui.Text(""))
	l := liveDoc(t, ui.Div(ui.Attr("data-name", "column"), stock, ui.Span(ui.Text("plain"))))
	stock.SetText("Marksman")
	l.frames(2)
	_, all := l.auto.Snapshot("", "")
	got := map[string]automation.Element{}
	for _, e := range all {
		got[e.ID] = e
	}
	assert.Equal(t, automation.RoleLabel, got["column/stock"].Role)
	assert.Equal(t, "Marksman", got["column/stock"].Label, "with the text it shows")
	_, box := got["column"]
	assert.False(t, box, "a named box only scopes")
	assert.Len(t, all, 1, "an unnamed text isn't recorded")
}
