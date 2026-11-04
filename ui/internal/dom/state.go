package dom

// State is the set of an element's dynamic pseudo-classes.
type State uint16

const (
	Hover State = 1 << iota
	Focus
	FocusVisible
	Active
	Disabled
	Checked
	Open
	// Modal is a dialog shown modal (HTML's :modal).
	Modal
)

func (s State) Has(f State) bool { return s&f == f }
