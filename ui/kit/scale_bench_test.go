package kit_test

import (
	"fmt"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/kit"
)

// viewport is the size the benchmarks lay out at.
func viewport() geom.Vec2 { return geom.V2(800, 600) }

// list is a column of rows of a label and a button, three elements a row, about n in all.
func list(n int) (*ui.Node, []*ui.Node) {
	var buttons []*ui.Node
	col := kit.Column()
	for i := range n / 3 {
		b := kit.Button("Edit")
		buttons = append(buttons, b)
		col.Append(kit.Row(kit.Label(fmt.Sprintf("tile %d", i)), b))
	}
	return col, buttons
}

// table is a table of rows of three cells, four elements a row, about n in all.
func table(n int) (*ui.Node, []*ui.Node) {
	var cells []*ui.Node
	body := ui.Tbody()
	for i := range n / 4 {
		c := ui.Td(ui.Text("walls"))
		cells = append(cells, c)
		body.Append(ui.Tr(ui.Td(ui.Text(fmt.Sprintf("tile %d", i))), c, ui.Td(ui.Text("true"))))
	}
	return ui.Table(body), cells
}

func laidOutDocument(content *ui.Node) *ui.Document {
	d := ui.NewDocument(kit.Theme())
	d.Body().Append(content)
	d.Layout(viewport())
	return d
}

// BenchmarkScale measures a static frame and a changed one (a hover in the middle of the
// page) for lists and tables of 200, 1,000 and 5,000 elements.
func BenchmarkScale(b *testing.B) {
	for _, kind := range []struct {
		name  string
		build func(int) (*ui.Node, []*ui.Node)
	}{{"list", list}, {"table", table}} {
		for _, n := range []int{200, 1000, 5000} {
			content, targets := kind.build(n)
			d := laidOutDocument(content)
			mid := targets[len(targets)/2]
			b.Run(fmt.Sprintf("%s/%d/static", kind.name, n), func(b *testing.B) {
				b.ReportAllocs()
				for range b.N {
					d.Layout(viewport())
				}
			})
			b.Run(fmt.Sprintf("%s/%d/changed", kind.name, n), func(b *testing.B) {
				b.ReportAllocs()
				on := false
				for range b.N {
					on = !on
					mid.SetState(dom.Hover, on)
					d.Layout(viewport())
				}
			})
		}
	}
}

// BenchmarkScrolling measures a frame of a 1,000-row list scrolling in a kit.Scroll.
func BenchmarkScrolling(b *testing.B) {
	content, _ := list(3000)
	sc := kit.Scroll(ui.Style("height: 400px"), content)
	d := laidOutDocument(sc)
	b.ReportAllocs()
	y := float32(0)
	for range b.N {
		y = float32(int(y+37) % 20000)
		dom.Data(sc).Scroll = geom.V2(0, y)
		d.Layout(viewport())
	}
}

// BenchmarkStartup measures making a Document with the kit theme, adding a page of 200
// and of 1,000 elements, and its first layout.
func BenchmarkStartup(b *testing.B) {
	for _, n := range []int{200, 1000} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				content, _ := list(n)
				laidOutDocument(content)
			}
		})
	}
}
