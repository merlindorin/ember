//go:build chrome && gpu

package kit_test

import (
	"testing"

	"github.com/merlindorin/ember/ui/internal/chrome"
)

func TestChoiceCard(t *testing.T) {
	chrome.Card(t, "../internal/pages/kit-17-choice", 380, 330, 0.005)
}
