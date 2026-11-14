//go:build chrome && gpu

package kit_test

import (
	"testing"

	"github.com/lantervale/ember/ui/internal/chrome"
)

func TestHintCard(t *testing.T) {
	chrome.Card(t, "../internal/pages/kit-33-hint", 420, 180, 0.004)
}
