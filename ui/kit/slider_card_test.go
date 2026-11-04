//go:build chrome && gpu

package kit_test

import (
	"testing"

	"github.com/merlindorin/ember/ui/internal/chrome"
)

func TestSliderCard(t *testing.T) {
	chrome.Card(t, "../internal/pages/kit-12-slider", 360, 296, 0.0011)
}
