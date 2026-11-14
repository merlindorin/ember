// Easing and curves: a walker per easing family slides across its lane and
// back with a ping-pong tween, so each shape of motion shows side by side, and
// one more walks round a Catmull-Rom path. A tween is the walker's child:
// it calls its function with the eased progress every frame.
//
//	go run ./examples/35-easing
package main

import (
	"embed"
	"log"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/ease"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/twod"
)

//go:embed walker.png
var files embed.FS

const (
	start, end = 130, 440
	top, row   = 74, 50
)

func main() {
	win := platform.NewWindow(
		platform.Title("ember · easing"),
		platform.Size(760, 470),
		platform.FixedSize(),
		platform.Background(color.Hex("#161b2c")),
	)
	walker, err := twod.LoadTexture(files, "walker.png", twod.Pixelated())
	if err != nil {
		log.Fatal(err)
	}
	side := twod.Strip(walker, twod.Cell(16, 16))[8]

	muted := color.Hex("#aab4d4")
	rows := twod.NewLayer(0)
	rows.Add(
		twod.NewText("InOut of each family", twod.At(16, 18), twod.Pivot(0, 0), twod.FontSize(15), twod.Color(muted)),
	)
	for i, e := range easings() {
		y := float32(top + i*row)
		from, to := geom.V2(start, y), geom.V2(end, y)
		sprite := twod.NewSprite(
			walker,
			twod.Still(side),
			twod.At(from.X, from.Y),
			twod.Scale(3, 3),
			twod.Pivot(0.5, 1),
		)
		sprite.Add(ease.NewTween(2, func(t float32) {
			sprite.SetPosition(from.Lerp(to, t))
		}, ease.Easing(e.f), ease.PingPong()))
		rows.Add(
			twod.NewLine(
				[]geom.Vec2{geom.V2(start-20, y), geom.V2(end+20, y)},
				twod.Width(2),
				twod.Color(color.Hex("#2a3352")),
			),
			twod.NewText(e.name, twod.At(16, y-12), twod.Pivot(0, 0.5), twod.FontSize(17), twod.Color(muted)),
			sprite,
		)
	}

	win.Add(rows, newPath(walker))

	ember.New().
		AddEntities(win).
		MustRun()
}
