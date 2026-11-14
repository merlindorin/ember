package desktop

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"time"
	"unicode"
)

// report is what the crash log says of a failure: what went wrong, the
// stack of a panic, and the machine as far as it is known.
type report struct {
	at       time.Time
	reason   string
	stack    []byte
	glfw     string
	graphics string
	vendor   string
	renderer string
	// errors are the lines the engine printed before it, oldest first.
	errors []string
}

func (r report) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Ember crashed at %s\n\n%s\n\n", r.at.Format(time.RFC3339), r.reason)
	fmt.Fprintf(&b, "Go:       %s %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	for _, line := range [][2]string{
		{"GLFW", r.glfw}, {"Graphics", r.graphics}, {"Vendor", r.vendor}, {"Renderer", r.renderer},
	} {
		if line[1] == "" {
			line[1] = "unknown"
		}
		fmt.Fprintf(&b, "%-9s %s\n", line[0]+":", line[1])
	}
	if len(r.errors) > 0 {
		b.WriteString("\nErrors before it:\n")
		for _, line := range r.errors {
			fmt.Fprintf(&b, "  %s\n", line)
		}
	}
	if len(r.stack) > 0 {
		fmt.Fprintf(&b, "\n%s", r.stack)
	}
	return b.String()
}

// writeCrash writes r to path, making its folder.
//
//nolint:gosec // the game's own crash log, under its config folder
func writeCrash(path string, r report) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(r.String()), 0o600)
}

// watchFatal makes the Go runtime write a fatal error or an unrecovered
// panic to path as well as stderr, which a windowed Windows build has not.
// The last crash's log waits beside it while the game runs; stop ends the
// watch and puts that log back when nothing was written, so a clean run
// neither leaves an empty log nor loses the last one. The log has to be
// open before the runtime needs it, so its folder is made at once; a run
// that made the folder and leaves nothing in it removes it.
//
//nolint:gosec // the game's own crash log, under its config folder
func watchFatal(path string) func() {
	dir := filepath.Dir(path)
	_, err := os.Stat(dir)
	made := os.IsNotExist(err)
	if err = os.MkdirAll(dir, 0o755); err != nil {
		return func() {}
	}
	last := path + ".last"
	kept := os.Rename(path, last) == nil
	f, err := os.Create(path)
	if err == nil {
		err = debug.SetCrashOutput(f, debug.CrashOptions{})
	}
	if err != nil {
		if f != nil {
			f.Close()
		}
		if kept {
			_ = os.Rename(last, path) // best effort: the last log stays as .last
		}
		if made {
			os.Remove(dir)
		}
		return func() {}
	}
	return func() {
		_ = debug.SetCrashOutput(nil, debug.CrashOptions{}) // nil only stops it
		f.Close()
		info, statErr := os.Stat(path)
		switch {
		case statErr == nil && info.Size() == 0 && kept:
			_ = os.Rename(last, path) // best effort, as above
		case statErr == nil && info.Size() == 0:
			os.Remove(path)
		case kept:
			os.Remove(last)
		}
		if made {
			os.Remove(dir) // refused while the folder holds anything
		}
	}
}

// crashPath is where a crash log goes: path when the game gave one, else
// crash.log in the user's config folder under the slug of title, the
// primary window's, such as ember-survivors for "Ember Survivors"; the
// temporary folder when there is no config folder. EMBER_CRASH_DIR names
// a folder that stands in for the config folder, for the tests and
// captures that run a program which may fail.
func crashPath(path, title string) string {
	if path != "" {
		return path
	}
	dir := os.Getenv("EMBER_CRASH_DIR")
	if dir == "" {
		var err error
		if dir, err = os.UserConfigDir(); err != nil {
			dir = os.TempDir()
		}
	}
	return filepath.Join(dir, slug(title), "crash.log")
}

// slug is title in lower case, words joined by hyphens: "ember" for none.
func slug(title string) string {
	words := strings.FieldsFunc(
		strings.ToLower(title),
		func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) },
	)
	if len(words) == 0 {
		return "ember"
	}
	return strings.Join(words, "-")
}

// noAdapter is the reason given when WebGPU finds no adapter to draw
// with: the backends it tried, then cause, wgpu-native's own words.
func noAdapter(backends []string, cause string) string {
	tried := strings.Join(backends, ", ")
	if i := strings.LastIndex(tried, ", "); i >= 0 {
		tried = tried[:i] + " or " + tried[i+2:]
	}
	msg := fmt.Sprintf(
		"Ember found no graphics adapter to draw with: WebGPU tried %s.\n\n"+
			"Update the graphics driver, or run it on the machine itself "+
			"rather than through Remote Desktop or a virtual machine.",
		tried,
	)
	if cause != "" {
		msg += "\n\n" + cause
	}
	return msg
}
