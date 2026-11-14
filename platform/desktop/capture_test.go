package desktop //nolint:testpackage // tests which frames a run's unexported settings draw

import "testing"

func TestAHiddenStillCaptureDrawsOnlyItsFrame(t *testing.T) {
	for _, c := range []struct {
		name          string
		hidden, shot  string
		draw          string
		frame         int
		drawn, always bool
	}{
		{"a frame before the captured one is ticked only", "1", "shot.png", "", 29, false, false},
		{"the captured frame is drawn", "1", "shot.png", "", 30, true, false},
		{"a window that shows draws every frame", "", "shot.png", "", 29, true, true},
		{"a GIF draws every frame", "1", "walk.gif", "", 29, true, true},
		{"a hidden run that captures nothing draws every frame", "1", "", "", 29, true, true},
		{"EMBER_CAPTURE_DRAW=all draws every frame", "1", "shot.png", "all", 29, true, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("EMBER_HIDDEN", c.hidden)
			t.Setenv("EMBER_CAPTURE", c.shot)
			t.Setenv("EMBER_CAPTURE_FRAME", "30")
			t.Setenv("EMBER_CAPTURE_DRAW", c.draw)
			set := readRunSettings()
			if got := set.draws(c.frame); got != c.drawn {
				t.Errorf("frame %d drawn: %v, want %v", c.frame, got, c.drawn)
			}
			if set.everyFrame != c.always {
				t.Errorf("every frame drawn: %v, want %v", set.everyFrame, c.always)
			}
		})
	}
}
