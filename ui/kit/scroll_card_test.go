//go:build chrome && gpu

package kit_test

import (
	"testing"

	"github.com/lantervale/ember/ui/internal/chrome"
)

func TestScrollCard(t *testing.T) {
	chrome.Card(t, "../internal/pages/kit-21-scroll", 420, 520, 0.0094)
}
