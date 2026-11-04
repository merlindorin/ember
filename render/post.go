package render

import "github.com/merlindorin/ember/geom"

// Post describes how a camera's rendered scene becomes the final image: at
// which resolution the scene renders, and which effects run on it after
// tonemapping. It describes, the presenter executes, so styles (pixel art,
// CRT, cel shading, colour grading) live in game code as data.
//
//	post := render.NewPost()
//	eye.Camera.Post = post
//	post.Resolution = func(fb geom.Vec2) (int, int) { return int(fb.X) / 4, int(fb.Y) / 4 }
//	post.MSAA = 1
//	post.Add(render.NewEffect(sepiaWGSL, render.Name("sepia")))
func NewPost() *Post { return &Post{} }

type Post struct {
	// Resolution picks the scene size for a framebuffer. Nil keeps the
	// framebuffer size scaled by the presenter's render scale.
	Resolution func(framebuffer geom.Vec2) (w, h int)
	// MSAA overrides the presenter's sample count when positive; 1 turns
	// anti-aliasing off.
	MSAA int
	// Effects run in order on the tonemapped image.
	Effects []*Effect
	// Upscale filters the final image onto the window when the scene is
	// smaller than it. A Screen effect at the end of the chain replaces it.
	Upscale Filter
}

type Filter int

const (
	FilterLinear Filter = iota
	FilterNearest
)

// Add appends an effect and returns it.
func (p *Post) Add(e *Effect) *Effect {
	p.Effects = append(p.Effects, e)
	return e
}

// Active returns the effects that run this frame: enabled ones, up to and
// including the first Screen effect.
func (p *Post) Active() []*Effect {
	var out []*Effect
	for _, e := range p.Effects {
		if e == nil || !e.Enabled() {
			continue
		}
		out = append(out, e)
		if e.Screen() {
			break
		}
	}
	return out
}

// SceneSize returns the resolution the scene renders at for a framebuffer.
func (p *Post) SceneSize(framebuffer geom.Vec2, scale float32) (int, int) {
	var w, h int
	if p != nil && p.Resolution != nil {
		w, h = p.Resolution(framebuffer)
	} else {
		if scale <= 0 {
			scale = 1
		}
		w, h = int(framebuffer.X*scale), int(framebuffer.Y*scale)
	}
	return max(w, 1), max(h, 1)
}

// Plain reports whether the presenter can skip the post chain and
// tonemap straight onto the window.
func (p *Post) Plain() bool {
	return p == nil || (len(p.Active()) == 0 && p.Upscale == FilterLinear)
}
