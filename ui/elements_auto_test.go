package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/automation"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui"
)

func recordedAs(t *testing.T, l *live, id string) automation.Element {
	t.Helper()
	_, all := l.auto.Snapshot("", "")
	for _, e := range all {
		if e.ID == id {
			return e
		}
	}
	require.Failf(t, "not recorded", "no element %q", id)
	return automation.Element{}
}

func TestAProgressAndAMeterAreRecordedWithTheirValues(t *testing.T) {
	l := liveDoc(t,
		ui.Label(ui.For("load"), ui.Text("Loading")), ui.Progress(ui.ID("load"), ui.Attr("value", "30"),
			ui.Attr("max", "120")),
		ui.Progress(ui.Attr("aria-label", "Waiting")),
		ui.Meter(ui.Attr("aria-label", "Health"), ui.Attr("value", "0.6")))
	load := recordedAs(t, l, "load")
	assert.Equal(t, automation.RoleBar, load.Role)
	assert.Equal(t, "Loading", load.Label, "named by its label")
	assert.InDelta(t, 0.25, load.Value, 1e-6, "value over max")
	assert.Nil(t, recordedAs(t, l, "waiting").Value, "an indeterminate progress has no value")
	health := recordedAs(t, l, "health")
	assert.Equal(t, automation.RoleMeter, health.Role)
	assert.InDelta(t, 0.6, health.Value, 1e-6)
}

func TestAnImageIsNamedByItsAlt(t *testing.T) {
	l := liveDoc(t, ui.Img(ui.Attr("alt", "Crate"), ui.Attr("src", "missing.png")), ui.Img(ui.Attr("src", "missing.png")))
	img := recordedAs(t, l, "crate")
	assert.Equal(t, automation.RoleLabel, img.Role)
	assert.Equal(t, "Crate", img.Label)
}

func TestAnElementWithAnARIAWidgetRoleIsRecordedWithItsValue(t *testing.T) {
	orb := ui.Canvas(func(*twod.Canvas) {}, ui.ID("orb"), ui.Attr("role", "meter"),
		ui.Attr("aria-label", "Health"), ui.Attr("aria-valuenow", "85"))
	bar := ui.Div(ui.Attr("role", "progressbar"), ui.Attr("aria-label", "Loading"), ui.Attr("aria-valuemin", "0"),
		ui.Attr("aria-valuemax", "60"), ui.Attr("aria-valuenow", "30"))
	knob := ui.Div(ui.Attr("role", "slider"), ui.Attr("aria-label", "Volume"), ui.Attr("aria-valuenow", "0.4"))
	l := liveDoc(t, orb, bar, knob)

	o := recordedAs(t, l, "orb")
	assert.Equal(t, automation.RoleMeter, o.Role)
	assert.Equal(t, "Health", o.Label)
	assert.InDelta(t, 85, o.Value, 1e-6, "a meter reads its value as is")

	b := recordedAs(t, l, "loading")
	assert.Equal(t, automation.RoleBar, b.Role)
	assert.InDelta(t, 0.5, b.Value, 1e-6, "a progress bar reads its share of min to max")

	k := recordedAs(t, l, "volume")
	assert.Equal(t, automation.RoleSlider, k.Role)
	assert.InDelta(t, 0.4, k.Value, 1e-6)

	orb.SetAttr("aria-valuenow", "40")
	l.frames(2)
	assert.InDelta(t, 40, recordedAs(t, l, "orb").Value, 1e-6, "follows the attribute")
}
