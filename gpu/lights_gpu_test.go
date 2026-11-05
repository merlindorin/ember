//go:build gpu

package gpu //nolint:testpackage // white-box: drives the renderer's unexported passes

import (
	"image"
	stdcolor "image/color"
	"math"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/merlindorin/ember/collide"
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/internal/draw"
	"github.com/merlindorin/ember/internal/wgpu"
	"github.com/merlindorin/ember/internal/wm"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/twod/light"
	"github.com/merlindorin/ember/world"
)

// device is the one device every GPU test here shares: the binding frees
// none, and a software adapter (WARP on CI) runs out of memory for one a
// test.
func device(t *testing.T) *wgpu.Device {
	t.Helper()
	gpuOnce.Do(func() {
		inst, err := wgpu.NewInstance()
		if err != nil {
			errDevice = err
			return
		}
		ad, err := inst.RequestAdapter(nil)
		if err != nil {
			errDevice, gpuMissing = err, true
			return
		}
		gpuDevice, errDevice = ad.RequestDevice()
	})
	if gpuMissing && os.Getenv(wgpu.OptionalEnv) == "1" {
		t.Skipf("no GPU adapter here, and %s=1: %v", wgpu.OptionalEnv, errDevice)
	}
	if errDevice != nil {
		t.Fatal(errDevice)
	}
	return gpuDevice
}

//nolint:gochecknoglobals // the shared device, made once for the package's tests
var (
	gpuOnce    sync.Once
	gpuDevice  *wgpu.Device
	errDevice  error
	gpuMissing bool
)

func filled(w, h int, c stdcolor.RGBA) *twod.Texture {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = c.R, c.G, c.B, c.A
	}
	return twod.NewTexture(img)
}

// lit draws a 200×100 window whose camera centres the world's origin: a
// white floor on a lit layer, a light at -60, 0, and whatever extra adds,
// and gives back the window's pixels.
func lit(t *testing.T, extra ...twod.Entity) *image.RGBA {
	t.Helper()
	return litDebug(t, nil, extra...)
}

