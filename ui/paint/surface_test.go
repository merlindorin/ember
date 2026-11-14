package paint //nolint:testpackage // white-box: builds box trees by hand and reads the painter's own state

import (
	"fmt"
	"strings"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/style"
)

var _ Surface = (*twod.Canvas)(nil)

// op is one call a recorder saw, in the root's space.
type op struct {
	Kind   string
	Rect   geom.Rect
	Radius float32
	Width  float32
	Color  color.Color
	Text   string
	At     geom.Vec2
	Clip   geom.Rect
	Xform  geom.Transform2D
	// Baseline is a text's: At is its baseline, not its top.
	Baseline bool
	// Radii are a per-corner shape's.
	Radii [4]float32
	// Points and Colors are a coloured mesh's, and Alpha each colour's.
	Points []geom.Vec2
	Colors []color.Color
	Alpha  []float32
	// Tex is an image's or a nine-patch's texture, and UV an image's
	// region of it.
	Tex *twod.Texture
	UV  geom.Rect
}

func (o op) String() string {
	switch o.Kind {
	case "text":
		return fmt.Sprintf("text %q at %v", o.Text, o.At)
	case "push", "pop":
		return fmt.Sprintf("%s %v", o.Kind, o.Rect)
	}
	return fmt.Sprintf("%s %v r%g w%g", o.Kind, o.Rect, o.Radius, o.Width)
}

// recorder is a Surface that writes down what is drawn.
type recorder struct {
	ops   []op
	xf    geom.Transform2D
	clips []geom.Rect
}

func newRecorder() *recorder { return &recorder{xf: geom.Identity2D()} }

func (r *recorder) clip() geom.Rect {
	if len(r.clips) == 0 {
		return geom.Rect{}
	}
	return r.clips[len(r.clips)-1]
}

func (r *recorder) add(o op) {
	o.Clip, o.Xform = r.clip(), r.xf
	r.ops = append(r.ops, o)
}

func (r *recorder) Rect(b geom.Rect, c color.Color) { r.add(op{Kind: "rect", Rect: b, Color: c}) }
func (r *recorder) RoundRect(b geom.Rect, radius float32, c color.Color) {
	r.add(op{Kind: "round", Rect: b, Radius: radius, Color: c})
}
func (r *recorder) RoundRectCorners(b geom.Rect, radii [4]float32, c color.Color) {
	r.add(op{Kind: "round4", Rect: b, Radii: radii, Color: c})
}
func (r *recorder) BorderCorners(b geom.Rect, radii [4]float32, w float32, c color.Color) {
	r.add(op{Kind: "border4", Rect: b, Radii: radii, Width: w, Color: c})
}
func (r *recorder) Border(b geom.Rect, radius, w float32, c color.Color) {
	r.add(op{Kind: "border", Rect: b, Radius: radius, Width: w, Color: c})
}
func (r *recorder) Glow(b geom.Rect, radius, spread float32, c color.Color) {
	r.add(op{Kind: "glow", Rect: b, Radius: radius, Width: spread, Color: c})
}
func (r *recorder) Line(a, b geom.Vec2, w float32, c color.Color) {
	r.add(op{Kind: "line", At: a, Rect: geom.Rect{X: b.X, Y: b.Y}, Width: w, Color: c})
}
func (r *recorder) Circle(at geom.Vec2, radius float32, c color.Color) {
	r.add(op{Kind: "circle", At: at, Radius: radius, Color: c})
}
func (r *recorder) Triangles(points []geom.Vec2, _ []int, c color.Color) {
	lo, hi := points[0], points[0]
	for _, p := range points {
		lo, hi = geom.V2(min(lo.X, p.X), min(lo.Y, p.Y)), geom.V2(max(hi.X, p.X), max(hi.Y, p.Y))
	}
	r.add(op{Kind: "triangles", Rect: geom.Rect{X: lo.X, Y: lo.Y, W: hi.X - lo.X, H: hi.Y - lo.Y}, Color: c})
}
func (r *recorder) TrianglesColored(points []geom.Vec2, colors []color.Color, indices []int) {
	lo, hi := points[0], points[0]
	for _, p := range points {
		lo, hi = geom.V2(min(lo.X, p.X), min(lo.Y, p.Y)), geom.V2(max(hi.X, p.X), max(hi.Y, p.Y))
	}
	alpha := make([]float32, len(colors))
	for i, c := range colors {
		alpha[i] = c.A
	}
	r.add(op{Kind: "gradient", Rect: geom.Rect{X: lo.X, Y: lo.Y, W: hi.X - lo.X, H: hi.Y - lo.Y}, Color: colors[0],
		Width:  float32(len(indices) / 3),
		Points: append([]geom.Vec2(nil), points...), Colors: append([]color.Color(nil), colors...), Alpha: alpha})
}
func (r *recorder) Text(at geom.Vec2, text string, opts ...twod.TextOption) geom.Vec2 {
	t := twod.NewText(text, opts...)
	r.add(op{Kind: "text", At: at, Text: text, Color: t.Color(), Baseline: t.Baseline()})
	return geom.Vec2{}
}
func (r *recorder) Image(b geom.Rect, tex *twod.Texture, uv geom.Rect, tint color.Color) {
	r.add(op{Kind: "image", Rect: b, Color: tint, Tex: tex, UV: uv})
}
func (r *recorder) NinePatch(b geom.Rect, tex *twod.Texture, left, _, _, _ float32, tint color.Color) {
	r.add(op{Kind: "nine", Rect: b, Width: left, Color: tint, Tex: tex})
}
func (r *recorder) PushClip(b geom.Rect) {
	r.clips = append(r.clips, b)
	r.add(op{Kind: "push", Rect: b})
}
func (r *recorder) PushClipRounded(b geom.Rect, radii [4]float32) {
	r.clips = append(r.clips, b)
	r.add(op{Kind: "push-round", Rect: b, Radii: radii})
}
func (r *recorder) PopClip() {
	r.add(op{Kind: "pop"})
	r.clips = r.clips[:len(r.clips)-1]
}
func (r *recorder) Transform() geom.Transform2D     { return r.xf }
func (r *recorder) SetTransform(t geom.Transform2D) { r.xf = t }

