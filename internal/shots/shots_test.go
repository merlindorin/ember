package shots_test

import (
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lantervale/ember/internal/shots"
)

// root is ember's root, from this package.
const root = "../.."

// The list is checked without a GPU: each shot is well formed and shown
// by a page, and what it runs or copies exists.
func TestEveryShotIsWellFormedAndShown(t *testing.T) {
	refs, err := shots.Refs(root)
	if err != nil {
		t.Fatal(err)
	}
	shown := map[string]bool{}
	for _, r := range refs {
		shown[r.Picture] = true
	}
	seen := map[string]bool{}
	for _, s := range shots.Shots {
		check(t, s, shown, seen)
	}
}

// check is one shot's checks: its path, its page, its kind, what it runs
// or copies, and its sizes.
func check(t *testing.T, s shots.Shot, shown, seen map[string]bool) {
	t.Helper()
	if seen[s.Out] {
		t.Errorf("%s: listed twice", s.Out)
	}
	seen[s.Out] = true
	if !shots.Made(s.Out) || strings.HasPrefix(s.Out, "/") || strings.Contains(s.Out, "..") {
		t.Errorf("%s: a .png or .gif path under docs/", s.Out)
	}
	if !shown[s.Out] {
		t.Errorf("%s: no page shows it", s.Out)
	}
	checkSource(t, s)
	checkSizes(t, s)
	for k := range s.Env {
		if k == "EMBER_CAPTURE" || k == "EMBER_HIDDEN" || k == "EMBER_SCALE" || k == "EMBER_CAPTURE_FRAME" {
			t.Errorf("%s: %s is the runner's (Frame sets the frame)", s.Out, k)
		}
	}
}

// checkSource checks that s runs, lines up or copies one thing, and that
// it and s's files exist.
func checkSource(t *testing.T, s shots.Shot) {
	t.Helper()
	kinds := 0
	for _, k := range []bool{s.Run != "", len(s.Panels) > 0, s.Copy != ""} {
		if k {
			kinds++
		}
	}
	if kinds != 1 {
		t.Errorf("%s: one of Run, Panels or Copy", s.Out)
	}
	if s.Run != "" {
		if _, err := os.Stat(filepath.Join(root, s.Run)); err != nil || !strings.HasPrefix(s.Run, "./") {
			t.Errorf("%s: Run %q is no package under ember", s.Out, s.Run)
		}
	}
	for _, f := range append(append([]string{}, s.Files...), s.Copy) {
		if _, err := os.Stat(filepath.Join(root, f)); f != "" && err != nil {
			t.Errorf("%s: no %s", s.Out, f)
		}
	}
}

// checkSizes checks s's crop and size: none for a GIF, a rectangle and a
// whole size otherwise.
func checkSizes(t *testing.T, s shots.Shot) {
	t.Helper()
	if path.Ext(s.Out) == ".gif" && (s.Crop != ([4]int{}) || s.Size != ([2]int{})) {
		t.Errorf("%s: a .gif is kept as recorded, with no Crop or Size", s.Out)
	}
	if c := s.Crop; c != ([4]int{}) && (c[2] <= 0 || c[3] <= 0 || c[0] < 0 || c[1] < 0) {
		t.Errorf("%s: Crop %v is no rectangle", s.Out, c)
	}
	if s.Size[0] < 0 || s.Size[1] < 0 || (s.Size[0] == 0) != (s.Size[1] == 0) {
		t.Errorf("%s: Size %v", s.Out, s.Size)
	}
}
