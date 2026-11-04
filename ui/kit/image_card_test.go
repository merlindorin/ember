//go:build chrome && gpu

package kit_test

import (
	"testing"

	"github.com/merlindorin/ember/ui/internal/chrome"
)

func TestImageCard(t *testing.T) {
	chrome.Card(t, "../internal/pages/kit-04-image", 400, 720, 0.0018)
}
