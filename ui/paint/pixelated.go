package paint

import (
	"sync"

	"github.com/merlindorin/ember/twod"
)

// sharp keeps each texture's pixelated copy, up to maxSharp, then starts
// over.
//
//nolint:gochecknoglobals // a cache every paint shares, under its mutex
var (
	sharp   = map[*twod.Texture]*twod.Texture{}
	sharpMu sync.Mutex
)

const maxSharp = 256

// pixelated is tex sampled by its nearest pixels: a copy sharing its
// picture, made once, given tex's new picture and touched whenever tex
// has been since, so the GPU uploads it again. A render texture has no picture to share and
// stays as it is.
func pixelated(tex *twod.Texture) *twod.Texture {
	if tex.Image == nil {
		return tex
	}
	sharpMu.Lock()
	defer sharpMu.Unlock()
	c, ok := sharp[tex]
	if !ok {
		if len(sharp) >= maxSharp {
			clear(sharp)
		}
		opts := []twod.TextureOption{twod.Pixelated()}
		if tex.Encoded() {
			opts = append(opts, twod.Encoded())
		}
		c = twod.NewTexture(tex.Image, opts...)
		c.Repeat = tex.Repeat
		sharp[tex] = c
	}
	if c.Image != tex.Image {
		c.Image = tex.Image
		c.Touch()
	}
	for c.Version() < tex.Version() {
		c.Touch()
	}
	return c
}
