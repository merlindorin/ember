package gpu

import (
	"math"

	"github.com/lantervale/ember/color"
)

// hdrBackground is the scene colour that the composite pass (exposure,
// ACES, then sRGB) turns back into bg, so a window shows its background
// as given behind a camera's scene too.
func hdrBackground(bg color.Color, exposure float32) [3]float32 {
	if exposure <= 0 {
		exposure = 1
	}
	var out [3]float32
	for i, v := range [3]float32{bg.R, bg.G, bg.B} {
		y := float64(max(v, 0))
		out[i] = float32(acesInverse(min(y, 0.999)) / float64(exposure))
	}
	return out
}

// acesInverse solves y = x(2.51x+0.03) / (x(2.43x+0.59)+0.14) for x ≥ 0.
func acesInverse(y float64) float64 {
	if y <= 0 {
		return 0
	}
	a, b, c := 2.43*y-2.51, 0.59*y-0.03, 0.14*y
	return (-b - math.Sqrt(b*b-4*a*c)) / (2 * a)
}
