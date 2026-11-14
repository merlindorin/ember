package twod

import (
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/render"
)

// Filter is how a picture is scaled: twod's name for render.Filter.
type Filter = render.Filter

const (
	// Nearest keeps pixels sharp.
	Nearest = render.FilterNearest
	// Linear smooths them.
	Linear = render.FilterLinear
)

// letterbox places a picture of size pixels in a window of screen points
// and fb pixels: scaled by the largest whole factor that fits, or by any
// factor when fit, centred on whole pixels. It maps the picture's points
// to the window's.
func letterbox(size, screen, fb geom.Vec2, fit bool) geom.Transform2D {
	if fb.X <= 0 || fb.Y <= 0 {
		fb = screen
	}
	density := fb.X / max(screen.X, 1)
	k := min(fb.X/size.X, fb.Y/size.Y)
	if !fit && k >= 1 {
		k = geom.Floor(k)
	}
	shown := size.Scale(k)
	origin := geom.V2(geom.Floor((fb.X-shown.X)/2), geom.Floor((fb.Y-shown.Y)/2)).Scale(1 / density)
	scale := k / density
	return geom.Transform2D{X: geom.V2(scale, 0), Y: geom.V2(0, scale), Origin: origin}
}
