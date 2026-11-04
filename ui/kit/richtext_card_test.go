//go:build chrome && gpu

package kit_test

import (
	"testing"

	"github.com/merlindorin/ember/ui/internal/chrome"
)

func TestRichTextCard(t *testing.T) {
	chrome.Card(t, "../internal/pages/kit-02-rich-text", 380, 470, 0.006)
}
