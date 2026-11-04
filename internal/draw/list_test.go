package draw //nolint:testpackage // white-box: reaches draw's unexported state

import (
	"image"
	"testing"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/render"
)

func TestImagesBatchByTextureAndClip(t *testing.T) {
	tex := &render.Texture{Image: image.NewRGBA(image.Rect(0, 0, 32, 16))}
	var c List
	c.Rect(geom.Rect{W: 10, H: 10}, color.White)
	c.Image(geom.Rect{X: 10, W: 32, H: 16}, tex, geom.Rect{W: 1, H: 1}, color.White)
	c.Image(geom.Rect{X: 50, W: 32, H: 16}, tex, geom.Rect{W: 1, H: 1}, color.White)
	if len(c.Batches) != 1 || c.Batches[0].Image() != tex || c.Batches[0].Count != 18 {
		t.Fatalf("shapes and one image share a batch: %+v", c.Batches)
	}
	other := &render.Texture{Image: image.NewRGBA(image.Rect(0, 0, 4, 4))}
	c.Image(geom.Rect{W: 4, H: 4}, other, geom.Rect{W: 1, H: 1}, color.White)
	if len(c.Batches) != 1 || c.Batches[0].Images[1] != other ||
		Slot(c.Unindexed()[len(c.Unindexed())-1].Shape.X) != 1 {
		t.Fatalf("another image takes the batch's next slot: %+v", c.Batches)
	}
	c.PushClip(geom.Rect{X: 0, Y: 0, W: 100, H: 100})
	c.PushClip(geom.Rect{X: 50, Y: 50, W: 100, H: 100})
	c.Rect(geom.Rect{W: 5, H: 5}, color.White)
	if b := c.Batches[len(c.Batches)-1]; !b.Clipped || b.Clip != (geom.Rect{X: 50, Y: 50, W: 50, H: 50}) {
		t.Fatalf("nested clips intersect: %+v", b)
	}
	c.PopClip()
	c.PopClip()
	c.Rect(geom.Rect{W: 5, H: 5}, color.White)
	if b := c.Batches[len(c.Batches)-1]; b.Clipped {
		t.Fatal("popping every clip ends clipping")
	}
}

func TestNinePatchKeepsItsCorners(t *testing.T) {
	tex := &render.Texture{Image: image.NewRGBA(image.Rect(0, 0, 30, 30))}
	var c List
	c.NinePatch(geom.Rect{X: 0, Y: 0, W: 200, H: 100}, tex, 10, 10, 10, 10, color.White)
	if len(c.Unindexed()) != 9*6 {
		t.Fatalf("%d vertices", len(c.Unindexed()))
	}
	corner, center := c.Unindexed()[0], c.Unindexed()[4*6]
	if corner.Pos != geom.V2(0, 0) || c.Unindexed()[2].Pos != geom.V2(10, 10) ||
		c.Unindexed()[2].UV != geom.V2(1.0/3, 1.0/3) {
		t.Fatalf("corner %v %v", c.Unindexed()[2].Pos, c.Unindexed()[2].UV)
	}
	if center.Pos != geom.V2(10, 10) || c.Unindexed()[4*6+2].Pos != geom.V2(190, 90) {
		t.Fatalf("center %v %v", center.Pos, c.Unindexed()[4*6+2].Pos)
	}
}

