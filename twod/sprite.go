package twod

import (
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	lt "github.com/merlindorin/ember/internal/lit"
	"github.com/merlindorin/ember/node"
)

//nolint:gochecknoinits // registers the package's node types on import
func init() {
	node.Register(node.Type{New: func() node.Noded { return NewSprite(nil) }})
}

// Sprite draws a picture centred on its position: a texture, a region of
// it, a still frame of a sheet cut by Strip, or the frames of the
// animation it plays.
type Sprite struct {
	Item
	texture    *Texture    `node:"prop"`
	region     geom.Rect   `node:"prop"`
	frame      int         `node:"prop"`
	pivot      geom.Vec2   `node:"prop"`
	size       geom.Vec2   `node:"prop"`
	flipX      bool        `node:"prop"`
	flipY      bool        `node:"prop"`
	animations *Animations `node:"prop"`
	current    string      `node:"prop"`
	speed      float32     `node:"prop"`
	playing    bool        `node:"prop"`
	progress   float32
	backward   bool
	// players is its System's list of playing sprites, which it joins as
	// it plays; listed while it is on it.
	players *players
	listed  bool
	// turn counts its times in a World, so an entry from an earlier one
	// is passed over.
	turn uint32
}

// players are the sprites a System steps each frame: those playing a
// clip, in the order they started. Paused, stopped, finished and removed
// ones leave as the System next steps them.
type players struct{ list []player }

type player struct {
	sprite *Sprite
	turn   uint32
}

// join lists s while it plays.
func (s *Sprite) join() {
	if s.players != nil && s.playing && !s.listed {
		s.players.list = append(s.players.list, player{s, s.turn})
		s.listed = true
	}
}

// NewSprite returns a sprite of tex set up by opts. An animated sprite may
// have no texture of its own: its frames carry theirs.
func NewSprite(tex *Texture, opts ...SpriteOption) *Sprite {
	s := Build(&Sprite{Item: Base(), texture: tex, pivot: geom.V2(0.5, 0.5), speed: 1}, "sprite")
	for _, o := range opts {
		o.sprite(s)
	}
	return s
}

// Region shows only r of the texture, in pixels.
func Region(r geom.Rect) SpriteOption { return spriteOption(func(s *Sprite) { s.region = r }) }

// Still shows one frame, such as one of those Strip cut from a sheet: its
// texture and region.
func Still(f Frame) SpriteOption { return spriteOption(func(s *Sprite) { s.SetStill(f) }) }

// Flip mirrors the picture horizontally, vertically or both.
func Flip(x, y bool) SpriteOption { return spriteOption(func(s *Sprite) { s.flipX, s.flipY = x, y }) }

// Animate gives the sprite clips to play; it shows the first until Play.
func Animate(anims *Animations) SpriteOption {
	return spriteOption(func(s *Sprite) {
		s.animations = anims
		if names := anims.Names(); len(names) > 0 && s.current == "" {
			s.current = names[0]
		}
	})
}

// Playing starts the sprite playing the clip name.
func Playing(name string) SpriteOption { return spriteOption(func(s *Sprite) { s.Play(name) }) }

// Speed plays clips s times faster (default 1).
func Speed(speed float32) SpriteOption {
	return spriteOption(func(s *Sprite) {
		if speed > 0 {
			s.speed = speed
		}
	})
}

func (s *Sprite) Texture() *Texture       { return s.texture }
func (s *Sprite) SetTexture(t *Texture)   { s.texture = t; s.refile() }
func (s *Sprite) Region() geom.Rect       { return s.region }
func (s *Sprite) SetRegion(r geom.Rect)   { s.region = r; s.refile() }
func (s *Sprite) SetFlip(x, y bool)       { s.flipX, s.flipY = x, y }
func (s *Sprite) Animations() *Animations { return s.animations }

// SetStill shows frame f: its texture and region.
func (s *Sprite) SetStill(f Frame) { s.texture, s.region = f.Texture, f.Region; s.refile() }

// Frame is the index of the frame shown in the clip.
func (s *Sprite) Frame() int { return s.frame }

// SetFrame shows frame i of the clip.
func (s *Sprite) SetFrame(i int) { s.frame = i; s.refile() }

