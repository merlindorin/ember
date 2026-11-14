package kit_test

import (
	"fmt"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/kit"
)

// wheeling scrolls f by the wheel over at, down for 200 frames then up.
func wheeling(f *framed, at geom.Vec2) func(i int) {
	f.win.Input().SetMouse(at)
	f.w.RunFrames(1)
	return func(i int) {
		d := float32(-1)
		if i/200%2 == 1 {
			d = 1
		}
		f.win.Input().AddWheel(geom.V2(0, d))
	}
}

// BenchmarkTableScrolling is a 5,000-row kit.Table scrolling by the wheel,
// as whole frames, beside a 5,000-row list in a kit.Scroll, the frame it
// must keep up with.
func BenchmarkTableScrolling(b *testing.B) {
	b.Run("list/5000", func(b *testing.B) {
		content, _ := list(15000)
		sc := kit.Scroll(ui.Style("height: 400px"), content)
		f := frame(sc)
		f.run(b, wheeling(f, sc.Box().Center()))
	})
	b.Run("table/5000", func(b *testing.B) {
		var rows []*ui.Node
		for i := range 5000 {
			rows = append(rows, kit.TableRow(fmt.Sprintf("tile_%04d.png", i), fmt.Sprintf("%d KB", i%97), "Picture"))
		}
		t := kit.Table([]*ui.Node{kit.TableColumn("Name", kit.Sortable()),
			kit.TableColumn("Size", kit.ColumnWidth(70), kit.ColumnAlign("right")),
			kit.TableColumn("Kind", kit.ColumnWidth(90))}, rows, ui.Style("height: 400px; width: 500px"))
		f := frame(t)
		f.run(b, wheeling(f, t.Box().Center()))
	})
}
