package shots

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/merlindorin/ember/internal/wgpu"
)

// version changes the key of every shot when what the runner does with a
// capture changes.
const version = "1"

// lightpanels is the package every Panels shot runs.
const lightpanels = "./examples/internal/lightpanels"

// Options are a run's settings.
type Options struct {
	// Root is ember's root, Docs the folder Out is under, Work where the
	// binaries, keys and captures go.
	Root, Docs, Work string
	// Jobs is how many shots run at once.
	Jobs int
	// Force redoes every shot, Only those whose Out contains it.
	Force bool
	Only  string
	// Log tells what each shot did.
	Log io.Writer
}

// Run makes every shot of list whose key changed since it was last made,
// building each package once, Jobs shots at a time.
func Run(list []Shot, o Options) error {
	if o.Jobs < 1 {
		o.Jobs = 1
	}
	if o.Log == nil {
		o.Log = io.Discard
	}
	for _, dir := range []string{"bin", "keys", "tmp"} {
		if err := os.MkdirAll(filepath.Join(o.Work, dir), 0o755); err != nil {
			return err
		}
	}
	env := os.Environ()
	if os.Getenv(wgpu.LibraryEnv) == "" {
		lib := filepath.Join(o.Root, "internal", "wgpu", "lib", runtime.GOOS+"-"+runtime.GOARCH, libraryName())
		if _, err := os.Stat(lib); err == nil {
			abs, _ := filepath.Abs(lib)
			env = append(env, wgpu.LibraryEnv+"="+abs)
		}
	}
	var todo []Shot
	for _, s := range list {
		if o.Only == "" || strings.Contains(s.Out, o.Only) {
			todo = append(todo, s)
		}
	}
	bins, err := build(todo, o)
	if err != nil {
		return err
	}
	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []error
		work = make(chan Shot)
	)
	for range o.Jobs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for s := range work {
				did, failed := one(s, bins, env, o)
				mu.Lock()
				switch {
				case failed != nil:
					errs = append(errs, fmt.Errorf("%s: %w", s.Out, failed))
					fmt.Fprintf(o.Log, "FAIL %s: %v\n", s.Out, failed)
				case did:
					fmt.Fprintf(o.Log, "made %s\n", s.Out)
				}
				mu.Unlock()
			}
		}()
	}
	for _, s := range todo {
		work <- s
	}
	close(work)
	wg.Wait()
	return errors.Join(errs...)
}

func libraryName() string {
	switch runtime.GOOS {
	case "windows":
		return "wgpu_native.dll"
	case "darwin":
		return "libwgpu_native.dylib"
	}
	return "libwgpu_native.so"
}

// binary is a package built for the shots and its content's hash.
type binary struct{ path, sum string }

// build builds each package the shots run once, with -trimpath so the
// same sources make the same bytes, which key the shots.
func build(list []Shot, o Options) (map[string]binary, error) {
	bins := map[string]binary{}
	for _, s := range list {
		pkg := s.pkg()
		if pkg == "" || bins[pkg].path != "" {
			continue
		}
		name := strings.NewReplacer("/", "_", ".", "_").Replace(strings.TrimPrefix(pkg, "./"))
		out, _ := filepath.Abs(filepath.Join(o.Work, "bin", name))
		if runtime.GOOS == "windows" {
			out += ".exe"
		}
		cmd := exec.CommandContext(context.Background(), "go", "build", "-trimpath", "-o", out, pkg)
		cmd.Dir = o.Root
		if b, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("build %s: %w: %s", pkg, err, b)
		}
		sum, err := fileSum(out)
		if err != nil {
			return nil, err
		}
		bins[pkg] = binary{out, sum}
	}
	return bins, nil
}

// pkg is the package s runs, empty for a copy.
func (s Shot) pkg() string {
	switch {
	case len(s.Panels) > 0:
		return lightpanels
	case s.Copy != "":
		return ""
	}
	return s.Run
}

