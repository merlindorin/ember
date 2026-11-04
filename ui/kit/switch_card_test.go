//go:build chrome && gpu

package kit_test

import (
	"testing"

	"github.com/merlindorin/ember/ui/internal/chrome"
)

func TestSwitchCard(t *testing.T) {
	chrome.Card(t, "../internal/pages/kit-11-switch", 360, 200, 0.004)
}
