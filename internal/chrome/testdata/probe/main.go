//go:build js

// Command probe runs a World in a page and publishes what its window's
// input saw each frame as window.emberProbe, for TestThePagesInputReachesTheWorld.
package main

import (
	"encoding/json"
	"os"
	"syscall/js"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/world"
)

type report struct {
	Down   []int      `json:"down"`
	Mouse  [2]float32 `json:"mouse"`
	Left   bool       `json:"left"`
	Right  bool       `json:"right"`
	Clicks int        `json:"clicks"`
	Wheel  float32    `json:"wheel"`
	Text   string     `json:"text"`
	Frame  int        `json:"frame"`
	Size   [2]float32 `json:"size"`
	Full   bool       `json:"full"`
}

type probe struct {
	win *platform.Window
	r   report
}

func (p *probe) Update(f world.Frame) {
	in := p.win.Input()
	p.r.Down = p.r.Down[:0]
	for _, k := range []input.Key{input.KeyW, input.KeyA, input.KeySpace, input.KeyLeftShift, input.KeyLeft} {
		if in.Down(k) {
			p.r.Down = append(p.r.Down, int(k))
		}
	}
	m := in.Mouse()
	p.r.Mouse = [2]float32{m.X, m.Y}
	p.r.Left, p.r.Right = in.Down(input.MouseLeft), in.Down(input.MouseRight)
	if in.Pressed(input.MouseLeft) {
		p.r.Clicks++
	}
	p.r.Wheel += in.Wheel().Y
	p.r.Text += string(in.Text())
	p.r.Frame++
	if os.Getenv("EMBER_PROBE_QUIT") != "" && p.r.Frame == 30 {
		f.Quit()
	}
	if in.Pressed(input.KeyF) {
		p.win.SetFullscreen(!p.win.Fullscreen())
	}
	size := p.win.Size()
	p.r.Size, p.r.Full = [2]float32{size.X, size.Y}, p.win.Fullscreen()
	data, _ := json.Marshal(p.r)
	js.Global().Set("emberProbe", string(data))
}

func main() {
	opts := []platform.Option{platform.Size(320, 200), platform.Title("probe"), platform.Background(color.Hex("#3366cc"))}
	if os.Getenv("EMBER_PROBE_FLUID") == "" {
		opts = append(opts, platform.FixedSize())
	}
	win := platform.NewWindow(opts...)
	ember.New().AddEntities(win).AddSystems(&probe{win: win}).MustRun()
}
