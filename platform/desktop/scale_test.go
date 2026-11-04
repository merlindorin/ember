package desktop //nolint:testpackage // white-box: framebuffer is the runner's own rule

import "testing"

func TestAHiddenRunDrawsAtTheScaleItAsksForWhateverTheScreen(t *testing.T) {
	for _, c := range []struct {
		name           string
		ww, wh, fw, fh int
		pixels         float32
		w, h           int
	}{
		{"no scale asked: the system's", 640, 360, 640, 360, 0, 640, 360},
		{"a 1× screen asked for 2: twice the points", 640, 360, 640, 360, 2, 1280, 720},
		{"a Retina screen asked for 2: as it is", 640, 360, 1280, 720, 2, 1280, 720},
		{"rounded to whole pixels", 333, 201, 333, 201, 1.5, 500, 302},
	} {
		if w, h := framebuffer(c.ww, c.wh, c.fw, c.fh, c.pixels); w != c.w || h != c.h {
			t.Errorf("%s: %d×%d, want %d×%d", c.name, w, h, c.w, c.h)
		}
	}
}
