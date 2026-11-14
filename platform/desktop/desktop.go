// Package desktop runs a World in native windows through GLFW and WebGPU.
//
// The plugin opens every window entity, the first added being the primary,
// and a default window when the game added none. Every window draws with
// one WebGPU device; removing the entity closes it. A window presents at
// the display's refresh unless the game gave it platform.VSync(false), so
// a visible frame takes a whole refresh however little work it holds; F3
// and EMBER_STATS show that work apart. Hidden windows draw offscreen and
// never wait.
//
// When the runner fails to start, no WebGPU adapter or no wgpu-native
// library among the reasons, or a panic escapes a frame, it writes a crash
// log (Plugin's CrashLog, ember.CrashLog) with the Go, GLFW and wgpu-native
// versions, the backend and the GPU, and on Windows shows a message box
// with the reason, so a windowed build never fails silently. A death no frame can catch, a panic in
// another goroutine, a fatal error or a fault in the driver, lands in the
// same file as the Go runtime prints it.
//
// Environment variables make every app scriptable, which is how the engine
// renders screenshots unattended:
//
//	EMBER_CAPTURE=shot.png  save frame EMBER_CAPTURE_FRAME (default 120), then quit;
//	                        extra windows are saved as shot-2.png, shot-3.png…
//	                        A hidden run ticks the frames before it without
//	                        drawing them, but those a light probe is answered
//	                        from; EMBER_CAPTURE_DRAW=all draws every frame
//	EMBER_CAPTURE=walk.gif  record EMBER_CAPTURE_FRAMES frames (default 90), one every
//	                        EMBER_CAPTURE_EVERY (default 2, so 30 fps), from frame
//	                        EMBER_CAPTURE_FRAME (default 1), as a looping GIF at the
//	                        window's size in points, then quit
//	EMBER_CAPTURE_KEYS=right:0-40;left:40-80
//	                        hold keys during those frames, in the primary window only;
//	                        every capture script separates its entries by ;
//	EMBER_CAPTURE_MOUSE=100,80@0;400,200@40;left:30-40
//	                        move the cursor through those points (window points) at
//	                        those frames, in straight lines, and hold mouse buttons,
//	                        in the primary window only
//	EMBER_CAPTURE_TOUCH=1:0.3,0.5@0;1:0.6,0.5@30;1:up@40;pinch:0.5@10-40
//	                        put trackpad fingers down (0 to 1 from the pad's
//	                        top-left), move and lift them, and pinch, rotate or
//	                        swipe, in the primary window only
//	EMBER_CAPTURE_SIZE=640,360@0;900,480@60
//	                        resize the primary window through those sizes (points)
//	                        at those frames, in straight lines; a GIF puts smaller
//	                        frames on a dark desk as large as the largest
//	EMBER_HIDDEN=1          create invisible windows, and play no sound
//	EMBER_SCALE=2           in a hidden run, pixels per point the windows draw
//	                        at, whatever the screen's (a CI screen is 1×)
//	EMBER_FRAMES=n          quit after n frames
//	EMBER_CRASH_DIR=dir     the folder crash.log and errors.log go under, in
//	                        place of the user's config folder, when the game
//	                        gave no CrashLog: a test's or a capture's own
//	EMBER_STATS=1           print, on quitting, the GPU and the backend it drew
//	                        through, the frame, tick and render times, each
//	                        system and the GPU's passes, averaged over the
//	                        frames after a warm-up of 60; GPU timers run as if
//	                        F3 showed, except with EMBER_STATS=cpu
//	EMBER_CPUPROFILE=f      write a CPU profile of the frames after that warm-up
//	EMBER_TRACE_INPUT=1     print every key event the windows receive
//
// A capture or a hidden run reads no real keyboard, mouse, trackpad or
// controller, and rumbles none: only its scripts play, so it gives the
// same frames whoever is at the machine.
package desktop

import (
	"errors"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/lantervale/ember/internal/inputfeed"
	"github.com/lantervale/ember/internal/wgpu"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/internal/audiodevice"
	"github.com/lantervale/ember/internal/draw"
	"github.com/lantervale/ember/internal/errlog"
	"github.com/lantervale/ember/internal/wm"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/platform/capture"
	"github.com/lantervale/ember/world"
)

//nolint:gochecknoinits // GLFW must be driven from the main OS thread, locked before main runs
func init() {
	runtime.LockOSThread()
}

