//go:build chrome && gpu

package kit_test

import (
	"testing"

	"github.com/merlindorin/ember/ui/internal/chrome"
)

func TestPathPopUpCard(t *testing.T) {
	chrome.Card(t, "../internal/pages/kit-19-path-pop-up", 480, 340, 0.0032)
}
