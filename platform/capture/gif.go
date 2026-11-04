package capture

import (
	"cmp"
	"image"
	"image/color"
	"image/gif"
	"io"
	"slices"

	xdraw "golang.org/x/image/draw"
)

// GIF collects frames and writes them as one looping animated GIF, with a
// palette shared by every frame so colours don't flicker between them.
// Colours are dithered with a fixed 4×4 Bayer pattern, so gradients don't
// band and still areas stay identical from frame to frame.
type GIF struct {
	frames []*image.RGBA
	delay  int
}

// NewGIF returns an empty GIF shown at fps frames a second.
func NewGIF(fps int) *GIF { return &GIF{delay: max(1, (100+fps/2)/fps)} }

// Add appends a copy of img scaled to w×h, such as a framebuffer at the
// window's size in points.
func (g *GIF) Add(img image.Image, w, h int) {
	frame := image.NewRGBA(image.Rect(0, 0, w, h))
	xdraw.ApproxBiLinear.Scale(frame, frame.Bounds(), img, img.Bounds(), xdraw.Src, nil)
	g.frames = append(g.frames, frame)
}

// Len is the number of frames so far; a nil GIF has none.
func (g *GIF) Len() int {
	if g == nil {
		return 0
	}
	return len(g.frames)
}

// Encode writes the frames, looping forever. After the first, each frame
// holds only the rectangle that changed, its unchanged pixels transparent,
// drawn over the one before: still parts cost nothing.
func (g *GIF) Encode(w io.Writer) error {
	frames := padded(g.frames)
	pal := append(palette(frames, 255), color.RGBA{})
	none := byte8(len(pal) - 1)
	q := newQuantizer(pal)
	anim := &gif.GIF{LoopCount: 0}
	var before *image.Paletted
	for _, f := range frames {
		p := q.dither(f)
		frame := p
		if before != nil {
			r := changed(before, p)
			frame = image.NewPaletted(r, pal)
			for y := r.Min.Y; y < r.Max.Y; y++ {
				for x := r.Min.X; x < r.Max.X; x++ {
					i := p.ColorIndexAt(x, y)
					if i == before.ColorIndexAt(x, y) {
						i = none
					}
					frame.SetColorIndex(x, y, i)
				}
			}
		}
		anim.Image = append(anim.Image, frame)
		anim.Delay = append(anim.Delay, g.delay)
		anim.Disposal = append(anim.Disposal, gif.DisposalNone)
		before = p
	}
	if len(frames) > 0 {
		anim.Config = image.Config{ColorModel: pal, Width: frames[0].Bounds().Dx(), Height: frames[0].Bounds().Dy()}
	}
	return gif.EncodeAll(w, anim)
}

// desk is what shows around a window smaller than the largest it was.
var desk = color.RGBA{0x10, 0x11, 0x14, 0xff} //nolint:gochecknoglobals // a colour, as a constant

// padded puts every frame on a canvas as large as the largest, at its
// top-left corner, on the desk, for a window that was resized. The desk is
// opaque: a GIF frame can't erase back to transparent where a larger one
// drew before.
func padded(frames []*image.RGBA) []*image.RGBA {
	var size image.Point
	for _, f := range frames {
		size = image.Pt(max(size.X, f.Bounds().Dx()), max(size.Y, f.Bounds().Dy()))
	}
	out := make([]*image.RGBA, len(frames))
	for i, f := range frames {
		if f.Bounds().Size() == size {
			out[i] = f
			continue
		}
		c := image.NewRGBA(image.Rectangle{Max: size})
		for px := 0; px < len(c.Pix); px += 4 {
			c.Pix[px], c.Pix[px+1], c.Pix[px+2], c.Pix[px+3] = desk.R, desk.G, desk.B, desk.A
		}
		copyRows(c, f)
		out[i] = c
	}
	return out
}

// copyRows copies src onto dst's top-left corner.
func copyRows(dst, src *image.RGBA) {
	w := src.Bounds().Dx() * 4
	for y := range src.Bounds().Dy() {
		copy(dst.Pix[y*dst.Stride:y*dst.Stride+w], src.Pix[y*src.Stride:y*src.Stride+w])
	}
}

// bayer is the 4×4 ordered-dither pattern, its thresholds from 0 to 15.
//
//nolint:gochecknoglobals // a lookup table
var bayer = [4][4]float32{{0, 8, 2, 10}, {12, 4, 14, 6}, {3, 11, 1, 9}, {15, 7, 13, 5}}

// spread is how far, in 8-bit levels, the pattern moves a colour: about
// the gap between neighbouring palette colours.
const spread = 24

