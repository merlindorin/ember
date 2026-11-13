// Audio effects: a corridor opens into a great hall. The sfx bus runs
// through a reverb that grows from the corridor's small room to the hall's
// large one as the hero walks in, so his footsteps and claps ring longer.
// Drums play behind a door, through a low-pass that muffles them until E
// opens the door. GIFs carry no sound: run it to hear it.
//
//	go run ./examples/55-audio-effects
//
// The sounds are synthesised by sounds_gen.go and free to use (CC0).
package main

//go:generate go run sounds_gen.go

import (
	"embed"
	"log"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/audio"
	_ "github.com/merlindorin/ember/audio/vorbis"
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
)

//go:embed *.ogg walker.png
var files embed.FS

func main() {
	s, err := load()
	if err != nil {
		log.Fatal(err)
	}
	win := platform.NewWindow(
		platform.Title("ember · audio effects"),
		platform.Size(640, 360),
		platform.FixedSize(),
		platform.Background(color.Hex("#1b1a22")),
	)
	room := audio.NewReverb(audio.RoomSize(corridor().size), audio.Wet(corridor().wet))
	mixer := audio.NewMixer(audio.WithBus("sfx", 1, audio.Effects(room)))
	hero, err := newHero()
	if err != nil {
		log.Fatal(err)
	}
	door := newDoor(s.drums)
	hud := newHUD()
	win.Add(twod.NewLayer(0).Add(newDungeon(), door, door.band, hero), hud.layer)

	ember.New().
		AddEntities(win, mixer).
		AddSystems(
			&Walk{hero: hero, in: win.Input(), mixer: mixer, sounds: s},
			&Acoustics{hero: hero, room: room},
			&Doorway{door: door, hero: hero, in: win.Input()},
			&Status{hud: hud, hero: hero, room: room, door: door},
		).
		MustRun()
}

type sounds struct {
	step, clap *audio.Clip
	drums      *audio.Clip
}

func load() (*sounds, error) {
	var s sounds
	var err error
	if s.step, err = audio.Load(files, "step.ogg"); err != nil {
		return nil, err
	}
	if s.clap, err = audio.Load(files, "clap.ogg"); err != nil {
		return nil, err
	}
	if s.drums, err = audio.Load(files, "drums.ogg"); err != nil {
		return nil, err
	}
	return &s, nil
}
