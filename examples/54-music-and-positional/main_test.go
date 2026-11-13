package main

import (
	"strings"
	"testing"
	"time"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/audio"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/world"
)

// meadow is the example's scene in a World, the mixer recorded.
type meadow struct {
	w           *world.World
	win         *platform.Window
	mixer       *audio.Mixer
	rec         *audio.Recorder
	day         *audio.Voice
	hero        *Hero
	fire, river *Source
	hud         *HUD
}

func newMeadow(t *testing.T) *meadow {
	t.Helper()
	s, err := load()
	if err != nil {
		t.Fatal(err)
	}
	win := platform.NewWindow(platform.Size(640, 360), platform.Background(dayColor()))
	mixer := audio.NewMixer(audio.WithBus("music", 0.5), audio.WithBus("sfx", 1))
	rec := audio.NewRecorder(mixer)
	day := mixer.PlayMusic(s.day, audio.Loop(), audio.FadeIn(time.Second), audio.Volume(dayVolume))
	hero, err := newHero()
	if err != nil {
		t.Fatal(err)
	}
	fire, river := newCampfire(s.crackle), newRiver(s.river)
	hud := newHUD(hero, fire, river)
	win.Add(twod.NewLayer(0).Add(river, fire, hero), hud.layer)
	music := &Music{window: win, mixer: mixer, sounds: s, hud: hud, fading: 1}
	w := ember.New().AddEntities(win, mixer).AddSystems(&Walk{hero: hero, in: win.Input()}, music, hud)
	return &meadow{w: w, win: win, mixer: mixer, rec: rec, day: day, hero: hero, fire: fire, river: river, hud: hud}
}

// hold presses k for frames, then lets it go for a frame.
func (m *meadow) hold(k input.Key, frames int) {
	in := m.win.Input()
	in.SetKey(k, true)
	m.w.RunFrames(frames)
	in.SetKey(k, false)
	m.w.RunFrames(1)
}

func TestTheHeroHearsTheMeadowFromWhereHeStandsAndFaces(t *testing.T) {
	m := newMeadow(t)
	m.w.RunFrames(1)
	if pan(m.fire) > -0.99 || pan(m.river) < 0.95 || !m.fire.Playing() || !m.river.Playing() {
		t.Fatalf("facing up, the fire is heard on the left (%v), the river on the right (%v)", pan(m.fire), pan(m.river))
	}

	m.hold(input.KeyE, 60)
	if d := m.hero.Degrees(); d < 175 || d > 185 {
		t.Fatalf("a second of E is a half turn: %v°", d)
	}
	if pan(m.fire) < 0.95 || pan(m.river) > -0.95 {
		t.Fatalf("after a half turn the fire is on the right (%v), the river on the left (%v)", pan(m.fire), pan(m.river))
	}
	if r := m.hero.body.Rotation(); r != -m.hero.Rotation() {
		t.Fatal("the sprite stays upright as the hero turns")
	}
	m.hold(input.KeyQ, 30)
	if d := m.hero.Degrees(); d < 85 || d > 95 || pan(m.fire) < -0.1 || pan(m.fire) > 0.1 {
		t.Fatalf(
			"half a second of Q turns him back a quarter, facing the river, fire behind: %v°, pan %v",
			d,
			pan(m.fire),
		)
	}

	before := gain(m.fire)
	m.hold(input.KeyA, 60)
	if gain(m.fire) <= before || m.hero.Position().X > 200 {
		t.Fatalf("walking to the fire makes it louder: %v then %v", before, gain(m.fire))
	}
	if d := m.hero.Degrees(); d < 265 || d > 275 || pan(m.fire) < -0.2 || pan(m.fire) > 0.2 {
		t.Fatalf("he turns to face where he walks, the fire ahead: %v°, pan %v", d, pan(m.fire))
	}

	m.hold(input.KeyD, 150)
	if gain(m.river) <= gain(m.fire) {
		t.Fatalf("by the river it is the louder one: fire %v, river %v", gain(m.fire), gain(m.river))
	}
	m.hold(input.KeyQ, 30)
	from := m.rec.Frames()
	m.w.RunFrames(30)
	if l, r := m.rec.PeakLR(from, m.rec.Frames()); r <= l || pan(m.river) < 0.5 {
		t.Fatalf("facing up by the river, the right ear hears more: %v %v, pan %v", l, r, pan(m.river))
	}

	for _, c := range []struct {
		name      string
		at        geom.Vec2
		near, far *Source
	}{{"river", geom.V2(560, 180), m.river, m.fire}, {"fire", geom.V2(110, 220), m.fire, m.river}} {
		m.hero.SetPosition(c.at)
		m.w.RunFrames(1)
		if gain(c.far) > gain(c.near)/4 {
			t.Fatalf("by the %s, it drowns the other out: %v against %v", c.name, gain(c.near), gain(c.far))
		}
	}

	if p := m.rec.Peak(0, m.rec.Frames()); p >= 1 {
		t.Fatalf("nothing clips at the example's levels: peak %v", p)
	}
}

func TestMCrossfadesTheMusicToNight(t *testing.T) {
	m := newMeadow(t)
	m.w.RunFrames(1)
	var finished world.Reader[audio.VoiceFinished]
	m.hold(input.KeyM, 1)
	night := m.mixer.Music()
	if night == m.day || !m.day.Playing() || night.Bus() != "music" || night.Volume() != nightVolume {
		t.Fatal("M starts the night ambience while the day one fades")
	}
	stopped := false
	for range 140 {
		m.w.RunFrames(1)
		for ev := range finished.Read(m.w) {
			stopped = stopped || ev.Voice == m.day && ev.Stopped
		}
	}
	if !stopped || m.day.Playing() || !night.Playing() || m.win.Background().Hex() != nightColor().Hex() {
		t.Fatalf("after the crossfade only the night plays, under a night sky: day stopped %v", stopped)
	}
	if m.hud.lines[1].Text() == "" || !strings.Contains(m.hud.lines[0].Text(), "facing") {
		t.Fatal("the HUD shows where the hero faces and what he hears")
	}
	if p := m.rec.Peak(0, m.rec.Frames()); p >= 1 {
		t.Fatalf("nothing clips at the example's levels: peak %v", p)
	}
}

type hearer interface{ Heard() (gain, pan float32) }

func gain(e hearer) float32 { g, _ := e.Heard(); return g }
func pan(e hearer) float32  { _, p := e.Heard(); return p }