func TestCanvasTransformModulateAndSnap(t *testing.T) {
	var c List
	c.SetTransform(geom.Transform2DFrom(geom.V2(100, 50), geom.Pi/2, 0, geom.V2(2, 2)))
	c.SetModulate(color.RGBA(0.5, 1, 1, 0.5))
	c.Rect(geom.Rect{W: 10, H: 5}, color.White)
	if p := c.Unindexed()[1].Pos; p.Sub(geom.V2(100, 70)).Len() > 1e-4 {
		t.Fatalf("the rect's x axis turns down: %v", p)
	}
	if col := c.Unindexed()[0].Color; col != color.RGBA(0.5, 1, 1, 0.5) {
		t.Fatalf("modulate multiplies colors: %v", col)
	}
	c.PushClip(geom.Rect{W: 10, H: 5})
	c.Rect(geom.Rect{W: 1, H: 1}, color.White)
	if b := c.Batches[len(c.Batches)-1]; b.Clip != (geom.Rect{X: 90, Y: 50, W: 10, H: 20}) {
		t.Fatalf("a clip covers its transformed rect: %v", b.Clip)
	}
	c.PopClip()

	c.SetTransform(geom.Transform2DFrom(geom.V2(10.4, 5.6), 0, 0, geom.V2(1, 1)))
	c.SetModulate(color.White)
	c.SnapVertices = true
	n := len(c.Unindexed())
	c.Rect(geom.Rect{W: 3.3, H: 2}, color.White)
	if p := c.Unindexed()[n].Pos; p != geom.V2(10, 6) {
		t.Fatalf("snapped vertices land on pixels: %v", p)
	}
	if p := c.Unindexed()[n+1].Pos; p != geom.V2(14, 6) {
		t.Fatalf("snapped: %v", p)
	}
	if c.Transform().Origin != geom.V2(10.4, 5.6) {
		t.Fatal("the transform reads back")
	}
}

func TestTrianglesFillPolygons(t *testing.T) {
	var c List
	c.Triangles(
		[]geom.Vec2{geom.V2(0, 0), geom.V2(10, 0), geom.V2(10, 10), geom.V2(0, 10)},
		[]int{0, 1, 2, 0, 2, 3},
		color.Red,
	)
	if len(c.Unindexed()) != 6 || c.Unindexed()[5].Pos != geom.V2(0, 10) || c.Batches[0].Count != 6 {
		t.Fatalf("%d vertices", len(c.Unindexed()))
	}
}

func TestMeasureScalesWithSizeAndLines(t *testing.T) {
	small, big := Regular.Size(12).Measure("Ember"), Regular.Size(24).Measure("Ember")
	if big.X < small.X*1.8 {
		t.Fatalf("size scaling %v vs %v", small, big)
	}
	two := Regular.Size(12).Measure("a\nb")
	if geom.Abs(two.Y-2*Regular.Size(12).LineHeight()) > 1e-3 {
		t.Fatalf("multiline height %v", two.Y)
	}
}

func TestCanvasBatchesShapesWithText(t *testing.T) {
	var c List
	c.Rect(geom.Rect{W: 10, H: 10}, color.White)
	c.Text(geom.V2(0, 0), "Hi", Style{Size: 14})
	c.Orb(geom.V2(50, 50), 20, 0.5, color.Red)
	if len(c.Batches) != 1 {
		t.Fatalf("expected a single batch, got %d", len(c.Batches))
	}
	if n := c.Batches[0].Count; n != len(c.Unindexed()) || n%6 != 0 {
		t.Fatalf("batch covers %d of %d vertices", n, len(c.Unindexed()))
	}
	c.Text(geom.V2(0, 0), "Mono", Style{Face: Mono, Size: 14})
	if len(c.Batches) != 2 {
		t.Fatalf("a new atlas must start a new batch, got %d", len(c.Batches))
	}
	c.Reset(1)
	if len(c.Unindexed()) != 0 || c.Time() != 1 {
		t.Fatal("reset failed")
	}
}

func TestOrbClampsFill(t *testing.T) {
	var c List
	c.Orb(geom.V2(0, 0), 10, 3, color.Red)
	if c.Unindexed()[0].Shape.W != 1 {
		t.Fatalf("fill %v", c.Unindexed()[0].Shape.W)
	}
}

func textures(n int) []*render.Texture {
	var out []*render.Texture
	for range n {
		out = append(out, &render.Texture{Image: image.NewRGBA(image.Rect(0, 0, 2, 2))})
	}
	return out
}

