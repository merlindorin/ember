package control_test

import (
	"testing"

	"github.com/lantervale/ember/ui/control"
)

func TestARadioChecksOnceAndNotWhenDisabled(t *testing.T) {
	r := &control.Radio{}
	if e := r.Click(); !e.Has(control.Changed) || !r.Checked {
		t.Errorf("first click: %v %v", e, r.Checked)
	}
	if e := r.Click(); e != 0 {
		t.Errorf("checked already: %v", e)
	}
	off := &control.Radio{Disabled: true}
	if off.Handles(control.Activate) || off.Click() != 0 || off.Checked {
		t.Error("disabled")
	}
	r2 := &control.Radio{}
	if e := r2.Do(control.Activate); !e.Has(control.Changed) || !r2.Checked {
		t.Errorf("Activate: %v", e)
	}
}
