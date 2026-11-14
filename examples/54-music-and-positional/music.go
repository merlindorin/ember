package main

import (
	"fmt"
	"time"

	"github.com/lantervale/ember/audio"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/world"
)

const crossfade = 2 * time.Second

// The loops are normalised when cut (CREDITS.md), so these only balance
// the two ambiences, under the campfire and the river.
const (
	dayVolume   = 0.85
	nightVolume = 1
)

// Music crossfades between the day and night ambiences when M is pressed,
// and the meadow's colour with them.
type Music struct {
	window *platform.Window
	mixer  *audio.Mixer
	sounds *sounds
	hud    *HUD
	night  bool
	fading float32
}

func (m *Music) Update(f world.Frame) {
	if m.window.Input().Pressed(input.KeyM) {
		m.night = !m.night
		track, volume := m.sounds.day, float32(dayVolume)
		if m.night {
			track, volume = m.sounds.night, nightVolume
		}
		m.mixer.PlayMusic(track, audio.Loop(), audio.Crossfade(crossfade), audio.Volume(volume))
		m.fading = 0
	}
	m.fading = min(m.fading+f.DT/float32(crossfade.Seconds()), 1)
	from, to := nightColor(), dayColor()
	if m.night {
		from, to = dayColor(), nightColor()
	}
	m.window.SetBackground(from.Lerp(to, m.fading))
	m.hud.night = m.night
}

// HUD shows what the hero hears: each source's gain and pan.
type HUD struct {
	hero    *Hero
	layer   *twod.Layer
	sources []*Source
	lines   []*twod.Text
	night   bool
}

func newHUD(hero *Hero, sources ...*Source) *HUD {
	h := &HUD{hero: hero, layer: twod.NewLayer(1, twod.Fixed()), sources: sources}
	for i := range len(sources) + 1 {
		t := twod.NewText("", twod.At(16, 12+20*float32(i)), twod.Pivot(0, 0), twod.FontSize(15))
		h.lines = append(h.lines, t)
		h.layer.Add(t)
	}
	return h
}

func (h *HUD) Update(world.Frame) {
	music := "day"
	if h.night {
		music = "night"
	}
	h.lines[0].SetText(
		fmt.Sprintf("WASD to walk · Q/E to turn (facing %3.0f°) · M for day or night (%s)", h.hero.Degrees(), music),
	)
	for i, s := range h.sources {
		gain, pan := s.Heard()
		h.lines[i+1].SetText(fmt.Sprintf("%s  gain %.2f  pan %+.2f", s.GetSceneNode().Name(), gain, pan))
	}
}
