//go:build gpu

package gpu //nolint:testpackage // white-box: drives the renderer's unexported passes

import (
	"image"
	stdcolor "image/color"
	"math"
	"testing"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/internal/draw"
	"github.com/lantervale/ember/internal/wgpu"
	"github.com/lantervale/ember/internal/wm"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/twod/light"
	"github.com/lantervale/ember/world"
)

// scene draws a 200×100 window under a black ambient holding ents, and
// hands back the picture and the window's lighting.
func scene(t *testing.T, ents ...twod.Entity) (*image.RGBA, *draw.Lighting) {
	t.Helper()
	d := device(t)
	r := &webgpu2D{}
	if err := r.init(d); err != nil {
		t.Fatal(err)
	}
	w := world.New().AddPlugins(wm.Plugin{}, twod.Plugin{}, light.Plugin{})
	win := platform.NewWindow(platform.Size(200, 100))
	win.Add(twod.NewCamera(light.Ambient(color.Black)))
	for _, e := range ents {
		win.Add(e)
	}
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
	return img, draw.LightingFor(w, win)
}

// white is a lit layer z of a white rectangle w×h centred at x, y, with
// opts on the layer.
func white(z, x, y, w, h int, opts ...twod.Option) *twod.Layer {
	all := []twod.LayerOption{light.Lit()}
	for _, o := range opts {
		all = append(all, o)
	}
	layer := twod.NewLayer(z, all...)
	return layer.Add(twod.NewSprite(filled(w, h, stdcolor.RGBA{255, 255, 255, 255}), twod.At(float32(x), float32(y))))
}

func lamp(x, y float32, opts ...light.Option) *light.Light {
	return light.NewPoint(300, color.White, append([]light.Option{light.Falloff(1), light.At(x, y)}, opts...)...)
}

func TestALightAffectsOnlyItsLayers(t *testing.T) {
	floor := white(0, 0, 0, 200, 100)
	crates := white(1, 40, 0, 20, 20)
	img, _ := scene(t, floor, crates, lamp(-60, 0, light.Affects(floor)))
	if v := at(img, -40, 30); v < 250 {
		t.Fatalf("the floor is lit: %d", v)
	}
	if v := at(img, 40, 0); v > 5 {
		t.Fatalf("the crates' layer is not: %d", v)
	}
}

func TestALightAffectsTheLayersNestedInItsLayers(t *testing.T) {
	inner := white(1, 40, 0, 20, 20)
	floor := white(0, 0, 0, 200, 100).Add(inner)
	other := white(2, -40, 0, 20, 20)
	img, _ := scene(t, floor, other, lamp(0, -40, light.Affects(floor)))
	if v := at(img, 40, 0); v < 250 {
		t.Fatalf("a layer nested in the floor is lit as the floor: %d", v)
	}
	if v := at(img, -40, 0); v > 5 {
		t.Fatalf("one outside it is not: %d", v)
	}
}

func TestEachLayerTakesTheLightsAffectingIt(t *testing.T) {
	floor := white(0, 0, 0, 200, 100)
	left := white(1, -40, 0, 20, 20)
	right := white(2, 40, 0, 20, 20)
	red := light.NewPoint(300, color.RGB(1, 0, 0), light.Falloff(1), light.Affects(left, floor))
	blue := light.NewPoint(300, color.RGB(0, 0, 1), light.Falloff(1), light.Affects(right, floor))
	green := light.NewPoint(300, color.RGB(0, 1, 0), light.Falloff(1))
	img, li := scene(t, floor, left, right, red, blue, green)
	if c := rgbAt(img, -40, 0); c.R < 250 || c.G < 250 || c.B > 5 {
		t.Fatalf("the left layer: red and green: %v", c)
	}
	if c := rgbAt(img, 40, 0); c.R > 5 || c.G < 250 || c.B < 250 {
		t.Fatalf("the right layer: blue and green: %v", c)
	}
	if c := rgbAt(img, 0, 40); c.R < 250 || c.G < 250 || c.B < 250 {
		t.Fatalf("the floor: all three: %v", c)
	}
	if len(li.Plan.Pictures) != 2 || len(li.Plan.Fallen) != 0 {
		t.Fatalf("two pictures, the floor reading the window's: %+v", li.Plan)
	}
}

func TestAffectsAndShadedByTogether(t *testing.T) {
	floor := white(0, 0, 0, 200, 100)
	crowd := white(1, 40, 0, 20, 20, light.ShadedBy(0, 1))
	box := twod.NewLayer(3).
		Add(twod.NewSprite(filled(4, 30, stdcolor.RGBA{0, 0, 0, 0}), twod.At(10, 0), light.Occluder(2)))
	only := lamp(-60, 0, light.Affects(floor))
	every := light.NewPoint(300, color.RGB(0, 0, 1), light.Falloff(1), light.At(-60, 0))
	img, _ := scene(t, floor, crowd, box, only, every)
	if c := rgbAt(img, 40, 0); c.R > 5 || c.B < 250 {
		t.Fatalf("the crowd: not the floor's lamp, and the blue one unshaded by group 2: %v", c)
	}
	if c := rgbAt(img, 40, 30); c.R < 250 {
		t.Fatalf("the floor beside it: its lamp: %v", c)
	}
}

