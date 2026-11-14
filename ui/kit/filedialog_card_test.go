//go:build chrome && gpu

package kit_test

import (
	"testing"

	"github.com/lantervale/ember/ui/internal/chrome"
)

func TestFileDialogCard(t *testing.T) {
	chrome.Card(t, "../internal/pages/kit-32-file-dialog", 940, 620, 0.0029)
}

func TestFileDialogSaveCard(t *testing.T) {
	chrome.Card(t, "../internal/pages/kit-32b-file-dialog-save", 940, 620, 0.0028)
}

func TestFileDialogReplaceCard(t *testing.T) {
	chrome.Card(t, "../internal/pages/kit-32c-file-dialog-replace", 940, 620, 0.0028)
}

func TestFileDialogOwnWindowCard(t *testing.T) {
	chrome.Card(t, "../internal/pages/kit-32d-file-dialog-own-window", 900, 600, 0.0031)
}
