//go:build chrome && gpu

package kit_test

import (
	"testing"

	"github.com/lantervale/ember/ui/internal/chrome"
)

func TestSeparatorCard(t *testing.T) {
	chrome.Card(t, "../internal/pages/kit-05-separator", 360, 370, 0.005)
}
