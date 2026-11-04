// Music and positional sound: the day ambience loops on the music bus, and
// M crossfades to the night one and back. The hero is the listener: the
// campfire and the river grow louder as he walks to them, and are heard
// from his side, so turning him with Q and E swaps them between his ears.
// GIFs carry no sound: run it to hear it.
//
//	go run ./examples/53-music-and-positional
//
// Every sound is a CC0 recording cut into a seamless loop (CREDITS.md).
package main

import (
	"embed"
	"log"
	"time"

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
		platform.Title("ember · music and positional sound"),
		platform.Size(640, 360),
		platform.FixedSize(),
		platform.Background(dayColor()),
	)
	mixer := audio.NewMixer(audio.WithBus("music", 0.5), audio.WithBus("sfx", 1))
	mixer.PlayMusic(s.day, audio.Loop(), audio.FadeIn(time.Second), audio.Volume(dayVolume))

	hero, err := newHero()
	if err != nil {
		log.Fatal(err)
	}
	fire := newCampfire(s.crackle)
	river := newRiver(s.river)
	hud := newHUD(hero, fire, river)
	win.Add(twod.NewLayer(0).Add(river, fire, hero), hud.layer)

	ember.New().
		AddEntities(win, mixer).
		AddSystems(
			&Walk{hero: hero, in: win.Input()},
			&Music{window: win, mixer: mixer, sounds: s, hud: hud, fading: 1},
			hud,
		).
		MustRun()
}

func dayColor() color.Color   { return color.Hex("#4f7d43") }
func nightColor() color.Color { return color.Hex("#1f2b3d") }

type sounds struct {
	day, night *audio.Stream
	crackle    *audio.Stream
	river      *audio.Stream
}

func load() (*sounds, error) {
	var s sounds
	var err error
	if s.day, err = audio.Open(files, "day.ogg"); err != nil {
		return nil, err
	}
	if s.night, err = audio.Open(files, "night.ogg"); err != nil {
		return nil, err
	}
	if s.crackle, err = audio.Open(files, "crackle.ogg"); err != nil {
		return nil, err
	}
	if s.river, err = audio.Open(files, "river.ogg"); err != nil {
		return nil, err
	}
	return &s, nil
}
