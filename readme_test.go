package ember_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
)

var readmeGo = regexp.MustCompile("(?s)```go\n(.*?)```")

// TestTheReadmesCodeBuilds builds every Go block of README.md, each a main
// package, so the first code a developer reads is today's API. The blocks are
// built inside the module, in a folder go's ./... patterns skip.
func TestTheReadmesCodeBuilds(t *testing.T) {
	readme, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatal(err)
	}
	blocks := readmeGo.FindAllSubmatch(readme, -1)
	if len(blocks) == 0 {
		t.Fatal("README.md shows no Go code")
	}
	dir, err := os.MkdirTemp(".", "_readme-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	for i, block := range blocks {
		pkg := filepath.Join(dir, strconv.Itoa(i+1))
		if err = os.MkdirAll(pkg, 0o750); err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(pkg, "main.go"), block[1], 0o600); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command("go", "vet", "./"+filepath.ToSlash(pkg))
		if out, vetErr := cmd.CombinedOutput(); vetErr != nil {
			t.Errorf("README.md's Go block %d doesn't build: %v\n%s", i+1, vetErr, out)
		}
	}
}
