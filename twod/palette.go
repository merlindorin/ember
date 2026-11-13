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

	"github.com/merlindorin/ember/color"
)

// ColorSwap is one swap of a Palette: From becomes To.
type ColorSwap struct {
	From, To color.Color
}

// Swap swaps a colour for another, exactly: what a palette is made of.
func Swap(from, to color.Color) ColorSwap { return ColorSwap{From: from, To: to} }

// Palette swaps colours: each From becomes its To where a picture holds it
// exactly, as 8-bit sRGB, the pixel's alpha kept; any other colour stays as
// drawn (decision 0069). Recolour applies it to a texture, Recoloured to one
// as it loads.
type Palette struct {
	swaps []ColorSwap
	table map[[3]uint8][3]uint8
	// key names the palette by what it swaps, so two equal palettes share
	// their recoloured textures.
	key string
}

// NewPalette is a palette of swaps; a colour swapped twice takes its last
// To.
func NewPalette(swaps ...ColorSwap) *Palette {
	p := &Palette{table: map[[3]uint8][3]uint8{}}
	for _, s := range swaps {
		from := rgb8(s.From)
		if _, again := p.table[from]; again {
			i := slices.IndexFunc(p.swaps, func(o ColorSwap) bool { return rgb8(o.From) == from })
			p.swaps[i] = s
		} else {
			p.swaps = append(p.swaps, s)
		}
		p.table[from] = rgb8(s.To)
	}
	keys := make([]string, 0, len(p.table))
	for from, to := range p.table {
		keys = append(keys, fmt.Sprintf("%02x%02x%02x>%02x%02x%02x", from[0], from[1], from[2], to[0], to[1], to[2]))
	}
	slices.Sort(keys)
	p.key = strings.Join(keys, ",")
	return p
}

// Swaps are the palette's swaps, in the order given.
func (p *Palette) Swaps() []ColorSwap { return slices.Clone(p.swaps) }

// LoadPalette reads the palette picture name in fsys: two rows of square
// swatches, each colour of the row above becoming the one below it, as an
// artist paints them in a pixel editor. Swatches are opaque, and a colour
// swapped twice, to two colours, is an error naming it.
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
	swatch := func(column, row int) (stdcolor.NRGBA, error) {
		c, _ := stdcolor.NRGBAModel.Convert(img.At(b.Min.X+column*side+side/2, b.Min.Y+row*side+side/2)).(stdcolor.NRGBA)
		if c.A != 0xff {
			return c, fmt.Errorf("twod: palette %s: swatch %d of row %d is not opaque", name, column+1, row+1)
		}
		return c, nil
	}
	var swaps []ColorSwap
	seen := map[[3]uint8][3]uint8{}
	for column := range b.Dx() / side {
		from, fromErr := swatch(column, 0)
		if fromErr != nil {
			return nil, fromErr
		}
		to, toErr := swatch(column, 1)
		if toErr != nil {
			return nil, toErr
		}
		key, want := [3]uint8{from.R, from.G, from.B}, [3]uint8{to.R, to.G, to.B}
		if had, twice := seen[key]; twice && had != want {
			return nil, fmt.Errorf("twod: palette %s: #%02x%02x%02x is swapped twice", name, from.R, from.G, from.B)
		}
		seen[key] = want
		swaps = append(swaps, Swap(color.RGB8(from.R, from.G, from.B), color.RGB8(to.R, to.G, to.B)))
	}
	return NewPalette(swaps...), nil
}

// rgb8 is c's 8-bit sRGB, as color.Hex writes it.
func rgb8(c color.Color) [3]uint8 {
	s := c.SRGB()
	b := func(v float32) uint8 { return uint8(math.Round(float64(v) * 255)) }
	return [3]uint8{b(s[0]), b(s[1]), b(s[2])}
}

// recolour is img through p: every pixel of a swapped colour takes its To,
// keeping its alpha. A picture holding none of p's colours is an error.
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
		if to, ok := p.table[[3]uint8{px[0], px[1], px[2]}]; ok {
			px[0], px[1], px[2], matched = to[0], to[1], to[2], true
		}
	}
	if !matched {
		return nil, fmt.Errorf("twod: the palette (%s) matches no colour of the picture", p.key)
	}
	return out, nil
}
