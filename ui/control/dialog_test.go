package control_test

import (
	"testing"

	"github.com/lantervale/ember/ui/control"

	"github.com/stretchr/testify/assert"
)

func TestAModalDialogClosesOnBackWithNoReturnValue(t *testing.T) {
	d := &control.Dialog{}
	d.ShowModal()
	assert.True(t, d.Open)
	assert.True(t, d.Modal)
	assert.True(t, d.Handles(control.Back))
	assert.Equal(t, control.Closed, d.Do(control.Back))
	assert.False(t, d.Open)
	assert.False(t, d.Modal)
	assert.Equal(t, "", d.ReturnValue)
}

func TestADialogClosesWithAReturnValueOnce(t *testing.T) {
	d := &control.Dialog{}
	d.Show()
	assert.False(t, d.Modal)
	assert.False(t, d.Handles(control.Back), "a non-modal dialog doesn't take Escape")
	assert.Equal(t, control.Closed, d.Close("ok"))
	assert.Equal(t, "ok", d.ReturnValue)
	assert.Equal(t, control.Events(0), d.Close("again"), "already closed")
	assert.Equal(t, "ok", d.ReturnValue)
}

func TestProgressIsAFractionOrIndeterminate(t *testing.T) {
	assert.Equal(t, 0.5, (&control.Progress{Value: 0.5, Max: 1}).Fraction())
	assert.Equal(t, 1.0, (&control.Progress{Value: 7, Max: 5}).Fraction())
	assert.Equal(t, 0.25, (&control.Progress{Value: 0.25}).Fraction(), "Max 0 means 1")
	p := &control.Progress{Value: -1, Indeterminate: true}
	assert.True(t, p.Indeterminate)
	assert.Equal(t, 0.0, p.Fraction())
}

func TestAMeterIsInTheRegionItsOptimumSays(t *testing.T) {
	health := func(v float64) control.Region {
		return (&control.Meter{Value: v, Min: 0, Max: 100, Low: 25, High: 75, Optimum: 90}).Region()
	}
	assert.Equal(t, control.Optimum, health(80))
	assert.Equal(t, control.Suboptimal, health(50))
	assert.Equal(t, control.EvenLessGood, health(10))

	heat := func(v float64) control.Region {
		return (&control.Meter{Value: v, Min: 0, Max: 100, Low: 25, High: 75, Optimum: 10}).Region()
	}
	assert.Equal(t, control.Optimum, heat(10))
	assert.Equal(t, control.Suboptimal, heat(50))
	assert.Equal(t, control.EvenLessGood, heat(90))

	mid := func(v float64) control.Region {
		return (&control.Meter{Value: v, Min: 0, Max: 100, Low: 25, High: 75, Optimum: 50}).Region()
	}
	assert.Equal(t, control.Optimum, mid(50))
	assert.Equal(t, control.Suboptimal, mid(10))
	assert.Equal(t, control.Suboptimal, mid(90))
}

func TestAMeterDefaultsLikeHTML(t *testing.T) {
	m := control.NewMeter(0.3)
	assert.Equal(t, 0.0, m.Min)
	assert.Equal(t, 1.0, m.Max)
	assert.Equal(t, 0.0, m.Low)
	assert.Equal(t, 1.0, m.High)
	assert.Equal(t, 0.5, m.Optimum)
	assert.Equal(t, control.Optimum, m.Region())
	assert.InDelta(t, 0.3, m.Fraction(), 1e-9)
}
