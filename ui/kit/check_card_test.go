//go:build chrome && gpu

package kit_test

import (
	"testing"

	"github.com/merlindorin/ember/ui/internal/chrome"
)

func TestCheckCard(t *testing.T) {
	chrome.Card(t, "../internal/pages/kit-10-check", 360, 240, 0.0052)
}
