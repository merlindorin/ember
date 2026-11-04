package shots //nolint:testpackage // white-box: environ is the runner's own

import (
	"slices"
	"testing"
)

func TestEveryShotIsTakenHiddenAtTwoPixelsAPoint(t *testing.T) {
	env := Shot{Env: map[string]string{"EMBER_CAPTURE_KEYS": "right:0-10"}}.environ(nil, t.TempDir(), "x.png", 30)
	for _, want := range []string{"EMBER_HIDDEN=1", "EMBER_SCALE=2", "EMBER_CAPTURE=x.png", "EMBER_CAPTURE_FRAME=30"} {
		if !slices.Contains(env, want) {
			t.Errorf("no %s in %v", want, env)
		}
	}
}
