package desktop //nolint:testpackage // tests the unexported run stats

import (
	"strings"
	"testing"
	"time"

	"github.com/merlindorin/ember/world"
)

func TestRunStatsAverageThePastWarmupCostliestFirst(t *testing.T) {
	w := world.New()
	w.On(world.Update, "slow", func(world.Frame) { time.Sleep(2 * time.Millisecond) })
	w.On(world.Update, "quick", func(world.Frame) {})
	var r runStats
	for frame := 1; frame <= statsWarmup+10; frame++ {
		w.Tick(1.0 / 60)
		w.Stats().Record("gpu 2d window", 50*time.Millisecond)
		r.add(w, frame, time.Duration(frame)*time.Millisecond)
	}
	if r.frames != 10 {
		t.Fatalf("the warm-up is skipped: %d frames", r.frames)
	}
	if r.worst != time.Duration(statsWarmup+10)*time.Millisecond {
		t.Fatalf("the worst frame is kept: %v", r.worst)
	}
	var out strings.Builder
	r.print(&out, 2)
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 3 || !strings.HasPrefix(lines[0], "stats over 10 frames") {
		t.Fatalf("a summary and the two costliest:\n%s", out.String())
	}
	if !strings.Contains(lines[0], " work ") || !strings.Contains(lines[0], "fps possible") {
		t.Fatalf("the work apart from the display's wait, and the rate it allows:\n%s", lines[0])
	}
	if !strings.Contains(lines[1], "gpu 2d window") || !strings.Contains(lines[1], "50.00 ms") ||
		!strings.Contains(lines[2], "slow (Update)") {
		t.Fatalf("recorded costs and systems, costliest first:\n%s", out.String())
	}
}

func TestASlowFrameIsReportedWithWhatItSpent(t *testing.T) {
	w := world.New()
	slow := false
	w.On(world.Update, "loader", func(world.Frame) {
		if slow {
			time.Sleep(20 * time.Millisecond)
		}
	})
	var r runStats
	var out strings.Builder
	for frame := 1; frame <= statsWarmup+3; frame++ {
		slow = frame == statsWarmup+2
		start := time.Now()
		w.Tick(1.0 / 60)
		w.Stats().Record("acquire", 15*time.Millisecond)
		r.spike(&out, w, frame, time.Since(start)+15*time.Millisecond)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 1 || !strings.HasPrefix(lines[0], "spike: frame 62") || !strings.Contains(lines[0], "loader") {
		t.Fatalf("the one slow frame, with the system that took it:\n%s", out.String())
	}
}
