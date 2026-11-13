// Collision: the walker of 20-sprite-sheet in a walled room. The walls are
// solid tiles, the hero has a body at its feet, and Walk moves it through
// the collision space, which stops it at walls and slides it along them.
// Stepping on the sand plate, a trigger, opens the door in the middle wall.
//
//	go run ./examples/27-collision
package main

import (
	"embed"
	"log"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/collide"
	"github.com/merlindorin/ember/collide/tiles"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
)

//go:embed tiles.png walker.png
var files embed.FS

func main() {
	win := platform.NewWindow(
		platform.Title("ember · collision"),
		platform.Size(640, 384),
		platform.FixedSize(),
	)

	sheet, err := twod.LoadTexture(files, "tiles.png", twod.Pixelated())
	if err != nil {
		log.Fatal(err)
	}
	walls := twod.NewTileMap(twod.NewTileset(sheet, twod.Cell(16, 16), tiles.Solid(wall)),
		twod.ASCII(room, legend()),
		twod.CellSize(32, 32),
	)
	hero, err := newHero(walls.Center(3, 8))
	if err != nil {
		log.Fatal(err)
	}
	plate := twod.NewSprite(nil, twod.At(walls.Center(plateX, plateY).X, walls.Center(plateX, plateY).Y))
	plate.Add(collide.NewBody(collide.Box(32, 32), collide.Trigger()))

	space := collide.NewSpace()
	space.AddSolid(tiles.Of(walls))
	win.Add(walls, plate, hero)

	ember.New().
		AddEntities(win).
		AddSystems(
			collide.NewTracker(space),
			&Walk{hero: hero, keys: win.Input(), space: space},
			&Door{walls: walls},
		).
		MustRun()
}