// quantizer maps colours to the nearest of a palette, its last entry being
// the transparent one, through a cache of 5-bit colours.
type quantizer struct {
	pal   color.Palette
	cache [32 * 32 * 32]int16
}

func newQuantizer(pal color.Palette) *quantizer {
	q := &quantizer{pal: pal}
	for i := range q.cache {
		q.cache[i] = -1
	}
	return q
}

// dither returns f in the palette, each pixel moved by the Bayer pattern
// before it is matched.
func (q *quantizer) dither(f *image.RGBA) *image.Paletted {
	b := f.Bounds()
	p := image.NewPaletted(b, q.pal)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := f.RGBAAt(x, y)
			if c.A == 0 {
				p.SetColorIndex(x, y, byte8(len(q.pal)-1))
				continue
			}
			d := (bayer[y&3][x&3]/16 - 0.5) * spread
			p.SetColorIndex(x, y, q.index(nudge(c.R, d), nudge(c.G, d), nudge(c.B, d)))
		}
	}
	return p
}

func nudge(v uint8, d float32) uint8 { return uint8(min(255, max(0, float32(v)+d))) }

func (q *quantizer) index(r, g, b uint8) uint8 {
	key := int(r>>3)<<10 | int(g>>3)<<5 | int(b>>3)
	if i := q.cache[key]; i >= 0 {
		return byte8(int(i))
	}
	opaque := q.pal[:len(q.pal)-1]
	i := opaque.Index(color.RGBA{r&^7 | 4, g&^7 | 4, b&^7 | 4, 255})
	q.cache[key] = int16(byte8(i))
	return byte8(i)
}

// changed is the smallest rectangle holding every pixel that differs
// between a and b, or a single pixel when none does.
func changed(a, b *image.Paletted) image.Rectangle {
	r := image.Rectangle{}
	bounds := a.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		row := (y - bounds.Min.Y) * a.Stride
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if a.Pix[row+x-bounds.Min.X] != b.Pix[row+x-bounds.Min.X] {
				r = r.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	if r.Empty() {
		return image.Rect(bounds.Min.X, bounds.Min.Y, bounds.Min.X+1, bounds.Min.Y+1)
	}
	return r
}

// palette is a median cut of the frames' colours: boxes of colours are
// split at the median of their widest channel until there are n.
func palette(frames []*image.RGBA, n int) color.Palette {
	var px []color.RGBA
	for _, f := range frames {
		step := max(1, len(f.Pix)/4/40000)
		for i := 0; i < len(f.Pix); i += 4 * step {
			if f.Pix[i+3] != 0 {
				px = append(px, color.RGBA{f.Pix[i], f.Pix[i+1], f.Pix[i+2], 255})
			}
		}
	}
	if len(px) == 0 {
		return color.Palette{color.Black}
	}
	boxes := [][]color.RGBA{px}
	for len(boxes) < n {
		i, ch := widest(boxes)
		if i < 0 {
			break
		}
		b := boxes[i]
		slices.SortFunc(b, func(p, q color.RGBA) int { return cmp.Compare(channel(p, ch), channel(q, ch)) })
		boxes[i], boxes = b[:len(b)/2], append(boxes, b[len(b)/2:])
	}
	pal := make(color.Palette, 0, len(boxes))
	for _, b := range boxes {
		var r, g, bl int
		for _, p := range b {
			r, g, bl = r+int(p.R), g+int(p.G), bl+int(p.B)
		}
		pal = append(pal, color.RGBA{byte8(r / len(b)), byte8(g / len(b)), byte8(bl / len(b)), 255})
	}
	return pal
}

// widest returns the box with the widest colour range and that channel, or
// -1 when no box can be split.
func widest(boxes [][]color.RGBA) (int, int) {
	best, bestCh, bestRange := -1, 0, 0
	for i, b := range boxes {
		if len(b) < 2 {
			continue
		}
		for ch := range 3 {
			lo, hi := 255, 0
			for _, p := range b {
				v := channel(p, ch)
				lo, hi = min(lo, v), max(hi, v)
			}
			if hi-lo > bestRange {
				best, bestCh, bestRange = i, ch, hi-lo
			}
		}
	}
	return best, bestCh
}

func channel(p color.RGBA, ch int) int {
	switch ch {
	case 0:
		return int(p.R)
	case 1:
		return int(p.G)
	}
	return int(p.B)
}

// byte8 is v as a byte: a palette index (a GIF's palette holds at most 256
// colours) or a channel's mean.
func byte8(v int) uint8 {
	if v < 0 || v > 255 {
		panic("capture: a palette index or channel out of range")
	}
	return uint8(v)
}
