package gpu //nolint:testpackage // tests the unexported GPU timing switch

import (
	"testing"

	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/world"
)

func TestTheGPUIsTimedOnlyWhileTheProfilerShows(t *testing.T) {
	w := world.New()
	w.AddEntities(platform.NewWindow())
	w.AddPlugins(twod.Plugin{})
	if timing(w) {
		t.Fatal("with F3 closed, no timer query runs")
	}
	twod.ShowProfiler(w, true)
	timed := 0
	for range 2 * gpuSample {
		w.Tick(1.0 / 60)
		if timing(w) {
			timed++
		}
	}
	if timed != 2 {
		t.Fatalf("F3 open times the passes one frame in %d: %d of %d", gpuSample, timed, 2*gpuSample)
	}
	twod.ShowProfiler(w, false)
	if timing(w) {
		t.Fatal("and closing it stops them")
	}
}
