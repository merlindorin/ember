package desktop //nolint:testpackage // tests the driver's unexported crash handling

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/merlindorin/ember/internal/errlog"
	"github.com/merlindorin/ember/internal/wm"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTheCrashLogSaysWhatWentWrongAndOnWhat(t *testing.T) {
	r := report{
		at:       time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC),
		reason:   "panic: boom",
		stack:    []byte("goroutine 1 [running]:\nmain.main()"),
		glfw:     "3.3.8 Win32",
		graphics: "WebGPU (wgpu-native v29.0.0.0, Direct3D 12)",
		vendor:   "NVIDIA",
		renderer: "NVIDIA GeForce GTX 1060",
	}
	text := r.String()
	for _, want := range []string{
		"2026-11-01T10:00:00Z", "panic: boom", runtime.Version(), runtime.GOOS + "/" + runtime.GOARCH,
		"GLFW:     3.3.8 Win32", "Graphics: WebGPU (wgpu-native v29.0.0.0, Direct3D 12)",
		"Vendor:   NVIDIA", "Renderer: NVIDIA GeForce GTX 1060", "goroutine 1 [running]",
	} {
		assert.Contains(t, text, want)
	}
	assert.Contains(t, report{reason: "x"}.String(), "Renderer: unknown", "what isn't known says so")
	path := filepath.Join(t.TempDir(), "new", "folder", "crash.log")
	require.NoError(t, writeCrash(path, r), "the folder is made")
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, text, string(got))
}

func TestTheCrashLogGoesInTheGamesFolder(t *testing.T) {
	assert.Equal(t, "/saves/crash.log", crashPath("/saves/crash.log", "Ember Survivors"), "the game's own path")
	config, err := os.UserConfigDir()
	require.NoError(t, err)
	assert.Equal(
		t,
		filepath.Join(config, "ember-survivors", "crash.log"),
		crashPath("", "Ember Survivors"),
		"else under the primary window's title",
	)
	assert.Equal(t, "ember", slug(""))
	assert.Equal(t, "my-game-2", slug("  My Game: 2! "))
}

func TestWithoutAnAdapterThePlayerIsToldWhichBackendsWereTried(t *testing.T) {
	msg := noAdapter([]string{"Direct3D 12", "Vulkan", "OpenGL"}, "wgpu: no adapter: No suitable adapter found")
	assert.True(
		t,
		strings.HasPrefix(
			msg,
			"Ember found no graphics adapter to draw with: WebGPU tried Direct3D 12, Vulkan or OpenGL.",
		),
		msg,
	)
	assert.Contains(t, msg, "Remote Desktop")
	assert.Contains(t, msg, "No suitable adapter found", "wgpu-native's own words follow")
	assert.True(
		t,
		strings.HasPrefix(
			noAdapter([]string{"Metal"}, ""),
			"Ember found no graphics adapter to draw with: WebGPU tried Metal.",
		),
	)
}

// failing is a hidden driver whose game is titled title, writing its
// crash log to path.
func failing(_ *testing.T, path string) *driver {
	w := world.New().AddPlugins(wm.Plugin{})
	w.AddEntities(platform.NewWindow(platform.Title("Ember Survivors")))
	return &driver{world: w, hidden: true, crashLog: path}
}

func TestAFailedStartIsLoggedAndStillReturned(t *testing.T) {
	path := filepath.Join(t.TempDir(), "crash.log")
	d := failing(t, path)
	err := d.guard(func() error { return errors.New(noAdapter([]string{"Vulkan", "OpenGL"}, "")) })
	require.Error(t, err)
	got, rerr := os.ReadFile(path)
	require.NoError(t, rerr)
	assert.Contains(t, string(got), "WebGPU tried Vulkan or OpenGL.")
}

func TestAPanicIsLoggedWithItsStackAndGoesOn(t *testing.T) {
	path := filepath.Join(t.TempDir(), "crash.log")
	d := failing(t, path)
	d.graphics, d.renderer = "WebGPU (wgpu-native v29.0.0.0, Metal)", "Apple M1"
	assert.PanicsWithValue(t, "boom", func() {
		_ = d.guard(func() error { panic("boom") })
	})
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	for _, want := range []string{"panic: boom", "Renderer: Apple M1", "crash_test.go"} {
		assert.Contains(t, string(got), want)
	}
}

func TestAGPUThatCantStartReachesTheCrashLogWithTheErrorsBeforeIt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "crash.log")
	d := failing(t, path)
	cause := errors.New(
		`gpu: the 2D renderer can't start on this GPU: lights: the "light shadows" pipeline drawing into r32float: boom`,
	)
	assert.Panics(t, func() {
		_ = d.guard(func() error {
			errlog.Print("gpu: webgpu: wgpu: an earlier error")
			panic(cause)
		})
	})
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	for _, want := range []string{"panic: " + cause.Error(), "Errors before it:\n  gpu: webgpu: wgpu: an earlier error"} {
		assert.Contains(t, string(got), want)
	}
	errs, err := os.ReadFile(filepath.Join(dir, "errors.log"))
	require.NoError(t, err, "every error line also lands beside the crash log as it comes")
	assert.Equal(t, "gpu: webgpu: wgpu: an earlier error\n", string(errs))

	require.NoError(t, failing(t, path).guard(func() error { return nil }))
	assert.NoFileExists(t, filepath.Join(dir, "errors.log"), "a run without errors clears the last run's")
}

func TestACleanRunLeavesTheLastCrashLogAsItWas(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "crash.log")
	require.NoError(t, failing(t, path).guard(func() error { return nil }))
	assert.NoFileExists(t, path, "no empty log")

	require.NoError(t, os.WriteFile(path, []byte("last crash"), 0o644))
	require.NoError(t, failing(t, path).guard(func() error { return nil }))
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "last crash", string(got))
	assert.NoFileExists(t, path+".last")
}

func TestAPanicOutsideTheFrameIsLoggedByTheRuntime(t *testing.T) {
	if path := os.Getenv("EMBER_TEST_CRASH_LOG"); path != "" {
		_ = failing(t, path).guard(func() error {
			go panic("boom in a goroutine")
			time.Sleep(10 * time.Second)
			return nil
		})
		return
	}
	path := filepath.Join(t.TempDir(), "crash.log")
	require.NoError(t, os.WriteFile(path, []byte("last crash"), 0o644))
	cmd := exec.Command(os.Args[0], "-test.run=^TestAPanicOutsideTheFrameIsLoggedByTheRuntime$")
	cmd.Env = append(os.Environ(), "EMBER_TEST_CRASH_LOG="+path)
	require.Error(t, cmd.Run(), "the process dies")
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(got), "panic: boom in a goroutine")
	assert.NotContains(t, string(got), "last crash", "the new crash replaces the last one")
}