// of keeps the ops of the kinds given.
func (r *recorder) of(kinds ...string) []op {
	var out []op
	for _, o := range r.ops {
		for _, k := range kinds {
			if o.Kind == k {
				out = append(out, o)
			}
		}
	}
	return out
}

// order is the colours of the fills, as names, in the order drawn.
func (r *recorder) order(names map[color.Color]string) string {
	var out []string
	for _, o := range r.ops {
		if n, ok := names[o.Color]; ok && (o.Kind == "rect" || o.Kind == "round") {
			out = append(out, n)
		}
	}
	return strings.Join(out, " ")
}

//nolint:gochecknoglobals // the tests' fixed values; Go has no constant colours or pictures
var (
	red   = color.RGB(1, 0, 0)
	green = color.RGB(0, 1, 0)
	blue  = color.RGB(0, 0, 1)
	white = color.RGB(1, 1, 1)
	black = color.RGB(0, 0, 0)
)

// el is an element box at pos of size, styled by edit.
func el(x, y, w, h float32, edit func(s *style.Computed), children ...*box.Box) *box.Box {
	s := style.Initial()
	s.Display = style.Block
	if edit != nil {
		edit(&s)
	}
	b := &box.Box{Style: &s, Pos: geom.V2(x, y), Size: geom.V2(w, h)}
	b.Border = box.Edges{Top: s.Border.Top, Right: s.Border.Right, Bottom: s.Border.Bottom, Left: s.Border.Left}
	b.Append(children...)
	return b
}

func bg(c color.Color) func(s *style.Computed) {
	return func(s *style.Computed) { s.Background = c }
}

// text is a text run of one line at pos in its parent.
func text(x, y float32, s string, parent *style.Computed) *box.Box {
	b := &box.Box{Kind: box.TextRun, Style: parent, Pos: geom.V2(x, y), Size: geom.V2(float32(len(s))*8, 20)}
	b.Lines = []box.Fragment{{Text: s, Rect: geom.Rect{W: float32(len(s)) * 8, H: 20}, Baseline: 15}}
	return b
}

func tree(root *box.Box) *box.Tree { return &box.Tree{Root: root} }
