//go:build chrome && gpu

package kit_test

import (
	"testing"

	"github.com/merlindorin/ember/ui/internal/chrome"
)

func TestRowColumnCard(t *testing.T) {
	chrome.Card(t, "../internal/pages/kit-06-row-column", 420, 680, 0.0026)
}
