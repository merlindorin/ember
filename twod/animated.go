package twod

import (
	"encoding/json"
	"fmt"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/node"
)

//nolint:gochecknoinits // registers the package's resource types on import
func init() {
	node.RegisterResource("twod.Animations", func() node.Resource { return NewAnimations() })
}

// Frame is one picture of an animation: a texture, or a region of it in
// pixels.
type Frame struct {
	Texture *Texture
	Region  geom.Rect
}

// Strip cuts tex into frames of cell, left to right then top to bottom:
// a sprite sheet.
func Strip(tex *Texture, cell Grid) []Frame {
	size := tex.Size()
	w, h := float32(max(cell.w, 1)), float32(max(cell.h, 1))
	columns, rows := max(int(size.X/w), 1), max(int(size.Y/h), 1)
	frames := make([]Frame, 0, columns*rows)
	for y := range rows {
		for x := range columns {
			frames = append(
				frames,
				Frame{Texture: tex, Region: geom.Rect{X: float32(x) * w, Y: float32(y) * h, W: w, H: h}},
			)
		}
	}
	return frames
}

// repeat is what a clip does at its end.
type repeat int

const (
	once repeat = iota
	loop
	pingPong
)

//nolint:gochecknoglobals // word table
var repeatWords = node.Words[repeat]{"once", "loop", "ping_pong"}

// ClipOption sets up a clip of Animations.
type ClipOption func(*clip)

// Once plays a clip to its end and stops there, the default.
func Once() ClipOption { return func(c *clip) { c.repeat = once } }

// Loop starts a clip over each time it ends.
func Loop() ClipOption { return func(c *clip) { c.repeat = loop } }

// PingPong plays a clip forth and back, again and again.
func PingPong() ClipOption { return func(c *clip) { c.repeat = pingPong } }

type clip struct {
	name   string
	fps    float32
	repeat repeat
	frames []Frame
}

// Animations are named clips of frames, which animated sprites play. They
// are plain data, shared by every sprite that plays them.
type Animations struct{ clips []*clip }

func NewAnimations() *Animations { return &Animations{} }

// Add adds a clip of frames played at fps frames per second, once unless
// opts say Loop or PingPong.
func (a *Animations) Add(name string, fps float32, frames []Frame, opts ...ClipOption) *Animations {
	c := &clip{name: name, fps: fps, frames: frames}
	for _, o := range opts {
		o(c)
	}
	a.clips = append(a.clips, c)
	return a
}

// Names lists the clips in the order they were added.
func (a *Animations) Names() []string {
	if a == nil {
		return nil
	}
	var out []string
	for _, c := range a.clips {
		out = append(out, c.name)
	}
	return out
}

func (a *Animations) clip(name string) *clip {
	if a == nil {
		return nil
	}
	for _, c := range a.clips {
		if c.name == name {
			return c
		}
	}
	return nil
}

func (*Animations) ResourceType() string { return "twod.Animations" }

type animationsJSON struct {
	Textures []*Texture `json:"textures"`
	Clips    []struct {
		Name   string  `json:"name"`
		FPS    float32 `json:"fps"`
		Repeat string  `json:"repeat,omitempty"`
		Loop   bool    `json:"loop,omitempty"`
		Frames []struct {
			Texture int       `json:"texture"`
			Region  geom.Rect `json:"region"`
		} `json:"frames"`
	} `json:"clips"`
}

// MarshalJSON stores each texture once, frames referring to it by index.
func (a *Animations) MarshalJSON() ([]byte, error) {
	var d animationsJSON
	index := map[*Texture]int{}
	for _, c := range a.clips {
		word, err := repeatWords.Text(c.repeat)
		if err != nil {
			return nil, err
		}
		out := struct {
			Name   string  `json:"name"`
			FPS    float32 `json:"fps"`
			Repeat string  `json:"repeat,omitempty"`
			Loop   bool    `json:"loop,omitempty"`
			Frames []struct {
				Texture int       `json:"texture"`
				Region  geom.Rect `json:"region"`
			} `json:"frames"`
		}{Name: c.name, FPS: c.fps, Repeat: string(word)}
		for _, f := range c.frames {
			i, ok := index[f.Texture]
			if !ok {
				i = len(d.Textures)
				index[f.Texture] = i
				d.Textures = append(d.Textures, f.Texture)
			}
			out.Frames = append(out.Frames, struct {
				Texture int       `json:"texture"`
				Region  geom.Rect `json:"region"`
			}{i, f.Region})
		}
		d.Clips = append(d.Clips, out)
	}
	return json.Marshal(d)
}

func (a *Animations) UnmarshalJSON(b []byte) error {
	var d animationsJSON
	if err := json.Unmarshal(b, &d); err != nil {
		return err
	}
	a.clips = nil
	for _, c := range d.Clips {
		var frames []Frame
		for _, f := range c.Frames {
			if f.Texture < 0 || f.Texture >= len(d.Textures) {
				return fmt.Errorf("twod: animation %s refers to texture %d of %d", c.Name, f.Texture, len(d.Textures))
			}
			frames = append(frames, Frame{Texture: d.Textures[f.Texture], Region: f.Region})
		}
		r := once
		switch {
		case c.Repeat != "":
			if err := repeatWords.Parse(&r, []byte(c.Repeat)); err != nil {
				return err
			}
		case c.Loop:
			r = loop
		}
		a.Add(c.Name, c.FPS, frames, func(c *clip) { c.repeat = r })
	}
	return nil
}

// AnimationFinished is emitted when a clip played Once reaches its end.
type AnimationFinished struct {
	Sprite *Sprite
	Name   string
}

// AnimationLooped is emitted each time a looping clip starts over.
type AnimationLooped struct {
	Sprite *Sprite
	Name   string
}
