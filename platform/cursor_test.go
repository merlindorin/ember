package platform //nolint:testpackage // tests the unexported cursor registry

import (
	"fmt"
	"image"
	"testing"

	"github.com/lantervale/ember/internal/cursorhint"
)

func TestAPictureCursorKeepsItsPictureAndHotSpot(t *testing.T) {
	small, big := image.NewNRGBA(image.Rect(0, 0, 16, 16)), image.NewNRGBA(image.Rect(0, 0, 32, 32))
	bow := NewCursor(small, 3, 5, HiDPI(big))
	plain := NewCursor(small, 0, 0)
	if bow == plain || bow == CursorArrow || plain == CursorHand {
		t.Fatalf("each picture is a cursor of its own: %v %v", bow, plain)
	}
	if x, y := bow.Hot(); bow.Image() != small || bow.HiDPI() != big || x != 3 || y != 5 {
		t.Fatalf("its picture, its HiDPI picture and its hot spot: %v, %v", x, y)
	}
	if plain.HiDPI() != nil || CursorHand.Image() != nil || CursorHand.HiDPI() != nil {
		t.Fatal("no HiDPI picture unless given, and none for a standard shape")
	}
}

func TestAPictureCursorShowsLikeAShapeAndTheUIStandsAside(t *testing.T) {
	bow := NewCursor(image.NewNRGBA(image.Rect(0, 0, 16, 16)), 8, 8)
	w := NewWindow()
	w.SetCursor(bow)
	if w.Cursor() != bow {
		t.Fatal("a window not yet open keeps it, for tests")
	}
	cursorhint.Show(w, int(CursorHand))
	if w.Cursor() != bow {
		t.Fatal("the UI's hand doesn't replace the game's cursor")
	}
	f := &fake{}
	w.Attach(f)
	if want := fmt.Sprintf("[cursor %d]", bow); fmt.Sprint(f.calls) != want {
		t.Fatalf("an opening window shows it at once: %v, want %v", f.calls, want)
	}
}
