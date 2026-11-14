//go:build js

// Package web runs a World in a browser page: the primary window is the
// page's canvas with id "ember", drawn through WebGPU, a frame each
// requestAnimationFrame: the page's keyboard, the canvas's pointer and
// wheel, the Gamepad API, cursors and the clipboard (W2 of
// docs/design/web.md), and sound through Web Audio once the player clicks
// or presses a key. Saves go through os to the page's fs.js, kept in
// localStorage; what stops the game shows over the page and in the
// console.
//
// The page passes settings as environment variables through go.env, as a
// desktop run takes them, the capture ones among them:
//
//	EMBER_CAPTURE=1          draw frame EMBER_CAPTURE_FRAME (default 120) at a
//	                         fixed step, then publish it as a PNG in
//	                         window.emberCapture (base64) and stop
//	EMBER_CAPTURE_KEYS, EMBER_CAPTURE_MOUSE   scripted input, as on desktop
//	EMBER_SCALE=2            pixels per point, else the page's devicePixelRatio
package web

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image/png"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"syscall/js"
	"time"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/internal/audiodevice"
	"github.com/lantervale/ember/internal/errlog"
	"github.com/lantervale/ember/internal/wgpu"
	"github.com/lantervale/ember/internal/wm"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/platform/capture"
	"github.com/lantervale/ember/world"
)

// Plugin makes the page's canvas the World's runner.
type Plugin struct{}

func (Plugin) Build(w *world.World) {
	r := &runner{world: w, sound: audiodevice.New()}
	w.AddView(r.sound)
	w.AddView(&r.windows).SetRunner(func(*world.World) error {
		defer func() {
			if p := recover(); p != nil {
				crash(fmt.Sprintf("panic: %v\n\n%s", p, debug.Stack()))
				panic(p)
			}
		}()
		err := r.run()
		if err != nil {
			crash(err.Error())
		}
		return err
	})
}

type runner struct {
	world   *world.World
	windows world.Group[platform.Windowed]
	// sound plays the World's mixer through Web Audio, which a browser
	// starts only after the player's first click or key.
	sound *audiodevice.Output
}

func (r *runner) run() error {
	dev, ad, err := openDevice()
	if err != nil {
		return err
	}
	win := wm.Primary(r.world)
	if win == nil {
		return fmt.Errorf("web: no primary window")
	}
	canvas, scale, err := pageCanvas()
	if err != nil {
		return err
	}
	surface, err := wgpu.CanvasSurface("ember")
	if err != nil {
		return err
	}
	view := fitTo(&fitter{canvas: canvas, win: win, surface: surface, dev: dev, scale: scale, own: win.Size()})
	shared := wgpu.Share(r.world, dev, ad.Info())

	capturing := os.Getenv("EMBER_CAPTURE") != ""
	captureFrame := envInt("EMBER_CAPTURE_FRAME", 120)
	keyScript, mouse, err := captureScripts()
	if err != nil {
		return err
	}

	frames, stop := animationFrames()
	defer stop()
	defer r.sound.Close()

	in := win.Input()
	if !capturing {
		defer r.attach(canvas, win)()
	}
	var gamepads pads
	last := time.Now()
	for frame := 1; !r.world.ShouldQuit(); frame++ {
		<-frames
		if capturing {
			keyScript.Apply(in, frame-1)
			mouse.Apply(in, frame-1)
		} else {
			gamepads.poll(in)
		}
		now := time.Now()
		dt := float32(now.Sub(last).Seconds())
		last = now
		if capturing {
			dt = 1.0 / 60
		}
		view.fit()
		r.world.Tick(dt)
		if frame == 1 && !capturing {
			// Start waits for the page's first click or key before Web
			// Audio runs, so it waits in a goroutine of its own, once the
			// game has added its mixer.
			go r.sound.Start()
		}
		t, presentErr := r.present(surface, shared, win)
		if presentErr != nil {
			return presentErr
		}
		if capturing && frame == captureFrame {
			return captureTo(dev, t)
		}
		shared.SetTarget(win, nil)
	}
	return nil
}

// present draws the World's frame into the surface's texture for win,
// which it gives, still set as win's target.
func (r *runner) present(surface *wgpu.Surface, shared *wgpu.Shared, win *platform.Window) (*wgpu.Texture, error) {
	t, err := surface.Current()
	if err != nil {
		return nil, err
	}
	shared.SetTarget(win, t)
	wgpu.Framing(true)
	wm.SetCurrent(r.world, win)
	r.world.Present()
	wm.SetCurrent(r.world, nil)
	wgpu.Framing(false)
	if err = wgpu.TakeError(); err != nil {
		return nil, err
	}
	return t, nil
}

func openDevice() (*wgpu.Device, *wgpu.Adapter, error) {
	inst, err := wgpu.NewInstance()
	if err != nil {
		return nil, nil, err
	}
	ad, err := inst.RequestAdapter(nil)
	if err != nil {
		return nil, nil, err
	}
	dev, err := ad.RequestDevice()
	if err != nil {
		return nil, nil, err
	}
	return dev, ad, nil
}

