package sound

import (
	"time"

	"github.com/merlindorin/ember/audio"
)

// Player is a game's mixer: its music on the "music" bus, its effects on
// "sfx". Add Mixer to the World.
type Player struct {
	Mixer *audio.Mixer
	music *audio.Clip
	on    bool
}

// NewPlayer starts music looping, at the volumes given.
func NewPlayer(music []float32, musicVolume, effects float32, musicOn bool) *Player {
	p := &Player{Mixer: audio.NewMixer(audio.WithBus("music", musicVolume), audio.WithBus("sfx", effects)),
		music: Clip(music)}
	p.Set(musicVolume, effects, musicOn)
	return p
}

// Play plays an effect.
func (p *Player) Play(c *audio.Clip, opts ...audio.Option) {
	p.Mixer.Play(c, append(opts, audio.OnBus("sfx"))...)
}

// Set changes the volumes and starts or stops the music.
func (p *Player) Set(music, effects float32, musicOn bool) {
	p.Mixer.Bus("music").SetVolume(music)
	p.Mixer.Bus("sfx").SetVolume(effects)
	switch {
	case musicOn && !p.on:
		p.Mixer.PlayMusic(p.music, audio.Loop(), audio.FadeIn(time.Second))
	case !musicOn && p.on:
		p.Mixer.StopMusic(audio.FadeOut(300 * time.Millisecond))
	}
	p.on = musicOn
}
