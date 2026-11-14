package twod //nolint:testpackage // white-box: shares twod's test helpers, reaches unexported state

import (
	"bytes"
	"image"
	stdcolor "image/color"
	"image/png"
	"testing"
	"testing/fstest"

	"github.com/lantervale/ember/automation"
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/internal/draw"
	"github.com/lantervale/ember/internal/inputfeed"
	"github.com/lantervale/ember/internal/slot"
	"github.com/lantervale/ember/internal/wm"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/world"
)

func TestPerWindowContexts(t *testing.T) {
	a := world.New()
	primary := platform.NewWindow(platform.Size(800, 600))
	a.AddEntities(primary)
	a.AddPlugins(canvasPlugin{})
	tool := platform.NewWindow(platform.Size(320, 240))
	a.AddEntities(tool)

	main := CanvasFor(a, nil)
	side := main.For(tool)
	if side == main || side.list == main.list {
		t.Fatal("each window needs its own context and canvas")
	}
	if side.Screen().W != 320 || main.Screen().W != 800 {
		t.Fatalf("screens %v %v", side.Screen(), main.Screen())
	}
	if main.For(nil) != main || main.For(primary) != main {
		t.Fatal("nil and the primary window resolve to the primary context")
	}

	tool.Input().SetMouse(geom.V2(10, 10))
	inputfeed.Of(tool.Input()).BeginFrame()
	if r := (geom.Rect{X: 5, Y: 5, W: 20, H: 20}); !side.Hovered(r) || main.Hovered(r) {
		t.Fatal("hover must use the window's own input")
	}

	side.Rect(geom.Rect{W: 5, H: 5}, color.White)
	if draw.FrontFor(a, tool) != side.list || len(side.list.Unindexed()) == 0 {
		t.Fatal("presenter must find the window canvas")
	}
	a.RunFrames(1)
	if len(side.list.Unindexed()) != 0 {
		t.Fatal("canvases reset every frame")
	}
	a.RemoveEntity(tool)
	a.RunFrames(1)
	if draw.FrontFor(a, tool) != nil {
		t.Fatal("closed window context must be dropped")
	}
}

func TestF3TogglesTheOverlay(t *testing.T) {
	a := world.New()
	primary := platform.NewWindow(platform.Size(800, 600))
	a.AddEntities(primary)
	a.AddPlugins(wm.Plugin{}, canvasPlugin{})
	in := wm.Input(a)
	overlay := &slot.Get[windows](a).overlay
	canvas := CanvasFor(a, nil).list

	in.SetKey(input.KeyF3, true)
	a.RunFrames(1)
	if !overlay.Visible || len(canvas.Unindexed()) == 0 {
		t.Fatalf("F3 shows the canvas overlay: visible %v, %d vertices", overlay.Visible, len(canvas.Unindexed()))
	}
	in.SetKey(input.KeyF3, false)
	a.RunFrames(1)
	in.SetKey(input.KeyF3, true)
	a.RunFrames(1)
	if overlay.Visible || len(canvas.Unindexed()) != 0 {
		t.Fatal("F3 again hides it")
	}
}

func TestBitmapFontsDrawCrispRunsOnTheScaleGrid(t *testing.T) {
	font := DefaultBitmapFont()
	if got := font.Measure("GEMS\n10"); got != geom.V2(4*7, 2*13) {
		t.Fatalf("measure %v", got)
	}
	c := &Canvas{list: &draw.List{}}
	font.Draw(c, geom.V2(10, 10), "I", 3, color.White)
	if len(c.list.Unindexed()) == 0 {
		t.Fatal("glyph drew nothing")
	}
	for _, v := range c.list.Unindexed() {
		if v.Pos.X/3 != geom.Floor(v.Pos.X/3) || v.Pos.Y/3 != geom.Floor(v.Pos.Y/3) {
			t.Fatalf("vertex %v is off the 3px grid", v.Pos)
		}
	}
	c.list.Vertices, c.list.Indices = c.list.Vertices[:0], c.list.Indices[:0]
	font.Draw(c, geom.V2(0, 0), " ", 3, color.White)
	if len(c.list.Unindexed()) != 0 {
		t.Fatal("space is empty")
	}
	if len(font.glyphOf('€').runs) == 0 {
		t.Fatal("unknown runes fall back to ?")
	}
}

