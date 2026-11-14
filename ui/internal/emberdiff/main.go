// Command emberdiff captures every kit page, every page of the UI kit
// example and the tileset tool's start card in ember, at a base revision
// and in the working tree, and reports each picture that changed. Chrome's
// comparisons can't see a change both engines share, such as a class a
// rename missed; this compares ember with itself.
//
//	go run ./ui/internal/emberdiff -base wip -out /tmp/ember-diff
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"image"
	_ "image/png"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

func main() { os.Exit(compare()) }

// compare captures both trees and lists what changed; 1 when anything did,
// after the base worktree is removed.
func compare() int {
	ctx := context.Background()
	base := flag.String("base", "wip", "the revision to compare the working tree with")
	out := flag.String("out", filepath.Join(os.TempDir(), "ember-diff"), "where the pictures go, base/ and head/")
	flag.Parse()
	head, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	tree := filepath.Join(*out, "base-tree")
	_ = exec.CommandContext(ctx, "git", "worktree", "remove", "--force", tree).Run()
	run(ctx, "", "git", "worktree", "add", "--detach", tree, *base)
	defer func() { _ = exec.CommandContext(ctx, "git", "worktree", "remove", "--force", tree).Run() }()
	baseDir := filepath.Join(tree, "ember")
	if _, statErr := os.Stat(baseDir); statErr != nil {
		baseDir = tree
	}
	shots := map[string][2]string{}
	for side, dir := range map[string]string{"base": baseDir, "head": head} {
		for name, path := range capture(ctx, dir, filepath.Join(*out, side)) {
			s := shots[name]
			if side == "base" {
				s[0] = path
			} else {
				s[1] = path
			}
			shots[name] = s
		}
	}
	changed := 0
	for name, s := range shots {
		switch n := differ(s[0], s[1]); {
		case n < 0:
			fmt.Fprintf(os.Stdout, "%s: only on one side\n", name)
			changed++
		case n > 0:
			fmt.Fprintf(os.Stdout, "%s: %d pixels differ\n", name, n)
			changed++
		}
	}
	fmt.Fprintf(os.Stdout, "%d pictures, %d changed (%s)\n", len(shots), changed, *out)
	if len(shots) == 0 {
		fmt.Fprintln(os.Stderr, "nothing compared: run task wgpu, or set EMBER_WGPU_LIB, in a worktree")
	}
	return verdict(len(shots), changed)
}

// verdict is the exit code of a comparison: 1 when a picture changed or
// none were taken, which is what a worktree without the wgpu library gives.
func verdict(pictures, changed int) int {
	if pictures == 0 || changed > 0 {
		return 1
	}
	return 0
}

// capture takes every picture in the ember module at dir into out.
func capture(ctx context.Context, dir, out string) map[string]string {
	if err := os.MkdirAll(out, 0o755); err != nil {
		log.Fatal(err)
	}
	shots := map[string]string{}
	shoot := func(name string, env []string, bin string, args ...string) {
		path := filepath.Join(out, name+".png")
		cmd := exec.CommandContext(ctx, bin, args...)
		cmd.Dir = dir
		cmd.Env = append(append(os.Environ(), "EMBER_HIDDEN=1", "EMBER_CAPTURE_FRAME=10", "EMBER_CAPTURE="+path,
			"EMBER_CRASH_DIR="+filepath.Join(out, "crash")), env...)
		if msg, err := cmd.CombinedOutput(); err != nil {
			log.Printf("%s: %v: %s", name, err, msg)
			return
		}
		shots[name] = path
	}
	for _, pkg := range kitPages(dir) {
		shoot(filepath.Base(pkg), nil, build(ctx, dir, out, pkg))
	}
	kit := build(ctx, dir, out, "./examples/92-ui-kit")
	for _, e := range elements(ctx, dir, kit) {
		shoot("92-"+e, nil, kit, "-element", e)
	}
	tiles := build(ctx, dir, out, "./tools/tileset")
	state, err := os.MkdirTemp("", "ember-diff-tileset")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(state)
	shoot("tileset-start", []string{"EMBER_TILESET_DIR=" + state, "EMBER_CAPTURE_FRAME=15"}, tiles)
	return shots
}

// kitPages are the kit's page packages in dir, under ui or, before the
// rename (decision 0059), uiv2, so a base and a tree on either side of it
// pair their pictures by page name.
func kitPages(dir string) []string {
	var pkgs []string
	for _, root := range []string{"ui", "uiv2"} {
		pages, _ := filepath.Glob(filepath.Join(dir, root, "internal", "pages", "kit-*"))
		for _, p := range pages {
			pkgs = append(pkgs, "./"+root+"/internal/pages/"+filepath.Base(p))
		}
	}
	return pkgs
}

// build compiles pkg in dir into out and returns the binary.
func build(ctx context.Context, dir, out, pkg string) string {
	bin := filepath.Join(out, "bin", strings.ReplaceAll(strings.TrimPrefix(pkg, "./"), "/", "-"))
	run(ctx, dir, "go", "build", "-o", bin, pkg)
	return bin
}

var oneOf = regexp.MustCompile(`try one of (.*)`)

// elements are the UI kit example's pages, read from the list it gives
// for a page it doesn't know.
func elements(ctx context.Context, dir, bin string) []string {
	cmd := exec.CommandContext(ctx, bin, "-element", "-")
	cmd.Dir = dir
	msg, _ := cmd.CombinedOutput()
	m := oneOf.FindSubmatch(msg)
	if m == nil {
		log.Fatalf("92-ui-kit lists no elements: %s", msg)
	}
	return strings.Split(string(bytes.TrimSpace(m[1])), ", ")
}

func run(ctx context.Context, dir string, name string, args ...string) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	if msg, err := cmd.CombinedOutput(); err != nil {
		log.Fatalf("%s %s: %v: %s", name, strings.Join(args, " "), err, msg)
	}
}

// differ is how many pixels of the pictures at a and b differ, -1 when one
// is missing or their sizes do.
func differ(a, b string) int {
	ia, ib := load(a), load(b)
	if ia == nil || ib == nil || ia.Bounds() != ib.Bounds() {
		return -1
	}
	n := 0
	r := ia.Bounds()
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			r1, g1, b1, _ := ia.At(x, y).RGBA()
			r2, g2, b2, _ := ib.At(x, y).RGBA()
			if r1 != r2 || g1 != g2 || b1 != b2 {
				n++
			}
		}
	}
	return n
}

func load(path string) image.Image {
	if path == "" {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil
	}
	return img
}
