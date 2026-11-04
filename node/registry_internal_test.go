package node

import "testing"

func TestPropertiesAreNamedAfterTheirFields(t *testing.T) {
	for field, want := range map[string]string{
		"WaitTime": "wait_time", "FOV": "fov", "ZIndex": "z_index", "YSortEnabled": "y_sort_enabled",
		"FlipH": "flip_h", "Hframes": "hframes", "Text": "text", "SizeFlagsStretchRatio": "size_flags_stretch_ratio",
	} {
		if got := snake(field); got != want {
			t.Errorf("%s: %s, want %s", field, got, want)
		}
	}
}
