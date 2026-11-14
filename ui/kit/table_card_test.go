//go:build chrome && gpu

package kit_test

import (
	"testing"

	"github.com/lantervale/ember/ui/internal/chrome"
)

func TestTableCard(t *testing.T) {
	chrome.Card(t, "../internal/pages/kit-24-table", 412, 340, 0.0052)
}