func TestEightImagesShareABatchAndANinthStartsAnother(t *testing.T) {
	texs := textures(9)
	var c List
	for round := range 3 {
		for i, tex := range texs[:8] {
			c.Image(geom.Rect{X: float32(round*8 + i)}, tex, geom.Rect{W: 1, H: 1}, color.White)
		}
	}
	if len(c.Batches) != 1 || c.Batches[0].ImageCount != 8 || c.Batches[0].Count != 3*8*6 {
		t.Fatalf("three rounds of eight interleaved images: one batch, %+v", c.Batches)
	}
	for i, v := range c.Unindexed() {
		if want := i / 6 % 8; Slot(v.Shape.X) != want || Kind(v.Shape.X) != KindImage {
			t.Fatalf("vertex %d: slot %d kind %d, want slot %d of an image", i, Slot(v.Shape.X), Kind(v.Shape.X), want)
		}
	}
	c.Image(geom.Rect{}, texs[8], geom.Rect{W: 1, H: 1}, color.White)
	c.Image(geom.Rect{}, texs[0], geom.Rect{W: 1, H: 1}, color.White)
	if len(c.Batches) != 2 || c.Batches[1].Images[0] != texs[8] || c.Batches[1].Images[1] != texs[0] {
		t.Fatalf("a ninth image starts the next batch, which then takes what follows: %+v", c.Batches[1])
	}
}

func TestClipBlendAndMasksStillBreakBatchesInOrder(t *testing.T) {
	texs := textures(3)
	var c List
	draw := func(i int) { c.Image(geom.Rect{}, texs[i], geom.Rect{W: 1, H: 1}, color.White) }
	draw(0)
	c.PushClip(geom.Rect{W: 10, H: 10})
	draw(1)
	c.PopClip()
	draw(2)
	c.SetBlend(Add)
	draw(0)
	c.SetBlend(Alpha)
	pass, mask := NewPass(), NewPass()
	c.Picture(geom.Rect{W: 4, H: 4}, pass, Alpha, mask)
	draw(1)
	if len(c.Batches) != 6 {
		t.Fatalf("a clip, a blend and a masked picture each break: %d batches", len(c.Batches))
	}
	next := 0
	for i, b := range c.Batches {
		if b.First != next {
			t.Fatalf("batch %d starts at %d, not %d: draw order kept", i, b.First, next)
		}
		next += b.Count
	}
	if c.Batches[4].Mask != mask.Target || c.Batches[4].Image() != pass.Target || c.Batches[5].Image() != texs[1] {
		t.Fatal("the picture keeps its batch to itself")
	}
	target := &render.Texture{}
	c.Image(geom.Rect{}, target, geom.Rect{W: 1, H: 1}, color.White)
	draw(2)
	if len(c.Batches) != 8 {
		t.Fatalf("a render texture, without pixels of its own, is drawn apart: %d batches", len(c.Batches))
	}
}

func TestTrianglesTakeAColourPerPoint(t *testing.T) {
	var c List
	c.SetModulate(color.RGBA(1, 1, 1, 0.5))
	centre, rim := color.RGBA(0, 0, 0, 1), color.RGBA(0, 0, 0, 0)
	points := []geom.Vec2{{}, {X: 10}, {Y: 10}, {X: 10, Y: 10}}
	colors := []color.Color{centre, rim, rim, rim}
	c.TrianglesColored(points, colors, []int{0, 1, 2, 1, 3, 2, 0, 1, 9})
	if len(c.Unindexed()) != 6 || len(c.Batches) != 1 || c.Batches[0].Count != 6 {
		t.Fatalf(
			"two whole triangles, the one with a bad index left out: %d vertices, %+v",
			len(c.Unindexed()),
			c.Batches,
		)
	}
	want := []color.Color{centre, rim, rim, rim, rim, rim}
	for i, v := range c.Unindexed() {
		if v.Color != want[i].Mul(color.RGBA(1, 1, 1, 0.5)) || Kind(v.Shape.X) != KindSolid {
			t.Fatalf("vertex %d: %v, want its point's colour, modulated", i, v.Color)
		}
	}
	tex := &render.Texture{Image: image.NewRGBA(image.Rect(0, 0, 2, 2))}
	c.Image(geom.Rect{W: 2, H: 2}, tex, geom.Rect{W: 1, H: 1}, color.White)
	if len(c.Batches) != 1 {
		t.Fatal("it batches with what follows, like Triangles")
	}
}

