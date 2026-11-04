package control

// Radio is an input of type radio: Activate or a click checks it, which
// ui follows by unchecking the rest of its group; the arrows move the
// check along the group, which ui does too, as it knows the group.
type Radio struct{ Checked, Disabled bool }

func (r *Radio) Handles(i Intent) bool { return i == Activate && !r.Disabled }

func (r *Radio) Do(i Intent) Events {
	if !r.Handles(i) {
		return 0
	}
	return r.Click()
}

// Click checks the radio: Input and Changed, nothing when it already was.
func (r *Radio) Click() Events {
	if r.Disabled || r.Checked {
		return 0
	}
	r.Checked = true
	return Input | Changed
}
