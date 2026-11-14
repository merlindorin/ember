//go:build chrome && gpu

package kit_test

import (
	"testing"

	"github.com/lantervale/ember/ui/internal/chrome"
)

func TestFormCard(t *testing.T) {
	chrome.Card(t, "../internal/pages/kit-22-form", 360, 280, 0.002)
}
