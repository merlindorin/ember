package draw //nolint:testpackage // white-box: reaches draw's unexported state

import (
	"image"
	"testing"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/render"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// page draws what a UI host would: shapes, clipped text and an image.
func page(c *List, img *render.Texture) {
	c.Rect(geom.Rect{X: 10, Y: 10, W: 200, H: 40}, color.RGB(0.2, 0.2, 0.2))
	c.PushClip(geom.Rect{X: 10, Y: 10, W: 120, H: 40})
	c.Text(geom.V2(14, 30), "Handgloves 12", Style{Size: 15, Baseline: true})
	c.PopClip()
	c.RoundRect(geom.Rect{X: 20, Y: 60, W: 80, H: 24}, 4, color.RGB(0.8, 0.6, 0.2))
	c.Image(geom.Rect{X: 120, Y: 60, W: 24, H: 24}, img, geom.Rect{W: 1, H: 1}, color.White)
	c.Text(geom.V2(30, 78), "Save", Style{Size: 13, Baseline: true})
}

func picture() *render.Texture { return &render.Texture{Image: image.NewRGBA(image.Rect(0, 0, 8, 8))} }

// framed draws before, then the page, then after, in a fresh list,
// recording the page into r when r is given.
func framed(img *render.Texture, r *Recording) *List {
	var c List
	c.Rect(geom.Rect{W: 4, H: 4}, color.White)
	m := c.Mark()
	page(&c, img)
	if r != nil {
		c.Record(m, r)
	}
	c.Rect(geom.Rect{X: 300, W: 4, H: 4}, color.White)
	return &c
}

func TestAReplayDrawsExactlyWhatWasRecorded(t *testing.T) {
	img := picture()
	var r Recording
	want := framed(img, &r)
	require.True(t, r.Ok())

	var c List
	c.Rect(geom.Rect{W: 4, H: 4}, color.White)
	require.True(t, c.Replay(&r))
	c.Rect(geom.Rect{X: 300, W: 4, H: 4}, color.White)
	assert.Equal(t, want.Vertices, c.Vertices)
	assert.Equal(t, want.Indices, c.Indices)
	assert.Equal(t, want.Batches, c.Batches, "the same batches, merged the same way")
}

func TestAReplayAllocatesNothingOnceItsListHasRoom(t *testing.T) {
	img := picture()
	var r Recording
	framed(img, &r)
	var c List
	c.Replay(&r)
	assert.Zero(t, testing.AllocsPerRun(100, func() {
		c.Reset(0)
		c.Replay(&r)
	}))
	var again Recording
	assert.Zero(t, testing.AllocsPerRun(100, func() {
		c.Reset(0)
		m := c.Mark()
		c.Replay(&r)
		c.Record(m, &again)
	}), "recording again into the same Recording")
}

func TestAReplayRefusesAListInAnotherState(t *testing.T) {
	var r Recording
	framed(picture(), &r)
	for name, set := range map[string]func(c *List){
		"transform": func(c *List) { c.SetTransform(geom.Transform2D{X: geom.V2(2, 0), Y: geom.V2(0, 2)}) },
		"modulate":  func(c *List) { c.SetModulate(color.RGBA(1, 1, 1, 0.5)) },
		"blend":     func(c *List) { c.SetBlend(Add) },
		"snap":      func(c *List) { c.SnapVertices = true },
		"clip":      func(c *List) { c.PushClip(geom.Rect{W: 50, H: 50}) },
	} {
		var c List
		set(&c)
		assert.False(t, c.Replay(&r), name)
		assert.Empty(t, c.Vertices, "%s: nothing appended", name)
	}
}

func TestAReplayRefusesAnotherRounding(t *testing.T) {
	panel := geom.Rect{W: 200, H: 100}
	var r Recording
	var c List
	c.PushClipRounded(panel, [4]float32{8, 8, 8, 8})
	m := c.Mark()
	page(&c, picture())
	require.True(t, c.Record(m, &r))

	var same List
	same.PushClipRounded(panel, [4]float32{8, 8, 8, 8})
	assert.True(t, same.Replay(&r), "the same rounded clip replays")
	var other List
	other.PushClipRounded(panel, [4]float32{10, 8, 8, 8})
	assert.False(t, other.Replay(&r), "one radius changed refuses it")
}

func TestLitDrawingIsNeverRecorded(t *testing.T) {
	var c List
	m := c.Mark()
	c.Rect(geom.Rect{W: 10, H: 10}, color.White)
	c.Vertices[0].LightSpread = geom.V2(-1, 0)
	var r Recording
	assert.False(t, c.Record(m, &r))
	assert.False(t, r.Ok())
	var d List
	assert.False(t, d.Replay(&r))
}

func TestAPassPictureIsNeverRecorded(t *testing.T) {
	var c List
	m := c.Mark()
	c.Image(geom.Rect{W: 10, H: 10}, &render.Texture{}, geom.Rect{W: 1, H: 1}, color.White)
	var r Recording
	assert.False(t, c.Record(m, &r), "a render texture's batch is drawn alone")
}

func TestARecordingStartingInsideAnOpenBatchReplaysItsOwnPart(t *testing.T) {
	var c List
	c.Rect(geom.Rect{W: 4, H: 4}, color.White)
	m := c.Mark()
	c.Rect(geom.Rect{X: 10, W: 4, H: 4}, color.White)
	var r Recording
	require.True(t, c.Record(m, &r))
	require.Len(t, c.Batches, 1, "drawn into the open batch")

	var d List
	d.Rect(geom.Rect{W: 4, H: 4}, color.White)
	require.True(t, d.Replay(&r))
	assert.Equal(t, c.Vertices, d.Vertices)
	assert.Equal(t, c.Batches, d.Batches, "merged back into the open batch")
}

func TestAReplayAtItsOwnBaseCopiesItsIndicesAsTheyAre(t *testing.T) {
	var r Recording
	want := framed(picture(), &r)
	var c List
	c.Rect(geom.Rect{W: 4, H: 4}, color.White)
	require.Equal(t, r.base, uint32(len(c.Vertices)), "the vertex the page was recorded at")
	require.True(t, c.Replay(&r))
	assert.Equal(t, want.Indices[6:len(want.Indices)-6], c.Indices[6:], "the page's indices, verbatim")
}

func TestAReplayElsewhereMovesItsIndices(t *testing.T) {
	img := picture()
	var r Recording
	framed(img, &r)

	var want List
	want.Rect(geom.Rect{W: 4, H: 4}, color.White)
	want.Rect(geom.Rect{X: 8, W: 4, H: 4}, color.White)
	page(&want, img)

	var c List
	c.Rect(geom.Rect{W: 4, H: 4}, color.White)
	c.Rect(geom.Rect{X: 8, W: 4, H: 4}, color.White)
	require.NotEqual(t, r.base, uint32(len(c.Vertices)))
	require.True(t, c.Replay(&r))
	assert.Equal(t, want.Indices, c.Indices, "every index moved by the four vertices drawn first")
	assert.Equal(t, want.Vertices, c.Vertices)
	assert.Equal(t, want.Batches, c.Batches)
}

func BenchmarkReplay(b *testing.B) {
	img := picture()
	var r Recording
	var c List
	c.Reset(0)
	c.Rect(geom.Rect{W: 4, H: 4}, color.White)
	m := c.Mark()
	for range 200 {
		page(&c, img)
	}
	if !c.Record(m, &r) {
		b.Fatal("not recorded")
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		c.Reset(0)
		c.Rect(geom.Rect{W: 4, H: 4}, color.White)
		if !c.Replay(&r) {
			b.Fatal("refused")
		}
	}
	b.ReportMetric(float64(len(c.Indices)), "indices")
}
