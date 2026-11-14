//go:build chrome && gpu

package kit_test

import (
	"testing"

	"github.com/lantervale/ember/ui/internal/chrome"
)

func TestLabelCard(t *testing.T) {
	chrome.Card(t, "../internal/pages/kit-01-label", 340, 360, 0.006)
}
