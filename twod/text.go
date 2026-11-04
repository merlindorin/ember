package twod

import (
	"strings"
	"sync"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/internal/draw"
	"github.com/merlindorin/ember/internal/emojinames"
	"github.com/merlindorin/ember/node"
)

//nolint:gochecknoinits // registers the package's node types on import
func init() {
	node.Register(node.Type{New: func() node.Noded { return NewText("") }})
	node.Register(node.Type{New: func() node.Noded { return NewNinePatch(nil) }})
}

// Text draws a string from its position, its top-left corner, wrapped to
// a width when asked.
type Text struct {
	Item
	text   string      `node:"prop"`
	size   float32     `node:"prop"`
	color  color.Color `node:"prop"`
	wrap   float32     `node:"prop"`
	pivot  geom.Vec2   `node:"prop"`
	shadow color.Color `node:"prop"`
	expand bool        `node:"prop"`
	// tracking is room added after each grapheme cluster, in points.
	tracking float32 `node:"prop"`
	// baseline takes the text's origin as its first baseline.
	baseline bool `node:"prop"`
	// pixelated draws the glyphs unsmoothed (Pixelated).
	pixelated bool `node:"prop"`
	face      *Face
	// expanded is text with its shortcodes replaced, for expandedFrom.
	expanded, expandedFrom string
}

// NewText returns white 16-point text.
func NewText(text string, opts ...TextOption) *Text {
	t := Build(
		&Text{Item: Base(), text: text, size: 16, color: color.White, face: Regular(), pivot: geom.V2(0.5, 0.5)},
		"text",
	)
	for _, o := range opts {
		o.text(t)
	}
	return t
}

// Typeface draws a text with face.
func Typeface(face *Face) TextOption {
	return textOption(func(t *Text) {
		if face != nil {
			t.face = face
		}
	})
}

// FontSize is a text's font size in points.
func FontSize(size float32) TextOption {
	return textOption(func(t *Text) {
		if size > 0 {
			t.size = size
		}
	})
}

// Tracking adds points of room after each character (each grapheme
// cluster), as letter spacing; negative tightens the text.
func Tracking(points float32) TextOption { return textOption(func(t *Text) { t.tracking = points }) }

// Baseline places a text by its first baseline: the origin, or a Canvas
// Text's at, is where the baseline starts, not the text's top-left; a
// pivot then only moves it across. For text placed by a layout's metrics.
func Baseline() TextOption { return textOption(func(t *Text) { t.baseline = true }) }

func (t *Text) Baseline() bool             { return t.baseline }
func (t *Text) SetBaseline(on bool)        { t.baseline = on }
func (t *Text) Tracking() float32          { return t.tracking }
func (t *Text) SetTracking(points float32) { t.tracking = points }

// Wrap breaks a text into lines no wider than width points, between words.
func Wrap(width float32) TextOption { return textOption(func(t *Text) { t.wrap = width }) }

// ExpandShortcodes draws a text's :name: shortcodes as their emoji, the
// names Slack and GitHub use (:fire:, :+1:, :wave::skin-tone-3:), and
// leaves unknown ones as written.
func ExpandShortcodes() TextOption { return textOption(func(t *Text) { t.expand = true }) }

// Shortcodes is s with its known :name: shortcodes replaced by their
// emoji, unknown ones left as written.
func Shortcodes(s string) string { return emojinames.Expand(s) }

func (t *Text) ExpandShortcodes() bool      { return t.expand }
func (t *Text) SetExpandShortcodes(on bool) { t.expand = on }

// shown is the text as drawn, shortcodes expanded when asked.
func (t *Text) shown(text string) string {
	if !t.expand {
		return text
	}
	if text != t.expandedFrom || t.expanded == "" {
		t.expanded, t.expandedFrom = emojinames.Expand(text), text
	}
	return t.expanded
}

func (t *Text) Text() string           { return t.text }
func (t *Text) SetText(s string)       { t.text = s }
func (t *Text) Color() color.Color     { return t.color }
func (t *Text) SetColor(c color.Color) { t.color = c }

func (t *Text) style() draw.Style {
	return draw.Style{
		Face:      t.face.draw(),
		Size:      t.size,
		Color:     t.color,
		Tracking:  t.tracking,
		Baseline:  t.baseline,
		Pixelated: t.pixelated,
	}
}

// Pixelated reports whether t's glyphs are drawn unsmoothed.
func (t *Text) Pixelated() bool { return t.pixelated }

// Lines are the text's lines once wrapped.
func (t *Text) Lines() []string {
	var out []string
	for _, para := range strings.Split(t.shown(t.text), "\n") {
		if t.wrap <= 0 {
			out = append(out, para)
			continue
		}
		line := ""
		for _, word := range strings.Fields(para) {
			next := word
			if line != "" {
				next = line + " " + word
			}
			if line != "" && t.style().Measure(next).X > t.wrap {
				out = append(out, line)
				next = word
			}
			line = next
		}
		out = append(out, line)
	}
	return out
}

