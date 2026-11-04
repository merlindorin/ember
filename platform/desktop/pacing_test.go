package desktop //nolint:testpackage // tests the driver's unexported frame pacing

import (
	"testing"
	"time"
)

func TestWithoutVSyncAWindowShowsAtMostOneFrameARefresh(t *testing.T) {
	refresh := refreshInterval(nil)
	if refresh != time.Second/60 {
		t.Fatalf("a display that doesn't say is taken for 60 Hz: %v", refresh)
	}
	if !held(false, 5*time.Millisecond, refresh) {
		t.Fatal("a frame drawn 5 ms after the last shown one stays offscreen")
	}
	if held(false, 17*time.Millisecond, refresh) {
		t.Fatal("once a refresh has passed, the next frame shows")
	}
	if held(true, 0, refresh) {
		t.Fatal("with vsync, every frame shows and waits for the display")
	}
}
