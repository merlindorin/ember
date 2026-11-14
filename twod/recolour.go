package twod

import (
	"sync"

	"github.com/lantervale/ember/render"
)

// variantKey names a texture made from another: its source, and the
// operation and its arguments as text.
type variantKey struct {
	source *Texture
	op     string
}

// variants are the textures Recolour and Masked made, so two equal asks
// share one texture and a frame costs nothing more.
//
//nolint:gochecknoglobals // the cache of variants, shared by every World as textures are
var variants = struct {
	sync.Mutex
	made map[variantKey]*Texture
}{made: map[variantKey]*Texture{}}

// variant is the texture key names, made once by build.
func variant(key variantKey, build func() (*Texture, error)) (*Texture, error) {
	variants.Lock()
	defer variants.Unlock()
	if t, ok := variants.made[key]; ok {
		return t, nil
	}
	t, err := build()
	if err != nil {
		return nil, err
	}
	variants.made[key] = t
	return t, nil
}

// derived is a texture of img that samples as source does.
func derived(source *Texture, img *Texture) *Texture {
	img.Filter, img.Mipmaps, img.Repeat, img.Texture.Encoded =
		source.Filter, source.Mipmaps, source.Repeat, source.Texture.Encoded
	return img
}

// Recolour is tex through pal: each colour pal swaps becomes its To, every
// frame and tile of a sheet where it was, alpha kept, other colours as
// drawn. It is made once, on the CPU: the same texture through an equal
// palette gives the same texture. A palette that matches no colour of tex
// is an error.
func Recolour(tex *Texture, pal *Palette) (*Texture, error) {
	return variant(variantKey{source: tex, op: "recolour " + pal.key}, func() (*Texture, error) {
		img, err := pal.recolour(tex.Image)
		if err != nil {
			return nil, err
		}
		return derived(tex, &Texture{Texture: render.Texture{Image: img}, recoloured: pal}), nil
	})
}

type recolouredOption struct{ pal *Palette }

func (o recolouredOption) texture(t *Texture) { t.recoloured = o.pal }

// Recoloured recolours a texture through pal as it loads, as Recolour does:
// LoadTexture refuses a palette that matches no colour of the picture,
// naming it; NewTexture panics on it.
func Recoloured(pal *Palette) TextureOption { return recolouredOption{pal} }

// Recoloured is the palette the texture was recoloured through, nil for
// none.
func (t *Texture) Recoloured() *Palette { return t.recoloured }
