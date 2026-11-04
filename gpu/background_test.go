package gpu //nolint:testpackage // tests the unexported tone curve of HDR backgrounds

import (
	"math"
	"testing"

	"github.com/merlindorin/ember/color"
)

func aces(x float64) float64 {
	return min(max(x*(2.51*x+0.03)/(x*(2.43*x+0.59)+0.14), 0), 1)
}

func TestTheBackgroundComesOutOfTheCompositeAsGiven(t *testing.T) {
	for _, bg := range []color.Color{
		color.Hex("#2b2f3a"), color.Hex("#7d828c"), color.Hex("#c8b9a0"), color.Black, color.Hex("#fefefe"),
	} {
		for _, exposure := range []float32{1, 0.95, 2} {
			hdr := hdrBackground(bg, exposure)
			want := bg.SRGB()
			for i := range 3 {
				got := float64(color.LinearToSRGB(float32(aces(float64(hdr[i]) * float64(exposure)))))
				if math.Abs(got-float64(want[i])) > 2e-3 {
					t.Fatalf("%v at exposure %v: channel %d is %.4f, want %.4f", bg, exposure, i, got, want[i])
				}
			}
		}
	}
}
