// Package golden checks what a game or an example draws against reference
// pictures, for tests on a machine with a GPU: each scene runs hidden, is
// captured at a fixed frame with scripted input (EMBER_CAPTURE and its
// settings), halved from pixels to points and compared with its PNG within
// one level a channel. The engine's own renderer test uses it, and a game
// can hold its screens to it the same way.
//
//	func TestGolden(t *testing.T) {
//		golden.Run(t, ".", "testdata/golden", []golden.Scene{{Name: "title", Pkg: ".", Frame: 60}})
//	}
//
// -update writes the references from the current renderer, of only the
// scenes that fail or that -run names whole (Golden/title), so the others
// keep theirs byte for byte. EMBER_GOLDEN_SHARE lets that share of a scene's
// pixels be over the levels, for a software adapter on a CI runner;
// EMBER_GOLDEN_REPORT names a Markdown file each comparison appends its row
// to; with EMBER_GPU_OPTIONAL=1 a machine without a GPU skips the scenes.
// A scene that fails writes its crash log in the test's own folder
// (EMBER_CRASH_DIR), never in the player's config folder.
package golden

import (
	"context"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// Update is -update: rewrite the references from the current renderer.
//
//nolint:gochecknoglobals // the -update flag go test passes
var Update = flag.Bool("update", false, "rewrite the golden references from the current renderer")

// Levels is how far apart a channel may be and the pixel still match.
const Levels = 1

const (
	shareEnv    = "EMBER_GOLDEN_SHARE"
	reportEnv   = "EMBER_GOLDEN_REPORT"
	optionalEnv = "EMBER_GPU_OPTIONAL"
)

// Scene is one picture to hold: the package Pkg run from the module's root
// (a path such as "examples/19-sprites", or "." for the root's own),
// captured at Frame with Env added (EMBER_CAPTURE_KEYS and the like), its
// references Name.png, then Name-2.png and on for more windows. Edges is
// the share of its pixels that may be over the levels, for multisampled
// 3D edges; Saves names an environment variable Run points at a fresh
// folder, for a game that keeps saves.
type Scene struct {
	Name, Pkg string
	Frame     int
	Env       []string
	Edges     float64
	Saves     string
}

// Run captures each scene in a subtest of its name, from root, and checks
// every window's picture against its reference in refs.
func Run(t *testing.T, root, refs string, scenes []Scene) {
	t.Helper()
	if *Update && Share(t) > 0 {
		t.Fatalf("-update with %s set: the references come from a real GPU only", shareEnv)
	}
	for _, s := range scenes {
		t.Run(s.Name, func(t *testing.T) {
			dir := t.TempDir()
			shot := filepath.Join(dir, s.Name+".png")
			capture(t, root, s, shot, dir)
			for n := 1; ; n++ {
				got := windowShot(shot, n)
				if _, err := os.Stat(got); err != nil {
					if n == 1 {
						t.Fatal("no capture")
					}
					return
				}
				check(t, s, windowShot(filepath.Join(refs, s.Name+".png"), n), ToPoints(t, Load(t, got)))
			}
		})
	}
}

// check holds half, a capture in points, to ref.
func check(t *testing.T, s Scene, ref string, half *image.NRGBA) {
	t.Helper()
	if _, err := os.Stat(ref); err != nil {
		if *Update {
			Save(t, ref, half)
			return
		}
		t.Fatalf("no reference %s: -update makes it", ref)
	}
	d, marks := Compare(t, Load(t, ref), half)
	allowed := int(s.Edges * float64(d.Total))
	if share := Share(t); share > 0 {
		allowed = max(allowed, int(share*float64(d.Total)))
	}
	t.Logf("%s: %d of %d pixels over %d levels, at most %d off", filepath.Base(ref), d.Over, d.Total, Levels, d.Most)
	Report(t, fmt.Sprintf("| %s | %d of %d | %d | %d |", filepath.Base(ref), d.Over, d.Total, d.Most, allowed))
	if *Update {
		if rewrite(flag.Lookup("test.run").Value.String(), s.Name, false, d.Over > allowed) {
			Save(t, ref, half)
		}
		return
	}
	if d.Over > allowed {
		diff := filepath.Join(os.TempDir(), "golden-"+filepath.Base(ref))
		Save(t, diff, marks)
		t.Errorf("%s: %d of %d pixels differ by more than %d levels (at most %d, %d allowed); where: %s",
			filepath.Base(ref), d.Over, d.Total, Levels, d.Most, allowed, diff)
	}
}

// environ is base with the capture's settings, then the scene's own: its
// saves in dir, and its crash log there too (EMBER_CRASH_DIR), so a scene
// that fails leaves nothing in the player's config folder.
func (s Scene) environ(base []string, shot, dir string) []string {
	// EMBER_SCALE draws at 2 pixels a point on every screen, as the
	// references were made, a 1× CI screen too.
	env := append(
		append([]string{}, base...),
		"EMBER_HIDDEN=1",
		"EMBER_SCALE=2",
		"EMBER_CAPTURE="+shot,
		fmt.Sprintf("EMBER_CAPTURE_FRAME=%d", s.Frame),
		"EMBER_CRASH_DIR="+filepath.Join(dir, "crash"),
	)
	if s.Saves != "" {
		env = append(env, s.Saves+"="+filepath.Join(dir, "saves"))
	}
	return append(env, s.Env...)
}

// run runs s hidden from root, its frame s.Frame captured into shot and
// its own files kept in dir, and gives what it printed.
func run(root string, s Scene, shot, dir string) ([]byte, error) {
	pkg := s.Pkg
	if pkg != "." {
		pkg = "./" + pkg
	}
	cmd := exec.CommandContext(context.Background(), "go", "run", pkg)
	cmd.Dir = root
	cmd.Env = s.environ(os.Environ(), shot, dir)
	return cmd.CombinedOutput()
}

// capture runs s hidden from root and captures its frame s.Frame into path.
func capture(t *testing.T, root string, s Scene, path, dir string) {
	t.Helper()
	if out, err := run(root, s, path, dir); err != nil {
		if _, statErr := os.Stat(path); statErr != nil {
			if os.Getenv(optionalEnv) == "1" && strings.Contains(string(out), "no graphics adapter") {
				t.Skipf("no GPU adapter here, and %s=1: %s", optionalEnv, out)
			}
			t.Fatalf("%v: %s", err, out)
		}
	}
}

// rewrite reports whether -update rewrites scene name's reference: when
// it has none, when the capture fails the comparison, or when run, the
// -run pattern, names the scene itself after its first "/", whole. A
// scene within tolerance is left as it is, so a refresh never churns the
// references of scenes it didn't change.
func rewrite(run, name string, missing, fails bool) bool {
	if missing || fails {
		return true
	}
	_, sub, ok := strings.Cut(run, "/")
	if !ok || sub == "" {
		return false
	}
	re, err := regexp.Compile("^(?:" + sub + ")$")
	return err == nil && re.MatchString(name)
}

// Share is EMBER_GOLDEN_SHARE: the share of a scene's pixels another
// renderer than the references' may have over the levels, 0 unset.
func Share(t *testing.T) float64 {
	t.Helper()
	v := os.Getenv(shareEnv)
	if v == "" {
		return 0
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil || f < 0 || f > 1 {
		t.Fatalf("%s=%q: a share from 0 to 1", shareEnv, v)
	}
	return f
}

// Report appends row to the EMBER_GOLDEN_REPORT file, its table header
// first when the file is empty: a CI job's summary.
func Report(t *testing.T, row string) {
	t.Helper()
	path := os.Getenv(reportEnv)
	if path == "" {
		return
	}
	//nolint:gosec // the CI job names its own summary file
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("%s: %v", reportEnv, err)
	}
	defer f.Close()
	if info, statErr := f.Stat(); statErr == nil && info.Size() == 0 {
		fmt.Fprintln(f, "| Picture | Pixels over | Most off | Allowed |\n|---|---|---|---|")
	}
	fmt.Fprintln(f, row)
}

// windowShot is window n's picture: path for the first, then path-2.png
// and on, as EMBER_CAPTURE names them.
func windowShot(path string, n int) string {
	if n == 1 {
		return path
	}
	return strings.TrimSuffix(path, ".png") + fmt.Sprintf("-%d.png", n)
}

// ToPoints averages each 2×2 block of img, from pixels to points.
func ToPoints(t *testing.T, img image.Image) *image.NRGBA {
	t.Helper()
	b := img.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, b.Dx()/2, b.Dy()/2))
	for y := range b.Dy() / 2 {
		for x := range b.Dx() / 2 {
			var s [3]int
			for _, d := range [4][2]int{{0, 0}, {1, 0}, {0, 1}, {1, 1}} {
				c, _ := color.NRGBAModel.Convert(img.At(b.Min.X+2*x+d[0], b.Min.Y+2*y+d[1])).(color.NRGBA) // as its model makes
				s[0] += int(c.R)
				s[1] += int(c.G)
				s[2] += int(c.B)
			}
			//nolint:gosec // the mean of four bytes is a byte
			out.SetNRGBA(x, y, color.NRGBA{uint8((s[0] + 2) / 4), uint8((s[1] + 2) / 4), uint8((s[2] + 2) / 4), 255})
		}
	}
	return out
}

