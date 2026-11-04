//go:build chrome && gpu

package kit_test

import (
	"testing"

	"github.com/merlindorin/ember/ui/internal/chrome"
)

func TestGridCard(t *testing.T) {
	chrome.Card(t, "../internal/pages/kit-07-grid", 420, 480, 0.0047)
}