// Plugin installs the windowed runner. It opens every window entity, the
// first added being the primary, and a default window when the game added
// none.
//
// CrashLog is where a failure to start or a panic while running is
// written, with the Go, GLFW and wgpu-native versions, the backend and the
// GPU; empty, it is
// crash.log in the user's config folder (EMBER_CRASH_DIR when set) under
// the slug of the primary window's title. On Windows a message box says
// why as well.
type Plugin struct{ CrashLog string }

func (p Plugin) Build(w *world.World) {
	d := &driver{
		crashLog: p.CrashLog,
		world:    w,
		hidden:   os.Getenv("EMBER_HIDDEN") != "",
		pixels:   hiddenScale(),
		scripted: os.Getenv("EMBER_HIDDEN") != "" || os.Getenv("EMBER_CAPTURE") != "",
		natives:  map[*platform.Window]*native{},
		gifs:     map[*native]*capture.GIF{},
		sound:    audiodevice.New(),
	}
	w.AddView(d.sound)
	w.AddView(&d.windows).SetRunner(func(*world.World) error { return d.guard(d.run) })
}

// guard runs run, and when it fails or panics, says so in the crash log
// and, on Windows, a message box, before the error or the panic goes on.
// Whatever kills the process outside it, a panic in another goroutine, a
// fatal error or a fault in the driver, the Go runtime writes there too.
func (d *driver) guard(run func() error) error {
	defer errlog.To(filepath.Join(filepath.Dir(d.crashPath()), "errors.log"))()
	stop := watchFatal(d.crashPath())
	defer func() {
		if p := recover(); p != nil {
			stop()
			d.fail(fmt.Sprintf("panic: %v", p), debug.Stack())
			panic(p)
		}
	}()
	err := run()
	stop()
	if err != nil {
		d.fail(err.Error(), nil)
		return err
	}
	return nil
}

func (d *driver) title() string {
	if win := wm.Primary(d.world); win != nil {
		return win.Title()
	}
	return ""
}

func (d *driver) crashPath() string { return crashPath(d.crashLog, d.title()) }

// fail writes the crash log for reason and tells the player where it is.
func (d *driver) fail(reason string, stack []byte) {
	title := d.title()
	path := d.crashPath()
	r := report{
		at:       time.Now(),
		reason:   reason,
		stack:    stack,
		glfw:     glfw.GetVersionString(),
		graphics: d.graphics,
		vendor:   d.vendor,
		renderer: d.renderer,
		errors:   errlog.Lines(),
	}
	where := "The details are in " + path + "."
	if err := writeCrash(path, r); err != nil {
		where = "The crash log couldn't be written: " + err.Error()
	}
	fmt.Fprintf(os.Stderr, "%s\n%s\n", reason, where)
	if !d.hidden {
		if title == "" {
			title = "Ember"
		}
		showError(title, reason+"\n\n"+where)
	}
}

type native struct {
	h       *glfw.Window
	entity  platform.Windowed
	window  *platform.Window
	focused bool
	order   int
	cursors map[platform.Cursor]*glfw.Cursor
	// scripted is set in a capture or a hidden run, where the cursor is
	// the script's.
	scripted bool
	// pixels is the driver's: pixels per point a hidden run draws at.
	pixels float32
	// surface is what the window shows through WebGPU, frame this frame's
	// texture of it, and configured the size it was last configured to.
	surface    *wgpu.Surface
	frame      *wgpu.Texture
	configured [2]int
	// hiddenFrame is what a hidden window draws into instead.
	hiddenFrame *wgpu.Texture
	// presented is when the window last showed a frame without vsync.
	presented time.Time
	// windowed is where the window was before going fullscreen, and
	// whether it was maximized.
	windowedX, windowedY, windowedW, windowedH int
	windowedMaximized                          bool
}

func (n *native) SetTitle(title string) { n.h.SetTitle(title) }

// standardCursor is GLFW's shape for c, the arrow for one it has none of.
func standardCursor(c platform.Cursor) glfw.StandardCursor {
	switch c {
	case platform.CursorText:
		return glfw.IBeamCursor
	case platform.CursorHand:
		return glfw.HandCursor
	case platform.CursorResizeEW:
		return glfw.HResizeCursor
	case platform.CursorResizeNS:
		return glfw.VResizeCursor
	case platform.CursorResizeNWSE:
		return glfw.CrosshairCursor
	case platform.CursorArrow, platform.CursorHidden:
	}
	return glfw.ArrowCursor
}

