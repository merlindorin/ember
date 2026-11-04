package pages_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/merlindorin/ember/ui/internal/pages"
)

// runPage builds the page in dir and runs it with EMBER_UI_BENCH, returning its frames.
func runPage(b *testing.B, dir string) pages.Frames {
	b.Helper()
	bin := filepath.Join(b.TempDir(), "page")
	if out, err := exec.Command("go", "build", "-o", bin, "./"+dir).CombinedOutput(); err != nil {
		b.Fatalf("%s: %v\n%s", dir, err, out)
	}
	cmd := exec.Command(bin)
	cmd.Env = append(os.Environ(), "EMBER_UI_BENCH=1")
	out, err := cmd.Output()
	if err != nil {
		b.Fatalf("%s: %v", dir, err)
	}
	var f pages.Frames
	if err = json.Unmarshal(out, &f); err != nil {
		b.Fatalf("%s: %v: %s", dir, err, out)
	}
	return f
}

func report(b *testing.B, f pages.Frames) {
	metric := func(name string, fr *pages.Frame) {
		if fr == nil {
			return
		}
		b.ReportMetric(float64(fr.NsPerOp)/1e3, name+"-µs/frame")
		b.ReportMetric(float64(fr.AllocsPerOp), name+"-allocs/frame")
	}
	b.ReportMetric(float64(f.First.NsPerOp)/1e3, "first-µs")
	metric("static", &f.Static)
	metric("hover", f.Hover)
	metric("edit", f.Edit)
	b.ReportMetric(float64(f.World.First.NsPerOp)/1e3, "frame-first-µs")
	metric("frame-static", &f.World.Static)
	metric("frame-hover", f.World.Hover)
	metric("frame-edit", f.World.Edit)
	b.ReportMetric(float64(f.World.Batches), "batches")
	b.ReportMetric(float64(f.World.Vertices), "vertices")
}

// BenchmarkKitPages measures every kit card page's frames: its first layout, a static
// frame, a hover change and a text edit (pages.Frames). Each page runs once per
// sub-benchmark, since it times its frames itself.
func BenchmarkKitPages(b *testing.B) {
	dirs, err := filepath.Glob("kit-*")
	if err != nil {
		b.Fatal(err)
	}
	for _, dir := range dirs {
		b.Run(strings.TrimPrefix(dir, "kit-"), func(b *testing.B) {
			report(b, runPage(b, dir))
		})
	}
}

// BenchmarkInspectorFrames measures the frames of mocka, the inspector mockup.
func BenchmarkInspectorFrames(b *testing.B) {
	report(b, runPage(b, "mocka"))
}
