package ember_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestEmberImportsOnlyItsOwnAndItsRequirements reads every Go file of the
// module, whatever its build tags: each import is the standard library,
// ember's own, or a module go.mod requires. A game built on ember is none
// of them, so the dependency goes one way, from the game to the engine.
func TestEmberImportsOnlyItsOwnAndItsRequirements(t *testing.T) {
	mod, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatal(err)
	}
	var self string
	var required []string
	block := false
	for _, line := range strings.Split(string(mod), "\n") {
		f := strings.Fields(line)
		switch {
		case len(f) == 0:
		case f[0] == "module":
			self = f[1]
		case f[0] == "require" && len(f) > 1 && f[1] == "(":
			block = true
		case f[0] == "require" && len(f) > 1:
			required = append(required, f[1])
		case f[0] == ")":
			block = false
		case block && !strings.HasPrefix(f[0], "//"):
			required = append(required, f[0])
		}
	}
	if self == "" || len(required) == 0 {
		t.Fatalf("go.mod read: module %q, %d requirements", self, len(required))
	}
	allowed := func(path string) bool {
		if first, _, _ := strings.Cut(path, "/"); !strings.Contains(first, ".") {
			return true // the standard library
		}
		for _, m := range append([]string{self}, required...) {
			if path == m || strings.HasPrefix(path, m+"/") {
				return true
			}
		}
		return false
	}
	files := token.NewFileSet()
	err = filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && path != "." && (strings.HasPrefix(d.Name(), ".") || d.Name() == "testdata") {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		f, err := parser.ParseFile(files, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range f.Imports {
			if p, _ := strconv.Unquote(imp.Path.Value); !allowed(p) {
				t.Errorf("%s imports %s, neither ember's nor a module go.mod requires", path, p)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
