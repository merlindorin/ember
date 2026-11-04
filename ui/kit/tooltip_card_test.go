//go:build chrome && gpu

package kit_test

import (
	"testing"

	"github.com/merlindorin/ember/ui/internal/chrome"
)

func TestTooltipCard(t *testing.T) {
	chrome.Card(t, "../internal/pages/kit-20-tooltip", 440, 470, 0.004)
}
