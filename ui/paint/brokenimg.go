package paint

import (
	"strings"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/twod"
)

// BrokenIcon is the side of the icon an img that didn't load shows, as
// Chrome's, in points.
const BrokenIcon = 16

// brokenFrame is the silver border Chrome draws round an img that didn't
// load and has no alt text, and brokenInset how far in its icon stands.
//
//nolint:gochecknoglobals // fixed values every paint reads; Go has no constant colours, tables or shapes
var brokenFrame = color.FromSRGB([3]float32{0xc0 / 255.0, 0xc0 / 255.0, 0xc0 / 255.0})

const brokenInset = 2

// brokenImage draws an img whose picture didn't load, as Chrome does:
// with alt text, the icon at its top-left and the text after it on one
// line; without, a silver frame with the icon inside it when the box
// holds both, else the icon alone; with an empty alt, nothing. All of it
// is cut to the content box.
func (p *painter) brokenImage(e entry, alpha float32) {
	alt, hasAlt := e.b.Node.Attr("alt")
	if hasAlt && alt == "" {
		return
	}
	box := content(e)
	if box.W <= 0 || box.H <= 0 {
		return
	}
	p.s.PushClip(box)
	defer p.s.PopClip()
	if hasAlt {
		p.brokenIcon(geom.V2(box.X, box.Y), alpha)
		p.altText(e, box, alt, alpha)
		return
	}
	if box.W < BrokenIcon+2*brokenInset || box.H < BrokenIcon+2*brokenInset {
		p.brokenIcon(geom.V2(box.X, box.Y), alpha)
		return
	}
	p.s.Border(box, 0, 1, fade(brokenFrame, alpha))
	p.brokenIcon(geom.V2(box.X+brokenInset, box.Y+brokenInset), alpha)
}

// altText writes alt after the icon, wrapped at spaces to box's width as
// Chrome wraps it: a word that doesn't fit beside the icon goes under it.
// Known difference: in a box only an icon high, Chrome lets the top of the
// second row show; here it starts below the box.
func (p *painter) altText(e entry, box geom.Rect, alt string, alpha float32) {
	st := shown(e.b)
	opts := []twod.TextOption{
		twod.Typeface(p.faceOf(st.Font)), twod.FontSize(st.Font.Size),
		twod.Color(fade(st.Color, alpha)), twod.Tracking(st.LetterSpacing),
	}
	if p.fonts == nil {
		p.s.Text(geom.V2(box.X+BrokenIcon, box.Y), alt, opts...)
		return
	}
	m := p.fonts.Metrics(st.Font)
	lh := m.Ascent + m.Descent + m.LineGap
	row := max(lh, BrokenIcon)
	x, y, line := float32(BrokenIcon), float32(0), ""
	for _, w := range strings.Fields(alt) {
		next := w
		if line != "" {
			next = line + " " + w
		}
		if x+p.fonts.Advance(st.Font, next) > box.W && (line != "" || x > 0) {
			if line != "" {
				p.s.Text(geom.V2(box.X+x, box.Y+y), line, opts...)
			}
			x, y, row, next = 0, y+row, lh, w
		}
		line = next
	}
	if line != "" {
		p.s.Text(geom.V2(box.X+x, box.Y+y), line, opts...)
	}
}

// The broken picture's colours: a sheet of sky with a cloud, a green
// hill, a torn diagonal, a grey edge and folded corner.
//
//nolint:gochecknoglobals // fixed values every paint reads; Go has no constant colours, tables or shapes
var (
	brokenSky   = color.FromSRGB([3]float32{0.80, 0.87, 0.97})
	brokenHill  = color.FromSRGB([3]float32{0.40, 0.69, 0.24})
	brokenEdge  = color.FromSRGB([3]float32{0.62, 0.62, 0.62})
	brokenPaper = color.FromSRGB([3]float32{1, 1, 1})
)

// brokenIcon draws the broken picture with its top-left at at.
func (p *painter) brokenIcon(at geom.Vec2, alpha float32) {
	r := geom.Rect{X: at.X, Y: at.Y, W: BrokenIcon, H: BrokenIcon}
	pt := func(x, y float32) geom.Vec2 { return geom.V2(at.X+x, at.Y+y) }
	p.s.PushClip(r)
	p.s.Rect(r, fade(brokenSky, alpha))
	p.s.Circle(pt(5.4, 5.2), 1.9, fade(brokenPaper, alpha))
	p.s.Circle(pt(3.6, 6.1), 1.2, fade(brokenPaper, alpha))
	p.s.Rect(geom.Rect{X: at.X + 2.6, Y: at.Y + 5.6, W: 5.4, H: 1.5}, fade(brokenPaper, alpha))
	p.s.Circle(pt(8.2, 17.6), 7.4, fade(brokenHill, alpha))
	p.s.Line(pt(7.2, 17), pt(17, 7.2), 1.8, fade(brokenPaper, alpha))
	fold := []geom.Vec2{pt(11, 0), pt(16, 0), pt(16, 5)}
	p.s.Triangles(fold, []int{0, 1, 2}, fade(brokenPaper, alpha))
	p.s.Border(r, 0, 0.5, fade(brokenEdge, alpha))
	p.s.Line(pt(11, 0), pt(16, 5), 0.6, fade(brokenEdge, alpha))
	p.s.PopClip()
}
