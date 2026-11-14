package golden //nolint:testpackage // white-box: checks which references Update rewrites

import (
	"context"
	"errors"
	"image"
	"image/color"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// elsewhere moves the player's home and config folder to a fresh folder for
// the test, Go's own folders staying where they are.
func elsewhere(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("go", "env", "GOCACHE", "GOMODCACHE", "GOPATH", "GOENV").Output()
	if err != nil {
		t.Fatal(err)
	}
	for i, v := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		t.Setenv([]string{"GOCACHE", "GOMODCACHE", "GOPATH", "GOENV"}[i], strings.TrimSpace(v))
	}
	home := t.TempDir()
	for _, k := range []string{"HOME", "XDG_CONFIG_HOME", "AppData"} {
		t.Setenv(k, home)
	}
	return home
}

func TestASceneThatFailsLeavesNothingInThePlayersConfigFolder(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs a program")
	}
	home, dir := elsewhere(t), t.TempDir()
	config, err := os.UserConfigDir()
	if err != nil || !strings.HasPrefix(config, home) {
		t.Fatalf("the config folder is not under the test's home: %s, %v", config, err)
	}
	s := Scene{Name: "fails", Pkg: "golden/testdata/fails", Frame: 5}
	out, _, err := run(context.Background(), "..", s, filepath.Join(dir, "fails.png"), dir)
	if err == nil {
		t.Fatalf("the scene fails: %s", out)
	}
	log := filepath.Join(dir, "crash", "golden-fails", "crash.log")
	if _, err = os.Stat(log); err != nil {
		t.Errorf("its crash log is in the test's folder: %v: %s", err, out)
	}
	_ = filepath.WalkDir(home, func(path string, d fs.DirEntry, _ error) error {
		if d != nil && !d.IsDir() && strings.HasSuffix(path, ".log") {
			t.Errorf("a log under the player's home: %s", path)
		}
		return nil
	})
}

func TestASceneStillRunningAtItsDeadlineIsKilledAndWhatItWroteKept(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs a program")
	}
	dir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	s := Scene{Name: "stalls", Pkg: "golden/testdata/stalls", Frame: 5}
	began := time.Now()
	out, took, err := run(ctx, "..", s, filepath.Join(dir, "stalls.png"), dir)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("the scene is stopped at its deadline: %v: %s", err, out)
	}
	if !strings.Contains(string(out), "frame 3") {
		t.Errorf("what it wrote before is kept: %q", out)
	}
	if took < time.Second || took > time.Minute || time.Since(began) > 5*time.Minute {
		t.Errorf("it ran for its 2 s, its build apart: %s", took)
	}
}

func TestASceneHasTheTestsTimeLessAMarginOrLessWhenTold(t *testing.T) {
	for _, c := range []struct {
		env  string
		left time.Duration
		want time.Duration
		bad  bool
	}{
		{"", 0, 0, false},
		{"", 30 * time.Minute, 30*time.Minute - margin, false},
		{"", margin / 2, margin / 4, false},
		{"10m", 0, 10 * time.Minute, false},
		{"10m", 30 * time.Minute, 10 * time.Minute, false},
		{"10m", 5 * time.Minute, 5*time.Minute - margin, false},
		{"soon", time.Minute, 0, true},
		{"-1s", time.Minute, 0, true},
	} {
		got, err := limit(c.env, c.left)
		if got != c.want || (err != nil) != c.bad {
			t.Errorf("%s=%q with %s left: %s, %v; want %s, bad %v", timeoutEnv, c.env, c.left, got, err, c.want, c.bad)
		}
	}
}

