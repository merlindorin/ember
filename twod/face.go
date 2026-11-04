package twod

import (
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/internal/draw"
)

// Face is a TrueType font file's typeface, drawn at any size.
type Face struct {
	face      *draw.Face
	fallbacks []*Face
	emoji     *EmojiSet
}

// FaceOption sets up a Face.
type FaceOption func(*Face)

// Fallback draws what the face lacks, such as emoji or another script,
// with the first of faces that has it.
func Fallback(faces ...*Face) FaceOption {
	return func(f *Face) { f.fallbacks = append(f.fallbacks, faces...) }
}

// Emoji draws the emoji set has as its pictures, inline at the text's
// size and untinted: colour emoji.
func Emoji(set *EmojiSet) FaceOption { return func(f *Face) { f.emoji = set } }

// Draws reports whether the face draws text as written: each character by
// its glyphs or its fallbacks', and each flag, family or keycap by an emoji
// picture, rather than as '?' or a first letter.
func (f *Face) Draws(text string) bool { return f.draw().Draws(text) }

// Fallbacks is the faces that draw what this one lacks.
func (f *Face) Fallbacks() []*Face { return f.fallbacks }

// Emoji is the set whose pictures replace emoji, nil without one.
func (f *Face) Emoji() *EmojiSet { return f.emoji }

// With is a new face drawing like f, set up by opts: an engine face with
// fallbacks, twod.Regular().With(twod.Fallback(emoji)).
func (f *Face) With(opts ...FaceOption) *Face {
	out := &Face{face: f.draw(), fallbacks: f.fallbacks, emoji: f.emoji}
	return out.setUp(opts)
}

func (f *Face) setUp(opts []FaceOption) *Face {
	for _, o := range opts {
		o(f)
	}
	if len(f.fallbacks) == 0 && f.emoji == nil {
		return f
	}
	var fallbacks []*draw.Face
	for _, fb := range f.fallbacks {
		fallbacks = append(fallbacks, fb.draw())
	}
	var emoji draw.Emoji
	if f.emoji != nil {
		emoji = pictures{f.emoji}
	}
	f.face = f.face.With(fallbacks, emoji)
	return f
}

//nolint:gochecknoglobals // the engine's faces, one value each that Regular()… return
var (
	regular   = &Face{face: draw.Regular}
	bold      = &Face{face: draw.Bold}
	italic    = &Face{face: draw.Italic}
	smallCaps = &Face{face: draw.SmallCaps}
	mono      = &Face{face: draw.Mono}
)

// The engine's faces: Go's and SourceCodePro for Mono.
func Regular() *Face   { return regular }
func Bold() *Face      { return bold }
func Italic() *Face    { return italic }
func SmallCaps() *Face { return smallCaps }
func Mono() *Face      { return mono }

// NewFace parses ttf, the bytes of a TrueType font file. TrueType is the
// one format: OpenType fonts with CFF outlines (.otf), collections and
// web fonts are refused.
func NewFace(ttf []byte, opts ...FaceOption) (*Face, error) {
	f, err := parseFace("font", ttf)
	if err != nil {
		return nil, err
	}
	return f.setUp(opts), nil
}

// LoadFace reads the TrueType font file name, a .ttf, in fsys, such as an
// embed.FS.
func LoadFace(fsys fs.FS, name string, opts ...FaceOption) (*Face, error) {
	if !strings.EqualFold(path.Ext(name), ".ttf") {
		return nil, fmt.Errorf("twod: %s: fonts are TrueType files, named .ttf", name)
	}
	ttf, err := fs.ReadFile(fsys, name)
	if err != nil {
		return nil, fmt.Errorf("twod: %w", err)
	}
	f, err := parseFace(name, ttf)
	if err != nil {
		return nil, err
	}
	return f.setUp(opts), nil
}

func parseFace(name string, ttf []byte) (*Face, error) {
	if len(ttf) < 4 {
		return nil, fmt.Errorf("twod: %s is not a TrueType font file", name)
	}
	switch string(ttf[:4]) {
	case "\x00\x01\x00\x00", "true":
	case "OTTO":
		return nil, fmt.Errorf(
			"twod: %s is an OpenType font with CFF outlines; only TrueType (.ttf) fonts are supported",
			name,
		)
	case "ttcf":
		return nil, fmt.Errorf("twod: %s is a font collection; only single TrueType (.ttf) fonts are supported", name)
	case "wOFF", "wOF2":
		return nil, fmt.Errorf("twod: %s is a web font; only TrueType (.ttf) fonts are supported", name)
	default:
		return nil, fmt.Errorf("twod: %s is not a TrueType font file", name)
	}
	f, err := draw.ParseFace(name, ttf)
	if err != nil {
		return nil, err
	}
	return &Face{face: f}, nil
}

func (f *Face) draw() *draw.Face {
	if f == nil {
		return draw.Regular
	}
	return f.face
}

// Measure is the size of text, one or more lines, in this face at size
// points.
func (f *Face) Measure(text string, size float32) geom.Vec2 {
	return draw.Style{Face: f.draw(), Size: size}.Font().Measure(text)
}
