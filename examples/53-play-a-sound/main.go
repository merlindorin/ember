// Play a sound: Space plays a short pling a little higher or lower each
// time, and a click plays it from the side of the window it lands on. The
// mixer is an entity like the window; the device opens when the game runs.
// GIFs carry no sound: run it to hear it.
//
//	go run ./examples/53-play-a-sound
//
// hit.wav is synthesised by sounds_gen.go and free to use (CC0).
package main

//go:generate go run sounds_gen.go

import (
	"embed"
	"log"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/audio"
	_ "github.com/lantervale/ember/audio/wav"
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/platform"
)

//go:embed hit.wav
var files embed.FS

func main() {
	hit, err := audio.Load(files, "hit.wav")
	if err != nil {
		log.Fatal(err)
	}
	mixer := audio.NewMixer()
	win := platform.NewWindow(
		platform.Title("ember · play a sound"),
		platform.Size(640, 360),
		platform.FixedSize(),
		platform.Background(color.Hex("#283040")),
	)
	player := newPlayer(win, mixer, hit)
	win.Add(player.layer)

	ember.New().
		AddEntities(win, mixer).
		AddSystems(player).
		MustRun()
}
