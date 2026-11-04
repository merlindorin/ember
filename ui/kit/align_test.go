package kit_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/merlindorin/ember/ui/align"
)

// alignSkip is why page name is skipped: another task is fixing it, until
// that lands.
func alignSkip(name string) (string, bool) {
	switch name {
	case "kit-30-window":
		return "TODO(ui-developer-1): FileDialog and window alignment", true
	case "kit-31-alert":
		return "TODO(ui-developer-1): Alert alignment", true
	}
	return "", false
}

// TestKitPagesLineUp lays out every kit page, as its card does, and checks
// it with align: the controls sharing a row are one height unless
// one carries class "tall", each control's label is in its middle, and the
// text of a row shares a baseline. A failure names each offender by its
// automation path, with the numbers.
func TestKitPagesLineUp(t *testing.T) {
	pages, err := filepath.Glob("../internal/pages/kit-*")
	require.NoError(t, err)
	bin := t.TempDir()
	build := exec.Command("go", append([]string{"build", "-o", bin + "/"}, pages...)...)
	if out, buildErr := build.CombinedOutput(); buildErr != nil {
		t.Fatalf("building the pages: %v: %s", buildErr, out)
	}
	for _, page := range pages {
		name := filepath.Base(page)
		t.Run(name, func(t *testing.T) {
			if why, skip := alignSkip(name); skip {
				t.Skip(why)
			}
			out := filepath.Join(t.TempDir(), "faults.json")
			run := exec.Command(filepath.Join(bin, name))
			run.Env = append(os.Environ(), "EMBER_UI_ALIGN="+out)
			if msg, runErr := run.CombinedOutput(); runErr != nil {
				t.Fatalf("%v: %s", runErr, msg)
			}
			data, readErr := os.ReadFile(out)
			require.NoError(t, readErr)
			var faults []align.Fault
			require.NoError(t, json.Unmarshal(data, &faults))
			if len(faults) > 0 {
				var b strings.Builder
				for _, f := range faults {
					b.WriteString("\n  " + f.String())
				}
				t.Errorf("%d out of line:%s", len(faults), b.String())
			}
		})
	}
}