// captureScripts are EMBER_CAPTURE_KEYS and EMBER_CAPTURE_MOUSE, the keys
// and pointer a capture plays.
func captureScripts() (capture.Keys, capture.Mouse, error) {
	keys, err := capture.ParseKeys(os.Getenv("EMBER_CAPTURE_KEYS"))
	if err != nil {
		return keys, capture.Mouse{}, err
	}
	mouse, err := capture.ParseMouse(os.Getenv("EMBER_CAPTURE_MOUSE"))
	return keys, mouse, err
}

// pageCanvas is the page's canvas #ember and the pixels a point it draws
// at: the page's pixel ratio, or EMBER_SCALE.
func pageCanvas() (js.Value, float64, error) {
	scale := envFloat("EMBER_SCALE", js.Global().Get("devicePixelRatio").Float())
	canvas := js.Global().Get("document").Call("getElementById", "ember")
	if canvas.IsNull() {
		return canvas, 0, fmt.Errorf("web: the page has no canvas #ember")
	}
	return canvas, scale, nil
}

// fitter keeps the canvas, the window and the surface the size the game
// shows at: the page's while it is full screen, whoever asked for it, or
// while the window is resizable, as a desktop window follows its frame (no
// smaller than MinSize); else the window's own.
type fitter struct {
	canvas  js.Value
	win     *platform.Window
	surface *wgpu.Surface
	dev     *wgpu.Device
	scale   float64
	own     geom.Vec2
	shown   geom.Vec2
}

// fitTo fits f now and gives it, to fit again each frame.
func fitTo(f *fitter) *fitter {
	f.fit()
	return f
}

func (f *fitter) fit() {
	full := !js.Global().Get("document").Get("fullscreenElement").IsNull()
	size := f.own
	if full || f.win.Resizable() {
		page := geom.V2(float32(js.Global().Get("innerWidth").Float()), float32(js.Global().Get("innerHeight").Float()))
		least := f.win.MinSize()
		size = geom.V2(max(page.X, least.X), max(page.Y, least.Y))
	}
	if full != f.win.Fullscreen() {
		f.win.SyncFullscreen(full)
	}
	if size == f.shown || size.X <= 0 || size.Y <= 0 {
		return
	}
	f.shown = size
	fw, fh := int(size.X*float32(f.scale)+0.5), int(size.Y*float32(f.scale)+0.5)
	f.canvas.Set("width", fw)
	f.canvas.Set("height", fh)
	f.canvas.Get("style").Set("width", fmt.Sprintf("%gpx", size.X))
	f.canvas.Get("style").Set("height", fmt.Sprintf("%gpx", size.Y))
	f.win.Sync(size, geom.V2(float32(fw), float32(fh)), true)
	f.surface.Configure(f.dev, fw, fh, wgpu.PreferredFormat(), true)
}

// animationFrames signals each of the browser's animation frames, at most
// one waiting; stop cancels the next and frees the callback.
func animationFrames() (<-chan struct{}, func()) {
	frames := make(chan struct{}, 1)
	var tick js.Func
	var next js.Value
	tick = js.FuncOf(func(js.Value, []js.Value) any {
		select {
		case frames <- struct{}{}:
		default:
		}
		next = js.Global().Call("requestAnimationFrame", tick)
		return nil
	})
	next = js.Global().Call("requestAnimationFrame", tick)
	return frames, func() {
		js.Global().Call("cancelAnimationFrame", next)
		tick.Release()
	}
}

// attach listens to the canvas's input, resumes the sound on its first
// click, key or touch, and titles the page; the func it gives removes and
// releases every handler it added.
func (r *runner) attach(canvas js.Value, win *platform.Window) func() {
	in := win.Input()
	ls := &listeners{}
	listen(ls, canvas, in)
	// The canvas keeps mousedown and mouseup from the page, which oto
	// waits for, so its own click, key and touch resume the sound.
	for _, event := range []string{"click", "keydown", "touchend"} {
		ls.on(canvas, event, func(js.Value) { r.sound.Resume() })
	}
	win.Attach(newCanvas(ls, canvas, in))
	if t := win.Title(); t != "" {
		js.Global().Get("document").Set("title", t)
	}
	return ls.release
}

// captureTo reads t back and hands it to the page as emberCapture, a
// base64 PNG.
func captureTo(dev *wgpu.Device, t *wgpu.Texture) error {
	img, err := dev.Read(t)
	if err != nil {
		return err
	}
	var b bytes.Buffer
	if err = png.Encode(&b, img); err != nil {
		return err
	}
	js.Global().Set("emberCapture", base64.StdEncoding.EncodeToString(b.Bytes()))
	return nil
}

// crash shows what stopped the game over the page, with the error lines
// the engine printed before it, as the desktop's crash log does, and in
// the console.
func crash(reason string) {
	text := reason
	if lines := errlog.Lines(); len(lines) > 0 {
		text += "\n\nErrors before it:\n  " + strings.Join(lines, "\n  ")
	}
	fmt.Fprintln(os.Stdout, "web:", text)
	if show := js.Global().Get("emberShowCrash"); show.Type() == js.TypeFunction {
		show.Invoke(text)
	} else {
		js.Global().Set("emberError", text)
	}
}

func envInt(name string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(name)); err == nil {
		return v
	}
	return def
}

func envFloat(name string, def float64) float64 {
	if v, err := strconv.ParseFloat(os.Getenv(name), 64); err == nil && v > 0 {
		return v
	}
	return def
}
