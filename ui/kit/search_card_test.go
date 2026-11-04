//go:build chrome && gpu

package kit_test

import (
	"testing"

	"github.com/merlindorin/ember/ui/internal/chrome"
)

func TestSearchCard(t *testing.T) {
	chrome.Card(t, "../internal/pages/kit-14b-search", 340, 200, 0.0038)
}