func (n *native) SetCursor(c platform.Cursor) {
	if c == platform.CursorHidden {
		n.h.SetInputMode(glfw.CursorMode, glfw.CursorHidden)
		return
	}
	n.h.SetInputMode(glfw.CursorMode, glfw.CursorNormal)
	if n.cursors == nil {
		n.cursors = make(map[platform.Cursor]*glfw.Cursor)
	}
	cur, ok := n.cursors[c]
	if !ok {
		cur = n.cursor(c)
		n.cursors[c] = cur
	}
	n.h.SetCursor(cur)
}

// cursor makes c's system cursor: its picture for one from NewCursor, the
// one at twice the size on a HiDPI screen where the system counts a
// cursor's pixels (Windows and Linux; macOS counts them as points).
func (n *native) cursor(c platform.Cursor) *glfw.Cursor {
	img := c.Image()
	if img == nil {
		return glfw.CreateStandardCursor(standardCursor(c))
	}
	x, y := c.Hot()
	if big := c.HiDPI(); big != nil && runtime.GOOS != "darwin" {
		if sx, _ := n.h.GetContentScale(); sx >= 1.5 {
			img, x, y = big, 2*x, 2*y
		}
	}
	return glfw.CreateCursor(img, x, y)
}

func (n *native) SetPosition(x, y int) { n.h.SetPos(x, y) }

func (n *native) WindowPosition() (int, int) { return n.h.GetPos() }

func (n *native) ScreenCursor() (float64, float64) {
	wx, wy := n.h.GetPos()
	if n.scripted {
		m := n.window.Input().Mouse()
		return float64(wx) + float64(m.X), float64(wy) + float64(m.Y)
	}
	cx, cy := n.h.GetCursorPos()
	return float64(wx) + cx, float64(wy) + cy
}
func (n *native) SetSize(w, h int) { n.h.SetSize(w, h) }

func (n *native) Clipboard() string        { return glfw.GetClipboardString() }
func (n *native) SetClipboard(text string) { glfw.SetClipboardString(text) }

func (n *native) sync() {
	ww, wh := n.h.GetSize()
	fw, fh := n.h.GetFramebufferSize()
	fw, fh = framebuffer(ww, wh, fw, fh, n.pixels)
	n.window.Sync(geom.V2(float32(ww), float32(wh)), geom.V2(float32(fw), float32(fh)), n.focused)
}

// framebuffer is the size a window of ww×wh points draws at: the system's
// fw×fh, or pixels a point when a hidden run asks for it. A hidden window
// draws offscreen, so its frame can be any size.
func framebuffer(ww, wh, fw, fh int, pixels float32) (int, int) {
	if pixels <= 0 {
		return fw, fh
	}
	return int(float32(ww)*pixels + 0.5), int(float32(wh)*pixels + 0.5)
}

// hiddenScale is EMBER_SCALE in a hidden run, else 0.
func hiddenScale() float32 {
	if os.Getenv("EMBER_HIDDEN") == "" {
		return 0
	}
	v, err := strconv.ParseFloat(os.Getenv("EMBER_SCALE"), 32)
	if err != nil {
		return 0
	}
	return float32(v)
}

type driver struct {
	world *world.World
	// crashLog is the game's crash log path, empty for the default;
	// graphics, vendor and renderer describe the device once it exists.
	crashLog                   string
	graphics, vendor, renderer string
	// gpu is the device every window draws with.
	inst   *wgpu.Instance
	gpu    *wgpu.Shared
	format wgpu.TextureFormat
	// interval is the primary display's refresh, the most often a window
	// without vsync shows a frame.
	interval time.Duration
	// pending are the windows drawn but not swapped yet.
	pending []*native
	hidden  bool
	// pixels is EMBER_SCALE in a hidden run, 0 for the screen's.
	pixels float32
	// scripted is set for a capture or a hidden run, which reads no real
	// input device.
	scripted bool
	windows  world.Group[platform.Windowed]
	natives  map[*platform.Window]*native
	sound    *audiodevice.Output
	primary  *native
	opened   int

	recording bool
	// live is set when frames are drawn from a window refresh, while the
	// OS holds the loop in a live resize; framing guards against nesting.
	live    bool
	framing bool
	gifFPS  int
	gifs    map[*native]*capture.GIF
}

