//go:build !js

package wgpu //nolint:testpackage // white-box: reaches the binding's unexported handles

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestTheLibraryIsLookedForWhereTaskWgpuPutsIt(t *testing.T) {
	t.Setenv(LibraryEnv, "/somewhere/libwgpu.dylib")
	got := candidatesFor("linux", "/games/survivors/survivors")
	if len(got) != 3 || got[0] != "/somewhere/libwgpu.dylib" || got[1] != "/games/survivors/libwgpu_native.so" {
		t.Fatalf("the env first, then beside the executable, then the source tree: %v", got)
	}
	want := filepath.Join("internal", "wgpu", "lib", runtime.GOOS+"-"+runtime.GOARCH, libraryName())
	if !strings.HasSuffix(got[2], want) {
		t.Fatalf("task wgpu's folder: %s, want …%s", got[2], want)
	}
}

func TestAMacAppFindsTheLibraryInItsFrameworks(t *testing.T) {
	t.Setenv(LibraryEnv, "")
	got := candidatesFor("darwin", "/Applications/Survivors.app/Contents/MacOS/survivors")
	if len(got) != 3 || got[0] != "/Applications/Survivors.app/Contents/MacOS/libwgpu_native.dylib" ||
		got[1] != "/Applications/Survivors.app/Contents/Frameworks/libwgpu_native.dylib" {
		t.Fatalf("beside the executable, then the app's Frameworks: %v", got)
	}
	if win := candidatesFor(
		"windows",
		`C:\Games\survivors.exe`,
	); len(win) != 2 ||
		!strings.HasSuffix(win[0], "wgpu_native.dll") {
		t.Fatalf("Windows looks beside the exe: %v", win)
	}
}
