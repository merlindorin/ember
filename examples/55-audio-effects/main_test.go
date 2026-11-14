package main

import (
	"testing"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/audio"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/world"
)

type game struct {
	w     *world.World
	in    *input.State
	hero  *Hero
	room  *audio.Reverb
	door  *Door
	rec   *audio.Recorder
	hud   *HUD
	mixer *audio.Mixer
}

func newGame(t *testing.T) *game {
	s, err := load()
	if err != nil {
		t.Fatal(err)
	}
	win := platform.NewWindow(platform.Size(640, 360))
	room := audio.NewReverb(audio.RoomSize(corridor().size), audio.Wet(corridor().wet))
	mixer := audio.NewMixer(audio.WithBus("sfx", 1, audio.Effects(room)))
	hero, err := newHero()
	if err != nil {
		t.Fatal(err)
	}
	door := newDoor(s.drums)
	hud := newHUD()
	win.Add(twod.NewLayer(0).Add(newDungeon(), door, door.band, hero), hud.layer)
	w := ember.New().AddEntities(win, mixer).AddSystems(
		&Walk{hero: hero, in: win.Input(), mixer: mixer, sounds: s},
		&Acoustics{hero: hero, room: room},
		&Doorway{door: door, hero: hero, in: win.Input()},
		&Status{hud: hud, hero: hero, room: room, door: door},
	)
	return &game{w, win.Input(), hero, room, door, audio.NewRecorder(mixer), hud, mixer}
}

func (g *game) press(k input.Key, frames int) {
	g.in.SetKey(k, true)
	g.w.RunFrames(frames)
	g.in.SetKey(k, false)
	g.w.RunFrames(1)
}

// tail is how loud a clap still rings from 0.3 to 0.8 s after it.
func (g *game) tail() float32 {
	g.rec.Reset()
	g.press(input.KeySpace, 1)
	g.w.RunFrames(50)
	return g.rec.Peak(48000*3/10, 48000*8/10)
}

// highs is the largest step between two samples: high notes such as the
// snare and hats step far, while a low-passed kick and bass barely move.
func highs(rec *audio.Recorder) float32 {
	s := rec.Samples()
	var step float32
	for i := 2; i < len(s); i += 2 {
		step = max(step, s[i]-s[i-2], s[i-2]-s[i])
	}
	return step
}

func TestTheHallRingsLongerThanTheCorridor(t *testing.T) {
	g := newGame(t)
	g.door.band.SetVolume(0)
	g.w.RunFrames(1)
	if g.room.RoomSize() != corridor().size || g.room.Wet() != corridor().wet {
		t.Fatalf("the corridor's reverb is small: %v %v", g.room.RoomSize(), g.room.Wet())
	}
	small := g.tail()

	g.press(input.KeyD, 200)
	if x := g.hero.Position().X; x < 300 || g.room.RoomSize() != hall().size || g.room.Wet() != hall().wet {
		t.Fatalf("walking into the hall at x %v grows the reverb: %v %v", x, g.room.RoomSize(), g.room.Wet())
	}
	g.w.RunFrames(30)
	large := g.tail()
	if small <= 0 || large < 3*small {
		t.Fatalf("a clap rings on far longer in the hall (%v) than in the corridor (%v)", large, small)
	}
	if g.mixer.Bus("sfx").Effects()[0] != audio.Effect(g.room) {
		t.Fatal("the reverb is the sfx bus's")
	}
	if g.rec.Peak(0, g.rec.Frames()) >= 1 {
		t.Fatal("nothing clips")
	}
}

func TestOpeningTheDoorUnmufflesTheDrums(t *testing.T) {
	g := newGame(t)
	g.hero.SetPosition(geom.V2(500, 120))
	g.w.RunFrames(60)
	g.rec.Reset()
	g.w.RunFrames(30)
	shut := highs(g.rec)
	if g.door.filter.Cutoff() != muffled || g.door.opened {
		t.Fatal("the door starts shut, the drums muffled")
	}

	g.press(input.KeyE, 1)
	g.w.RunFrames(40)
	if !g.door.opened || !g.door.Hidden() || g.door.filter.Cutoff() < open*0.99 {
		t.Fatalf("E opens the door and sweeps the filter open: %v Hz", g.door.filter.Cutoff())
	}
	g.rec.Reset()
	g.w.RunFrames(30)
	if opened := highs(g.rec); opened < 3*shut {
		t.Fatalf("the open door lets the drums' snare and hats through: %v against %v", opened, shut)
	}
	if g.hud.lines[2].Text() == "" {
		t.Fatal("the HUD shows the filter")
	}

	g.press(input.KeyE, 1)
	g.w.RunFrames(40)
	if g.door.opened || g.door.filter.Cutoff() > muffled*1.01 {
		t.Fatal("E shuts it again")
	}
	g.hero.SetPosition(geom.V2(40, 196))
	g.w.RunFrames(1)
	g.press(input.KeyE, 1)
	if g.door.opened {
		t.Fatal("the door only opens from near it")
	}
}

func TestFootstepsPlayWhileWalking(t *testing.T) {
	g := newGame(t)
	var played world.Reader[audio.VoiceFinished]
	steps := 0
	g.in.SetKey(input.KeyD, true)
	for range 70 {
		g.w.RunFrames(1)
		for range played.Read(g.w) {
			steps++
		}
	}
	g.in.SetKey(input.KeyD, false)
	g.press(input.KeyW, 1)
	if steps < 2 {
		t.Fatalf("a second of walking has three footsteps: %d finished", steps)
	}
	if !onFloor(g.hero.Position()) || onFloor(geom.V2(100, 100)) {
		t.Fatal("the hero stays on the floor")
	}
}
