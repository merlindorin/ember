//go:build chrome && gpu

package kit_test

import (
	"testing"

	"github.com/lantervale/ember/ui/internal/chrome"
)

func TestThumbnailsCard(t *testing.T) {
	chrome.Card(t, "../internal/pages/kit-26-thumbnails", 420, 500, 0.002)
}