// Measure is the text's size in points once wrapped.
func (t *Text) Measure() geom.Vec2 {
	var w float32
	lines := t.Lines()
	for _, line := range lines {
		w = max(w, t.style().Measure(line).X)
	}
	return geom.V2(w, float32(len(lines))*t.size*1.25)
}

func (t *Text) Draw(c *Canvas) {
	at := t.Measure()
	at = geom.V2(-at.X*t.pivot.X, -at.Y*t.pivot.Y)
	if t.baseline {
		at.Y = 0
	}
	for _, line := range t.Lines() {
		t.paint(c, at, line)
		at.Y += t.size * 1.25
	}
}

// paint draws one line of t from at, its top-left corner, over its
// shadow, and returns its size.
func (t *Text) paint(c *Canvas, at geom.Vec2, line string) geom.Vec2 {
	s := t.style()
	if t.shadow.A > 0 {
		shade := s
		shade.Color = t.shadow
		c.list.Text(at.Add(geom.V2(1, 1.5)), line, shade)
	}
	return c.list.Text(at, line, s)
}

// Text draws text from at, its top-left corner unless Pivot says
// otherwise, with the options of a Text entity but Wrap: white 16-point
// Regular by default. It returns the text's size, measured once while
// drawing unless the pivot needs it first.
func (u *Canvas) Text(at geom.Vec2, text string, opts ...TextOption) geom.Vec2 {
	t := u.scratchText(text, opts)
	text = t.shown(text)
	if t.pivot.X == 0 && (t.baseline || t.pivot.Y == 0) {
		return t.paint(u, at, text)
	}
	size := t.style().Measure(text)
	off := geom.V2(size.X*t.pivot.X, size.Y*t.pivot.Y)
	if t.baseline {
		off.Y = 0
	}
	t.paint(u, at.Sub(off), text)
	return size
}

// Measure is the size Text would draw text at with opts.
func (u *Canvas) Measure(text string, opts ...TextOption) geom.Vec2 {
	t := u.scratchText(text, opts)
	return t.style().Measure(t.shown(text))
}

// Measure is the size text takes drawn with opts, without a canvas: a
// layout's measure, the same as Canvas.Measure's.
func Measure(text string, opts ...TextOption) geom.Vec2 {
	measuring.Lock()
	defer measuring.Unlock()
	t := &measured
	*t = Text{text: text, size: 16, color: color.White, face: Regular()}
	for _, o := range opts {
		o.text(t)
	}
	return t.style().Measure(t.shown(text))
}

//nolint:gochecknoglobals // the one Text every Measure reuses, under its mutex
var (
	measured  Text
	measuring sync.Mutex
)

// scratchText is the canvas's one Text set up by opts, reused by every
// call so drawing text allocates nothing.
func (u *Canvas) scratchText(text string, opts []TextOption) *Text {
	if u.text == nil {
		u.text = &Text{}
	}
	t := u.text
	*t = Text{text: text, size: 16, color: color.White, face: Regular()}
	for _, o := range opts {
		o.text(t)
	}
	return t
}

// NinePatch stretches a texture to a size, keeping its borders: a frame or
// a speech bubble at any size.
type NinePatch struct {
	Item
	texture                  *Texture  `node:"prop"`
	left, top, right, bottom float32   `node:"prop"`
	size                     geom.Vec2 `node:"prop"`
	pivot                    geom.Vec2 `node:"prop"`
}

// NewNinePatch returns tex stretched to its Size, its Margins kept as
// they are.
func NewNinePatch(tex *Texture, opts ...NinePatchOption) *NinePatch {
	p := Build(&NinePatch{Item: Base(), texture: tex, size: tex.Size(), pivot: geom.V2(0.5, 0.5)}, "nine-patch")
	for _, o := range opts {
		o.ninePatch(p)
	}
	return p
}

// Margins are a nine-patch's left, top, right and bottom borders, in
// texture pixels, kept unstretched.
func Margins(left, top, right, bottom float32) NinePatchOption {
	return ninePatchOption(func(p *NinePatch) { p.SetMargins(left, top, right, bottom) })
}

func (p *NinePatch) Margins() (float32, float32, float32, float32) {
	return p.left, p.top, p.right, p.bottom
}

func (p *NinePatch) SetMargins(left, top, right, bottom float32) {
	p.left, p.top, p.right, p.bottom = left, top, right, bottom
}

func (p *NinePatch) Size() geom.Vec2     { return p.size }
func (p *NinePatch) SetSize(s geom.Vec2) { p.size = s }

func (p *NinePatch) Draw(c *Canvas) {
	if p.texture == nil || p.texture.Image == nil {
		return
	}
	c.NinePatch(
		geom.Rect{X: -p.size.X * p.pivot.X, Y: -p.size.Y * p.pivot.Y, W: p.size.X, H: p.size.Y},
		p.texture,
		p.left,
		p.top,
		p.right,
		p.bottom,
		color.White,
	)
}
