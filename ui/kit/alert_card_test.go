//go:build chrome && gpu

package kit_test

import (
	"testing"

	"github.com/merlindorin/ember/ui/internal/chrome"
)

func TestAlertCard(t *testing.T) {
	chrome.Card(t, "../internal/pages/kit-31-alert", 560, 300, 0.002)
}
