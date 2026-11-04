//go:build chrome && gpu

package kit_test

import (
	"testing"

	"github.com/merlindorin/ember/ui/internal/chrome"
)

func TestTableSpansCard(t *testing.T) {
	chrome.Card(t, "../internal/pages/kit-24c-table-spans", 412, 220, 0.004)
}
