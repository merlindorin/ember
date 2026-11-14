//go:build chrome && gpu

package kit_test

import (
	"testing"

	"github.com/lantervale/ember/ui/internal/chrome"
)

func TestStatusBarCard(t *testing.T) {
	chrome.Card(t, "../internal/pages/kit-27-status-bar", 380, 150, 0.0026)
}