func TestAQuadIsFourCornersAndTwoTrianglesByIndex(t *testing.T) {
	var c List
	c.Reset(0)
	c.Rect(geom.Rect{X: 1, Y: 2, W: 10, H: 20}, color.White)
	if len(c.Vertices) != 4 || len(c.Indices) != 6 {
		t.Fatalf("four corners, six indices: %d, %d", len(c.Vertices), len(c.Indices))
	}
	want := []geom.Vec2{{X: 1, Y: 2}, {X: 11, Y: 2}, {X: 11, Y: 22}, {X: 1, Y: 2}, {X: 11, Y: 22}, {X: 1, Y: 22}}
	for i, v := range c.Unindexed() {
		if v.Pos != want[i] {
			t.Fatalf("triangle corner %d at %v, want %v", i, v.Pos, want[i])
		}
	}
	if b := c.Batches[0]; b.First != 0 || b.Count != 6 {
		t.Fatalf("its batch is a range of the indices: %d, %d", b.First, b.Count)
	}
}

func TestAMeshMakesOneVertexAPointHoweverManyTrianglesShareIt(t *testing.T) {
	var c List
	c.Reset(0)
	points := []geom.Vec2{{}, {X: 1}, {X: 1, Y: 1}, {Y: 1}, {X: 9, Y: 9}}
	colors := []color.Color{color.White, color.White, color.White, color.White, color.White}
	c.TrianglesColored(points, colors, []int{0, 1, 2, 0, 2, 3})
	if len(c.Vertices) != 4 || len(c.Indices) != 6 {
		t.Fatalf("four points used, six indices: %d, %d", len(c.Vertices), len(c.Indices))
	}
	c.Rect(geom.Rect{W: 1, H: 1}, color.White)
	c.Triangles(points, []int{4, 1, 2}, color.White)
	if len(c.Vertices) != 4+4+3 || c.Indices[len(c.Indices)-3] != 8 {
		t.Fatalf(
			"each mesh its own vertices, indexed after the others: %d, %v",
			len(c.Vertices),
			c.Indices[len(c.Indices)-3:],
		)
	}
}

func TestARoundedClipRoundsItsBatchesAndTheClipsInsideIt(t *testing.T) {
	var c List
	radii := [4]float32{8, 8, 4, 4}
	panel := geom.Rect{X: 10, Y: 10, W: 100, H: 80}
	c.PushClipRounded(panel, radii)
	c.Rect(geom.Rect{W: 200, H: 200}, color.White)
	if b := c.Batches[0]; !b.Clipped || b.Clip != panel || b.Round != (Rounded{Rect: panel, Radii: radii}) {
		t.Fatalf("the batch is clipped to the panel and rounded by its radii: %+v", b)
	}
	c.PushClip(geom.Rect{X: 20, Y: 20, W: 30, H: 30})
	c.Rect(geom.Rect{W: 5, H: 5}, color.Black)
	if b := c.Batches[len(c.Batches)-1]; len(c.Batches) != 2 || b.Round.Rect != panel ||
		b.Clip != (geom.Rect{X: 20, Y: 20, W: 30, H: 30}) {
		t.Fatalf("a square clip inside keeps the panel's rounding: %+v", c.Batches)
	}
	inner := geom.Rect{X: 30, Y: 30, W: 10, H: 10}
	c.PushClipRounded(inner, [4]float32{2, 2, 2, 2})
	c.Rect(geom.Rect{W: 5, H: 5}, color.Black)
	if b := c.Batches[len(c.Batches)-1]; b.Round.Rect != inner {
		t.Fatalf("an inner rounded clip replaces the outer one: %+v", b.Round)
	}
	c.PopClip()
	c.PopClip()
	c.Rect(geom.Rect{W: 5, H: 5}, color.Black)
	if b := c.Batches[len(c.Batches)-1]; b.Round.Rect != panel {
		t.Fatalf("popping brings the panel's rounding back: %+v", b.Round)
	}
	c.PopClip()
	c.Rect(geom.Rect{W: 5, H: 5}, color.Black)
	if b := c.Batches[len(c.Batches)-1]; b.Clipped || !b.Round.IsZero() {
		t.Fatalf("and the last pop ends both: %+v", b)
	}
}

