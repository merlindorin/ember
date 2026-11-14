//go:build chrome && gpu

package kit_test

import (
	"testing"

	"github.com/lantervale/ember/ui/internal/chrome"
)

func TestNumberFieldCard(t *testing.T) {
	chrome.Card(t, "../internal/pages/kit-16-number-field", 420, 260, 0.004)
}
