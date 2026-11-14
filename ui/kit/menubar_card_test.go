//go:build chrome && gpu

package kit_test

import (
	"testing"

	"github.com/lantervale/ember/ui/internal/chrome"
)

func TestMenuBarCard(t *testing.T) {
	chrome.Card(t, "../internal/pages/kit-29-menu-bar", 420, 260, 0.0035)
}
