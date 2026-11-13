package twod

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io/fs"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/node"
	"github.com/merlindorin/ember/render"
)

//nolint:gochecknoinits // registers the package's resource types on import
func init() {
	node.RegisterResource("twod.Texture", func() node.Resource { return &Texture{} })
}

// Texture is an image sprites draw, stored in scene files as PNG.
type Texture struct {
	render.Texture
	// width and height are a render texture's size, which has no image.
	width, height int
	// recoloured is the palette the image went through, nil for none.
	recoloured *Palette
}

// TextureOption sets up a Texture.
type TextureOption interface{ texture(*Texture) }

// FilterOption sets how a texture or a tiled picture is sampled.
type FilterOption interface {
	TextureOption
	TiledPictureOption
}

type filterOption render.Filter

func (o filterOption) texture(t *Texture)           { t.Filter = render.Filter(o) }
func (o filterOption) tiledPicture(p *TiledPicture) { p.filter = render.Filter(o) }

// PixelatedOption is Pixelated: a filter for a texture or a tiled picture,
// and a text option.
type PixelatedOption interface {
	FilterOption
	TextOption
}

func (o filterOption) text(t *Text) { t.pixelated = render.Filter(o) == render.FilterNearest }

// Pixelated keeps pixels sharp: a texture or a tiled picture scaled with
// nearest filtering, for pixel art and maps of cells, both smooth (linear)
// by default; a text's glyphs drawn unsmoothed, each pixel covered or
// not, for pixel fonts, as Chrome draws -webkit-font-smoothing: none.
func Pixelated() PixelatedOption { return filterOption(render.FilterNearest) }

type encodedOption struct{}

func (encodedOption) texture(t *Texture) { t.Texture.Encoded = true }

// Encoded filters a texture in encoded sRGB, as a browser filters images,
// instead of in linear light: a UI's pictures, so they scale as Chrome
// shows them (decision 0056). Sprites keep the default.
func Encoded() TextureOption { return encodedOption{} }

// Encoded reports whether t filters in encoded sRGB.
func (t *Texture) Encoded() bool { return t.Texture.Encoded }

// NewTexture wraps img.
func NewTexture(img image.Image, opts ...TextureOption) *Texture {
	t, err := newTexture(img, opts)
	if err != nil {
		panic(err)
	}
	return t
}

// newTexture wraps img, recoloured when an option asks for it.
func newTexture(img image.Image, opts []TextureOption) (*Texture, error) {
	t := &Texture{Texture: render.Texture{Image: img}}
	for _, opt := range opts {
		opt.texture(t)
	}
	if t.recoloured != nil && img != nil {
		recoloured, err := t.recoloured.recolour(img)
		if err != nil {
			return nil, err
		}
		t.Image = recoloured
	}
	return t, nil
}

// NewRenderTexture returns a texture of w×h pixels with no image of its
// own, that a camera draws Into: a minimap, a portrait. Sprites show it
// like any texture.
func NewRenderTexture(w, h int, opts ...TextureOption) *Texture {
	t := &Texture{width: max(w, 1), height: max(h, 1)}
	for _, opt := range opts {
		opt.texture(t)
	}
	return t
}

// LoadTexture decodes the PNG name in fsys, such as an embed.FS.
func LoadTexture(fsys fs.FS, name string, opts ...TextureOption) (*Texture, error) {
	f, err := fsys.Open(name)
	if err != nil {
		return nil, fmt.Errorf("twod: %w", err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("twod: %s: %w", name, err)
	}
	t, err := newTexture(img, opts)
	if err != nil {
		return nil, fmt.Errorf("twod: %s: %w", name, err)
	}
	return t, nil
}

func (*Texture) ResourceType() string { return "twod.Texture" }

// Size is the image's size in pixels.
func (t *Texture) Size() geom.Vec2 {
	if t == nil {
		return geom.Vec2{}
	}
	if t.Image == nil {
		return geom.V2(float32(t.width), float32(t.height))
	}
	b := t.Image.Bounds()
	return geom.Vec2{X: float32(b.Dx()), Y: float32(b.Dy())}
}

// uv is region, in pixels, as texture coordinates.
func (t *Texture) uv(region geom.Rect) geom.Rect {
	size := t.Size()
	return geom.Rect{X: region.X / size.X, Y: region.Y / size.Y, W: region.W / size.X, H: region.H / size.Y}
}

func (t *Texture) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	if t.Image != nil {
		if err := png.Encode(&buf, t.Image); err != nil {
			return nil, err
		}
	}
	return json.Marshal(
		map[string]any{
			"png":    buf.Bytes(),
			"filter": t.Filter,
			"repeat": t.Repeat,
			"width":  t.width,
			"height": t.height,
		},
	)
}

func (t *Texture) UnmarshalJSON(b []byte) error {
	var d struct {
		PNG    string        `json:"png"`
		Filter render.Filter `json:"filter"`
		Repeat bool          `json:"repeat"`
		Width  int           `json:"width"`
		Height int           `json:"height"`
	}
	if err := json.Unmarshal(b, &d); err != nil {
		return err
	}
	raw, err := base64.StdEncoding.DecodeString(d.PNG)
	if err != nil {
		return fmt.Errorf("twod: texture: %w", err)
	}
	var img image.Image
	if len(raw) > 0 {
		if img, err = png.Decode(bytes.NewReader(raw)); err != nil {
			return fmt.Errorf("twod: texture: %w", err)
		}
	}
	t.Image, t.Filter, t.Repeat, t.width, t.height = img, d.Filter, d.Repeat, d.Width, d.Height
	t.Touch()
	return nil
}
