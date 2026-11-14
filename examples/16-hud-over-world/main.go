// A HUD over the world: three walkers wander a meadow, each with its name
// and a health bar that follow it on screen, while the score stays in the
// window's top-right corner. Click a walker to hurt it.
//
//	go run ./examples/16-hud-over-world
package main

import (
	"embed"
	"log"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/kit"
)

//go:embed walker.png tiles.png
var files embed.FS

// page draws each tag as its name over a 48 × 6 bar, and keeps the score
// in the window's top-right corner.
const page = `.tag { display: flex; flex-direction: column; align-items: center; gap: 2px; font-size: 12px; }
.tag > .kit-progress { width: 48px; height: 6px; } #score { position: fixed; right: 16px; top: 12px; font-size: 20px; }`

func main() {
	win := platform.NewWindow(
		platform.Title("ember · HUD over the world"),
		platform.Size(640, 384),
		platform.FixedSize(),
	)
	sheet, err := twod.LoadTexture(files, "walker.png", twod.Pixelated())
	if err != nil {
		log.Fatal(err)
	}
	tiles, err := twod.LoadTexture(files, "tiles.png", twod.Pixelated())
	if err != nil {
		log.Fatal(err)
	}
	walkers := newWalkers(sheet)
	hud, score := newHUD(walkers)

	win.Add(newMeadow(tiles))
	for _, w := range walkers {
		win.Add(w.sprite)
	}
	win.Add(hud)

	ember.New().
		AddEntities(win).
		AddSystems(&Wander{walkers: walkers}, &Hit{walkers: walkers, mouse: win.Input(), score: score}).
		MustRun()
}

// newHUD is the Document over the meadow: the walkers' tags and the score.
func newHUD(walkers []*walker) (*ui.Document, *ui.Node) {
	doc := ui.NewDocument(kit.Theme(), ui.CSS(page))
	score := kit.Label("Hits 0", ui.ID("score"))
	for _, w := range walkers {
		doc.Body().Append(w.tag)
	}
	doc.Body().Append(score)
	return doc, score
}