// Diff is how far a picture is from its reference: Over pixels more than
// Levels away on some channel, the Most any channel is off, of Total.
type Diff struct{ Over, Most, Total int }

// Compare counts the pixels of got more than Levels away from want, and
// marks them red on a dimmed copy of want.
func Compare(t *testing.T, want image.Image, got *image.NRGBA) (Diff, *image.NRGBA) {
	t.Helper()
	r := want.Bounds()
	if r != got.Bounds() {
		t.Fatalf("sizes differ: %v and %v", r, got.Bounds())
	}
	marks := image.NewNRGBA(r)
	d := Diff{Total: r.Dx() * r.Dy()}
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			a, _ := color.NRGBAModel.Convert(want.At(x, y)).(color.NRGBA) // as its model makes
			b := got.NRGBAAt(x, y)
			m := max(apart(a.R, b.R), apart(a.G, b.G), apart(a.B, b.B))
			d.Most = max(d.Most, m)
			if m > Levels {
				d.Over++
				marks.SetNRGBA(x, y, color.NRGBA{255, 0, 0, 255})
			} else {
				marks.SetNRGBA(x, y, color.NRGBA{a.R / 4, a.G / 4, a.B / 4, 255})
			}
		}
	}
	return d, marks
}

func apart(a, b uint8) int {
	if a > b {
		return int(a - b)
	}
	return int(b - a)
}

// Load reads the PNG at path.
func Load(t *testing.T, path string) image.Image {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	return img
}

// Save writes img to path as a PNG, compressed its best.
func Save(t *testing.T, path string, img image.Image) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err = (&png.Encoder{CompressionLevel: png.BestCompression}).Encode(f, img); err != nil {
		t.Fatal(err)
	}
}
