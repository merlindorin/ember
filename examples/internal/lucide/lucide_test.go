package lucide_test

import (
	"testing"

	"github.com/lantervale/ember/examples/internal/lucide"
)

func TestTheFontNamesItsIcons(t *testing.T) {
	icons, err := lucide.Kit()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"sun", "play", "eye", "settings"} {
		if !icons.Has(name) {
			t.Errorf("no icon %q", name)
		}
	}
}
