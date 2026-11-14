package capture //nolint:testpackage // tests the unexported parsing and GIF helpers

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"slices"
	"strings"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/internal/inputfeed"
)

func TestKeysAreHeldDuringTheirFrames(t *testing.T) {
	keys, err := ParseKeys("right:0-40, LEFT: 40 - 80")
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 2 || keys[0] != (Hold{input.KeyRight, 0, 40}) || keys[1] != (Hold{input.KeyLeft, 40, 80}) {
		t.Fatalf("parsed %+v", keys)
	}
	var in input.State
	keys.Apply(&in, 39)
	inputfeed.Of(&in).BeginFrame()
	if !in.Down(input.KeyRight) || in.Down(input.KeyLeft) {
		t.Fatal("frame 39 holds right only")
	}
	keys.Apply(&in, 40)
	inputfeed.Of(&in).BeginFrame()
	if in.Down(input.KeyRight) || !in.Down(input.KeyLeft) {
		t.Fatal("frame 40 holds left only")
	}
	keys.Apply(&in, 80)
	inputfeed.Of(&in).BeginFrame()
	if in.Down(input.KeyLeft) {
		t.Fatal("frame 80 holds nothing")
	}
}

func TestAKeyHeldTwiceIsDownDuringBothHolds(t *testing.T) {
	keys, err := ParseKeys("up:0-20,left:20-34,up:34-54")
	if err != nil {
		t.Fatal(err)
	}
	var in input.State
	for frame, want := range map[int]bool{0: true, 19: true, 20: false, 34: true, 53: true, 54: false} {
		keys.Apply(&in, frame)
		inputfeed.Of(&in).BeginFrame()
		if in.Down(input.KeyUp) != want {
			t.Errorf("frame %d: up down %v, want %v", frame, in.Down(input.KeyUp), want)
		}
	}
}

func TestBadKeyScriptsAreRefused(t *testing.T) {
	for _, s := range []string{"right", "right:10", "nope:0-4", "up:5-5", "up:a-4"} {
		if _, err := ParseKeys(s); err == nil {
			t.Errorf("%q was accepted", s)
		}
	}
}

func TestKeyScriptsSeparateBySemicolonLikeTheOthers(t *testing.T) {
	semi, err := ParseKeys("right:100-400;down:400-600")
	if err != nil {
		t.Fatal(err)
	}
	comma, err := ParseKeys("right:100-400,down:400-600")
	if err != nil {
		t.Fatal(err)
	}
	want := Keys{{input.KeyRight, 100, 400}, {input.KeyDown, 400, 600}}
	if !slices.Equal(semi, want) || !slices.Equal(comma, want) {
		t.Fatalf("; and , read the same: %v %v", semi, comma)
	}
	_, err = ParseKeys("right:100-400; down:400")
	if err == nil || !strings.Contains(err.Error(), `"down:400"`) || !strings.Contains(err.Error(), "separated by ;") {
		t.Fatalf("a bad entry is quoted and the separator named: %v", err)
	}
}

func solid(c color.RGBA, w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), image.NewUniform(c), image.Point{}, draw.Src)
	return img
}

