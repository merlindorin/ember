package twod

import (
	"fmt"
	"image"
	stdcolor "image/color"
	"image/draw"
	"image/png"
	"io/fs"
	"math"
	"slices"
	"strings"

	"github.com/lantervale/ember/color"
)

// ColorSwap is one swap of a Palette: From becomes To. With AtAlpha it
// matches only pixels of that Alpha; without, any alpha.
type ColorSwap struct {
	From, To color.Color
	Alpha    uint8
	AtAlpha  bool
}

// Swap swaps a colour for another, exactly, at any alpha: what a palette is
// made of.
func Swap(from, to color.Color) ColorSwap { return ColorSwap{From: from, To: to} }

// SwapAt swaps a colour for another only where a pixel has that 8-bit
// alpha, so one colour drawn opaque and half clear can be told apart. It
// wins over a Swap of the same colour; alpha 0 matches nothing.
func SwapAt(from color.Color, alpha uint8, to color.Color) ColorSwap {
	return ColorSwap{From: from, To: to, Alpha: alpha, AtAlpha: true}
}

// Palette swaps colours: each From becomes its To where a picture holds it
// exactly, as 8-bit sRGB, at the alpha it names or at any; the pixel's alpha
// is kept, From's and To's own alpha unused, and any other colour stays as
// drawn (decision 0069). Recolour applies it to a texture, Recoloured to one
// as it loads.
type Palette struct {
	swaps []ColorSwap
	// table is keyed by slot: a colour at one alpha, or at any.
	table map[uint32][3]uint8
	// atAlpha says a swap names an alpha, so pixels look theirs up first.
	atAlpha bool
	// key names the palette by what it swaps, so two equal palettes share
	// their recoloured textures.
	key string
}

// swapSlot is rgb at alpha in a palette's table. Alpha 0 is the colour at any
// alpha: no swap matches a clear pixel.
func swapSlot(rgb [3]uint8, alpha uint8) uint32 {
	return uint32(alpha)<<24 | uint32(rgb[0])<<16 | uint32(rgb[1])<<8 | uint32(rgb[2])
}

// NewPalette is a palette of swaps; a colour swapped twice at the same
// alpha, or twice at any, takes its last To.
func NewPalette(swaps ...ColorSwap) *Palette {
	p := &Palette{table: map[uint32][3]uint8{}}
	same := func(a, b ColorSwap) bool {
		return rgb8(a.From) == rgb8(b.From) && a.AtAlpha == b.AtAlpha && (!a.AtAlpha || a.Alpha == b.Alpha)
	}
	for _, s := range swaps {
		if i := slices.IndexFunc(p.swaps, func(o ColorSwap) bool { return same(o, s) }); i >= 0 {
			p.swaps[i] = s
		} else {
			p.swaps = append(p.swaps, s)
		}
	}
	keys := make([]string, 0, len(p.swaps))
	for _, s := range p.swaps {
		from, to := rgb8(s.From), rgb8(s.To)
		at := ""
		switch {
		case !s.AtAlpha:
			p.table[swapSlot(from, 0)] = to
		case s.Alpha != 0:
			p.table[swapSlot(from, s.Alpha)] = to
			p.atAlpha = true
		}
		if s.AtAlpha {
			at = fmt.Sprintf("@%02x", s.Alpha)
		}
		keys = append(keys, fmt.Sprintf("%02x%02x%02x%s>%02x%02x%02x", from[0], from[1], from[2], at, to[0], to[1], to[2]))
	}
	slices.Sort(keys)
	p.key = strings.Join(keys, ",")
	return p
}

// Swaps are the palette's swaps, in the order given.
func (p *Palette) Swaps() []ColorSwap { return slices.Clone(p.swaps) }

// LoadPalette reads the palette picture name in fsys: two rows of square
// swatches, each colour of the row above becoming the one below it, as an
// artist paints them in a pixel editor. An opaque swatch swaps its colour at
// any alpha; a half-clear one only at its own alpha, as SwapAt, and the
// swatch below it is as clear. A clear swatch, or a colour swapped twice at
// one alpha, to two colours, is an error naming it.
func LoadPalette(fsys fs.FS, name string) (*Palette, error) {
	f, err := fsys.Open(name)
	if err != nil {
		return nil, fmt.Errorf("twod: palette %s: %w", name, err)
	}
	defer func() { _ = f.Close() }()
	img, err := png.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("twod: palette %s: %w", name, err)
	}
	b := img.Bounds()
	side := b.Dy() / 2
	if side == 0 || b.Dy()%2 != 0 || b.Dx()%side != 0 {
		return nil, fmt.Errorf("twod: palette %s: %dx%d is not two rows of square swatches", name, b.Dx(), b.Dy())
	}
	swatch := func(column, row int) stdcolor.NRGBA {
		c, _ := stdcolor.NRGBAModel.Convert(img.At(b.Min.X+column*side+side/2, b.Min.Y+row*side+side/2)).(stdcolor.NRGBA)
		return c
	}
	var swaps []ColorSwap
	seen := map[[4]uint8][3]uint8{}
	for column := range b.Dx() / side {
		from, to := swatch(column, 0), swatch(column, 1)
		if from.A == 0 {
			return nil, fmt.Errorf("twod: palette %s: swatch %d of row 1 is clear", name, column+1)
		}
		if to.A != from.A {
			return nil, fmt.Errorf("twod: palette %s: swatch %d of row 2 is not as clear as the one above it", name, column+1)
		}
		key, want := [4]uint8{from.R, from.G, from.B, from.A}, [3]uint8{to.R, to.G, to.B}
		if had, twice := seen[key]; twice && had != want {
			return nil, fmt.Errorf("twod: palette %s: #%02x%02x%02x%02x is swapped twice", name, from.R, from.G, from.B, from.A)
		}
		seen[key] = want
		swap := Swap(color.RGB8(from.R, from.G, from.B), color.RGB8(to.R, to.G, to.B))
		if from.A != 0xff {
			swap = SwapAt(swap.From, from.A, swap.To)
		}
		swaps = append(swaps, swap)
	}
	return NewPalette(swaps...), nil
}

// rgb8 is c's 8-bit sRGB, as color.Hex writes it.
func rgb8(c color.Color) [3]uint8 {
	s := c.SRGB()
	b := func(v float32) uint8 { return uint8(math.Round(float64(v) * 255)) }
	return [3]uint8{b(s[0]), b(s[1]), b(s[2])}
}

// recolour is img through p: every pixel of a swapped colour takes the To
// of its alpha, else the one at any alpha, keeping its alpha. A picture
// holding none of p's colours is an error.
func (p *Palette) recolour(img image.Image) (*image.NRGBA, error) {
	b := img.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(out, out.Bounds(), img, b.Min, draw.Src)
	matched := false
	for i := 0; i < len(out.Pix); i += 4 {
		px := out.Pix[i : i+4 : i+4]
		if px[3] == 0 {
			continue
		}
		rgb := [3]uint8{px[0], px[1], px[2]}
		to, ok := [3]uint8{}, false
		if p.atAlpha {
			to, ok = p.table[swapSlot(rgb, px[3])]
		}
		if !ok {
			to, ok = p.table[swapSlot(rgb, 0)]
		}
		if ok {
			px[0], px[1], px[2], matched = to[0], to[1], to[2], true
		}
	}
	if !matched {
		return nil, fmt.Errorf("twod: the palette (%s) matches no colour of the picture", p.key)
	}
	return out, nil
}
