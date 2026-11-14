//go:build chrome && gpu

package kit_test

import (
	"testing"

	"github.com/lantervale/ember/ui/internal/chrome"
)

func TestProgressCard(t *testing.T) {
	chrome.Card(t, "../internal/pages/kit-13-progress", 400, 230, 0.002)
}
