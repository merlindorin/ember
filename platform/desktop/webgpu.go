package desktop

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"time"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/lantervale/ember/internal/wgpu"
)

// attach gives n its WebGPU surface, and with the first window, the
// instance, adapter and device every window shares.
func (d *driver) attach(n *native) error {
	if d.inst == nil {
		inst, err := wgpu.NewInstance()
		if err != nil {
			return fmt.Errorf(
				"ember couldn't load its graphics library, which ships beside the game. Reinstalling the game puts it back.\n\n%w",
				err,
			)
		}
		d.inst = inst
	}
	src, err := surfaceSource(n.h)
	if err != nil {
		return err
	}
	surface, err := d.inst.Surface(src)
	if err != nil {
		return err
	}
	n.surface = surface
	if d.gpu == nil {
		ad, adapterErr := d.inst.RequestAdapter(surface)
		if adapterErr != nil {
			return errors.New(noAdapter(wgpu.Backends(), adapterErr.Error()))
		}
		dev, deviceErr := ad.RequestDevice()
		if deviceErr != nil {
			return deviceErr
		}
		info := ad.Info()
		d.gpu = wgpu.Share(d.world, dev, info)
		d.graphics = fmt.Sprintf("WebGPU (wgpu-native %s, %s)", wgpu.Version(), info.Backend)
		d.vendor, d.renderer = info.Vendor, info.Device
		d.interval = refreshInterval(glfw.GetPrimaryMonitor())
		d.format = wgpu.BGRA8Unorm
		if formats := surface.Formats(ad); len(formats) > 0 && !slices.Contains(formats, wgpu.BGRA8Unorm) {
			d.format = formats[0]
		}
	}
	return nil
}

// acquire takes n's texture for this frame, configuring its surface for
// the window's framebuffer when that changed, and reports whether there is
// one to draw into. A frame read back, for a capture or a GIF, draws
// offscreen, since the surface's own textures can only be shown. Without
// vsync a window shows at most one frame a display refresh and draws the
// others offscreen: macOS lets about two drawables through a refresh and
// holds nextDrawable for the rest, 7 ms a frame on average and up to
// 60 ms (docs/design/performance.md).
func (d *driver) acquire(n *native, read bool) bool {
	fb := n.window.FramebufferSize()
	w, h := int(fb.X), int(fb.Y)
	if w <= 0 || h <= 0 {
		return false
	}
	if d.hidden || read || held(n.window.VSync(), time.Since(n.presented), d.interval) {
		if n.frame != nil {
			d.show(n)
		}
		if n == d.primary && !d.hidden {
			d.world.Stats().Record("acquire", 0)
		}
		return d.offscreen(n, w, h)
	}
	n.presented = time.Now()
	for range 2 {
		if n.configured != [2]int{w, h} {
			n.surface.Configure(d.gpu.Device, w, h, d.format, n.window.VSync())
			n.configured = [2]int{w, h}
		}
		if n.frame != nil {
			d.show(n)
		}
		began := time.Now()
		t, err := n.surface.Current()
		if n == d.primary {
			d.world.Stats().Record("acquire", time.Since(began))
		}
		if errors.Is(err, wgpu.ErrOccluded) {
			return false
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "desktop:", err)
			return false
		}
		if t != nil {
			n.frame = t
			d.gpu.SetTarget(n.window, t)
			return true
		}
		n.configured = [2]int{}
	}
	return false
}

// offscreen gives a window a texture of its framebuffer's size to draw
// into when its frame isn't shown: a hidden window's surface has no
// drawable to give (macOS counts it occluded), and captures read it.
func (d *driver) offscreen(n *native, w, h int) bool {
	if n.hiddenFrame == nil || n.hiddenFrame.Width != w || n.hiddenFrame.Height != h {
		if n.hiddenFrame != nil {
			n.hiddenFrame.Release()
		}
		n.hiddenFrame = d.gpu.Device.CreateTexture(
			w,
			h,
			d.format,
			wgpu.UsageRenderAttachment|wgpu.UsageCopySrc|wgpu.UsageTextureBinding,
		)
	}
	n.frame = n.hiddenFrame
	d.gpu.SetTarget(n.window, n.frame)
	return true
}

// show presents n's frame, if it drew one on its surface.
func (d *driver) show(n *native) {
	if n.frame == nil {
		return
	}
	if n.frame != n.hiddenFrame {
		n.surface.Present()
		n.frame.Release()
	}
	n.frame = nil
	d.gpu.SetTarget(n.window, nil)
}

// held is whether a frame drawn since the last shown one is kept
// offscreen: without vsync, until a display refresh has passed.
func held(vsync bool, since, interval time.Duration) bool { return !vsync && since < interval }

// refreshInterval is how long m shows a frame, a 60 Hz display's when it
// doesn't say.
func refreshInterval(m *glfw.Monitor) time.Duration {
	hz := 60
	if m != nil {
		if mode := m.GetVideoMode(); mode != nil && mode.RefreshRate > 0 {
			hz = mode.RefreshRate
		}
	}
	return time.Second / time.Duration(hz)
}
