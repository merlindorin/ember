//go:build gpu

package gpu //nolint:testpackage // white-box: drives the renderer's unexported passes

import (
	"image"
	stdcolor "image/color"
	"math"
	"testing"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/internal/draw"
	"github.com/merlindorin/ember/internal/wgpu"
	"github.com/merlindorin/ember/internal/wm"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/world"
)

// plain draws a 200×100 window of background bg holding ents, unlit, a
// point a pixel.
func plain(t *testing.T, bg color.Color, ents ...twod.Entity) *image.RGBA {
	t.Helper()
	d := device(t)
	r := &webgpu2D{}
	if err := r.init(d); err != nil {
		t.Fatal(err)
	}
	w := world.New().AddPlugins(wm.Plugin{}, twod.Plugin{})
	win := platform.NewWindow(platform.Size(200, 100), platform.Background(bg))
	win.Add(twod.NewCamera(twod.At(100, 50)))
	layer := twod.NewLayer(0)
	for _, e := range ents {
		layer.Add(e)
	}
	win.Add(layer)
	w.AddEntities(win)
	w.RunFrames(2)
	target := d.CreateTexture(200, 100, wgpu.RGBA8Unorm, wgpu.UsageRenderAttachment|wgpu.UsageCopySrc)
	r.window(w, win, target, d.NewTimers(), draw.PassesFor(w, win), draw.BackFor(w, win), draw.FrontFor(w, win))
	if err := wgpu.TakeError(); err != nil {
		t.Fatal(err)
	}
	img, err := d.Read(target)
	if err != nil {
		t.Fatal(err)
	}
	return img
}

func rect(x, y, w, h float32, hex string) *twod.Polygon {
	return twod.NewPolygon(
		[]geom.Vec2{{X: x, Y: y}, {X: x + w, Y: y}, {X: x + w, Y: y + h}, {X: x, Y: y + h}},
		twod.Color(color.Hex(hex)),
	)
}

func TestColoursComeOutByteForByte(t *testing.T) {
	img := plain(t, color.Hex("#1e1e1e"),
		rect(10, 10, 40, 40, "#808080"),
		rect(60, 10, 40, 40, "#f0f0f0"),
		rect(110, 10, 40, 40, "#1e1e1e"),
		rect(160, 10, 30, 40, "#0a0b0c"))
	for _, c := range []struct {
		x, y int
		want stdcolor.RGBA
	}{
		{5, 90, stdcolor.RGBA{0x1e, 0x1e, 0x1e, 0xff}},
		{30, 30, stdcolor.RGBA{0x80, 0x80, 0x80, 0xff}},
		{80, 30, stdcolor.RGBA{0xf0, 0xf0, 0xf0, 0xff}},
		{130, 30, stdcolor.RGBA{0x1e, 0x1e, 0x1e, 0xff}},
		{175, 30, stdcolor.RGBA{0x0a, 0x0b, 0x0c, 0xff}},
	} {
		if got := img.RGBAAt(c.x, c.y); got != c.want {
			t.Errorf("at %d,%d: %v, want %v, as color.Hex wrote it", c.x, c.y, got, c.want)
		}
	}
}

func TestAPictureRoundTripsByteForByte(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 128, 64))
	for y := range 64 {
		for x := range 128 {
			v := uint8((x + y*128) % 256)
			src.SetRGBA(x, y, stdcolor.RGBA{v, uint8(255 - int(v)), uint8(x * 2), 255})
		}
	}
	img := plain(t, color.Black, twod.NewSprite(twod.NewTexture(src), twod.At(36, 18), twod.Pivot(0, 0)))
	for y := range 64 {
		for x := range 128 {
			if got, want := img.RGBAAt(36+x, 18+y), src.RGBAAt(x, y); got != want {
				t.Fatalf("texel %d,%d drawn 1:1: %v, want %v", x, y, got, want)
			}
		}
	}
}

// midway draws a 2×1 image of greys 59 and 193 stretched over the 200×100
// window, its texel centres at x 50 and 150, and gives the grey midway.
func midway(t *testing.T, opts ...twod.TextureOption) uint8 {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 1))
	img.Pix = []byte{59, 59, 59, 255, 193, 193, 193, 255}
	s := twod.NewSprite(twod.NewTexture(img, opts...), twod.At(100, 50), twod.Size(200, 100))
	return plain(t, color.Black, s).RGBAAt(100, 50).G
}

func TestASpriteFiltersInLinearLightAndAUIImageInEncodedSRGB(t *testing.T) {
	t.Logf("midway between 59 and 193: sprite %d, Encoded %d", midway(t), midway(t, twod.Encoded()))
	if v := midway(t); v < 145 || v > 149 {
		t.Fatalf("a sprite blends 59 and 193 in linear light, 147: %d", v)
	}
	if v := midway(t, twod.Encoded()); v < 124 || v > 128 {
		t.Fatalf("an Encoded texture blends them as Chrome does, in encoded sRGB, 126: %d", v)
	}
}

func TestARenderTexturesMaskLeavesItsCornersClear(t *testing.T) {
	d := device(t)
	r := &webgpu2D{}
	if err := r.init(d); err != nil {
		t.Fatal(err)
	}
	w := world.New().AddPlugins(wm.Plugin{}, twod.Plugin{})
	win := platform.NewWindow(platform.Size(200, 100), platform.Background(color.White))
	var ring []geom.Vec2
	for i := range 48 {
		a := float64(i) / 48 * 2 * math.Pi
		ring = append(ring, geom.V2(float32(32*math.Cos(a)), float32(32*math.Sin(a))))
	}
	disc := twod.NewLayer(5, twod.AsMask()).Add(twod.NewPolygon(ring, twod.Color(color.White)))
	inside := twod.NewLayer(6, twod.MaskedBy(disc)).Add(rect(-32, -32, 64, 64, "#ff0000"))
	minimap := twod.NewRenderTexture(64, 64)
	hud := twod.NewLayer(10, twod.Fixed()).Add(twod.NewSprite(minimap, twod.At(100, 50), twod.Size(64, 64)))
	win.Add(
		twod.NewCamera(twod.Layers(hud)),
		twod.NewCamera(twod.Into(minimap), twod.Layers(disc, inside)),
		disc,
		inside,
		hud,
	)
	w.AddEntities(win)
	w.RunFrames(2)
	target := d.CreateTexture(200, 100, wgpu.RGBA8Unorm, wgpu.UsageRenderAttachment|wgpu.UsageCopySrc)
	r.window(w, win, target, d.NewTimers(), draw.PassesFor(w, win), draw.BackFor(w, win), draw.FrontFor(w, win))
	if err := wgpu.TakeError(); err != nil {
		t.Fatal(err)
	}
	img, err := d.Read(target)
	if err != nil {
		t.Fatal(err)
	}
	if c := img.RGBAAt(100, 50); c.R < 250 || c.G > 5 {
		t.Fatalf("inside the disc the texture is red: %v", c)
	}
	if c := img.RGBAAt(70, 20); c.R < 250 || c.G < 250 || c.B < 250 {
		t.Fatalf("its corner is clear, the window showing through: %v", c)
	}
}