// Play plays the clip name from its first frame, or carries on when it is
// the one playing.
func (s *Sprite) Play(name string) {
	defer s.refile()
	if name != s.current {
		s.current, s.frame, s.progress, s.backward = name, 0, 0, false
	}
	s.playing = true
	s.join()
}

func (s *Sprite) Pause() { s.playing = false }

// Stop stops and goes back to the clip's first frame.
func (s *Sprite) Stop() {
	s.playing, s.frame, s.progress, s.backward = false, 0, 0, false
	s.refile()
}

// Playing is the clip shown, and whether it plays.
func (s *Sprite) Playing() (string, bool) { return s.current, s.playing }

// step advances the clip by dt seconds and returns the events it raised.
func (s *Sprite) step(dt float32) []any {
	var events []any
	if !s.playing {
		return nil
	}
	c := s.animations.clip(s.current)
	if c == nil || len(c.frames) == 0 || c.fps <= 0 {
		return nil
	}
	s.progress += dt * c.fps * s.speed
	for s.progress >= 1 {
		s.progress--
		switch {
		case s.backward && s.frame > 0:
			s.frame--
			if s.frame == 0 {
				s.backward = false
				events = append(events, AnimationLooped{Sprite: s, Name: c.name})
			}
		case !s.backward && s.frame+1 < len(c.frames):
			s.frame++
		case c.repeat == once:
			s.playing, s.progress = false, 0
			return append(events, AnimationFinished{Sprite: s, Name: c.name})
		case c.repeat == pingPong && len(c.frames) > 1:
			s.backward = true
			s.frame--
		default:
			s.frame = 0
			events = append(events, AnimationLooped{Sprite: s, Name: c.name})
		}
	}
	return events
}

// Picture is the texture and the part of it, in pixels, the sprite shows
// now: its animation's current frame, else its still picture.
func (s *Sprite) Picture() (*Texture, geom.Rect) {
	if c := s.animations.clip(s.current); c != nil && len(c.frames) > 0 {
		f := c.frames[min(max(s.frame, 0), len(c.frames)-1)]
		src := f.Region
		if src.W == 0 || src.H == 0 {
			size := f.Texture.Size()
			src = geom.Rect{W: size.X, H: size.Y}
		}
		return f.Texture, src
	}
	src := s.region
	if src.W == 0 || src.H == 0 {
		size := s.texture.Size()
		src = geom.Rect{W: size.X, H: size.Y}
	}
	return s.texture, src
}

// Size is the box the sprite fills, in points before Scale: its Size, else
// its picture's size in pixels.
func (s *Sprite) Size() geom.Vec2 {
	if s.size != (geom.Vec2{}) {
		return s.size
	}
	_, src := s.Picture()
	return geom.V2(src.W, src.H)
}

// SetSize fits the sprite's picture to size, in points; zero shows it at
// its own size.
func (s *Sprite) SetSize(size geom.Vec2) { s.size = size; s.refile() }

// Rect is the sprite's rectangle in its own space.
func (s *Sprite) Rect() geom.Rect {
	size := s.Size()
	return geom.Rect{X: -size.X * s.pivot.X, Y: -size.Y * s.pivot.Y, W: size.X, H: size.Y}
}

func (s *Sprite) Draw(c *Canvas) {
	tex, src := s.Picture()
	if tex.Size() == (geom.Vec2{}) {
		return
	}
	uv := tex.uv(src)
	if s.flipX {
		uv.X, uv.W = uv.X+uv.W, -uv.W
	}
	if s.flipY {
		uv.Y, uv.H = uv.Y+uv.H, -uv.H
	}
	if st := s.GetDrawLook().light; st.Occluder && st.Mode == lt.Inherit {
		k := max(c.Transform().X.Len(), 1e-6)
		r := s.Rect()
		saved := c.list.Light()
		c.list.SetLightFacing(
			geom.Rect{X: r.X - baseFront/k, Y: r.Y - baseFront/k, W: r.W + 2*baseFront/k, H: r.H + 2*baseFront/k},
		)
		c.Image(r, tex, uv, color.White)
		c.list.RestoreLight(saved)
		return
	}
	c.Image(s.Rect(), tex, uv, color.White)
}