// litDebug is lit with debug, if any, setting the window's debug view up.
func litDebug(t *testing.T, debug func(w *world.World, win *platform.Window), extra ...twod.Entity) *image.RGBA {
	t.Helper()
	d := device(t)
	r := &webgpu2D{}
	if err := r.init(d); err != nil {
		t.Fatal(err)
	}
	w := world.New().AddPlugins(wm.Plugin{}, twod.Plugin{}, light.Plugin{})
	win := platform.NewWindow(platform.Size(200, 100))
	win.Add(twod.NewCamera(light.Ambient(color.Black)))
	floor := twod.NewLayer(0, light.Lit()).Add(twod.NewSprite(filled(200, 100, stdcolor.RGBA{255, 255, 255, 255})))
	things := twod.NewLayer(1).Add(light.NewPoint(120, color.White, light.Falloff(1), light.At(-60, 0)))
	win.Add(floor, things)
	for _, e := range extra {
		if l, ok := e.(*twod.Layer); ok {
			win.Add(l)
		} else {
			things.Add(e)
		}
	}
	w.AddEntities(win)
	w.RunFrames(2)
	if debug != nil {
		debug(w, win)
	}
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

// at is the grey level of the window's pixel showing the world's x, y.
func at(img *image.RGBA, x, y int) uint8 { return img.RGBAAt(100+x, 50+y).R }

func TestLightPicturesTheDeviceRefusesAreAnErrorNamingTheirFormatAndSize(t *testing.T) {
	d := device(t)
	r := &webgpu2D{}
	if err := r.init(d); err != nil {
		t.Fatal(err)
	}
	win := platform.NewWindow(platform.Size(200, 100))
	if _, err := r.lightPicturesFor(win, geom.V2(200, 100), geom.V2(200000, 100)); err == nil ||
		!strings.Contains(err.Error(), "rgba16float") || !strings.Contains(err.Error(), "100000×50") {
		t.Fatalf("a refused light picture names its format and size: %v", err)
	} else {
		t.Log(err)
	}
	if r.lights.pictures[win] != nil {
		t.Fatal("nothing half-made is kept")
	}
	if p, err := r.lightPicturesFor(win, geom.V2(200, 100), geom.V2(200, 100)); err != nil || p == nil {
		t.Fatalf("pictures the device can make are made: %v", err)
	}
}

func TestALightBrightensALitLayerAndAmbientDarknessTheRest(t *testing.T) {
	img := lit(t)
	if v := at(img, -60, 0); v < 250 {
		t.Fatalf("under the light the floor is white: %d", v)
	}
	if v := at(img, 30, 0); v < 250 {
		t.Fatalf("nothing blocks it at 90 points, within its radius: %d", v)
	}
	if v := at(img, 99, 49); v > 5 {
		t.Fatalf("past its radius the floor takes the black ambient: %d", v)
	}
}

func TestAnOccluderCastsAShadowAndAnXRayLightShinesThrough(t *testing.T) {
	wall := twod.NewSprite(filled(6, 100, stdcolor.RGBA{}), twod.At(0, 0), light.Occluder())
	img := lit(t, wall)
	if v := at(img, -30, 0); v < 250 {
		t.Fatalf("before the wall the floor is lit: %d", v)
	}
	if v := at(img, 30, 0); v > 5 {
		t.Fatalf("behind the wall it is in shadow: %d", v)
	}
	wall = twod.NewSprite(filled(6, 100, stdcolor.RGBA{}), twod.At(0, 0), light.Occluder())
	img = lit(t, wall, light.NewPoint(120, color.White, light.Falloff(1), light.At(-60, 0), light.Shadows(false)))
	if v := at(img, 30, 0); v < 250 {
		t.Fatalf("an x-ray light reaches behind the wall: %d", v)
	}
}

func TestUnlitAndLitAtSpritesOnALitLayer(t *testing.T) {
	marker := filled(4, 4, stdcolor.RGBA{255, 255, 255, 255})
	d := device(t)
	r := &webgpu2D{}
	if err := r.init(d); err != nil {
		t.Fatal(err)
	}
	w := world.New().AddPlugins(wm.Plugin{}, twod.Plugin{}, light.Plugin{})
	win := platform.NewWindow(platform.Size(200, 100))
	win.Add(twod.NewCamera(light.Ambient(color.Black)))
	layer := twod.NewLayer(0, light.Lit()).Add(
		twod.NewSprite(marker, twod.At(80, 0), light.Unlit()),
		twod.NewSprite(marker, twod.At(80, 30), light.LitAt(-140, -30)),
		light.NewPoint(30, color.White, light.Falloff(1), light.At(-60, 0)),
	)
	win.Add(layer)
	w.AddEntities(win)
	w.RunFrames(2)
	target := d.CreateTexture(200, 100, wgpu.RGBA8Unorm, wgpu.UsageRenderAttachment|wgpu.UsageCopySrc)
	r.window(w, win, target, d.NewTimers(), draw.PassesFor(w, win), draw.BackFor(w, win), draw.FrontFor(w, win))
	out, err := d.Read(target)
	if err != nil {
		t.Fatal(err)
	}
	if v := at(out, 80, 0); v < 250 {
		t.Fatalf("an unlit sprite in the dark shows as it is: %d", v)
	}
	if v := at(out, 80, 30); v < 250 {
		t.Fatalf("a sprite lit at a point under the light is lit whole, far from it: %d", v)
	}
}

// shine draws a 200×100 window whose camera centres the world's origin,
// in the dark: a white floor on a lit layer, and ents over it.
func shine(t *testing.T, ents ...twod.Entity) *image.RGBA {
	t.Helper()
	d := device(t)
	r := &webgpu2D{}
	if err := r.init(d); err != nil {
		t.Fatal(err)
	}
	w := world.New().AddPlugins(wm.Plugin{}, twod.Plugin{}, light.Plugin{})
	win := platform.NewWindow(platform.Size(200, 100))
	win.Add(twod.NewCamera(light.Ambient(color.Black)))
	floor := twod.NewLayer(0, light.Lit()).Add(twod.NewSprite(filled(200, 100, stdcolor.RGBA{255, 255, 255, 255})))
	things := twod.NewLayer(1)
	for _, e := range ents {
		things.Add(e)
	}
	win.Add(floor, things)
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

func wallAt(x, y float32, w, h int, opts ...twod.Option) *twod.Sprite {
	s := twod.NewSprite(filled(w, h, stdcolor.RGBA{}), twod.At(x, y))
	for _, o := range append([]twod.Option{light.Occluder()}, opts...) {
		o(s)
	}
	return s
}

func TestAConeShinesOnlyWhereItPoints(t *testing.T) {
	img := shine(t, light.NewCone(120, color.White, 0, math.Pi/2, light.Falloff(1), light.At(-60, 0)))
	if v := at(img, -20, 0); v < 250 {
		t.Fatalf("ahead of it: %d", v)
	}
	if v := at(img, -60, -40); v > 5 {
		t.Fatalf("90° aside, out of its 90° spread: %d", v)
	}
	if v := at(img, -90, 0); v > 5 {
		t.Fatalf("behind it: %d", v)
	}
}

func TestADirectionalLightCrossesTheViewAndCastsLongShadows(t *testing.T) {
	sun := light.NewDirectional(color.White, math.Pi/2, light.ShadowLength(60))
	img := shine(t, sun, wallAt(0, -30, 20, 6))
	if v := at(img, 80, 40); v < 250 {
		t.Fatalf("it lights the whole view: %d", v)
	}
	if v := at(img, 0, -10); v > 5 {
		t.Fatalf("under the wall, its rays going down, is its shadow: %d", v)
	}
	if v := at(img, 0, 40); v < 250 {
		t.Fatalf("past ShadowLength the shadow ends: %d", v)
	}
	img = shine(t, light.NewDirectional(color.White, math.Pi/2, light.Shadows(false)), wallAt(0, -30, 20, 6))
	if v := at(img, 0, -10); v < 250 {
		t.Fatalf("an x-ray one casts none: %d", v)
	}
}

func TestAShapeLightIsFullInsideItsShapeAndFadesOverItsReach(t *testing.T) {
	img := shine(t, light.NewShape(light.Rect(40, 20), color.White, 30, light.At(-60, 0)))
	if v := at(img, -45, 5); v < 250 {
		t.Fatalf("inside the rectangle: %d", v)
	}
	if v := at(img, -60+20+15, 0); v < 20 || v > 235 {
		t.Fatalf("halfway through its reach, part lit: %d", v)
	}
	if v := at(img, -60+20+32, 0); v > 5 {
		t.Fatalf("past its reach: %d", v)
	}
	img = shine(
		t,
		light.NewShape(
			light.Polygon(geom.V2(0, -30), geom.V2(40, 30), geom.V2(-40, 30)),
			color.White,
			10,
			light.At(40, 0),
		),
	)
	if v := at(img, 40, 15); v < 250 {
		t.Fatalf("inside the triangle: %d", v)
	}
	if v := at(img, 10, -20); v > 5 {
		t.Fatalf("well outside it: %d", v)
	}
	img = shine(t, light.NewShape(light.Line(geom.V2(-40, 0), geom.V2(40, 0)), color.White, 20, light.Falloff(1)))
	if v := at(img, 30, 10); v < 250 {
		t.Fatalf("a line's light, within its reach: %d", v)
	}
}

func TestAShapeLightWithShadowsIsSoftenedByItsOutline(t *testing.T) {
	window := light.NewShape(
		light.Rect(60, 10),
		color.White,
		140,
		light.At(-60, 0),
		light.Shadows(true),
		light.Falloff(1),
	)
	img := shine(t, window, wallAt(0, 0, 6, 30))
	if v := at(img, -20, 0); v < 250 {
		t.Fatalf("before the wall: %d", v)
	}
	if v := at(img, 40, 0); v > 250 {
		t.Fatalf("behind the wall it is darker: %d", v)
	}
}

func TestOccluderGroupsAndBlockedBy(t *testing.T) {
	glass := wallAt(0, 0, 6, 100, light.Occluder(1))
	img := shine(t, light.NewPoint(120, color.White, light.Falloff(1), light.At(-60, 0), light.BlockedBy(0)), glass)
	if v := at(img, 30, 0); v < 250 {
		t.Fatalf("a light blocked only by group 0 shines through group 1: %d", v)
	}
	glass = wallAt(0, 0, 6, 100, light.Occluder(1))
	img = shine(t, light.NewPoint(120, color.White, light.Falloff(1), light.At(-60, 0)), glass)
	if v := at(img, 30, 0); v > 5 {
		t.Fatalf("by default every group blocks: %d", v)
	}
}

func TestAnEmissiveSpriteGlowsAndLightsRoundIt(t *testing.T) {
	lava := twod.NewSprite(
		filled(20, 20, stdcolor.RGBA{255, 255, 255, 255}),
		twod.At(0, 0),
		light.Emissive(20),
		light.Unlit(),
	)
	img := shine(t, lava)
	if v := at(img, 0, 0); v < 250 {
		t.Fatalf("the sprite itself shows: %d", v)
	}
	if v := at(img, 15, 0); v < 20 || v > 250 {
		t.Fatalf("5 points past its edge its glow lights the floor, softly: %d", v)
	}
	if v := at(img, 60, 0); v > 5 {
		t.Fatalf("past its reach, dark: %d", v)
	}
}

func TestASmallEmissiveSpriteGlowsFullAtItsEdgeHoweverFarItReaches(t *testing.T) {
	ember := twod.NewSprite(
		filled(6, 4, stdcolor.RGBA{255, 255, 255, 255}),
		twod.At(0, 0),
		light.Emissive(60),
		light.Unlit(),
	)
	img := shine(t, ember)
	if v := at(img, 8, 0); v < 150 {
		t.Fatalf("just past its edge, its glow is strong, not spread thin over its reach: %d", v)
	}
	if v := at(img, 3+50, 0); v < 3 || v > 150 {
		t.Fatalf("most of its reach away, faint: %d", v)
	}
}

func TestNoLightLeaksBetweenOccludersTouchingAtACorner(t *testing.T) {
	img := shine(t, light.NewPoint(140, color.White, light.Falloff(1), light.At(-40, 40)),
		wallAt(-5, -5, 10, 10), wallAt(5, 5, 10, 10))
	for _, d := range []int{12, 20, 30} {
		if v := at(img, d, -d); v > 5 {
			t.Fatalf("past the corner the two walls share, %d points along the diagonal, it is dark: %d", d, v)
		}
	}
}

func TestProbesAnswerTheLightTheFloorShows(t *testing.T) {
	d := device(t)
	r := &webgpu2D{}
	if err := r.init(d); err != nil {
		t.Fatal(err)
	}
	w := world.New().AddPlugins(wm.Plugin{}, twod.Plugin{}, light.Plugin{})
	win := platform.NewWindow(platform.Size(200, 100))
	win.Add(twod.NewCamera(light.Ambient(color.RGB(0.05, 0.05, 0.05))))
	floor := twod.NewLayer(0, light.Lit()).Add(twod.NewSprite(filled(200, 100, stdcolor.RGBA{255, 255, 255, 255})))
	points := []geom.Vec2{{X: -40}, {X: 30}, {X: -60, Y: -45}, {X: 400}}
	var probes []*light.Probe
	things := twod.NewLayer(1).
		Add(light.NewPoint(120, color.White, light.Falloff(0.2), light.At(-60, 0)), wallAt(0, 0, 6, 60))
	for _, p := range points {
		probe := light.NewProbe(twod.At(p.X, p.Y))
		probes = append(probes, probe)
		things.Add(probe)
	}
	win.Add(floor, things)
	w.AddEntities(win)
	target := d.CreateTexture(200, 100, wgpu.RGBA8Unorm, wgpu.UsageRenderAttachment|wgpu.UsageCopySrc)
	timers := d.NewTimers()
	for range 8 {
		w.RunFrames(1)
		r.window(w, win, target, timers, draw.PassesFor(w, win), draw.BackFor(w, win), draw.FrontFor(w, win))
		d.Poll()
	}
	if err := wgpu.TakeError(); err != nil {
		t.Fatal(err)
	}
	img, err := d.Read(target)
	if err != nil {
		t.Fatal(err)
	}
	for i, p := range points[:3] {
		if !probes[i].Ready() || probes[i].Stale() {
			t.Fatalf("probe %d at %v is answered: ready %v, stale %v", i, p, probes[i].Ready(), probes[i].Stale())
		}
		drawn := math.Pow(float64(at(img, int(p.X), int(p.Y)))/255, 2.2)
		if got := float64(probes[i].Light().R); math.Abs(got-drawn) > 0.04 {
			t.Fatalf("probe %d at %v reads %.3f, the floor shows %.3f", i, p, got, drawn)
		}
	}
	if l := probes[1].Light().R; l > 0.1 {
		t.Fatalf("behind the wall, a probe reads the shadow: %.3f", l)
	}
	if l := probes[0].Light().R; l < 0.9 {
		t.Fatalf("near the light, it reads it full: %.3f", l)
	}
	if !probes[3].Stale() {
		t.Fatal("a probe out of view is stale")
	}
}

func TestATiledPictureShowsWhatIsPaintedAndARepaintPatchesItsPage(t *testing.T) {
	d := device(t)
	r := &webgpu2D{}
	if err := r.init(d); err != nil {
		t.Fatal(err)
	}
	shade := uint8(200)
	pic := twod.NewTiledPicture(func(pt twod.PictureTile) {
		for y := pt.Area.Min.Y; y < pt.Area.Max.Y; y++ {
			for x := pt.Area.Min.X; x < pt.Area.Max.X; x++ {
				pt.Image.SetRGBA(x, y, stdcolor.RGBA{shade, shade, shade, 255})
			}
		}
	}, twod.TexelSize(10, 10), twod.TileTexels(16), twod.Pixelated())
	w := world.New().AddPlugins(wm.Plugin{}, twod.Plugin{})
	win := platform.NewWindow(platform.Size(200, 100))
	win.Add(twod.NewCamera(), twod.NewLayer(0).Add(pic))
	w.AddEntities(win)
	target := d.CreateTexture(200, 100, wgpu.RGBA8Unorm, wgpu.UsageRenderAttachment|wgpu.UsageCopySrc)
	frame := func() *image.RGBA {
		w.RunFrames(1)
		r.window(w, win, target, d.NewTimers(), draw.PassesFor(w, win), draw.BackFor(w, win), draw.FrontFor(w, win))
		img, err := d.Read(target)
		if err != nil {
			t.Fatal(err)
		}
		return img
	}
	if v := at(frame(), 20, 20); v < 198 || v > 202 {
		t.Fatalf("the painted texels show: %d", v)
	}
	before := map[*wgpu.Texture]bool{}
	for _, c := range r.images {
		before[c.gpu] = true
	}
	shade = 60
	pic.MarkDirty(geom.Rect{X: 20, Y: 20, W: 1, H: 1})
	img := frame()
	if v := at(img, 25, 25); v < 58 || v > 62 {
		t.Fatalf("the repainted texel shows its new shade: %d", v)
	}
	if v := at(img, 45, 25); v < 198 || v > 202 {
		t.Fatalf("its neighbour is untouched: %d", v)
	}
	for _, c := range r.images {
		if !before[c.gpu] {
			t.Fatal("the page was patched, not made again")
		}
	}
}

// ambientShine draws the lit white floor of shine under an ambient map
// whose texels, 10 points each, are 0.5 of light left of x = 0 and 0.1
// right of it.
func ambientShine(t *testing.T, opts ...twod.TiledPictureOption) *image.RGBA {
	t.Helper()
	d := device(t)
	r := &webgpu2D{}
	if err := r.init(d); err != nil {
		t.Fatal(err)
	}
	half, tenth := color.Gray(0.5).SRGB(), color.Gray(0.1).SRGB()
	ambient := twod.NewTiledPicture(func(pt twod.PictureTile) {
		for y := pt.Area.Min.Y; y < pt.Area.Max.Y; y++ {
			for x := pt.Area.Min.X; x < pt.Area.Max.X; x++ {
				c := half
				if cx, _ := pt.Cell(x, y); cx >= 0 {
					c = tenth
				}
				pt.Image.SetRGBA(
					x,
					y,
					stdcolor.RGBA{uint8(c[0]*255 + 0.5), uint8(c[1]*255 + 0.5), uint8(c[2]*255 + 0.5), 255},
				)
			}
		}
	}, append([]twod.TiledPictureOption{twod.TexelSize(10, 10), twod.TileTexels(16)}, opts...)...)
	w := world.New().AddPlugins(wm.Plugin{}, twod.Plugin{}, light.Plugin{})
	win := platform.NewWindow(platform.Size(200, 100))
	win.Add(twod.NewCamera(light.AmbientMap(ambient)))
	win.Add(twod.NewLayer(0, light.Lit()).Add(twod.NewSprite(filled(200, 100, stdcolor.RGBA{255, 255, 255, 255}))))
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

func TestAnAmbientMapLightsEachSpotByItsTexel(t *testing.T) {
	img := ambientShine(t, twod.Pixelated())
	if v := at(img, -50, 0); v < 184 || v > 190 {
		t.Fatalf("left, half the light: %d", v)
	}
	if v := at(img, 50, 0); v < 86 || v > 92 {
		t.Fatalf("right, a tenth: %d", v)
	}
	if v := at(img, 2, 0); v < 86 || v > 92 {
		t.Fatalf("Pixelated, the step is sharp at x = 0: %d", v)
	}
	smooth := ambientShine(t)
	if v := at(smooth, 0, 0); v <= 92 || v >= 184 {
		t.Fatalf("by default it blends between the texels round x = 0: %d", v)
	}
}

// wallTile shines a light at -60, 0 on the lit floor over a column of
// tile 1 at x 0, a tileset made with opts, the map a solid of the
// camera's Occluders; and walks a body from -30, 35 to the right, off
// the light's way to the points checked.
func wallTile(t *testing.T, opts ...twod.TilesetOption) (*image.RGBA, float32) {
	t.Helper()
	d := device(t)
	r := &webgpu2D{}
	if err := r.init(d); err != nil {
		t.Fatal(err)
	}
	tiles := twod.NewTileset(filled(10, 10, stdcolor.RGBA{}), twod.Cell(10, 10), opts...)
	m := twod.NewTileMap(
		tiles,
		twod.Cells([][]int{{1}, {1}, {1}, {1}, {1}, {1}, {1}, {1}, {1}, {1}}),
		twod.CellSize(10, 10),
		twod.At(0, -50),
	)
	space := collide.NewSpace()
	space.AddSolid(m)
	w := world.New().AddPlugins(wm.Plugin{}, twod.Plugin{}, light.Plugin{})
	w.AddSystems(collide.NewTracker(space))
	win := platform.NewWindow(platform.Size(200, 100))
	win.Add(twod.NewCamera(light.Ambient(color.Black), light.OccludersFrom(space)))
	walker := twod.NewSprite(filled(4, 4, stdcolor.RGBA{}), twod.At(-30, 35))
	walker.Add(collide.NewBody(collide.Box(4, 4)))
	floor := twod.NewLayer(0, light.Lit()).Add(twod.NewSprite(filled(200, 100, stdcolor.RGBA{255, 255, 255, 255})))
	things := twod.NewLayer(1).Add(light.NewPoint(120, color.White, light.Falloff(1), light.At(-60, 0)), m, walker)
	win.Add(floor, things)
	w.AddEntities(win)
	w.RunFrames(2)
	space.Move(walker, geom.V2(60, 0))
	target := d.CreateTexture(200, 100, wgpu.RGBA8Unorm, wgpu.UsageRenderAttachment|wgpu.UsageCopySrc)
	r.window(w, win, target, d.NewTimers(), draw.PassesFor(w, win), draw.BackFor(w, win), draw.FrontFor(w, win))
	if err := wgpu.TakeError(); err != nil {
		t.Fatal(err)
	}
	img, err := d.Read(target)
	if err != nil {
		t.Fatal(err)
	}
	return img, walker.Position().X
}

func TestATilesShadowShapesBlockLightAndItsNoShadowLetsItThrough(t *testing.T) {
	img, walked := wallTile(t, twod.ShadowShapes(1, geom.Rect{W: 10, H: 10}))
	if v := at(img, 40, 0); v > 5 {
		t.Fatalf("a shadow-only tile casts a shadow: %d", v)
	}
	if walked < 29 {
		t.Fatalf("and lets the body walk through it: it reached %v", walked)
	}
	img, walked = wallTile(t, twod.Solid(1), twod.CastShadow(false, 1))
	if v := at(img, 40, 0); v < 250 {
		t.Fatalf("a NoShadow tile lets the light through: %d", v)
	}
	if walked > -2 {
		t.Fatalf("but stops the body: it reached %v", walked)
	}
}

// tallColumn draws a white tall solid tile, 10 wide and 30 high, standing
// on the cell 0..10 of a map whose layer is lit, the map a solid of the
// camera's Occluders, lit by a light at from; in front of it at x
// -20..-10, behind, a solid wall from top to bottom (2), another tall
// column on the same row (1), or nothing (0).
func tallColumn(t *testing.T, from geom.Vec2, behind int) *image.RGBA {
	t.Helper()
	d := device(t)
	r := &webgpu2D{}
	if err := r.init(d); err != nil {
		t.Fatal(err)
	}
	tiles := twod.NewTileset(
		filled(20, 30, stdcolor.RGBA{255, 255, 255, 255}),
		twod.Cell(10, 30),
		twod.Solid(1, 2),
		twod.Tall(1),
	)
	cells := make([][]int, 10)
	for y := range cells {
		cells[y] = []int{0, 0, 0}
		if behind == 2 {
			cells[y][0] = 2
		}
	}
	if behind == 1 {
		cells[5][0] = 1
	}
	cells[5][2] = 1
	m := twod.NewTileMap(tiles, twod.Cells(cells), twod.CellSize(10, 10), twod.At(-20, -50))
	space := collide.NewSpace()
	space.AddSolid(m)
	w := world.New().AddPlugins(wm.Plugin{}, twod.Plugin{}, light.Plugin{})
	win := platform.NewWindow(platform.Size(200, 100))
	win.Add(twod.NewCamera(light.Ambient(color.Black), light.OccludersFrom(space)))
	floor := twod.NewLayer(0, light.Lit()).Add(twod.NewSprite(filled(200, 100, stdcolor.RGBA{0, 0, 0, 255})))
	things := twod.NewLayer(1, light.Lit()).
		Add(light.NewPoint(150, color.White, light.Falloff(0.2), light.At(from.X, from.Y)), m)
	win.Add(floor, things)
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

func TestATallSolidTileIsLitFromAnySideNotByItsOwnShadow(t *testing.T) {
	for _, from := range []geom.Vec2{{X: 5, Y: 45}, {X: -40, Y: -40}, {X: 5, Y: -45}, {X: 50, Y: 5}} {
		img := tallColumn(t, from, 0)
		if v := at(img, 5, -5); v < 80 {
			t.Fatalf("a tall solid tile lit from %v draws dark: %d", from, v)
		}
	}
	if v := at(tallColumn(t, geom.V2(-60, 0), 2), 5, -5); v > 5 {
		t.Fatalf("behind a wall it stays dark: %d", v)
	}
}

// A tall tile whose blocker's shadow edge runs within baseFront of its
// sides takes light past them through the half-resolution light picture's
// blurred edge: the shadow-edge limit on the lighting roadmap's later list
// (docs/design/2d-lighting.md §8).
func TestATallTileAtAnotherTallTilesShadowEdgeStaysDark(t *testing.T) {
	t.Skip("known limit, docs/design/2d-lighting.md §8: draws 165")
	if v := at(tallColumn(t, geom.V2(-60, 5), 1), 5, -5); v > 5 {
		t.Fatalf("in another tall tile's shadow it stays dark: %d", v)
	}
}

// wallRow draws three white solid tiles, 10 points square, in a row from
// x -15, y 0, on a map whose layer is lit, the map a solid of the
// camera's Occluders, lit by a light at from.
func wallRow(t *testing.T, from geom.Vec2) *image.RGBA {
	t.Helper()
	d := device(t)
	r := &webgpu2D{}
	if err := r.init(d); err != nil {
		t.Fatal(err)
	}
	tiles := twod.NewTileset(filled(10, 10, stdcolor.RGBA{255, 255, 255, 255}), twod.Cell(10, 10), twod.Solid(1))
	m := twod.NewTileMap(tiles, twod.Cells([][]int{{1, 1, 1}}), twod.CellSize(10, 10), twod.At(-15, 0))
	space := collide.NewSpace()
	space.AddSolid(m)
	w := world.New().AddPlugins(wm.Plugin{}, twod.Plugin{}, light.Plugin{})
	win := platform.NewWindow(platform.Size(200, 100))
	win.Add(twod.NewCamera(light.Ambient(color.Black), light.OccludersFrom(space)))
	floor := twod.NewLayer(0, light.Lit()).Add(twod.NewSprite(filled(200, 100, stdcolor.RGBA{0, 0, 0, 255})))
	things := twod.NewLayer(1, light.Lit()).
		Add(light.NewPoint(150, color.White, light.Falloff(0.2), light.At(from.X, from.Y)), m)
	win.Add(floor, things)
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

func TestASolidTilesFaceIsLitFromTheRoomNeverFromBehindIt(t *testing.T) {
	if v := at(wallRow(t, geom.V2(0, 40)), 0, 5); v < 80 {
		t.Fatalf("a wall lit from in front draws its face lit, not in its own shadow: %d", v)
	}
	if v := at(wallRow(t, geom.V2(0, -40)), 0, 5); v > 5 {
		t.Fatalf("lit from behind, it stays dark: %d", v)
	}
}

// post is an occluder sprite 10 points wide and 40 high at x, standing
// height points tall when height > 0.
func post(x, height float32) *twod.Sprite {
	opts := []twod.SpriteOption{twod.At(x, 0), light.Occluder(), light.Unlit()}
	if height > 0 {
		opts = append(opts, light.Height(height))
	}
	return twod.NewSprite(filled(10, 40, stdcolor.RGBA{}), opts...)
}

func TestALightsElevationEndsTheShadowsOfWhatIsLowerThanIt(t *testing.T) {
	lamp := func(opts ...light.Option) *light.Light {
		return light.NewPoint(180, color.White, append([]light.Option{light.Falloff(1), light.At(-60, 0)}, opts...)...)
	}
	img := shine(t, lamp(), post(0, 20))
	if v := at(img, 80, 0); v > 5 {
		t.Fatalf("without an elevation, the shadow runs to the radius: %d", v)
	}
	img = shine(t, lamp(light.Elevation(40)), post(0, 20))
	if v := at(img, 30, 0); v > 5 {
		t.Fatalf("a post half the lamp's height shadows out to twice its distance (x 70): dark at 30, %d", v)
	}
	if v := at(img, 85, 0); v < 200 {
		t.Fatalf("and the floor is lit past it: %d", v)
	}
	img = shine(t, lamp(light.Elevation(40)), post(0, 0))
	if v := at(img, 85, 0); v > 5 {
		t.Fatalf("a post of no height given blocks all the way: %d", v)
	}
	img = shine(t, lamp(light.Elevation(40)), post(0, 20), post(80, 0))
	if v := at(img, 75, 0); v < 200 {
		t.Fatalf("between the first shadow's end and a second post, the floor is lit: %d", v)
	}
	if v := at(img, 92, 0); v > 5 {
		t.Fatalf("and the second post shadows behind it: %d", v)
	}
}

func TestALightsSizeSoftensShadowsFarFromWhatCastsThem(t *testing.T) {
	lamp := func(opts ...light.Option) *light.Light {
		return light.NewPoint(190, color.White, append([]light.Option{light.Falloff(1), light.At(-80, 0)}, opts...)...)
	}
	block := func() *twod.Sprite {
		return twod.NewSprite(filled(10, 10, stdcolor.RGBA{}), twod.At(-40, 0), light.Occluder(), light.Unlit())
	}
	// penumbra is how many rows of column x, below the shadow's middle,
	// are neither shadowed nor lit: between a sixth and five sixths of
	// the column's brightest.
	penumbra := func(img *image.RGBA, x int) int {
		top := 0
		for y := 0; y <= 48; y++ {
			top = max(top, int(at(img, x, y)))
		}
		n := 0
		for y := 0; y <= 48; y++ {
			if v := int(at(img, x, y)); v > top/6 && v < top*5/6 {
				n++
			}
		}
		return n
	}
	sharp := penumbra(shine(t, lamp(), block()), 90)
	soft := shine(t, lamp(light.Size(12)), block())
	near, far := penumbra(soft, -20), penumbra(soft, 90)
	if far < sharp+3 || far < near+3 {
		t.Fatalf(
			"with a size, the shadow's edge is wider far behind what casts it (%d rows part lit) "+
				"than near it (%d) or without one (%d)",
			far,
			near,
			sharp,
		)
	}
}

func TestAFadingShadowLightensAlongItsLength(t *testing.T) {
	img := shine(
		t,
		light.NewPoint(180, color.White, light.Falloff(1), light.At(-60, 0), light.Elevation(40), light.ShadowFade()),
		post(0, 20),
	)
	near, far := at(img, 8, 0), at(img, 60, 0)
	if far < near+60 {
		t.Fatalf("dark by the post (%d), lighter towards the shadow's end (%d)", near, far)
	}
}

func TestATilesHeightSetsItsShadowsLength(t *testing.T) {
	d := device(t)
	r := &webgpu2D{}
	if err := r.init(d); err != nil {
		t.Fatal(err)
	}
	draw := func(height float32) *image.RGBA {
		tiles := twod.NewTileset(filled(10, 10, stdcolor.RGBA{}), twod.Cell(10, 10), twod.Solid(1))
		if height > 0 {
			tiles = twod.NewTileset(
				filled(10, 10, stdcolor.RGBA{}),
				twod.Cell(10, 10),
				twod.Solid(1),
				twod.Height(height, 1),
			)
		}
		m := twod.NewTileMap(tiles, twod.Cells([][]int{{1}, {1}, {1}}), twod.CellSize(10, 10), twod.At(0, -15))
		return wallsLitBy(
			t,
			r,
			d,
			m,
			light.NewPoint(180, color.White, light.Falloff(1), light.At(-60, 0), light.Elevation(40)),
		)
	}
	if v := at(draw(0), 60, 0); v < 200 {
		t.Fatalf(
			"a solid tile stands its cell high, 10: its shadow ends at 70·40/30 ≈ 93 from the lamp: lit at 60, %d",
			v,
		)
	}
	if v := at(draw(30), 60, 0); v > 5 {
		t.Fatalf("one 30 high shadows to 70·40/10 = 280: dark at 60, %d", v)
	}
}

// wallsLitBy draws map m, a solid of the camera's Occluders, on the lit
// floor of a 200×100 window centred on the origin, lit by lamp.
func wallsLitBy(t *testing.T, r *webgpu2D, d *wgpu.Device, m *twod.TileMap, lamp *light.Light) *image.RGBA {
	t.Helper()
	space := collide.NewSpace()
	space.AddSolid(m)
	w := world.New().AddPlugins(wm.Plugin{}, twod.Plugin{}, light.Plugin{})
	win := platform.NewWindow(platform.Size(200, 100))
	win.Add(twod.NewCamera(light.Ambient(color.Black), light.OccludersFrom(space)))
	floor := twod.NewLayer(0, light.Lit()).Add(twod.NewSprite(filled(200, 100, stdcolor.RGBA{255, 255, 255, 255})))
	win.Add(floor, twod.NewLayer(1).Add(lamp, m))
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

func TestASmoothFadeIsDarkerThanLinearEarlyAndLighterLate(t *testing.T) {
	fade := func(curve light.Fade) *image.RGBA {
		return shine(
			t,
			light.NewPoint(
				180,
				color.White,
				light.Falloff(1),
				light.At(-60, 0),
				light.Elevation(40),
				light.ShadowFade(curve),
			),
			post(0, 20),
		)
	}
	// The post's shadow runs from 55 to 130 from the lamp: a quarter
	// along at x 14, three quarters at x 51.
	linear, smooth := fade(light.FadeLinear), fade(light.FadeSmooth)
	if l, s := at(linear, 14, 0), at(smooth, 14, 0); s+8 > l {
		t.Fatalf("a quarter along, smoothstep has faded less than linear (darker): linear %d, smooth %d", l, s)
	}
	if l, s := at(linear, 51, 0), at(smooth, 51, 0); s < l+8 {
		t.Fatalf("three quarters along, more (lighter): linear %d, smooth %d", l, s)
	}
	soft := fade(light.FadeSoft)
	if l, s := at(linear, 14, 0), at(soft, 14, 0); s < l+8 {
		t.Fatalf("soft fades fastest near what casts it: linear %d, soft %d", l, s)
	}
	from := shine(
		t,
		light.NewPoint(
			180,
			color.White,
			light.Falloff(1),
			light.At(-60, 0),
			light.Elevation(40),
			light.ShadowFade(),
			light.FadeFrom(0.5),
		),
		post(0, 20),
	)
	if v := at(from, 14, 0); v > 5 {
		t.Fatalf("FadeFrom(0.5) keeps the first half full: %d", v)
	}
}

func TestTheOccluderDebugViewShowsOccludersInTheirGroupsColourBesideF3sPanel(t *testing.T) {
	wall := func() *twod.Sprite {
		return twod.NewSprite(filled(6, 100, stdcolor.RGBA{}), twod.At(0, 0), light.Occluder())
	}
	panel := geom.Rect{W: 50, H: 100}
	img := litDebug(t, func(w *world.World, win *platform.Window) {
		light.SetDebug(win, light.DebugOccluders)
		draw.SetDebugPanel(w, panel)
	}, wall())
	plain := lit(t, wall())
	if v, want := at(img, -60, 0), at(plain, -60, 0); v != want {
		t.Fatalf("F3's panel is left clear: %d, not %d", v, want)
	}
	if v := at(img, -30, 0); v > 60 {
		t.Fatalf("where nothing blocks light the view is dark: %d", v)
	}
	if c := img.RGBAAt(100, 50); int(c.R) < int(c.G)+100 {
		t.Fatalf("the wall shows in group 0's red: %v", c)
	}
}

func TestTheDebugViewShowsOnlyInItsArea(t *testing.T) {
	wall := twod.NewSprite(filled(6, 100, stdcolor.RGBA{}), twod.At(0, 0), light.Occluder())
	clipped := twod.NewLayer(2, twod.ClipTo(geom.Rect{X: 90, Y: 40, W: 10, H: 10})).
		Add(twod.NewSprite(filled(4, 4, stdcolor.RGBA{0, 0, 255, 255}), twod.At(95, 45)))
	img := litDebug(t, func(_ *world.World, win *platform.Window) {
		light.SetDebug(win, light.DebugOccluders)
		light.SetDebugArea(win, geom.Rect{W: 80, H: 100})
	}, wall, clipped)
	if v := at(img, -30, 0); v > 60 {
		t.Fatalf("inside its area the view shows, whatever clip the window drew last: %d", v)
	}
	if c := img.RGBAAt(100, 50); int(c.R) > int(c.G)+20 {
		t.Fatalf("past it the wall's red doesn't: %v", c)
	}
}

func TestTheShadowMapViewShowsWhereALightIsBlockedAgainstTealWhereNothingIs(t *testing.T) {
	wall := twod.NewSprite(filled(6, 100, stdcolor.RGBA{}), twod.At(0, 0), light.Occluder())
	img := litDebug(t, func(_ *world.World, win *platform.Window) { light.SetDebug(win, light.DebugShadows) }, wall)
	if c := img.RGBAAt(100, 50); c.R <= c.B {
		t.Fatalf("towards the wall, half the radius away, a warm shade: %v", c)
	}
	if c := img.RGBAAt(20, 50); c.B <= c.R || c.G <= c.R {
		t.Fatalf("away from it, teal: %v", c)
	}
}

func TestAnOccluderSpriteIsLitFromInFrontAndDarkBehindAWall(t *testing.T) {
	crate := func(x float32) *twod.Layer {
		return twod.NewLayer(2, light.Lit()).
			Add(twod.NewSprite(filled(10, 10, stdcolor.RGBA{255, 255, 255, 255}), twod.At(x, 0), light.Occluder()))
	}
	img := lit(t, crate(-30))
	if v := at(img, -30, 0); v < 200 {
		t.Fatalf("lit from in front, the crate is bright, not in its own shadow: %d", v)
	}
	wall := twod.NewSprite(filled(6, 100, stdcolor.RGBA{}), twod.At(0, 0), light.Occluder())
	img = lit(t, wall, crate(40))
	if v := at(img, 40, 0); v > 30 {
		t.Fatalf("behind a wall it is dark: %d", v)
	}
}

func TestAFadingShadowIsDarkestRightBehindWhatCastsItHoweverDeep(t *testing.T) {
	block := twod.NewSprite(
		filled(40, 40, stdcolor.RGBA{}),
		twod.At(0, 0),
		light.Occluder(),
		light.Unlit(),
		light.Height(20),
	)
	img := shine(
		t,
		light.NewPoint(180, color.White, light.Falloff(1), light.At(-60, 0), light.Elevation(40), light.ShadowFade()),
		block,
	)
	if v := at(img, 24, 0); v > 100 {
		t.Fatalf(
			"the fade starts where the shadow leaves the block, not where the light enters it (about 170 then): "+
				"%d just behind it",
			v,
		)
	}
	if v := at(img, 80, 0); v < 180 {
		t.Fatalf("and lightens towards its end: %d", v)
	}
}

func TestAStaticLightIsBakedAsItIsDrawnLiveShadowsIncluded(t *testing.T) {
	d := device(t)
	r := &webgpu2D{}
	if err := r.init(d); err != nil {
		t.Fatal(err)
	}
	column := func() *twod.TileMap {
		tiles := twod.NewTileset(filled(10, 10, stdcolor.RGBA{}), twod.Cell(10, 10), twod.Solid(1))
		return twod.NewTileMap(tiles, twod.Cells([][]int{{1}, {1}, {1}}), twod.CellSize(10, 10), twod.At(0, -15))
	}
	lamp := func(opts ...light.Option) *light.Light {
		return light.NewPoint(
			140,
			color.RGB(1, 0.8, 0.5),
			append([]light.Option{light.Falloff(0.3), light.At(-60, 0)}, opts...)...)
	}
	live := wallsLitBy(t, r, d, column(), lamp())
	baked := wallsLitBy(t, r, d, column(), lamp(light.Static()))
	if r.lights.static.baked == 0 {
		t.Fatal("the static lamp was baked into tiles")
	}
	for _, p := range [][2]int{{-60, 0}, {-30, 20}, {-90, -40}, {40, 45}, {70, -45}} {
		a, b := live.RGBAAt(100+p[0], 50+p[1]), baked.RGBAAt(100+p[0], 50+p[1])
		for i, pair := range [][2]uint8{{a.R, b.R}, {a.G, b.G}, {a.B, b.B}} {
			if d := int(pair[0]) - int(pair[1]); d < -4 || d > 4 {
				t.Fatalf("at %v, channel %d: live %v, baked %v", p, i, a, b)
			}
		}
	}
	if v := at(baked, 30, 0); v > 5 {
		t.Fatalf("behind the column, the baked light is in its shadow: %d", v)
	}
}

func TestStaticTilesAreBakedOnceAndAgainOnlyWhenTheirLightChanges(t *testing.T) {
	d := device(t)
	r := &webgpu2D{}
	if err := r.init(d); err != nil {
		t.Fatal(err)
	}
	w := world.New().AddPlugins(wm.Plugin{}, twod.Plugin{}, light.Plugin{})
	win := platform.NewWindow(platform.Size(200, 100))
	torch := light.NewPoint(100, color.White, light.Static())
	win.Add(
		twod.NewCamera(light.Ambient(color.Black)),
		twod.NewLayer(0, light.Lit()).Add(twod.NewSprite(filled(200, 100, stdcolor.RGBA{255, 255, 255, 255}))),
		twod.NewLayer(1).Add(torch),
	)
	w.AddEntities(win)
	target := d.CreateTexture(200, 100, wgpu.RGBA8Unorm, wgpu.UsageRenderAttachment|wgpu.UsageCopySrc)
	frame := func() int {
		w.RunFrames(1)
		before := r.lights.static.baked
		r.window(w, win, target, d.NewTimers(), draw.PassesFor(w, win), draw.BackFor(w, win), draw.FrontFor(w, win))
		if err := wgpu.TakeError(); err != nil {
			t.Fatal(err)
		}
		return r.lights.static.baked - before
	}
	if n := frame(); n == 0 {
		t.Fatal("the first frame bakes the tiles in view")
	}
	if n := frame(); n != 0 {
		t.Fatalf("the next one bakes none: %d", n)
	}
	torch.SetColor(color.RGB(1, 0, 0))
	if n := frame(); n == 0 {
		t.Fatal("a change of colour bakes its tiles again")
	}
}

func TestALayerShadedBySomeGroupsIsShadedOnlyByThem(t *testing.T) {
	crowd := func(box *twod.Sprite) *image.RGBA {
		monster := twod.NewSprite(filled(10, 10, stdcolor.RGBA{255, 255, 255, 255}), twod.At(30, 0))
		return shine(t, light.NewPoint(120, color.White, light.Falloff(1), light.At(-60, 0)), box,
			twod.NewLayer(2, light.Lit(), light.ShadedBy(0, 1)).Add(monster))
	}
	img := crowd(wallAt(0, 0, 6, 100, light.Occluder(2)))
	if v := at(img, 30, 0); v < 250 {
		t.Fatalf("a monster behind a group 2 box stays lit: %d", v)
	}
	if v := at(img, 30, 30); v > 5 {
		t.Fatalf("while the floor beside it is in the box's shadow: %d", v)
	}
	img = crowd(wallAt(0, 0, 6, 100, light.Occluder(0)))
	if v := at(img, 30, 0); v > 5 {
		t.Fatalf("a group 0 wall still shades it: %d", v)
	}

	monster := twod.NewSprite(filled(10, 10, stdcolor.RGBA{255, 255, 255, 255}), twod.At(0, -10))
	img = shine(
		t,
		light.NewDirectional(color.White, math.Pi/2, light.ShadowLength(60)),
		wallAt(0, -30, 20, 6, light.Occluder(2)),
		twod.NewLayer(2, light.Lit(), light.ShadedBy(0, 1)).Add(monster),
	)
	if v := at(img, 0, -10); v < 250 {
		t.Fatalf("a directional light's group 2 shadow doesn't fall on it either: %d", v)
	}
	if v := at(img, 8, -2); v > 5 {
		t.Fatalf("but falls on the floor: %d", v)
	}
}

func TestAShadedPictureRedrawsOnlyTheLightsAnExcludedOccluderReaches(t *testing.T) {
	d := device(t)
	r := &webgpu2D{}
	if err := r.init(d); err != nil {
		t.Fatal(err)
	}
	w := world.New().AddPlugins(wm.Plugin{}, twod.Plugin{}, light.Plugin{})
	win := platform.NewWindow(platform.Size(200, 100))
	win.Add(twod.NewCamera(light.Ambient(color.Black)))
	floor := twod.NewLayer(0, light.Lit()).Add(twod.NewSprite(filled(200, 100, stdcolor.RGBA{255, 255, 255, 255})))
	near := light.NewPoint(60, color.White, light.Falloff(1), light.At(-60, 0))
	far := light.NewPoint(40, color.White, light.Falloff(1), light.At(60, 0))
	box := wallAt(-30, 0, 6, 100, light.Occluder(2))
	monster := twod.NewSprite(filled(10, 10, stdcolor.RGBA{255, 255, 255, 255}), twod.At(-15, 0))
	crowd := twod.NewLayer(2, light.Lit(), light.ShadedBy(0, 1)).
		Add(monster, twod.NewSprite(filled(10, 10, stdcolor.RGBA{255, 255, 255, 255}), twod.At(60, 20)))
	win.Add(floor, twod.NewLayer(1).Add(near, far, box), crowd)
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
	if r.lights.redrawn != 1 {
		t.Fatalf("only the light by the box is redrawn: %d", r.lights.redrawn)
	}
	if v := at(img, -15, 0); v < 250 {
		t.Fatalf("the monster behind the box stays lit: %d", v)
	}
	if v := at(img, -15, 30); v > 5 {
		t.Fatalf("the floor behind it is shaded: %d", v)
	}
	if v := at(img, 60, 20); v < 250 {
		t.Fatalf("the far light still lights the crowd, copied: %d", v)
	}
}

func TestASilhouetteShadowsTheFloorWithItsPictureAwayFromTheLight(t *testing.T) {
	half := image.NewRGBA(image.Rect(0, 0, 10, 20))
	for y := range 20 {
		for x := 5; x < 10; x++ {
			half.SetRGBA(x, y, stdcolor.RGBA{255, 255, 255, 255})
		}
	}
	sprite := twod.NewSprite(twod.NewTexture(half), twod.At(0, 0), light.Silhouette())
	img := shine(t, light.NewPoint(120, color.White, light.Falloff(1), light.At(-60, 10)), sprite)
	if v := at(img, 12, 7); v > 5 {
		t.Fatalf("beyond its foot, under its opaque half, the floor is in its shadow: %d", v)
	}
	if v := at(img, 12, 13); v < 250 {
		t.Fatalf("under its transparent half the light passes: %d", v)
	}
	if v := at(img, 40, 7); v < 250 {
		t.Fatalf("past its height, away from its foot, the shadow ends: %d", v)
	}
	if v := at(img, -20, 7); v < 250 {
		t.Fatalf("on the light's side it casts nothing: %d", v)
	}
}

func TestADirectionalLightCastsASilhouetteShadowLengthLong(t *testing.T) {
	sprite := twod.NewSprite(filled(10, 20, stdcolor.RGBA{255, 255, 255, 255}), twod.At(0, -20), light.Silhouette())
	img := shine(t, light.NewDirectional(color.White, math.Pi/2, light.ShadowLength(30)), sprite)
	if v := at(img, 0, 0); v > 5 {
		t.Fatalf("below its foot, the rays going down, is its shadow: %d", v)
	}
	if v := at(img, 0, 25); v < 250 {
		t.Fatalf("past ShadowLength it ends: %d", v)
	}
	if v := at(img, 20, 0); v < 250 {
		t.Fatalf("beside it the sun shines: %d", v)
	}
}

func TestALightNotCastingASilhouetteIsNotShadowedByAnothersMask(t *testing.T) {
	sprite := twod.NewSprite(filled(10, 20, stdcolor.RGBA{255, 255, 255, 255}), twod.At(0, 0), light.Silhouette())
	other := twod.NewSprite(filled(10, 20, stdcolor.RGBA{255, 255, 255, 255}), twod.At(85, 0), light.Silhouette())
	img := shine(t,
		light.NewPoint(120, color.White, light.Falloff(1), light.At(-60, 10)),
		light.NewPoint(60, color.White, light.Falloff(1), light.At(60, 10)),
		sprite, other)
	if v := at(img, 12, 7); v < 250 {
		t.Fatalf("the second lamp, casting only the other sprite, still lights the first one's shadow: %d", v)
	}
	if v := at(img, -20, 7); v < 250 {
		t.Fatalf("on the first lamp's side it is lit: %d", v)
	}
}

// shineWith is shine, handing back the renderer too.
func shineWith(t *testing.T, ents ...twod.Entity) (*image.RGBA, *webgpu2D) {
	t.Helper()
	d := device(t)
	r := &webgpu2D{}
	if err := r.init(d); err != nil {
		t.Fatal(err)
	}
	w := world.New().AddPlugins(wm.Plugin{}, twod.Plugin{}, light.Plugin{})
	win := platform.NewWindow(platform.Size(200, 100))
	win.Add(twod.NewCamera(light.Ambient(color.Black)))
	floor := twod.NewLayer(0, light.Lit()).Add(twod.NewSprite(filled(200, 100, stdcolor.RGBA{255, 255, 255, 255})))
	things := twod.NewLayer(1)
	for _, e := range ents {
		things.Add(e)
	}
	win.Add(floor, things)
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
	return img, r
}

func TestTwoLightsSilhouetteTilesNeverBleedIntoEachOther(t *testing.T) {
	wide := twod.NewSprite(filled(300, 6, stdcolor.RGBA{255, 255, 255, 255}), twod.At(-50, 15), light.Silhouette())
	half := image.NewRGBA(image.Rect(0, 0, 20, 6))
	for y := range 6 {
		for x := range 10 {
			half.SetRGBA(x, y, stdcolor.RGBA{255, 255, 255, 255})
		}
	}
	small := twod.NewSprite(twod.NewTexture(half), twod.At(50, 15), light.Silhouette())
	img, r := shineWith(t,
		light.NewPoint(40, color.White, light.Falloff(1), light.At(-50, 0)),
		light.NewPoint(40, color.White, light.Falloff(1), light.At(50, 0)),
		wide, small)
	if len(r.lights.casting) != 2 || r.lights.casting[0].page != r.lights.casting[1].page {
		t.Fatalf("both lights' tiles on one page: %+v", r.lights.casting)
	}
	if v := at(img, 45, 21); v > 5 {
		t.Fatalf("under the second sprite's opaque half, its shadow: %d", v)
	}
	if v := at(img, 55, 21); v < 250 {
		t.Fatalf(
			"under its transparent half the second lamp lights the floor, whatever the first one's wide silhouette spills: %d",
			v,
		)
	}
	if v := at(img, -50, 21); v > 5 {
		t.Fatalf("under the wide sprite's shadow, from the first lamp: %d", v)
	}
}

func TestSilhouetteTilesThatDontFitGoOnAnotherPage(t *testing.T) {
	side := draw.SilhouetteSide
	draw.SilhouetteSide = 10
	defer func() { draw.SilhouetteSide = side }()
	a := twod.NewSprite(filled(60, 20, stdcolor.RGBA{255, 255, 255, 255}), twod.At(-30, 0), light.Silhouette())
	b := twod.NewSprite(filled(60, 20, stdcolor.RGBA{255, 255, 255, 255}), twod.At(30, 0), light.Silhouette())
	img, r := shineWith(t,
		light.NewPoint(68, color.White, light.Falloff(1), light.At(-70, 10)),
		light.NewPoint(68, color.White, light.Falloff(1), light.At(70, 10)),
		a, b)
	pages := map[int]bool{}
	for _, c := range r.lights.casting {
		pages[c.page] = true
	}
	if len(pages) != 2 {
		t.Fatalf("two tiles too big to share a small page: a page each: %+v", r.lights.casting)
	}
	if v, open := at(img, -18, 25), at(img, -85, -42); v+20 > open {
		t.Fatalf("beyond a's foot the left lamp is shaded: %d, as far on its open side %d", v, open)
	}
	if v, open := at(img, 18, 25), at(img, 85, -42); v+20 > open {
		t.Fatalf("and beyond b's the right one: %d, as far on its open side %d", v, open)
	}
}

func TestAShadedPictureTellsALightsSilhouettesApartByGroup(t *testing.T) {
	kept := twod.NewSprite(filled(10, 20, stdcolor.RGBA{255, 255, 255, 255}), twod.At(-30, 0), light.Silhouette(0))
	left := twod.NewSprite(filled(10, 20, stdcolor.RGBA{255, 255, 255, 255}), twod.At(30, 0), light.Silhouette(2))
	patches := twod.NewLayer(0, light.Lit(), light.ShadedBy(0, 1)).Add(
		twod.NewSprite(filled(20, 4, stdcolor.RGBA{255, 255, 255, 255}), twod.At(-45, 7)),
		twod.NewSprite(filled(20, 4, stdcolor.RGBA{255, 255, 255, 255}), twod.At(45, 7)))
	img := shine(t, light.NewPoint(120, color.White, light.Falloff(1), light.At(0, 10)), kept, left, patches)
	if v := at(img, -42, 7); v > 5 {
		t.Fatalf("on a layer ShadedBy(0, 1), one light's group 0 silhouette shades: %d", v)
	}
	if v := at(img, 42, 7); v < 250 {
		t.Fatalf("and the same light's group 2 one doesn't: %d", v)
	}
	if v := at(img, 42, 11); v > 5 {
		t.Fatalf("on the floor below the patch, the group 2 one does: %d", v)
	}
}
