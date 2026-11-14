package main

import (
	"fmt"
	"math"
	"math/rand/v2"

	"github.com/lantervale/ember/audio"
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/world"
)

const ringLife = 0.4

// A ring grows and fades where a sound was played.
type ring struct {
	line *twod.Line
	age  float32
}

// Player plays the sound on Space at a random pitch, and on a click panned
// to where the click is, from -1 at the left edge to 1 at the right. It
// counts the voices playing and the sounds that finished.
type Player struct {
	window *platform.Window
	mixer  *audio.Mixer
	hit    *audio.Clip
	layer  *twod.Layer
	status *twod.Text
	rings  []*ring
	next   int

	finished world.Reader[audio.VoiceFinished]
	played   int
}

func newPlayer(win *platform.Window, mixer *audio.Mixer, hit *audio.Clip) *Player {
	p := &Player{window: win, mixer: mixer, hit: hit, layer: twod.NewLayer(0)}
	circle := make([]geom.Vec2, 32)
	for i := range circle {
		a := float64(i) / float64(len(circle)) * 2 * math.Pi
		circle[i] = geom.V2(float32(math.Cos(a)), float32(math.Sin(a)))
	}
	for range 8 {
		l := twod.NewLine(circle, twod.Closed(), twod.Width(0.08), twod.Color(color.Hex("#ffd27a")))
		l.SetHidden(true)
		p.rings = append(p.rings, &ring{line: l, age: ringLife})
		p.layer.Add(l)
	}
	p.status = twod.NewText("", twod.At(16, 12), twod.Pivot(0, 0), twod.FontSize(15))
	p.layer.Add(p.status)
	return p
}

func (p *Player) Update(f world.Frame) {
	in := p.window.Input()
	size := p.window.Size()
	if in.Pressed(input.KeySpace) {
		p.mixer.Play(p.hit, audio.Pitch(0.85+0.3*rand.Float32())) //nolint:gosec // a game's randomness, not a secret
		p.ring(size.Scale(0.5))
	}
	if in.Pressed(input.MouseLeft) {
		at := in.Mouse()
		p.mixer.Play(p.hit, audio.Pan(at.X/size.X*2-1))
		p.ring(at)
	}
	for range p.finished.Read(f.World) {
		p.played++
	}
	for _, r := range p.rings {
		r.age += f.DT
		t := min(r.age/ringLife, 1)
		r.line.SetHidden(t >= 1)
		r.line.SetScale(geom.V2(20+60*t, 20+60*t))
		r.line.SetTint(color.White.Alpha(1 - t))
	}
	p.status.SetText(fmt.Sprintf("Space or click to play · playing %d · finished %d", p.mixer.Voices(), p.played))
}

func (p *Player) ring(at geom.Vec2) {
	r := p.rings[p.next]
	p.next = (p.next + 1) % len(p.rings)
	r.age = 0
	r.line.SetPosition(at)
}