func TestTheGIFLoopsAndSharesOnePalette(t *testing.T) {
	g := NewGIF(30)
	red, blue := color.RGBA{220, 40, 30, 255}, color.RGBA{30, 60, 200, 255}
	g.Add(solid(red, 64, 32), 32, 16)
	g.Add(solid(blue, 64, 32), 32, 16)
	g.Add(solid(red, 64, 32), 32, 16)
	var buf bytes.Buffer
	if err := g.Encode(&buf); err != nil {
		t.Fatal(err)
	}
	anim, err := gif.DecodeAll(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if len(anim.Image) != 3 || anim.LoopCount != 0 || anim.Delay[0] != 3 {
		t.Fatalf("%d frames, loop %d, delay %v", len(anim.Image), anim.LoopCount, anim.Delay)
	}
	if b := anim.Image[0].Bounds(); b.Dx() != 32 || b.Dy() != 16 {
		t.Fatalf("frames are scaled to the size asked: %v", b)
	}
	if len(anim.Image[0].Palette) > 256 || len(anim.Image[0].Palette) != len(anim.Image[1].Palette) {
		t.Fatal("one palette of at most 256 colours")
	}
	at := func(i int) color.RGBA {
		c, _ := color.RGBAModel.Convert(anim.Image[i].At(4, 4)).(color.RGBA)
		return c
	}
	if at(0) != red || at(1) != blue || at(2) != red {
		t.Fatalf("frames keep their colours: %v %v %v", at(0), at(1), at(2))
	}
}

func TestTheCursorMovesInStraightLinesBetweenWaypoints(t *testing.T) {
	m, err := ParseMouse("100,80@0; 400,200@40; 200,300@80; left: 30 - 40")
	if err != nil {
		t.Fatal(err)
	}
	for frame, want := range map[int]geom.Vec2{
		0: geom.V2(100, 80), 20: geom.V2(250, 140), 40: geom.V2(400, 200),
		60: geom.V2(300, 250), 80: geom.V2(200, 300), 200: geom.V2(200, 300),
	} {
		if at, _ := m.At(frame); at != want {
			t.Errorf("frame %d: cursor at %v, want %v", frame, at, want)
		}
	}
	var in input.State
	m.Apply(&in, 35)
	inputfeed.Of(&in).BeginFrame()
	if in.Mouse() != geom.V2(362.5, 185) || !in.Down(input.MouseLeft) {
		t.Fatalf("frame 35: cursor %v, left down %v", in.Mouse(), in.Down(input.MouseLeft))
	}
	m.Apply(&in, 40)
	inputfeed.Of(&in).BeginFrame()
	if in.Down(input.MouseLeft) {
		t.Fatal("frame 40 releases the button")
	}
}

func TestAScriptWithoutPathLeavesTheCursorAlone(t *testing.T) {
	m, err := ParseMouse("right:0-5; back:1-3")
	if err != nil {
		t.Fatal(err)
	}
	var in input.State
	in.SetMouse(geom.V2(7, 9))
	m.Apply(&in, 2)
	inputfeed.Of(&in).BeginFrame()
	if in.Mouse() != geom.V2(7, 9) || !in.Down(input.MouseRight) || !in.Down(input.MouseBack) {
		t.Fatal("buttons only: the cursor stays, the right and back buttons are down")
	}
}

func TestTheWheelTurnsOnItsFrames(t *testing.T) {
	m, err := ParseMouse("wheel: -2 @ 10; wheel:1.5@20")
	if err != nil {
		t.Fatal(err)
	}
	for frame, want := range map[int]float32{9: 0, 10: -2, 11: 0, 20: 1.5} {
		var in input.State
		m.Apply(&in, frame)
		inputfeed.Of(&in).BeginFrame()
		if got := in.Wheel(); got != geom.V2(0, want) {
			t.Errorf("frame %d: wheel %v, want %v lines", frame, got, want)
		}
	}
}

func TestBadMouseScriptsAreRefused(t *testing.T) {
	for _, s := range []string{
		"100,80", "100@4", "a,b@0", "10,10@5;20,20@5", "10,10@5;20,20@2",
		"thumb:0-4", "left:4-4", "wheel:up@3", "wheel:1@-1", "wheel:1",
	} {
		if _, err := ParseMouse(s); err == nil {
			t.Errorf("%q was accepted", s)
		}
	}
}

func TestLaterFramesHoldOnlyWhatChanged(t *testing.T) {
	g := NewGIF(30)
	base := solid(color.RGBA{40, 120, 60, 255}, 64, 32)
	moved := solid(color.RGBA{40, 120, 60, 255}, 64, 32)
	draw.Draw(moved, image.Rect(10, 4, 14, 8), image.NewUniform(color.RGBA{250, 240, 90, 255}), image.Point{}, draw.Src)
	g.Add(base, 64, 32)
	g.Add(moved, 64, 32)
	g.Add(moved, 64, 32)
	var buf bytes.Buffer
	if err := g.Encode(&buf); err != nil {
		t.Fatal(err)
	}
	anim, err := gif.DecodeAll(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if b := anim.Image[1].Bounds(); b != image.Rect(10, 4, 14, 8) {
		t.Fatalf("the second frame is only the square that changed: %v", b)
	}
	if b := anim.Image[2].Bounds(); b.Dx()*b.Dy() != 1 {
		t.Fatalf("an unchanged frame is one pixel: %v", b)
	}
	canvas := image.NewRGBA(image.Rect(0, 0, 64, 32))
	for _, f := range anim.Image {
		draw.Draw(canvas, f.Bounds(), f, f.Bounds().Min, draw.Over)
	}
	if got := canvas.RGBAAt(12, 6); got != (color.RGBA{250, 240, 90, 255}) {
		t.Fatalf("the square shows on the composed picture: %v", got)
	}
	if got := canvas.RGBAAt(40, 20); got != (color.RGBA{40, 120, 60, 255}) {
		t.Fatalf("the rest stays as it was: %v", got)
	}
}

func TestDitheringMixesNeighboursAndStaysStill(t *testing.T) {
	dark, light := color.RGBA{20, 20, 20, 255}, color.RGBA{230, 230, 230, 255}
	ramp := image.NewRGBA(image.Rect(0, 0, 64, 16))
	for x := range 64 {
		v := uint8(20 + x*210/63)
		draw.Draw(ramp, image.Rect(x, 0, x+1, 16), image.NewUniform(color.RGBA{v, v, v, 255}), image.Point{}, draw.Src)
	}
	g := NewGIF(30)
	g.Add(solid(dark, 64, 16), 64, 16)
	g.Add(solid(light, 64, 16), 64, 16)
	g.Add(ramp, 64, 16)
	g.Add(ramp, 64, 16)
	var buf bytes.Buffer
	if err := g.Encode(&buf); err != nil {
		t.Fatal(err)
	}
	anim, err := gif.DecodeAll(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if b := anim.Image[3].Bounds(); b.Dx()*b.Dy() != 1 {
		t.Fatalf("the same picture dithers the same way, so the repeat costs one pixel: %v", b)
	}
	ramped := anim.Image[2]
	seen := map[uint8]bool{}
	for y := range 4 {
		for x := 28; x < 32; x++ {
			seen[ramped.ColorIndexAt(x, y)] = true
		}
	}
	if len(seen) < 2 {
		t.Fatal("a mid-grey between palette colours is a pattern of both, not one band")
	}
}

func TestTheWindowIsResizedAlongItsWaypoints(t *testing.T) {
	s, err := ParseSizes("640,360@0;900,480@60")
	if err != nil {
		t.Fatal(err)
	}
	if at, _ := s.At(30); at != geom.V2(770, 420) {
		t.Fatalf("halfway: %v", at)
	}
	if at, _ := s.At(90); at != geom.V2(900, 480) {
		t.Fatalf("after the last waypoint: %v", at)
	}
	for _, bad := range []string{"0,360@0", "640,360@0;left:0-4", "640x360@0"} {
		if _, err = ParseSizes(bad); err == nil {
			t.Errorf("%q was accepted", bad)
		}
	}
}

func TestFramesOfDifferentSizesShareTheLargestCanvas(t *testing.T) {
	g := NewGIF(30)
	big, small := color.RGBA{200, 60, 40, 255}, color.RGBA{40, 160, 90, 255}
	g.Add(solid(big, 64, 32), 64, 32)
	g.Add(solid(small, 32, 16), 32, 16)
	var buf bytes.Buffer
	if err := g.Encode(&buf); err != nil {
		t.Fatal(err)
	}
	anim, err := gif.DecodeAll(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if anim.Config.Width != 64 || anim.Config.Height != 32 {
		t.Fatalf("the canvas is the largest frame: %dx%d", anim.Config.Width, anim.Config.Height)
	}
	canvas := image.NewRGBA(image.Rect(0, 0, 64, 32))
	for _, f := range anim.Image {
		draw.Draw(canvas, f.Bounds(), f, f.Bounds().Min, draw.Over)
	}
	if got := canvas.RGBAAt(8, 8); got != small {
		t.Fatalf("the smaller window shows at the top-left: %v", got)
	}
	if got := canvas.RGBAAt(50, 24); got != desk {
		t.Fatalf("around it is the desk, not the larger frame before: %v", got)
	}
}

func TestTextIsTypedOneCharacterEveryFourFrames(t *testing.T) {
	ty, err := ParseText("Hé n@10; ok@40")
	if err != nil {
		t.Fatal(err)
	}
	typed := map[int]string{}
	for frame := range 60 {
		var in input.State
		ty.Apply(&in, frame)
		inputfeed.Of(&in).BeginFrame()
		if r := in.Text(); len(r) > 0 {
			typed[frame] = string(r)
		}
	}
	want := map[int]string{10: "H", 14: "é", 18: " ", 22: "n", 40: "o", 44: "k"}
	if len(typed) != len(want) {
		t.Fatalf("typed %v, want %v", typed, want)
	}
	for f, s := range want {
		if typed[f] != s {
			t.Errorf("frame %d: typed %q, want %q", f, typed[f], s)
		}
	}
	for _, s := range []string{"abc", "abc@x", "@-1", "abc@-2"} {
		if _, err = ParseText(s); err == nil {
			t.Errorf("%q was accepted", s)
		}
	}
}