// scripts are the EMBER_CAPTURE_* scripts a run plays into its primary
// window: keys, mouse, typed text, pads, touches and window sizes.
type scripts struct {
	keys    capture.Keys
	text    capture.Text
	pads    capture.Pads
	touches capture.Touches
	mouse   capture.Mouse
	sizes   capture.Sizes
}

// readScripts reads the scripts from the environment.
func readScripts() (scripts, error) {
	var s scripts
	var err error
	if s.keys, err = capture.ParseKeys(os.Getenv("EMBER_CAPTURE_KEYS")); err != nil {
		return s, err
	}
	if s.text, err = capture.ParseText(os.Getenv("EMBER_CAPTURE_TEXT")); err != nil {
		return s, err
	}
	if s.pads, err = capture.ParsePads(os.Getenv("EMBER_CAPTURE_PAD")); err != nil {
		return s, err
	}
	if s.touches, err = capture.ParseTouches(os.Getenv("EMBER_CAPTURE_TOUCH")); err != nil {
		return s, err
	}
	if s.mouse, err = capture.ParseMouse(os.Getenv("EMBER_CAPTURE_MOUSE")); err != nil {
		return s, err
	}
	s.sizes, err = capture.ParseSizes(os.Getenv("EMBER_CAPTURE_SIZE"))
	return s, err
}

// apply plays frame of the scripts into in and win, as its own events
// would.
func (s scripts) apply(in *input.State, win *platform.Window, frame int) {
	s.keys.Apply(in, frame)
	s.mouse.Apply(in, frame)
	s.text.Apply(in, frame)
	s.pads.Apply(in, frame)
	s.touches.Apply(in, frame)
	if size, ok := s.sizes.At(frame); ok && size != win.Size() {
		win.SetSize(size)
	}
}

// drawAll draws every window, in order, capturing each into path (the
// later ones suffixed) when final.
func (d *driver) drawAll(final bool, path string) {
	for i, n := range d.sorted() {
		if final {
			n.window.Capture(path)
			if i > 0 {
				n.window.Capture(suffixed(path, i+1))
			}
		}
		d.draw(n)
		d.pending = append(d.pending, n)
	}
	wm.SetCurrent(d.world, nil)
}

// readsBack reports whether the game waits on what drawing this frame
// brings back, so a run that skips frames draws it all the same.
func (d *driver) readsBack() bool {
	for _, n := range d.natives {
		if draw.ReadsBack(d.world, n.window) {
			return true
		}
	}
	return false
}

// afterTick runs what follows a frame's tick: the pads' rumble, the
// windows opened and closed, and the screenshot Shift+F12 asks for.
func (d *driver) afterTick(in *input.State) error {
	if !d.scripted {
		rumbleGamepads(in)
	}
	if err := d.open(); err != nil {
		return err
	}
	d.close()
	if in.Pressed(input.KeyF12) && (in.Down(input.KeyLeftShift) || in.Down(input.KeyRightShift)) {
		d.primary.window.Capture(time.Now().Format("ember-20060102-150405.png"))
	}
	return nil
}

// runSettings are a run's EMBER_* settings: what it captures, and how
// long it runs.
type runSettings struct {
	capturePath         string
	animated            bool
	captureFrame        int
	gifFrames, gifEvery int
	maxFrames           int
	profile             string
	// everyFrame is unset for a hidden run capturing a still, unless
	// EMBER_CAPTURE_DRAW=all.
	everyFrame bool
}

// draws reports whether frame is drawn for what the settings alone say: a
// hidden run capturing a still ticks the frames before the captured one
// without drawing them, as nobody sees them (decision 0072).
func (s runSettings) draws(frame int) bool { return s.everyFrame || frame >= s.captureFrame }

// readRunSettings reads them from the environment.
func readRunSettings() runSettings {
	s := runSettings{capturePath: os.Getenv("EMBER_CAPTURE"), profile: os.Getenv("EMBER_CPUPROFILE")}
	s.animated = strings.HasSuffix(s.capturePath, ".gif")
	firstFrame := 120
	if s.animated {
		firstFrame = 1
	}
	s.captureFrame = envInt("EMBER_CAPTURE_FRAME", firstFrame)
	s.gifFrames, s.gifEvery = envInt("EMBER_CAPTURE_FRAMES", 90), max(1, envInt("EMBER_CAPTURE_EVERY", 2))
	s.maxFrames = envInt("EMBER_FRAMES", 0)
	still := os.Getenv("EMBER_HIDDEN") != "" && s.capturePath != "" && !s.animated
	s.everyFrame = !still || os.Getenv("EMBER_CAPTURE_DRAW") == "all"
	return s
}

