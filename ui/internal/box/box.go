// Package box is the box tree: what layout produces from the styled node
// tree and paint, hit testing and Node.Box read.
package box

import (
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/ui/internal/dom"
	"github.com/merlindorin/ember/ui/internal/style"
)

// Kind is what generated a box.
type Kind uint8

const (
	// Element is an element's principal box.
	Element Kind = iota
	// AnonymousBlock wraps inline content beside blocks.
	AnonymousBlock
	// TextRun is a text node's box; its Lines hold the text.
	TextRun
	// Marker is a summary's disclosure triangle, an inline box at the start
	// of its first line as wide as Chrome's ::marker; Node is the summary.
	Marker
)

// Edges is a resolved margin, border or padding, in points.
type Edges struct{ Top, Right, Bottom, Left float32 }

func (e Edges) Horizontal() float32 { return e.Left + e.Right }
func (e Edges) Vertical() float32   { return e.Top + e.Bottom }

// Fragment is a piece of one line of text, in the box's space.
type Fragment struct {
	Text string
	// Rect is the line box's part the text covers; Baseline is its
	// baseline's y.
	Rect     geom.Rect
	Baseline float32
	// Ellipsis is set when Text was cut and ends with "…".
	Ellipsis bool
}

// Box is one box. Positions are relative to the parent box's border box,
// before scrolling, as Taffy's are.
type Box struct {
	Kind Kind
	// Node is the element or text node, nil for anonymous boxes.
	Node *dom.Node
	// Style is Node's computed style. An anonymous box's is Initial()
	// with its parent's Text, so nothing paints the parent's background
	// or border twice.
	Style    *style.Computed
	Parent   *Box
	Children []*Box
	// Pos is the border box's top-left in the parent's border box; Size
	// is the border box's size.
	Pos     geom.Vec2
	Size    geom.Vec2
	Margin  Edges
	Border  Edges
	Padding Edges
	// Content is the size of the scrollable overflow, and Scroll the
	// node's dom Scroll offset clamped to it by layout.
	Content geom.Vec2
	Scroll  geom.Vec2
	// Scrollbar is the room the box's scrollbars take along its right and bottom
	// padding edges: overflow: scroll's, and auto's once the content overflows.
	Scrollbar geom.Vec2
	// Sticky is how far position: sticky moved the box from its place in the
	// flow; Pos includes it.
	Sticky geom.Vec2
	// Escapes is set on an absolutely or fixed positioned box and on each
	// box above it up to its containing block: what it draws lies where
	// that containing block placed it, outside these boxes' own overflow.
	Escapes bool
	// Lines is a TextRun's text, laid out.
	Lines []Fragment
	// Fragments is an inline element's border box on each line it spans,
	// in the box's space: the first holds its left edge and the last its
	// right, as Chrome's getClientRects.
	Fragments []geom.Rect
	// Baseline is the first baseline's y, for alignment.
	Baseline float32
	// Wrap is the room a table's captions take inside its box, above and below
	// its grid: its background, border and shadows frame the box less it, as
	// Chrome's table wrapper leaves the captions out.
	Wrap Edges
}

// Append adds children to b.
func (b *Box) Append(children ...*Box) {
	for _, c := range children {
		c.Parent = b
	}
	b.Children = append(b.Children, children...)
}

// Rect is the border box in the root's space, scrolling applied.
func (b *Box) Rect() geom.Rect {
	p := b.Pos
	for a := b.Parent; a != nil; a = a.Parent {
		p = p.Add(a.Pos).Sub(a.Scroll)
	}
	return geom.Rect{X: p.X, Y: p.Y, W: b.Size.X, H: b.Size.Y}
}

// ContentRect is the content box, relative to the border box.
func (b *Box) ContentRect() geom.Rect {
	return geom.Rect{
		X: b.Border.Left + b.Padding.Left,
		Y: b.Border.Top + b.Padding.Top,
		W: b.Size.X - b.Border.Horizontal() - b.Padding.Horizontal(),
		H: b.Size.Y - b.Border.Vertical() - b.Padding.Vertical(),
	}
}

// Tree is a laid-out tree and each element's principal box.
type Tree struct {
	Root *Box
	// Of is each node's box. A retained layout updates one map in place, so it holds the
	// latest layout's alone: Written are the nodes that layout gave a box, or whose kept
	// box isn't where their Placed points, and Dropped those it took out of it.
	Of      map[*dom.Node]*Box
	Written []*dom.Node
	Dropped []*dom.Node
	// Live are the scroll containers and sticky boxes, parents first: what can change
	// without a layout; Applied is each one's scroll and sticky offsets as the last layout
	// or refresh left them.
	Live    []*Box
	Applied [][2]geom.Vec2
}

// Metrics are a font's vertical metrics at a size, in points: CSS's
// normal line height is Ascent+Descent+LineGap.
type Metrics struct{ Ascent, Descent, LineGap float32 }

// Fonts measures text for layout, which knows no faces: the root package
// implements it over twod's.
type Fonts interface {
	Metrics(f style.Font) Metrics
	// Advance is the width of text on one line, letter spacing apart.
	Advance(f style.Font, text string) float32
}

// FallbackFonts is Fonts that draws some characters from other faces:
// FallbackMetrics are the largest metrics of the faces besides f's that
// draw text, false when f's own face draws it all. Chrome stretches a
// line to fit them, not the inline box holding them.
type FallbackFonts interface {
	FallbackMetrics(f style.Font, text string) (Metrics, bool)
}

// Images gives layout the natural size of a replaced element's picture,
// such as an img's src, in points; ok false when there is none.
type Images interface {
	ImageSize(url string) (size geom.Vec2, ok bool)
}
