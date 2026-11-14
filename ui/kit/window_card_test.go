//go:build chrome && gpu

package kit_test

import (
	"testing"

	"github.com/lantervale/ember/ui/internal/chrome"
)

func TestWindowCard(t *testing.T) {
	chrome.Card(t, "../internal/pages/kit-30-window", 580, 440, 0.0025)
}