func TestASceneRunsHiddenWithItsFilesInTheTestsFolder(t *testing.T) {
	dir := t.TempDir()
	s := Scene{Name: "title", Frame: 60, Saves: "GAME_DIR", Env: []string{"EMBER_CAPTURE_KEYS=enter:5-7"}}
	env := s.environ([]string{"HOME=/h"}, filepath.Join(dir, "title.png"), dir)
	for _, want := range []string{
		"HOME=/h", "EMBER_HIDDEN=1", "EMBER_SCALE=2", "EMBER_CAPTURE=" + filepath.Join(dir, "title.png"),
		"EMBER_CAPTURE_FRAME=60", "GAME_DIR=" + filepath.Join(dir, "saves"), "EMBER_CAPTURE_KEYS=enter:5-7",
		"EMBER_CRASH_DIR=" + filepath.Join(dir, "crash"),
	} {
		if !slices.Contains(env, want) {
			t.Errorf("no %s in %v", want, env)
		}
	}
	own := Scene{Env: []string{"EMBER_CRASH_DIR=/elsewhere"}}.environ(nil, "x.png", dir)
	if own[len(own)-1] != "EMBER_CRASH_DIR=/elsewhere" {
		t.Errorf("a scene's own Env comes last, so it wins: %v", own)
	}
}

func TestUpdateRewritesOnlyWhatFailsOrIsNamed(t *testing.T) {
	for _, c := range []struct {
		run, name      string
		missing, fails bool
		want           bool
	}{
		{"Golden", "09-window", false, false, false},
		{"Golden", "09-window", false, true, true},
		{"Golden", "new-scene", true, false, true},
		{"Golden/look3d", "look3d", false, false, true},
		{"Golden/look3d", "look3d-post", false, false, false},
		{"Golden/look3d.*", "look3d-post", false, false, true},
		{"Golden/(", "look3d", false, false, false},
	} {
		if got := rewrite(c.run, c.name, c.missing, c.fails); got != c.want {
			t.Errorf("-run %q, %s (missing %v, fails %v): rewrite %v, want %v", c.run, c.name, c.missing, c.fails, got, c.want)
		}
	}
}

func TestAPictureIsHalvedToPointsByAveraging(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 4, 2))
	for x, v := range []uint8{0, 100, 200, 201} {
		for y := range 2 {
			img.SetNRGBA(x, y, color.NRGBA{v, v, v, 255})
		}
	}
	half := ToPoints(t, img)
	if half.Bounds().Dx() != 2 || half.Bounds().Dy() != 1 {
		t.Fatalf("half the size: %v", half.Bounds())
	}
	if got := half.NRGBAAt(0, 0).R; got != 50 {
		t.Fatalf("each point the mean of its 2×2 pixels: %d", got)
	}
	if got := half.NRGBAAt(1, 0).R; got != 201 {
		t.Fatalf("rounded to nearest: %d", got)
	}
}

func TestAPixelMatchesWithinOneLevelAndIsMarkedPast(t *testing.T) {
	want := image.NewNRGBA(image.Rect(0, 0, 3, 1))
	got := image.NewNRGBA(image.Rect(0, 0, 3, 1))
	for x, d := range []uint8{0, Levels, Levels + 3} {
		want.SetNRGBA(x, 0, color.NRGBA{100, 100, 100, 255})
		got.SetNRGBA(x, 0, color.NRGBA{100, 100 + d, 100, 255})
	}
	d, marks := Compare(t, want, got)
	if d.Over != 1 || d.Most != Levels+3 || d.Total != 3 {
		t.Fatalf("one pixel past the levels, by %d: %+v", Levels+3, d)
	}
	if marks.NRGBAAt(2, 0) != (color.NRGBA{255, 0, 0, 255}) || marks.NRGBAAt(1, 0).R == 255 {
		t.Fatal("the pixel past them is marked red, the others dimmed")
	}
}

func TestASceneLikeItsReferencePassesAndAPixelOffCounts(t *testing.T) {
	ref := filepath.Join(t.TempDir(), "scene.png")
	pic := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	Save(t, ref, pic)
	check(t, Scene{Name: "scene"}, ref, pic, 0) // the same picture passes: it would fail t
	off := image.NewNRGBA(pic.Bounds())
	off.SetNRGBA(0, 0, color.NRGBA{200, 0, 0, 255})
	if d, _ := Compare(t, Load(t, ref), off); d.Over != 1 {
		t.Fatalf("a pixel off is counted: %+v", d)
	}
}
