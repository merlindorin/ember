// A level made in Tiled: level.tmj and tiles.tsj are what the editor saves.
// LoadTiled turns its tile layers into tile maps, whose wall tiles are
// solid by a property set on the tileset, and gives back its objects, from
// which the game spawns the hero and the pillars.
//
//	go run ./examples/29-tiled
package main

import (
	"embed"
	"log"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/collide"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
)

//go:embed level.tmj tiles.tsj tiles.png walker.png objects.png
var files embed.FS

func main() {
	win := platform.NewWindow(
		platform.Title("ember · Tiled"),
		platform.Size(640, 384),
		platform.FixedSize(),
	)

	level, err := twod.LoadTiled(files, "level.tmj")
	if err != nil {
		log.Fatal(err)
	}
	space := collide.NewSpace()
	for _, layer := range level.Layers() {
		win.Add(layer)
		space.AddSolid(layer)
	}

	things := twod.NewLayer(1, twod.SortByY())
	hero, err := spawn(things, level.Objects("things"))
	if err != nil {
		log.Fatal(err)
	}
	win.Add(things)

	ember.New().
		AddEntities(win).
		AddSystems(
			collide.NewTracker(space),
			&Walk{hero: hero, keys: win.Input(), space: space},
		).
		MustRun()
}
