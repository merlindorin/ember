package render_test

import (
	"image"
	"testing"

	"github.com/merlindorin/ember/render"
)

func TestATextureTellsItsPresenterWhatChanged(t *testing.T) {
	var tex render.Texture
	v := tex.Version()
	tex.TouchRect(image.Rect(1, 2, 3, 4))
	tex.TouchRect(image.Rect(5, 5, 6, 6))
	rects, whole := tex.Changes()
	if tex.Version() == v || whole || len(rects) != 2 || rects[0] != image.Rect(1, 2, 3, 4) {
		t.Fatalf("rectangles touched, the version moves: %v %v", rects, whole)
	}
	tex.Uploaded()
	if after, all := tex.Changes(); len(after) != 0 || all {
		t.Fatal("once uploaded, nothing changed")
	}
	tex.TouchRect(image.Rect(0, 0, 1, 1))
	tex.Touch()
	if _, all := tex.Changes(); !all {
		t.Fatal("Touch is the whole image")
	}
}
