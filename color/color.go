// Package color provides linear float RGBA colors. Values above 1 are valid
// and meaningful: the renderer works in HDR, so an emissive color of
// RGB(4, 1.5, 0.3) glows and blooms.
package color

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

// Color is a linear RGBA colour, each channel 0–1; in JSON its fields keep
// their names.
type Color struct {
	R float32 `json:"R"`
	G float32 `json:"G"`
	B float32 `json:"B"`
	A float32 `json:"A"`
}

// UnmarshalJSON reads a colour as {"R", "G", "B", "A"} or as hex text,
// "#rrggbb" or "#rrggbbaa".
func (c *Color) UnmarshalJSON(data []byte) error {
	if len(data) > 0 && data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		v, err := ParseHex(s)
		if err != nil {
			return err
		}
		*c = v
		return nil
	}
	type plain Color
	return json.Unmarshal(data, (*plain)(c))
}

func RGB(r, g, b float32) Color     { return Color{r, g, b, 1} }
func RGBA(r, g, b, a float32) Color { return Color{r, g, b, a} }
func Gray(v float32) Color          { return Color{v, v, v, 1} }

// RGB8 builds a color from 0-255 sRGB components, converting to linear.
func RGB8(r, g, b uint8) Color {
	return Color{srgbToLinear(float32(r) / 255), srgbToLinear(float32(g) / 255), srgbToLinear(float32(b) / 255), 1}
}

// Hex parses "#rgb", "#rrggbb" or "#rrggbbaa" sRGB notation into linear color.
func Hex(s string) Color {
	c, err := ParseHex(s)
	if err != nil {
		panic(err)
	}
	return c
}

func ParseHex(s string) (Color, error) {
	h := strings.TrimPrefix(s, "#")
	var v [8]uint8
	switch len(h) {
	case 3:
		for i := range 3 {
			d, ok := hexDigit(h[i])
			if !ok {
				return Color{}, fmt.Errorf("color: invalid hex %q", s)
			}
			v[2*i], v[2*i+1] = d, d
		}
		v[6], v[7] = 15, 15
	case 6, 8:
		for i := range len(h) {
			d, ok := hexDigit(h[i])
			if !ok {
				return Color{}, fmt.Errorf("color: invalid hex %q", s)
			}
			v[i] = d
		}
		if len(h) == 6 {
			v[6], v[7] = 15, 15
		}
	default:
		return Color{}, fmt.Errorf("color: invalid hex %q", s)
	}
	c := RGB8(v[0]<<4|v[1], v[2]<<4|v[3], v[4]<<4|v[5])
	c.A = float32(v[6]<<4|v[7]) / 255
	return c, nil
}

// hexDigit is the value of the hex digit b, of either case.
func hexDigit(b byte) (uint8, bool) {
	switch {
	case '0' <= b && b <= '9':
		return b - '0', true
	case 'a' <= b && b <= 'f':
		return b - 'a' + 10, true
	case 'A' <= b && b <= 'F':
		return b - 'A' + 10, true
	}
	return 0, false
}

// HSV builds a linear color from hue (0-360), saturation and value (0-1).
func HSV(h, s, v float32) Color {
	h = float32(math.Mod(float64(h), 360))
	if h < 0 {
		h += 360
	}
	c := v * s
	x := c * (1 - float32(math.Abs(math.Mod(float64(h/60), 2)-1)))
	m := v - c
	var r, g, b float32
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	return Color{srgbToLinear(r + m), srgbToLinear(g + m), srgbToLinear(b + m), 1}
}

// Scale multiplies RGB by k, leaving alpha untouched. Use it for intensity.
func (c Color) Scale(k float32) Color { return Color{c.R * k, c.G * k, c.B * k, c.A} }

func (c Color) Mul(o Color) Color { return Color{c.R * o.R, c.G * o.G, c.B * o.B, c.A * o.A} }

func (c Color) Add(o Color) Color { return Color{c.R + o.R, c.G + o.G, c.B + o.B, c.A} }

func (c Color) Alpha(a float32) Color { return Color{c.R, c.G, c.B, a} }

func (c Color) Lerp(o Color, t float32) Color {
	return Color{c.R + (o.R-c.R)*t, c.G + (o.G-c.G)*t, c.B + (o.B-c.B)*t, c.A + (o.A-c.A)*t}
}

func (c Color) Array() [4]float32 { return [4]float32{c.R, c.G, c.B, c.A} }

func srgbToLinear(v float32) float32 {
	if v <= 0.04045 {
		return v / 12.92
	}
	return float32(math.Pow(float64((v+0.055)/1.055), 2.4))
}

//nolint:gochecknoglobals // the named colours are the package's API, read-only values
var (
	White       = RGB(1, 1, 1)
	Black       = RGB(0, 0, 0)
	Transparent = RGBA(0, 0, 0, 0)
	Red         = RGB(1, 0, 0)
	Green       = RGB(0, 1, 0)
	Blue        = RGB(0, 0, 1)
	Yellow      = RGB(1, 1, 0)
	Cyan        = RGB(0, 1, 1)
	Magenta     = RGB(1, 0, 1)
	Orange      = RGB(1, 0.35, 0.02)
)

func linearToSRGB(v float32) float32 {
	v = max(0, min(v, 1))
	if v <= 0.0031308 {
		return v * 12.92
	}
	return 1.055*float32(math.Pow(float64(v), 1/2.4)) - 0.055
}

// SRGBToLinear and LinearToSRGB are the exact sRGB transfer of one 0-1
// component, as CSS defines it.
func SRGBToLinear(v float32) float32 { return srgbToLinear(v) }
func LinearToSRGB(v float32) float32 { return linearToSRGB(v) }

// SRGB returns the 0-1 sRGB components, as color pickers show them.
func (c Color) SRGB() [3]float32 {
	return [3]float32{linearToSRGB(c.R), linearToSRGB(c.G), linearToSRGB(c.B)}
}

// FromSRGB builds an opaque linear color from 0-1 sRGB components.
func FromSRGB(c [3]float32) Color {
	return Color{srgbToLinear(c[0]), srgbToLinear(c[1]), srgbToLinear(c[2]), 1}
}

// Hex formats the color in "#rrggbb" sRGB notation, the inverse of Hex.
func (c Color) Hex() string {
	s := c.SRGB()
	b := func(v float32) uint8 { return uint8(math.Round(float64(v) * 255)) }
	return fmt.Sprintf("#%02x%02x%02x", b(s[0]), b(s[1]), b(s[2]))
}
