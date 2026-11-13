// Tileset is a tool for a Tiled tileset's collision and shadow shapes:
// pick a tile, draw its rectangles, ellipses and polygons over it,
// magnified and snapped to its art pixels, collision (orange) or light
// only (blue), set its properties, and see them in a second window: the
// tile lit by a light the mouse drags, with live shadows, and a body
// walked into it with WASD, through the engine's own lighting and
// collision. Save writes the shapes and properties back into the file,
// keeping everything else as it was, and refuses when the file changed
// on disk since it was opened. File › Open… (or no argument) shows a
// dialog of the folder's tilesets and the recent ones, remembered in
// EMBER_TILESET_DIR, else ~/.ember-tileset.
//
//	go run ./tools/tileset [-room path/to/room.tmj] [path/to/walls.tsj]
//
// -room, or the automation action preview-room with a "path", opens the
// room preview at once, for captures and scripts.
package main

import (
	"flag"
	"fmt"
	"image"
	"os"

	"github.com/merlindorin/ember"
)

func main() {
	props := flag.String("props", "", "a JSON file of the game's tile properties, beside the engine's")
	flag.StringVar(&startDir, "dir", "", "the folder the open dialog starts in, until a tileset is opened")
	project := flag.Bool(
		"tiled-project",
		false,
		"print the Tiled project file of the engine's and the game's tile properties, and stop",
	)
	room := flag.String("room", "", "a Tiled map (.tmj) to show the tileset in, as View › Preview room… does")
	flag.Usage = func() {
		fmt.Fprintln(
			os.Stderr,
			"usage: go run ./tools/tileset [-props game.json] [-dir folder] [-room room.tmj] [-tiled-project] "+
				"[path/to/tileset.tsj]",
		)
		flag.PrintDefaults()
	}
	flag.Parse()
	if *props != "" {
		defs, err := readProperties(*props)
		if err != nil {
			fmt.Fprintln(os.Stderr, "tileset:", err)
			os.Exit(1)
		}
		declared = append(engineTileProperties(), defs...)
	}
	if *project {
		if _, err := os.Stdout.Write(tiledProject()); err != nil {
			fmt.Fprintln(os.Stderr, "tileset:", err)
			os.Exit(1)
		}
		return
	}
	if flag.NArg() > 1 {
		flag.Usage()
		os.Exit(2)
	}
	path := flag.Arg(0)
	auto := recorder()
	w, tool, err := build(path, ember.Automation(auto))
	if err != nil {
		fmt.Fprintln(os.Stderr, "tileset:", err)
		os.Exit(1)
	}
	tool.define(auto)
	if *room != "" {
		if err = tool.previewRoom(*room); err != nil {
			fmt.Fprintln(os.Stderr, "tileset:", err)
			os.Exit(1)
		}
	}
	w.MustRun()
}

func newRGBA(w, h int) *image.RGBA { return image.NewRGBA(image.Rect(0, 0, w, h)) }
