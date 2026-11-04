//go:build chrome && gpu

package kit_test

import (
	"testing"

	"github.com/merlindorin/ember/ui/internal/chrome"
)

func TestSidebarCard(t *testing.T) {
	chrome.Card(t, "../internal/pages/kit-25-sidebar", 420, 250, 0.0029)
}
