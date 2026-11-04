package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func picture(t *testing.T, name string, c color.Color) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 4, 2))
	for x := range 4 {
		img.Set(x, 0, color.Black)
		img.Set(x, 1, color.Black)
	}
	img.Set(3, 1, c)
	path := filepath.Join(t.TempDir(), name)
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err = png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestDifferCountsTheChangedPixels(t *testing.T) {
	a, same, other := picture(t, "a.png", color.Black), picture(t, "b.png", color.Black), picture(t, "c.png", color.White)
	if n := differ(a, same); n != 0 {
		t.Errorf("the same picture: %d", n)
	}
	if n := differ(a, other); n != 1 {
		t.Errorf("one pixel changed: %d", n)
	}
	if n := differ(a, ""); n != -1 {
		t.Errorf("a missing picture: %d", n)
	}
}

func TestTheUIKitsElementsComeFromItsList(t *testing.T) {
	m := oneOf.FindSubmatch([]byte(`2026/11/03 no element "-"; try one of alert, button, canvas` + "\n"))
	if m == nil || string(m[1]) != "alert, button, canvas" {
		t.Errorf("%q", m)
	}
}

func TestKitPagesAreFoundUnderUIOrTheOldUIv2(t *testing.T) {
	for _, root := range []string{"ui", "uiv2"} {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, root, "internal", "pages", "kit-01-label"), 0o755); err != nil {
			t.Fatal(err)
		}
		if got := kitPages(dir); len(got) != 1 || got[0] != "./"+root+"/internal/pages/kit-01-label" {
			t.Errorf("%s: %v", root, got)
		}
	}
}

func TestAnEmptyComparisonFails(t *testing.T) {
	if code := verdict(0, 0); code == 0 {
		t.Error("no pictures compared passed: a worktree without the wgpu library takes none")
	}
	if code := verdict(77, 0); code != 0 {
		t.Errorf("77 pictures, none changed: exit %d", code)
	}
	if code := verdict(77, 1); code == 0 {
		t.Error("a changed picture passed")
	}
}
