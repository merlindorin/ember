//go:build chrome && gpu

package kit_test

import (
	"testing"

	"github.com/lantervale/ember/ui/internal/chrome"
)

func TestPopUpCard(t *testing.T) {
	chrome.Card(t, "../internal/pages/kit-18-popup", 320, 230, 0.0044)
}
