package twod

import (
	"fmt"
	"image"
	"image/draw"
	"math"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/render"
)

// MaskOption sets up a mask Masked cuts with.
type MaskOption interface{ mask(*maskSpec) }

type maskSpec struct{ inverted bool }

type invertedOption struct{}

func (invertedOption) mask(m *maskSpec) { m.inverted = true }

// Inverted keeps where the mask is clear and cuts where it is opaque, so
// one shape gives the hole and the plug.
func Inverted() MaskOption { return invertedOption{} }

// Masked is region of tex kept where maskRegion of mask is opaque, cut
// where it is clear and faded in between, as a new texture of region's size
// (decision 0069): each pixel's colour as drawn, its alpha times the
// mask's. The regions are in pixels, whole, within their textures, and of
// one size. It is made once, on the CPU: the same cut asked again gives the
// same texture. Mask a recoloured texture to recolour, then mask.
func Masked(tex *Texture, region geom.Rect, mask *Texture, maskRegion geom.Rect, opts ...MaskOption) (*Texture, error) {
	var m maskSpec
	for _, o := range opts {
		o.mask(&m)
	}
	r, err := pixels(tex, region)
	if err != nil {
		return nil, fmt.Errorf("twod: masked region: %w", err)
	}
	mr, err := pixels(mask, maskRegion)
	if err != nil {
		return nil, fmt.Errorf("twod: mask region: %w", err)
	}
	if r.Size() != mr.Size() {
		return nil, fmt.Errorf("twod: a mask of %v cuts a region of its own size, not %v", mr.Size(), r.Size())
	}
	op := fmt.Sprintf("mask %p %v %v %v", mask, r, mr, m.inverted)
	return variant(variantKey{source: tex, op: op}, func() (*Texture, error) {
		out := image.NewNRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
		draw.Draw(out, out.Bounds(), tex.Image, r.Min, draw.Src)
		alpha := image.NewNRGBA(out.Bounds())
		draw.Draw(alpha, alpha.Bounds(), mask.Image, mr.Min, draw.Src)
		for i := 3; i < len(out.Pix); i += 4 {
			keep := uint32(alpha.Pix[i])
			if m.inverted {
				keep = 0xff - keep
			}
			out.Pix[i] = uint8((uint32(out.Pix[i])*keep + 0x7f) / 0xff) //nolint:gosec // a byte times a byte over 0xff
		}
		return derived(tex, &Texture{Texture: render.Texture{Image: out}}), nil
	})
}

// pixels is region of tex as whole pixels within its picture.
func pixels(tex *Texture, region geom.Rect) (image.Rectangle, error) {
	whole := func(v float32) bool { return v == float32(math.Trunc(float64(v))) }
	if tex == nil || tex.Image == nil {
		return image.Rectangle{}, fmt.Errorf("a texture with no picture")
	}
	if !whole(region.X) || !whole(region.Y) || !whole(region.W) || !whole(region.H) || region.W <= 0 || region.H <= 0 {
		return image.Rectangle{}, fmt.Errorf("%v is not whole pixels", region)
	}
	b := tex.Image.Bounds()
	r := image.Rect(int(region.X), int(region.Y), int(region.X+region.W), int(region.Y+region.H)).Add(b.Min)
	if !r.In(b) {
		return image.Rectangle{}, fmt.Errorf("%v is outside its picture of %v", region, b.Size())
	}
	return r, nil
}
