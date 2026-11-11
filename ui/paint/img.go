package paint

import (
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/ui/internal/style"
)

// img draws an img element's picture, its src found through Images, in
// its content box by object-fit and object-position, cut to the box where
// it overflows; a src Images doesn't find draws as a broken image. The box
// and the picture snap to device pixels as Chrome's do, so a pixelated
// picture's texels fall on whole pixels as a background's do.
func (p *painter) img(e entry, alpha float32) {
	src, hasSrc := e.b.Node.Attr("src")
	st := shown(e.b)
	tex := p.find(style.Image{Source: src}, st)
	if tex == nil {
		if hasSrc && p.image != nil {
			p.brokenImage(e, alpha)
		}
		return
	}
	size := tex.Size()
	if size.X <= 0 || size.Y <= 0 {
		return
	}
	box := p.snap(content(e))
	at, overflows := objectFit(st.ObjectFit, box, size, st.ObjectPosition)
	at = p.snap(at)
	if overflows {
		p.s.PushClip(box)
		defer p.s.PopClip()
	}
	p.s.Image(at, tex, whole, fade(color.White, alpha))
}

// objectFit is where a picture of natural size goes in box by fit, placed
// by position as a background image is, and whether it overflows box.
func objectFit(
	fit style.ObjectFit, box geom.Rect, natural geom.Vec2, position style.BackgroundPosition,
) (geom.Rect, bool) {
	size := natural
	switch fit {
	case style.FitFill:
		return box, false
	case style.FitContain:
		size = natural.Scale(min(box.W/natural.X, box.H/natural.Y))
	case style.FitCover:
		size = natural.Scale(max(box.W/natural.X, box.H/natural.Y))
	case style.FitScaleDown:
		size = natural.Scale(min(1, box.W/natural.X, box.H/natural.Y))
	case style.FitNone:
	}
	r := geom.Rect{
		X: place(position.X, position.FromRight, box.X, box.W, size.X),
		Y: place(position.Y, position.FromBottom, box.Y, box.H, size.Y),
		W: size.X, H: size.Y,
	}
	return r, size.X > box.W || size.Y > box.H
}
