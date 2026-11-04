package ember_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"unicode"
)

const module = "github.com/merlindorin/ember"

// TestTheEngineExposesNoResources keeps decision 0015: engine packages
// share state through values the game holds, entities and private
// registries, never through World resources of their exported types.
// Games, the examples among them, and tests may use resources for their own
// types.
func TestTheEngineExposesNoResources(t *testing.T) {
	fset := token.NewFileSet()
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch path {
			case "world", "internal", "examples", "third_party", "docs", ".hugo", "public":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		for _, bad := range engineResources(f) {
			t.Errorf(
				"%s: a World resource of an engine type: hold it as a value, an entity or a private registry instead",
				fset.Position(bad.Pos()),
			)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// engineResources are the calls in f that put or read a World resource of
// an exported engine type.
func engineResources(f *ast.File) []*ast.CallExpr {
	resourceCalls := map[string]bool{"Resource": true, "TryResource": true, "ResourceOrInit": true, "SetResource": true}
	imports := map[string]string{}
	for _, imp := range f.Imports {
		p, _ := strconv.Unquote(imp.Path.Value)
		name := p[strings.LastIndex(p, "/")+1:]
		if imp.Name != nil {
			name = imp.Name.Name
		}
		imports[name] = p
	}
	engine := func(t ast.Expr) bool {
		switch t := t.(type) {
		case *ast.StarExpr:
			return engineType(t.X, imports)
		default:
			return engineType(t, imports)
		}
	}
	var out []*ast.CallExpr
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		fn, typ := call.Fun, ast.Expr(nil)
		if ix, isIndex := fn.(*ast.IndexExpr); isIndex {
			fn, typ = ix.X, ix.Index
		}
		sel, ok := fn.(*ast.SelectorExpr)
		if !ok || !resourceCalls[sel.Sel.Name] || imports[identName(sel.X)] != module+"/world" {
			return true
		}
		if typ == nil && sel.Sel.Name == "SetResource" && len(call.Args) == 2 {
			typ = valueType(call.Args[1])
		}
		if typ == nil || engine(typ) {
			out = append(out, call)
		}
		return true
	})
	return out
}

// engineType reports whether t names an exported type of an engine package:
// this one, or another package of the module outside the examples.
func engineType(t ast.Expr, imports map[string]string) bool {
	switch t := t.(type) {
	case *ast.Ident:
		return unicode.IsUpper([]rune(t.Name)[0])
	case *ast.SelectorExpr:
		p := imports[identName(t.X)]
		return strings.HasPrefix(p, module) && !strings.HasPrefix(p, module+"/examples")
	}
	return true
}

// valueType is the type of &T{…} or T{…}, nil when it cannot be read.
func valueType(v ast.Expr) ast.Expr {
	if u, ok := v.(*ast.UnaryExpr); ok && u.Op == token.AND {
		v = u.X
	}
	if lit, ok := v.(*ast.CompositeLit); ok {
		return lit.Type
	}
	return nil
}

func identName(e ast.Expr) string {
	if id, ok := e.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}
