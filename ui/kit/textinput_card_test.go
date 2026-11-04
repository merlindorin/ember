//go:build chrome && gpu

package kit_test

import (
	"testing"

	"github.com/merlindorin/ember/ui/internal/chrome"
)

func TestTextInputCard(t *testing.T) {
	chrome.Card(t, "../internal/pages/kit-14-textinput", 470, 370, 0.004)
}
