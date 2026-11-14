//go:build chrome && gpu

package kit_test

import (
	"testing"

	"github.com/lantervale/ember/ui/internal/chrome"
)

func TestIconsCard(t *testing.T) {
	chrome.Card(t, "../internal/pages/kit-03-icons", 340, 400, 0.005)
}
