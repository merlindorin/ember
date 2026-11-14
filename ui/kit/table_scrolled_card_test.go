//go:build chrome && gpu

package kit_test

import (
	"testing"

	"github.com/lantervale/ember/ui/internal/chrome"
)

func TestTableScrolledCard(t *testing.T) {
	chrome.Card(t, "../internal/pages/kit-24b-table-scrolled", 412, 232, 0.0075)
}