// start opens the windows, a default one when the World has none, and
// starts the sound unless hidden; stop undoes what start did.
func (d *driver) start() (func(), error) {
	loadGamepadMappings()
	if wm.Primary(d.world) == nil {
		d.world.AddEntity(platform.NewWindow())
	}
	if err := d.open(); err != nil {
		return func() {}, err
	}
	if d.primary == nil {
		return func() {}, errors.New("desktop: no primary window")
	}
	if d.hidden {
		return func() {}, nil
	}
	d.sound.Start()
	return func() {
		stopRumbles()
		d.sound.Close()
	}, nil
}

// endFrame records frame's stats, starts the CPU profile when its time
// comes and writes the GIFs once they are full; it reports whether the run
// is over.
func (d *driver) endFrame(
	set runSettings, stats *runStats, frame int, dt float32, begun time.Time, final bool,
) (bool, error) {
	if stats != nil {
		stats.add(d.world, frame, time.Duration(float64(dt)*float64(time.Second)))
		stats.spike(os.Stderr, d.world, frame, time.Since(begun))
	}
	if set.profile != "" && frame == statsWarmup {
		if err := startProfile(set.profile); err != nil {
			return true, err
		}
	}
	if set.animated && d.gifs[d.primary].Len() >= set.gifFrames {
		return true, d.writeGIFs(set.capturePath)
	}
	return final || (set.maxFrames > 0 && frame >= set.maxFrames), nil
}

// statsFromEnv makes the run's stats under EMBER_STATS, nil otherwise, and
// report prints them at its end.
func (d *driver) statsFromEnv() (*runStats, func()) {
	if os.Getenv("EMBER_STATS") == "" {
		return nil, func() {}
	}
	stats := &runStats{}
	return stats, func() {
		fmt.Fprintf(os.Stderr, "graphics: %s, %s\n", d.graphics, d.renderer)
		stats.print(os.Stderr, 24)
	}
}

// frameTime is the time since *last, which it moves to now: a fixed 1/60 s
// for a capture, so its frames are the same on every machine.
func frameTime(last *time.Time, fixed bool) float32 {
	now := time.Now()
	dt := float32(now.Sub(*last).Seconds())
	*last = now
	if fixed {
		return 1.0 / 60
	}
	return dt
}

func (d *driver) run() error {
	if err := glfw.Init(); err != nil {
		return fmt.Errorf("desktop: init glfw: %w", err)
	}
	defer glfw.Terminate()
	stop, err := d.start()
	defer stop()
	if err != nil {
		return err
	}
	set := readRunSettings()
	d.live = !d.hidden && set.capturePath == ""
	d.gifFPS = max(1, 60/set.gifEvery)
	played, err := readScripts()
	if err != nil {
		return err
	}
	stats, report := d.statsFromEnv()
	defer report()
	defer pprof.StopCPUProfile()
	in := d.primary.window.Input()

	last := time.Now()
	for frame := 1; !d.world.ShouldQuit(); frame++ {
		begun := time.Now()
		glfw.PollEvents()
		d.framing = true
		if d.primary.h.ShouldClose() {
			break
		}
		d.closeRequested()
		pollGamepads(in, devices(d.scripted, joysticks))
		played.apply(in, d.primary.window, frame-1)
		dt := frameTime(&last, set.capturePath != "")

		d.world.Tick(dt)
		d.swapPending()
		if err = d.afterTick(in); err != nil {
			return err
		}
		final := set.capturePath != "" && !set.animated && frame == set.captureFrame
		d.recording = set.animated && frame >= set.captureFrame && (frame-set.captureFrame)%set.gifEvery == 0
		if set.draws(frame) || d.readsBack() {
			d.drawAll(final, set.capturePath)
		}

		d.world.EndFrame()
		d.framing = false
		over, endErr := d.endFrame(set, stats, frame, dt, begun, final)
		if endErr != nil {
			return endErr
		}
		if over {
			break
		}
	}
	d.swapPending()
	for _, n := range d.natives {
		if n != d.primary {
			d.destroy(n)
		}
	}
	return nil
}

