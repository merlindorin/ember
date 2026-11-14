//go:build chrome && gpu

package kit_test

import (
	"testing"

	"github.com/lantervale/ember/ui/internal/chrome"
)

func TestTextAreaCard(t *testing.T) {
	chrome.Card(t, "../internal/pages/kit-15-textarea", 560, 550, 0.012)
}
