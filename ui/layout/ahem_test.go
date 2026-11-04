package layout //nolint:testpackage // tests the layout tree and its unexported helpers

import "testing"

func TestAhemContentWidths(t *testing.T) {
	for _, c := range []struct {
		text     string
		min, max float32
	}{
		{"", 0, 0},
		{"HH", 20, 20},
		{"HHHH\u200bHH", 40, 60},
		{"HH\u200bHHHH\u200bH", 40, 70},
	} {
		lo, hi := ahemContentWidths(c.text)
		if lo != c.min || hi != c.max {
			t.Errorf("%q: %v, %v, want %v, %v", c.text, lo, hi, c.min, c.max)
		}
	}
}

func TestAhemHeight(t *testing.T) {
	for _, c := range []struct {
		text   string
		width  float32
		height float32
	}{
		{"HHHH", 40, 10},
		{"HHHH", 10, 10},
		{"HH\u200bHH\u200bHH", 60, 10},
		{"HH\u200bHH\u200bHH", 59, 20},
		{"HH\u200bHH\u200bHH", 40, 20},
		{"HH\u200bHH\u200bHH", 20, 30},
		{"HHHH\u200bH\u200bHHHH", 50, 20},
	} {
		if got := ahemHeight(c.text, c.width); got != c.height {
			t.Errorf("%q at %v: height %v, want %v", c.text, c.width, got, c.height)
		}
	}
}
