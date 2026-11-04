package pages

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/internal/dom"
)

// Frame is one kind of frame's cost: CPU time and allocations for styling and laying
// out the page.
type Frame struct {
	NsPerOp     int64 `json:"ns_per_op"`
	AllocsPerOp int64 `json:"allocs_per_op"`
	BytesPerOp  int64 `json:"bytes_per_op"`
}

// Frames are a page's frame costs: its first layout, a frame where nothing changed, one
// where a button's hover changes, and one where a text field's value changes; the last
// two are missing when the page has no such element.
type Frames struct {
	First  Frame  `json:"first"`
	Static Frame  `json:"static"`
	Hover  *Frame `json:"hover,omitempty"`
	Edit   *Frame `json:"edit,omitempty"`
	// World are the whole frames, paint and draw list included.
	World WorldFrames `json:"world"`
}

// bench measures d's frames in a viewport of w×h points and writes them as JSON to
// stdout.
func bench(d *ui.Document, w, h int) {
	size := geom.V2(float32(w), float32(h))
	start := time.Now()
	d.Layout(size)
	var f Frames
	f.First = Frame{NsPerOp: time.Since(start).Nanoseconds()}
	f.Static = measure(func() { d.Layout(size) })
	if b := first(d, func(n *ui.Node) bool { return n.Tag() == dom.Button }); b != nil {
		on := false
		r := measure(func() {
			on = !on
			b.SetState(dom.Hover, on)
			d.Layout(size)
		})
		f.Hover = &r
	}
	if in := first(d, func(n *ui.Node) bool {
		return n.Tag() == dom.Textarea || n.Tag() == dom.Input &&
			isText(n)
	}); in != nil {
		text := []string{"a", "ab"}
		i := 0
		r := measure(func() {
			i++
			d.SetValue(in, text[i%2])
			d.Layout(size)
		})
		f.Edit = &r
	}
	f.World = worldFrames(d, w, h)
	if err := json.NewEncoder(os.Stdout).Encode(f); err != nil {
		panic(err)
	}
}

func measure(frame func()) Frame {
	r := testing.Benchmark(func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			frame()
		}
	})
	return Frame{NsPerOp: r.NsPerOp(), AllocsPerOp: r.AllocsPerOp(), BytesPerOp: r.AllocedBytesPerOp()}
}

func first(d *ui.Document, match func(*ui.Node) bool) *ui.Node {
	var found *ui.Node
	d.Body().Walk(func(n *ui.Node) bool {
		if found == nil && !n.IsText() && match(n) {
			found = n
		}
		return found == nil
	})
	return found
}

func isText(n *ui.Node) bool {
	t, ok := n.Attr("type")
	return !ok || t == "text" || t == "search" || t == "password"
}
