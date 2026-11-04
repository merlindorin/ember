//go:build chrome && gpu

package kit_test

import (
	"testing"

	"github.com/merlindorin/ember/ui/internal/chrome"
)

func TestMenuCard(t *testing.T) {
	chrome.Card(t, "../internal/pages/kit-28-menu", 420, 320, 0.0031)
}