func fileSum(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// key is what s's picture depends on: the runner, its binary, its copied
// files and itself.
func (s Shot) key(bins map[string]binary, root string) (string, error) {
	h := sha256.New()
	fmt.Fprintln(h, version, bins[s.pkg()].sum)
	entry, _ := json.Marshal(s)
	h.Write(entry)
	for _, src := range append(s.Files, s.Copy) {
		if src == "" {
			continue
		}
		err := filepath.WalkDir(filepath.Join(root, src), func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			sum, err := fileSum(path)
			fmt.Fprintln(h, filepath.ToSlash(path), sum)
			return err
		})
		if err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// one makes s unless its picture is there with the same key, and says
// whether it did.
func one(s Shot, bins map[string]binary, env []string, o Options) (bool, error) {
	key, err := s.key(bins, o.Root)
	if err != nil {
		return false, err
	}
	out := filepath.Join(o.Docs, filepath.FromSlash(s.Out))
	keyFile := filepath.Join(o.Work, "keys", strings.ReplaceAll(s.Out, "/", "__")+".key")
	if !o.Force {
		if old, readErr := os.ReadFile(keyFile); readErr == nil && string(old) == key {
			if _, statErr := os.Stat(out); statErr == nil {
				return false, nil
			}
		}
	}
	tmp, err := os.MkdirTemp(filepath.Join(o.Work, "tmp"), "shot-")
	if err != nil {
		return false, err
	}
	defer os.RemoveAll(tmp)
	if err = os.MkdirAll(filepath.Dir(out), 0o750); err != nil {
		return false, err
	}
	switch {
	case s.Copy != "":
		err = copyFile(filepath.Join(o.Root, s.Copy), out)
	case len(s.Panels) > 0:
		err = s.strip(bins[lightpanels].path, env, tmp, out, o)
	default:
		err = s.capture(bins[s.Run].path, env, tmp, out, o)
	}
	if err != nil {
		return false, err
	}
	return true, os.WriteFile(keyFile, []byte(key), 0o600)
}

// capture runs s's binary in tmp and turns its capture into out.
func (s Shot) capture(bin string, env []string, tmp, out string, o Options) error {
	if err := s.prepare(tmp, o.Root); err != nil {
		return err
	}
	ext := filepath.Ext(s.Out)
	shot := filepath.Join(tmp, "capture"+ext)
	frame := s.Frame
	if frame == 0 && ext == ".png" {
		frame = 30
	}
	if err := run(bin, s.expand(s.Args, tmp), s.environ(env, tmp, shot, frame), o.Root); err != nil {
		return err
	}
	got := windowShot(shot, s.Window)
	if ext == ".gif" {
		if s.Crop != ([4]int{}) || s.Size != ([2]int{}) {
			return errors.New("a .gif is kept as recorded: no Crop or Size")
		}
		return copyFile(got, out)
	}
	img, err := load(got)
	if err != nil {
		return err
	}
	return save(out, s.finish(img))
}

// finish crops img and sizes it: to Size, else to half, from pixels to
// points.
func (s Shot) finish(img image.Image) image.Image {
	if c := s.Crop; c != ([4]int{}) {
		img = crop(img, image.Rect(c[0], c[1], c[0]+c[2], c[1]+c[3]))
	}
	if s.Size != ([2]int{}) {
		return resize(img, s.Size[0], s.Size[1])
	}
	return halve(img)
}

// strip captures each of s's panels and lines them up, each 320×240.
func (s Shot) strip(bin string, env []string, tmp, out string, o Options) error {
	frame := s.Frame
	if frame == 0 {
		frame = 10
	}
	var panels []image.Image
	for i, p := range s.Panels {
		shot := filepath.Join(tmp, "panel-"+strconv.Itoa(i)+".png")
		if err := run(bin, []string{"-panel", p}, s.environ(env, tmp, shot, frame), o.Root); err != nil {
			return fmt.Errorf("panel %s: %w", p, err)
		}
		img, err := load(shot)
		if err != nil {
			return err
		}
		panels = append(panels, resize(img, 320, 240))
	}
	strip := hstack(panels)
	if s.Size != ([2]int{}) {
		strip = resize(strip, s.Size[0], s.Size[1])
	}
	return save(out, strip)
}

// prepare copies s's files and writes its Write ones into tmp.
func (s Shot) prepare(tmp, root string) error {
	for _, src := range s.Files {
		from := filepath.Join(root, src)
		to := filepath.Join(tmp, filepath.Base(src))
		err := filepath.WalkDir(from, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(from, path)
			dst := filepath.Join(to, rel)
			if d.IsDir() {
				return os.MkdirAll(dst, 0o755)
			}
			return copyFile(path, dst)
		})
		if err != nil {
			return err
		}
	}
	for name, content := range s.Write {
		path := filepath.Join(tmp, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			return err
		}
	}
	return nil
}

// expand puts tmp for {tmp} in each of vs.
func (s Shot) expand(vs []string, tmp string) []string {
	out := make([]string, len(vs))
	for i, v := range vs {
		out[i] = strings.ReplaceAll(v, "{tmp}", tmp)
	}
	return out
}

// environ is base with the capture's settings and s's Env.
func (s Shot) environ(base []string, tmp, shot string, frame int) []string {
	// EMBER_SCALE draws at 2 pixels a point on every screen, the 1×
	// one Pages captures on too.
	env := append(append([]string{}, base...), "EMBER_HIDDEN=1", "EMBER_SCALE=2", "EMBER_CAPTURE="+shot)
	if frame > 0 {
		env = append(env, "EMBER_CAPTURE_FRAME="+strconv.Itoa(frame))
	}
	for k, v := range s.Env {
		env = append(env, k+"="+strings.ReplaceAll(v, "{tmp}", tmp))
	}
	return env
}

func run(bin string, args, env []string, dir string) error {
	cmd := exec.CommandContext(context.Background(), bin, args...)
	cmd.Dir = dir
	cmd.Env = env
	if b, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: %s", err, lastLines(b, 5))
	}
	return nil
}

func lastLines(b []byte, n int) string {
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	return strings.Join(lines[max(len(lines)-n, 0):], "\n")
}

// windowShot is window n's capture: path for the first, path-n then.
func windowShot(path string, n int) string {
	if n <= 1 {
		return path
	}
	ext := filepath.Ext(path)
	return strings.TrimSuffix(path, ext) + "-" + strconv.Itoa(n) + ext
}

func copyFile(from, to string) error {
	b, err := os.ReadFile(from)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(to), 0o750); err != nil {
		return err
	}
	return os.WriteFile(to, b, 0o600) //nolint:gosec // paths from the repository's own shot list
}

func load(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return png.Decode(f)
}

func save(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err = (&png.Encoder{CompressionLevel: png.BestCompression}).Encode(f, img); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