func TestLoadBitmapFontCutsCellsInCharsetOrder(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 8, 4))
	img.Set(0, 0, stdcolor.White)
	img.Set(5, 3, stdcolor.White)
	img.Set(6, 3, stdcolor.White)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	font, err := LoadBitmapFont(fstest.MapFS{"font.png": {Data: buf.Bytes()}}, "font.png", Cell(4, 4), Charset("AB"))
	if err != nil {
		t.Fatal(err)
	}
	if a := font.glyphOf('A').runs; len(a) != 1 || a[0] != (geom.Rect{W: 1, H: 1}) {
		t.Fatalf("A: %v", a)
	}
	if b := font.glyphOf('B').runs; len(b) != 1 || b[0] != (geom.Rect{X: 1, Y: 3, W: 2, H: 1}) {
		t.Fatalf("B, the second cell, as one run: %v", b)
	}
	if font.Measure("AB") != geom.V2(8, 4) {
		t.Fatal("cells of 4×4")
	}
	if _, err = LoadBitmapFont(
		fstest.MapFS{"font.png": {Data: buf.Bytes()}},
		"font.png",
		Cell(16, 4),
		Charset("AB"),
	); err == nil {
		t.Fatal("cells wider than the picture")
	}
}

func TestBitmapTextsAreEntitiesWithAShadow(t *testing.T) {
	w, win, c := app()
	plain := NewBitmapText("I", At(100, 100), Scale(3, 3), Pivot(0, 0))
	win.AppendChild(plain)
	w.AddEntities(plain)
	w.RunFrames(1)
	n := len(c.Unindexed())
	lo, _ := corners(c.Unindexed())
	if n == 0 || lo.X < 100 || lo.Y < 100 {
		t.Fatalf("drawn from its top-left at 100, 100: %d vertices from %v", n, lo)
	}
	TextShadow(color.Black).bitmapText(plain)
	w.RunFrames(1)
	if len(c.Unindexed()) != 2*n {
		t.Fatalf("a shadow doubles the runs: %d vs %d", len(c.Unindexed()), n)
	}
	if NewBitmapText("GEMS").Measure() != geom.V2(28, 13) {
		t.Fatal("size in font pixels")
	}
}

func TestTheMouseAndRecordsFollowTheTransform(t *testing.T) {
	w := world.New().AddPlugins(canvasPlugin{})
	win := platform.NewWindow(platform.Size(400, 300))
	w.AddEntities(win)
	w.RunFrames(1)
	u := CanvasFor(w, nil).For(win)
	reg := automation.NewRegistry()
	reg.SetRecording(true)
	slot.Set(w, reg)
	win.Input().SetMouse(geom.V2(110, 220))
	inputfeed.Of(win.Input()).BeginFrame()
	u.SetTransform(geom.Transform2DFrom(geom.V2(100, 200), 0, 0, geom.V2(2, 2)))
	if m := u.Mouse(); m != geom.V2(5, 10) {
		t.Fatalf("the mouse in the element's space: %v", m)
	}
	if !u.Hovered(geom.Rect{W: 10, H: 20}) || u.Hovered(geom.Rect{X: 6, W: 10, H: 20}) {
		t.Fatal("hovering is tested where the element shows")
	}
	u.Expose("box", automation.RoleMeter, "Box", 1, geom.Rect{W: 10, H: 20})
	reg.EndFrame(1)
	if _, els := reg.Snapshot("", ""); len(els) != 1 || els[0].Bounds != (geom.Rect{X: 100, Y: 200, W: 20, H: 40}) {
		t.Fatalf("automation sees screen bounds: %+v", els)
	}
	u.SetTransform(geom.Identity2D())
	if u.Mouse() != geom.V2(110, 220) {
		t.Fatal("without a transform, the window's mouse")
	}
}