// refresh draws a frame while the OS holds the loop, in a live resize or
// a move to fullscreen, so the window never shows its last frame
// stretched. The frame takes no time: nothing moves, but everything is
// laid out and drawn at the new size.
func (d *driver) refresh() {
	if !d.live || d.framing || d.primary == nil {
		return
	}
	d.framing = true
	defer func() { d.framing = false }()
	for _, n := range d.natives {
		n.sync()
	}
	d.world.Tick(0)
	d.swapPending()
	for _, n := range d.sorted() {
		d.present(n)
	}
	wm.SetCurrent(d.world, nil)
	d.world.EndFrame()
}

func (d *driver) sorted() []*native {
	out := make([]*native, 0, len(d.natives))
	for _, n := range d.natives {
		out = append(out, n)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].order < out[j].order })
	return out
}

// present draws one window and swaps it at once.
func (d *driver) present(n *native) {
	d.draw(n)
	d.swap(n)
}

// swapPending swaps the windows the last frame drew. A frame is drawn,
// then the next one simulated while the GPU draws it, and only then
// swapped, since the swap waits for the GPU: the CPU and the GPU work at
// the same time, for one tick more before a frame shows.
func (d *driver) swapPending() {
	for _, n := range d.pending {
		d.swap(n)
	}
	d.pending = d.pending[:0]
}

func (d *driver) swap(n *native) {
	start := time.Now()
	d.show(n)
	if n == d.primary {
		d.world.Stats().Record("swap", time.Since(start))
	}
}

// draw runs every presenter into one window and sends its commands to the
// GPU, without swapping.
func (d *driver) draw(n *native) {
	wm.SetCurrent(d.world, n.window)
	path := n.window.TakeCapture()
	if !d.acquire(n, d.recording || path != "") {
		return
	}
	d.world.Present()
	if d.recording {
		size := n.window.Size()
		if d.gifs[n] == nil {
			d.gifs[n] = capture.NewGIF(d.gifFPS)
		}
		if img, err := d.read(n); err == nil {
			d.gifs[n].Add(img, int(size.X), int(size.Y))
		}
	}
	if path != "" {
		img, err := d.read(n)
		if err == nil {
			err = savePNG(path, img)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "desktop: capture:", err)
		} else {
			fmt.Fprintln(os.Stderr, "desktop: saved", path)
		}
	}
}

// read is what n shows this frame, opaque, top row first.
func (d *driver) read(n *native) (*image.RGBA, error) {
	if n.frame == nil {
		return nil, errors.New("no frame drawn")
	}
	return d.gpu.Device.Read(n.frame)
}

// open creates native windows for new Windowed entities.
func (d *driver) open() error {
	for e := range d.windows.All() {
		win := e.GetWindow()
		if _, ok := d.natives[win]; ok {
			continue
		}
		n, err := d.create(e)
		if err != nil {
			return err
		}
		d.natives[win] = n
		if win == wm.Primary(d.world) {
			d.primary = n
		}
	}
	return nil
}

// close destroys native windows whose entity left the world.
func (d *driver) close() {
	for win, n := range d.natives {
		if n == d.primary {
			continue
		}
		if _, ok := d.windows.Get(n.entity.ID()); !ok {
			d.destroy(n)
			delete(d.natives, win)
		}
	}
}

// closeRequested turns native close buttons on secondary windows into
// entity removal plus a WindowClosed event.
func (d *driver) closeRequested() {
	for _, n := range d.natives {
		if n != d.primary && n.h.ShouldClose() {
			n.h.SetShouldClose(false)
			world.Emit(d.world, platform.WindowClosed{Window: n.window})
			d.world.RemoveEntity(n.entity)
		}
	}
}

