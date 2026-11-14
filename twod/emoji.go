package twod

import (
	"fmt"
	"io/fs"
	"path"
	"strconv"
	"strings"
	"unicode"

	"github.com/lantervale/ember/render"
)

// EmojiSet is colour emoji as pictures, by the grapheme cluster each
// draws: a face with Emoji(set) draws them inline in its text.
type EmojiSet struct {
	pictures map[string]*Texture
}

// LoadEmoji reads the PNGs of dir in fsys, each named by the codepoints
// of the emoji it draws in hexadecimal: Twemoji's "1f468-200d-1f469.png"
// or Noto's "emoji_u1f468_200d_1f469.png". Variation selectors (FE0F)
// are ignored, on the files' side and the text's.
func LoadEmoji(fsys fs.FS, dir string) (*EmojiSet, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, fmt.Errorf("twod: %w", err)
	}
	set := &EmojiSet{pictures: map[string]*Texture{}}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.EqualFold(path.Ext(name), ".png") {
			continue
		}
		key, ok := emojiKey(strings.TrimSuffix(name, path.Ext(name)))
		if !ok {
			continue
		}
		tex, loadErr := LoadTexture(fsys, path.Join(dir, name))
		if loadErr != nil {
			return nil, loadErr
		}
		set.pictures[key] = tex
	}
	return set, nil
}

// emojiKey is the codepoints a file name spells, without FE0F.
func emojiKey(name string) (string, bool) {
	name = strings.TrimPrefix(strings.ToLower(name), "emoji_u")
	var b strings.Builder
	for _, part := range strings.FieldsFunc(name, func(r rune) bool { return r == '-' || r == '_' }) {
		cp, err := strconv.ParseUint(part, 16, 32)
		if err != nil {
			return "", false
		}
		if cp > unicode.MaxRune {
			return "", false
		}
		if cp != 0xfe0f {
			b.WriteRune(rune(cp))
		}
	}
	return b.String(), b.Len() > 0
}

// Len is how many emoji the set draws.
func (s *EmojiSet) Len() int { return len(s.pictures) }

// Has reports whether the set draws cluster.
func (s *EmojiSet) Has(cluster string) bool {
	_, ok := s.picture(cluster)
	return ok
}

// pictures is a set as the renderer reads it.
type pictures struct{ set *EmojiSet }

func (p pictures) Picture(cluster string) (*render.Texture, bool) { return p.set.picture(cluster) }

func (s *EmojiSet) picture(cluster string) (*render.Texture, bool) {
	t, ok := s.pictures[strings.ReplaceAll(cluster, "️", "")]
	if !ok {
		return nil, false
	}
	return &t.Texture, true
}
