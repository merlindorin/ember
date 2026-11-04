package ember_test

import (
	"bytes"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
)

var goBlock = regexp.MustCompile("(?s)```go\n(.*?)```")

// TestTheReadmesCodeBuilds builds every Go block of README.md, each a main
// package, so the first code a developer reads is today's API.
func TestTheReadmesCodeBuilds(t *testing.T) {
	readme, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatal(err)
	}
	blocks := programs(readme, false)
	if len(blocks) == 0 {
		t.Fatal("README.md shows no Go code")
	}
	vetPrograms(t, "README.md", blocks)
}

// TestTheSkillsCodeBuilds builds every Go block of the ember skill that is a
// whole program (it starts with package main), so what agents copy builds.
// Shorter fragments are left alone.
func TestTheSkillsCodeBuilds(t *testing.T) {
	const skill = ".claude/skills/ember"
	found := false
	err := filepath.WalkDir(skill, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() || filepath.Ext(path) != ".md" {
			return walkErr
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if blocks := programs(body, true); len(blocks) > 0 {
			found = true
			vetPrograms(t, path, blocks)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatalf("%s shows no whole program", skill)
	}
}

// programs returns the Go blocks of a Markdown file, only those that are a
// main package when mainOnly is set.
func programs(md []byte, mainOnly bool) [][]byte {
	var out [][]byte
	for _, m := range goBlock.FindAllSubmatch(md, -1) {
		if mainOnly && !bytes.HasPrefix(bytes.TrimSpace(m[1]), []byte("package main")) {
			continue
		}
		out = append(out, m[1])
	}
	return out
}

// vetPrograms vets each block as its own main package inside the module, in a
// folder go's ./... patterns skip.
func vetPrograms(t *testing.T, from string, blocks [][]byte) {
	t.Helper()
	dir, err := os.MkdirTemp(".", "_snippets-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	for i, block := range blocks {
		pkg := filepath.Join(dir, strconv.Itoa(i+1))
		if err = os.MkdirAll(pkg, 0o750); err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(pkg, "main.go"), block, 0o600); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command("go", "vet", "./"+filepath.ToSlash(pkg))
		if out, vetErr := cmd.CombinedOutput(); vetErr != nil {
			t.Errorf("%s: Go block %d doesn't build: %v\n%s", from, i+1, vetErr, out)
		}
	}
}
