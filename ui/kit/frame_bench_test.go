package kit_test

import (
	"fmt"
	"testing"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/internal/draw"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/kit"
	"github.com/lantervale/ember/world"
)

// framed is content in a Document in an 800×600 window of a headless World:
// a frame is the whole of it, input, styling, layout, paint and the draw
// list the GPU takes, without the GPU.
type framed struct {
	w   *world.World
	win *platform.Window
	doc *ui.Document
}

func frame(content *ui.Node) *framed {
	f := &framed{doc: ui.NewDocument(kit.Theme()),
		win: platform.NewWindow(platform.Size(int(viewport().X), int(viewport().Y)))}
	f.doc.Body().Append(content)
	f.win.Add(f.doc)
	f.w = ember.New().AddEntities(f.win)
	f.w.RunFrames(3)
	return f
}

// run times b.N frames, each after step, and reports the draw list of the
// last: its draw batches and vertices.
func (f *framed) run(b *testing.B, step func(i int)) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		if step != nil {
			step(i)
		}
		f.w.RunFrames(1)
	}
	b.StopTimer()
	if l := draw.FrontFor(f.w, f.win); l != nil {
		b.ReportMetric(float64(len(l.Batches)), "batches")
		b.ReportMetric(float64(len(l.Vertices)), "vertices")
	}
}

// BenchmarkScaleFrame is BenchmarkScale's lists and tables as whole frames:
// static, and with the pointer moving between two of their buttons or
// cells, so hover changes every frame.
func BenchmarkScaleFrame(b *testing.B) {
	for _, kind := range []struct {
		name  string
		build func(int) (*ui.Node, []*ui.Node)
	}{{"list", list}, {"table", table}} {
		for _, n := range []int{200, 1000, 5000} {
			content, targets := kind.build(n)
			f := frame(content)
			b.Run(fmt.Sprintf("%s/%d/static", kind.name, n), func(b *testing.B) { f.run(b, nil) })
			a, z := targets[0].Box().Center(), targets[1].Box().Center()
			b.Run(fmt.Sprintf("%s/%d/hover", kind.name, n), func(b *testing.B) {
				f.run(b, func(i int) {
					at := a
					if i%2 == 1 {
						at = z
					}
					f.win.Input().SetMouse(at)
				})
			})
		}
	}
}

// BenchmarkScrollingFrame is a 1,000-row list scrolling by the wheel in a
// kit.Scroll, as whole frames.
func BenchmarkScrollingFrame(b *testing.B) {
	content, _ := list(3000)
	sc := kit.Scroll(ui.Style("height: 400px"), content)
	f := frame(sc)
	f.win.Input().SetMouse(sc.Box().Center())
	f.w.RunFrames(1)
	f.run(b, func(i int) {
		d := float32(-1)
		if i/200%2 == 1 {
			d = 1
		}
		f.win.Input().AddWheel(geom.V2(0, d))
	})
}

// BenchmarkEmptyFrame is the World's frame with an empty Document: what
// the engine costs around ui.
func BenchmarkEmptyFrame(b *testing.B) {
	frame(ui.Div()).run(b, nil)
}
