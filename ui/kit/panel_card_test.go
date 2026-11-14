//go:build chrome && gpu

package kit_test

import (
	"testing"

	"github.com/lantervale/ember/ui/internal/chrome"
)

func TestPanelCard(t *testing.T) {
	chrome.Card(t, "../internal/pages/kit-08-panel", 420, 720, 0.0034)
}
