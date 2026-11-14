//go:build chrome && gpu

package kit_test

import (
	"testing"

	"github.com/lantervale/ember/ui/internal/chrome"
)

func TestButtonCard(t *testing.T) {
	chrome.Card(t, "../internal/pages/kit-09-button", 470, 240, 0.0066)
}
