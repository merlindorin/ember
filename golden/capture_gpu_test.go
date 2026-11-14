//go:build gpu

package golden //nolint:testpackage // runs scenes through the unexported run

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestACaptureIsTheSamePictureWithOnlyItsFrameDrawn holds decision 0072:
// a hidden still capture ticks the frames before the captured one without
// drawing them, and the picture is byte for byte the one every frame
// drawn gives; 41-probes' slimes wait on their light probes, answered
// from the frames drawn.
func TestACaptureIsTheSamePictureWithOnlyItsFrameDrawn(t *testing.T) {
	for _, s := range []Scene{
		{Name: "lights2d", Pkg: "examples/internal/lights2d", Frame: 12},
		{Name: "41-probes", Pkg: "examples/41-probes", Frame: 8},
		{Name: "look3d", Pkg: "examples/internal/look3d", Frame: 6},
		{Name: "10-multi-window", Pkg: "examples/10-multi-window", Frame: 6},
	} {
		t.Run(s.Name, func(t *testing.T) {
			shoot := func(name string, env ...string) string {
				dir := t.TempDir()
				shot := filepath.Join(dir, name+".png")
				s.Env = env
				if out, _, err := run(context.Background(), "..", s, shot, dir); err != nil {
					if os.Getenv(optionalEnv) == "1" && strings.Contains(string(out), "no graphics adapter") {
						t.Skipf("no GPU adapter here, and %s=1: %s", optionalEnv, out)
					}
					t.Fatalf("%v: %s", err, out)
				}
				return shot
			}
			every, one := shoot("every", "EMBER_CAPTURE_DRAW=all"), shoot("one")
			for n := 1; ; n++ {
				want, err := os.ReadFile(windowShot(every, n))
				if err != nil {
					if n == 1 {
						t.Fatal("no capture")
					}
					return
				}
				got, err := os.ReadFile(windowShot(one, n))
				if err != nil || !bytes.Equal(got, want) {
					t.Errorf("window %d differs with only frame %d drawn (%v)", n, s.Frame, err)
				}
			}
		})
	}
}
