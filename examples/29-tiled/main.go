// A level made in Tiled: level.tmj, tiles.tsj and objects.tsj are what the
// editor saves. LoadTiled gives back its layers in the file's order: tile
// layers as tile maps, whose wall tiles are solid by a property set on the
// tileset, and an object layer, from which the game spawns the hero and the
// pillars, tile objects drawn as Tiled draws them.
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

//go:embed level.tmj tiles.tsj tiles.png objects.tsj objects.png walker.png
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
	things := twod.NewLayer(1, twod.SortByY())
	var hero *twod.Sprite
	for _, layer := range level.Layers() {
		switch layer.Kind {
		case twod.TiledTiles:
			win.Add(layer.TileMap())
			space.AddSolid(layer.TileMap())
		case twod.TiledObjects:
			if hero, err = spawn(things, level, layer.Objects()); err != nil {
				log.Fatal(err)
			}
			win.Add(things)
		}
	}

	ember.New().
		AddEntities(win).
		AddSystems(
			collide.NewTracker(space),
			&Walk{hero: hero, keys: win.Input(), space: space},
		).
		MustRun()
}
