package errlog //nolint:testpackage // tests the unexported line limit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLinesPrintedGoToTheFileAsTheyComeAndAreKept(t *testing.T) {
	path := filepath.Join(t.TempDir(), "game", "errors.log")
	stop := To(path)
	Print("gpu: webgpu: the shader failed")
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "gpu: webgpu: the shader failed\n" {
		t.Fatalf("written at once, its folder made: %q %v", got, err)
	}
	Print("wgpu: second")
	stop()
	got, _ = os.ReadFile(path)
	if !strings.HasSuffix(string(got), "wgpu: second\n") {
		t.Fatalf("each line appended: %q", got)
	}
	if l := Lines(); len(l) != 2 || l[1] != "wgpu: second" {
		t.Fatalf("kept for the crash log: %q", l)
	}
}

func TestARunWithoutErrorsLeavesNoFileAndClearsTheLastOne(t *testing.T) {
	path := filepath.Join(t.TempDir(), "errors.log")
	if err := os.WriteFile(path, []byte("last run"), 0o644); err != nil {
		t.Fatal(err)
	}
	To(path)()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("no file: %v", err)
	}
}

func TestOnlyTheLastLinesAreKept(t *testing.T) {
	defer To(filepath.Join(t.TempDir(), "errors.log"))()
	for range keep + 10 {
		Print("x")
	}
	if n := len(Lines()); n != keep {
		t.Fatalf("%d kept", n)
	}
}