func (d *driver) create(e platform.Windowed) (*native, error) {
	win := e.GetWindow()
	glfw.DefaultWindowHints()
	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
	glfw.WindowHint(glfw.ScaleToMonitor, glfw.True)
	if win.Hidden() || d.hidden {
		glfw.WindowHint(glfw.Visible, glfw.False)
	}
	if win.Undecorated() {
		glfw.WindowHint(glfw.Decorated, glfw.False)
	}
	hints(win)
	size, at := win.Size(), win.Position()
	h, err := glfw.CreateWindow(int(size.X), int(size.Y), win.Title(), nil, nil)
	if err != nil {
		return nil, fmt.Errorf("desktop: create window %q: %w", win.Title(), err)
	}
	if at != (geom.Vec2{}) {
		h.SetPos(int(at.X), int(at.Y))
	}
	d.opened++
	n := &native{
		h:        h,
		entity:   e,
		window:   win,
		focused:  true,
		order:    d.opened,
		scripted: d.scripted,
		pixels:   d.pixels,
	}
	if err = d.attach(n); err != nil {
		h.Destroy()
		return nil, err
	}

	n.apply(win)
	win.Attach(n)
	n.sync()
	px, py := h.GetPos()
	win.SyncPosition(geom.V2(float32(px), float32(py)))
	h.SetPosCallback(func(_ *glfw.Window, x, y int) { win.SyncPosition(geom.V2(float32(x), float32(y))) })
	h.SetSizeCallback(func(*glfw.Window, int, int) { n.sync() })
	h.SetFramebufferSizeCallback(func(*glfw.Window, int, int) { n.sync() })
	h.SetRefreshCallback(func(*glfw.Window) { d.refresh() })
	h.SetFocusCallback(func(_ *glfw.Window, f bool) { n.focused = f; n.sync() })
	if !d.scripted {
		d.listen(h, win)
	}
	return n, nil
}

// listen feeds the real keyboard, mouse, trackpad, typed text and dropped
// files of h into win's input.
func (d *driver) listen(h *glfw.Window, win *platform.Window) {
	in := win.Input()
	trace := os.Getenv("EMBER_TRACE_INPUT") != ""
	h.SetKeyCallback(func(_ *glfw.Window, k glfw.Key, scancode int, action glfw.Action, _ glfw.ModifierKey) {
		if trace {
			fmt.Fprintf(
				os.Stderr,
				"input: %-16s %-7s scancode %-3d window %q\n",
				input.Key(k),
				[...]string{"up", "down", "repeat"}[action],
				scancode,
				win.Title(),
			)
		}
		if action == glfw.Repeat {
			inputfeed.Of(in).AddRepeat(int(k))
		} else {
			in.SetKey(input.Key(k), action == glfw.Press)
		}
	})
	h.SetCharCallback(func(_ *glfw.Window, r rune) { in.AddText(r) })
	h.SetMouseButtonCallback(func(_ *glfw.Window, b glfw.MouseButton, action glfw.Action, _ glfw.ModifierKey) {
		in.SetMouseButton(input.MouseButton(b), action == glfw.Press)
	})
	h.SetCursorPosCallback(func(_ *glfw.Window, x, y float64) { in.SetMouse(geom.V2(float32(x), float32(y))) })
	h.SetScrollCallback(func(_ *glfw.Window, x, y float64) { in.AddWheel(geom.V2(float32(x), float32(y))) })
	listenTouches(h, in)
	listenText(h, in)
	h.SetDropCallback(func(_ *glfw.Window, paths []string) {
		target := in
		if d.primary != nil {
			target = d.primary.window.Input()
		}
		target.AddDrop(paths...)
	})
}

func (d *driver) destroy(n *native) {
	d.pending = slices.DeleteFunc(d.pending, func(p *native) bool { return p == n })
	forgetTouches(n.h)
	forgetText(n.h)
	n.window.Attach(nil)
	for _, c := range n.cursors {
		c.Destroy()
	}
	if n.surface != nil {
		d.show(n)
		n.surface.Release()
	}
	n.h.Destroy()
}

// writeGIFs writes each window's recording, the primary to path and the
// others suffixed.
func (d *driver) writeGIFs(path string) error {
	for i, n := range d.sorted() {
		g := d.gifs[n]
		if g == nil {
			continue
		}
		out := path
		if i > 0 {
			out = suffixed(path, i+1)
		}
		f, err := os.Create(out)
		if err != nil {
			return err
		}
		if err = g.Encode(f); err != nil {
			f.Close()
			return err
		}
		if err = f.Close(); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "desktop: saved", out)
	}
	return nil
}

func suffixed(path string, i int) string {
	ext := filepath.Ext(path)
	return strings.TrimSuffix(path, ext) + "-" + strconv.Itoa(i) + ext
}

func envInt(name string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(name)); err == nil {
		return v
	}
	return def
}