func TestADirectionalLightAffectsOnlyItsLayers(t *testing.T) {
	floor := white(0, 0, 0, 200, 100)
	crates := white(1, 40, 0, 20, 20)
	img, _ := scene(t, floor, crates, light.NewDirectional(color.White, math.Pi/2, light.Affects(floor)))
	if v := at(img, -40, 30); v < 250 {
		t.Fatalf("the sun lights the floor: %d", v)
	}
	if v := at(img, 40, 0); v > 5 {
		t.Fatalf("not the crates' layer: %d", v)
	}
}

func TestALightLeftOutTakesItsSilhouettesWithIt(t *testing.T) {
	floor := white(0, 0, 0, 200, 100)
	patch := white(1, 20, 10, 30, 6)
	hero := twod.NewLayer(2).
		Add(twod.NewSprite(filled(10, 20, stdcolor.RGBA{255, 255, 255, 255}), twod.At(0, 0), light.Silhouette()))
	only := lamp(-60, 10, light.Affects(floor))
	half := light.NewPoint(
		300,
		color.White,
		light.Falloff(1),
		light.At(-60, 10),
		light.Intensity(0.5),
		light.Shadows(false),
	)
	img, _ := scene(t, floor, patch, hero, only, half)
	if v := at(img, 12, 5); !near(v, srgb(0.5), 6) {
		t.Fatalf("on the floor, in the hero's shadow, only the half lamp: %d", v)
	}
	if v := at(img, 12, 10); !near(v, srgb(0.5), 6) {
		t.Fatalf(
			"on the patch, left out of the floor's lamp, the half lamp whole, the shadow gone with its lamp: %d",
			v,
		)
	}
}

func TestPastFourPicturesALayerFallsBackToEveryLight(t *testing.T) {
	floor := white(0, 0, 0, 200, 100)
	var ents []twod.Entity
	ents = append(ents, floor)
	var layers []*twod.Layer
	for i := range 5 {
		layers = append(layers, white(1+i, -80+40*i, 0, 10, 10))
	}
	for _, l := range layers {
		ents = append(ents, l, lamp(0, -45, light.Affects(l, floor), light.Intensity(0.1)))
	}
	img, li := scene(t, ents...)
	if len(li.Plan.Pictures) != draw.MaxPictures || len(li.Plan.Fallen) == 0 {
		t.Fatalf("four pictures and the rest fallen back: %+v", li.Plan)
	}
	brighter := 0
	for i := range layers {
		if at(img, -80+40*i, 0) > at(img, -80, 0) || at(img, -80+40*i, 0) > at(img, -40, 0) {
			brighter++
		}
	}
	if brighter != len(li.Plan.Fallen) {
		t.Fatalf(
			"each layer fallen back is lit by every lamp, brighter than those lit by their own: %d of %d",
			brighter,
			len(li.Plan.Fallen),
		)
	}
}

func TestExposureScalesTheLightLitDrawingsRead(t *testing.T) {
	read := func(opts ...twod.Option) (uint8, uint8) {
		d := device(t)
		r := &webgpu2D{}
		if err := r.init(d); err != nil {
			t.Fatal(err)
		}
		w := world.New().AddPlugins(wm.Plugin{}, twod.Plugin{}, light.Plugin{})
		win := platform.NewWindow(platform.Size(200, 100))
		cam := twod.NewCamera(append([]twod.CameraOption{light.Ambient(color.Black)}, func() []twod.CameraOption {
			var out []twod.CameraOption
			for _, o := range opts {
				out = append(out, o)
			}
			return out
		}()...)...)
		win.Add(cam, white(0, 0, 0, 200, 100),
			light.NewPoint(30, color.White, light.Falloff(1), light.At(-50, 0), light.Intensity(0.25)),
			light.NewPoint(30, color.White, light.Falloff(1), light.At(50, 0)))
		w.AddEntities(win)
		w.RunFrames(2)
		target := d.CreateTexture(200, 100, wgpu.RGBA8Unorm, wgpu.UsageRenderAttachment|wgpu.UsageCopySrc)
		r.window(w, win, target, d.NewTimers(), draw.PassesFor(w, win), draw.BackFor(w, win), draw.FrontFor(w, win))
		img, err := d.Read(target)
		if err != nil {
			t.Fatal(err)
		}
		return at(img, -50, 0), at(img, 50, 0)
	}
	half, full := read()
	if !near(half, srgb(0.25), 6) || full < 250 {
		t.Fatalf("as is: a quarter and full: %d %d", half, full)
	}
	half, full = read(light.Exposure(2))
	if !near(half, srgb(0.5), 6) || full < 250 {
		t.Fatalf("Exposure(2) doubles the quarter, full stays full: %d %d", half, full)
	}
	half, full = read(light.Exposure(0.5))
	if !near(half, srgb(0.125), 6) || !near(full, srgb(0.5), 6) {
		t.Fatalf("Exposure(0.5) halves both: %d %d", half, full)
	}
}
