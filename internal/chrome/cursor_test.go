//go:build chrome

package chrome_test

import (
	"strings"
	"testing"
	"time"

	"github.com/lantervale/ember/internal/chrome"
)

// TestAPictureCursorShowsOnTheCanvas runs testdata/cursor: its picture
// cursor becomes the canvas's CSS cursor, which the browser keeps only
// when it accepts it, its HiDPI picture in an image-set.
func TestAPictureCursorShowsOnTheCanvas(t *testing.T) {
	srv := servePackage(t, "./internal/chrome/testdata/cursor")
	b, err := chrome.Launch(320, 200, 1)
	if err != nil {
		t.Skipf("no Chrome to drive: %v", err)
	}
	defer b.Close()
	if err = b.Navigate(srv.URL + "/index.html"); err != nil {
		t.Fatal(err)
	}
	cursor := `window.emberError ? "error: " + window.emberError : document.getElementById("ember").style.cursor`
	got, err := b.Wait(cursor, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "image-set(") || !strings.Contains(got, "data:image/png;base64,") ||
		!strings.HasSuffix(got, " 3 5, auto") {
		t.Fatalf("the canvas shows the picture, 2x beside it, its hot spot at 3, 5: %.120s…%s",
			got, got[max(len(got)-20, 0):])
	}
}
