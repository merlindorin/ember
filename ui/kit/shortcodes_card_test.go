//go:build chrome && gpu

package kit_test

import (
	"testing"

	"github.com/lantervale/ember/ui/internal/chrome"
)

func TestShortcodesCard(t *testing.T) {
	chrome.Card(t, "../internal/pages/kit-14b-shortcodes", 300, 200, 0.0064)
}
