//go:build chrome && gpu

package kit_test

import (
	"testing"

	"github.com/lantervale/ember/ui/internal/chrome"
)

func TestTabsCard(t *testing.T) {
	chrome.Card(t, "../internal/pages/kit-23-tabs", 440, 200, 0.003)
}