func TestARoundedClipFollowsAScaleButNotARotation(t *testing.T) {
	var c List
	c.SetTransform(geom.Transform2D{X: geom.V2(2, 0), Y: geom.V2(0, 2), Origin: geom.V2(5, 5)})
	c.PushClipRounded(geom.Rect{W: 10, H: 10}, [4]float32{3, 3, 3, 3})
	c.Rect(geom.Rect{W: 1, H: 1}, color.White)
	if r := c.Batches[0].Round; r.Rect != (geom.Rect{X: 5, Y: 5, W: 20, H: 20}) || r.Radii[0] != 6 {
		t.Fatalf("scaled, the rect and its radii scale: %+v", r)
	}
	var mirrored List
	mirrored.SetTransform(geom.Transform2D{X: geom.V2(-1, 0), Y: geom.V2(0, 1), Origin: geom.V2(100, 0)})
	mirrored.PushClipRounded(geom.Rect{W: 10, H: 10}, [4]float32{1, 2, 3, 4})
	mirrored.Rect(geom.Rect{W: 1, H: 1}, color.White)
	if r := mirrored.Batches[0].Round.Radii; r != [4]float32{2, 1, 4, 3} {
		t.Fatalf("mirrored left to right, each corner's radius goes with it: %v", r)
	}
	var flipped List
	flipped.SetTransform(geom.Transform2D{X: geom.V2(1, 0), Y: geom.V2(0, -1), Origin: geom.V2(0, 100)})
	flipped.PushClipRounded(geom.Rect{W: 10, H: 10}, [4]float32{1, 2, 3, 4})
	flipped.Rect(geom.Rect{W: 1, H: 1}, color.White)
	if r := flipped.Batches[0].Round.Radii; r != [4]float32{4, 3, 2, 1} {
		t.Fatalf("flipped upside down, too: %v", r)
	}
	var turned List
	turned.SetTransform(geom.Transform2DFrom(geom.Vec2{}, 0.3, 0, geom.V2(1, 1)))
	turned.PushClipRounded(geom.Rect{W: 10, H: 10}, [4]float32{3, 3, 3, 3})
	turned.Rect(geom.Rect{W: 1, H: 1}, color.White)
	if b := turned.Batches[0]; !b.Clipped || !b.Round.IsZero() {
		t.Fatalf("turned, it clips to its bounds, square: %+v", b)
	}
}

func TestARoundedClipSplitsBatchesAndShiftsWithTheList(t *testing.T) {
	var c List
	c.Rect(geom.Rect{W: 5, H: 5}, color.White)
	c.PushClipRounded(geom.Rect{W: 50, H: 50}, [4]float32{4, 4, 4, 4})
	c.Rect(geom.Rect{W: 5, H: 5}, color.White)
	c.PopClip()
	c.PushClipRounded(geom.Rect{W: 50, H: 50}, [4]float32{6, 6, 6, 6})
	c.Rect(geom.Rect{W: 5, H: 5}, color.White)
	if len(c.Batches) != 3 {
		t.Fatalf("another rounding is another batch: %d", len(c.Batches))
	}
	c.Shift(geom.V2(7, 3))
	if r := c.Batches[2].Round.Rect; r.X != 7 || r.Y != 3 {
		t.Fatalf("shifting moves the rounded rect with the clip: %+v", r)
	}
}
