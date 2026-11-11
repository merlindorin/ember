package style

import (
	"github.com/merlindorin/ember/color"
)

// Computed is one node's computed style. The zero value is CSS's initial
// value for every property except where noted; css starts from Initial().
type Computed struct {
	Box
	Flex
	Grid
	Text
	Look
	// ColorInherited is set when color comes from the parent (inherit,
	// unset, currentcolor or no declaration), so a parent's colour
	// transition carries on to it.
	ColorInherited bool
	// CurrentColor marks the colours that are currentColor, so they follow
	// color, its transitions included.
	CurrentColor CurrentColorFields
	// Vars is the custom properties (--name) as CSS text, set and inherited
	// by css.
	Vars map[string]string
}

// Initial is CSS's initial values, with ui's box-sizing: border-box.
func Initial() Computed {
	return Computed{
		Box: Box{
			Display:   Inline,
			BoxSizing: BorderBox,
			MaxSize:   Size[Length]{Length{Kind: None}, Length{Kind: None}},
			Margin:    All(Px(0)),
			Padding:   All(Px(0)),
			ZAuto:     true,
		},
		Flex: Flex{
			Shrink: 1, AlignItems: AlignNormal, AlignContent: AlignNormal, JustifyContent: AlignNormal,
			AlignSelf: AlignAuto, JustifyItems: AlignNormal, JustifySelf: AlignAuto,
		},
		Text: Text{
			Font:       Font{Family: []string{"sans-serif"}, Size: 16, Weight: 400},
			LineHeight: LineHeight{Normal: true},
			Color:      color.Color{A: 1},
			TextAlign:  Start,
		},
		Look: Look{Opacity: 1, Visible: true, TransformOrigin: Size[Length]{Percent(50), Percent(50)}},
	}
}

// Display is the outer and inner display type, as one keyword.
type Display uint8

const (
	Inline Display = iota
	Block
	InlineBlock
	FlexDisplay
	InlineFlex
	GridDisplay
	InlineGrid
	DisplayNone
	Contents
)

type BoxSizing uint8

const (
	BorderBox BoxSizing = iota
	ContentBox
)

type Position uint8

const (
	Static Position = iota
	Relative
	Absolute
	FixedPosition
	Sticky
)

type Overflow uint8

const (
	Visible Overflow = iota
	Hidden
	Clip
	Scroll
	OverflowAuto
)

// Box holds what places and sizes the node's box.
type Box struct {
	Display Display
	// ListItem is display: list-item, laid out as Block, with a marker
	// where the element has one (a summary's).
	ListItem  bool
	BoxSizing BoxSizing
	Position  Position
	// Inset is top, right, bottom, left; Auto where unset.
	Inset   Sides[Length]
	Size    Size[Length]
	MinSize Size[Length]
	MaxSize Size[Length]
	// AspectRatio is width over height, 0 for auto.
	AspectRatio float32
	Margin      Sides[Length]
	Padding     Sides[Length]
	// Border is the used border widths: 0 where border-style is none.
	Border   Sides[float32]
	Overflow Size[Overflow]
	// ScrollbarWidth is the gutter, in points, that overflow: scroll reserves
	// across the scrolling axis (scrollbar-width: none is 0).
	ScrollbarWidth float32
	// ScrollbarNone is scrollbar-width: none: no gutter and no bars, the
	// box still scrolling by the wheel.
	ScrollbarNone bool
	// ZIndex applies when ZAuto is false.
	ZIndex int
	ZAuto  bool
}

type FlexDirection uint8

const (
	Row FlexDirection = iota
	RowReverse
	Column
	ColumnReverse
)

type FlexWrap uint8

const (
	NoWrap FlexWrap = iota
	Wrap
	WrapReverse
)

// Align is a value of align-*, justify-* and place-*.
type Align uint8

const (
	AlignNormal Align = iota
	AlignAuto
	AlignStretch
	AlignStart
	AlignEnd
	AlignFlexStart
	AlignFlexEnd
	AlignCenter
	AlignBaseline
	AlignSpaceBetween
	AlignSpaceAround
	AlignSpaceEvenly
)

// Flex holds the flex and box-alignment properties.
type Flex struct {
	Direction FlexDirection
	Wrap      FlexWrap
	Grow      float32
	Shrink    float32
	Basis     Length
	// Gap is column-gap (Width) and row-gap (Height).
	Gap            Size[Length]
	AlignItems     Align
	AlignSelf      Align
	AlignContent   Align
	JustifyContent Align
	JustifyItems   Align
	JustifySelf    Align
	// Order is the flex and grid order.
	Order int
}

// Track is one grid track size: minmax(Min, Max), or a single size in
// both.
type Track struct{ Min, Max Length }

// Repeat counts of a TrackList entry.
const (
	AutoFill = -1
	AutoFit  = -2
)

// TrackList entry: Tracks repeated Count times (1 for no repeat()), or
// AutoFill/AutoFit.
type TrackRepeat struct {
	Count  int
	Tracks []Track
}

// GridLine is one end of a placement: auto, a line number (negative from
// the end), or a span.
type GridLine struct {
	Line int
	Span int
}

// IsAuto reports a line that is neither a number nor a span.
func (g GridLine) IsAuto() bool { return g.Line == 0 && g.Span == 0 }

type GridFlow uint8

const (
	FlowRow GridFlow = iota
	FlowColumn
	FlowRowDense
	FlowColumnDense
)

type Grid struct {
	Columns     []TrackRepeat
	Rows        []TrackRepeat
	AutoColumns []Track
	AutoRows    []Track
	Flow        GridFlow
	// Column and Row are grid-column and grid-row, Start/End.
	Column [2]GridLine
	Row    [2]GridLine
}

// Font is what picks and sizes a face; the root package maps it to a
// twod.Face.
type Font struct {
	// Family is font-family's list, generic names last.
	Family []string
	Size   float32
	Weight int
	Italic bool
}

// LineHeight is normal, a factor of the font size, or points.
type LineHeight struct {
	Normal bool
	Factor float32
	Px     float32
}

// CaptionSide is caption-side: a table's caption above it or below.
type CaptionSide uint8

const (
	CaptionTop CaptionSide = iota
	CaptionBottom
)

// TableLayout is table-layout: columns from every cell's content, or from
// the first row's and the table's width alone.
type TableLayout uint8

const (
	TableAuto TableLayout = iota
	TableFixed
)

// TextTransform is text-transform: how text shows cased, its source kept.
type TextTransform uint8

const (
	NoTransform TextTransform = iota
	Uppercase
	Lowercase
	Capitalize
)

type TextAlign uint8

const (
	Start TextAlign = iota
	End
	Left
	Right
	Center
	Justify
)

type WhiteSpace uint8

const (
	Normal WhiteSpace = iota
	NoWrapSpace
	Pre
	PreWrap
	PreLine
)

type Text struct {
	Font          Font
	LineHeight    LineHeight
	LetterSpacing float32
	Color         color.Color
	TextAlign     TextAlign
	WhiteSpace    WhiteSpace
	// PlaceholderColor is the color of the element's ::placeholder, set
	// when HasPlaceholder is (a ::placeholder rule matched).
	PlaceholderColor color.Color
	HasPlaceholder   bool
	// Backdrop is a dialog's ::backdrop background-color, drawn under it
	// while it is modal; transparent draws none.
	Backdrop color.Color
	// SelectionColor is the ::selection background-color, set when
	// HasSelection is (a ::selection rule matched).
	SelectionColor color.Color
	HasSelection   bool
	// TextShadows is text-shadow: Spread and Inset are unused.
	TextShadows []Shadow
	// AccentColor and CaretColor are accent-color and caret-color,
	// resolved: their element's color when AccentAuto or CaretAuto (auto
	// or currentColor, kept as such when inherited).
	AccentColor, CaretColor color.Color
	AccentAuto, CaretAuto   bool
	// ListStyle is list-style-type: a summary's disclosure triangle, or
	// none.
	ListStyle ListStyle
	// BorderSpacing is a table's border-spacing, across and down, in points.
	BorderSpacing [2]float32
	CaptionSide   CaptionSide
	TableLayout   TableLayout
	TextTransform TextTransform
	// PixelText draws the text's glyphs unsmoothed, as
	// -webkit-font-smoothing: none does in Chrome: for pixel fonts.
	PixelText bool
	// Ellipsis is text-overflow: ellipsis.
	Ellipsis bool
	// EllipsisStart cuts the start of the text rather than its end, as a
	// path keeps its last folders: class truncate-start, an ember
	// extension.
	EllipsisStart bool
	// BreakWord is overflow-wrap: anywhere or break-word.
	BreakWord bool
	// BreakAnywhere is overflow-wrap: anywhere, whose breaks also count
	// in min-content widths.
	BreakAnywhere bool
}

type BorderStyle uint8

const (
	BorderNone BorderStyle = iota
	BorderSolid
	BorderDashed
	BorderDotted
	// BorderInset and BorderOutset draw a side in its colour or a darker
	// shade of it, as Chrome does: inset darker on top and left, outset
	// on bottom and right.
	BorderInset
	BorderOutset
)

// Shadow is one box-shadow.
type Shadow struct {
	X, Y, Blur, Spread float32
	Color              color.Color
	Inset              bool
	// CurrentColor is set when Color is currentColor: the colour of the
	// element it is on, inherited as such.
	CurrentColor bool
}

// Image is a background-image or border-image source by name, with
// border-image's slice for nine-patches, or a background's
// linear-gradient().
type Image struct {
	Source   string
	Slice    Sides[float32]
	Gradient *Gradient
}

// Gradient is a linear-gradient(), a conic-gradient() or a
// radial-gradient(): its kind, its geometry and its colour stops.
type Gradient struct {
	Kind GradientKind
	// Angle is a linear gradient's line, in degrees, 0 up and 90 to the
	// right, while Corner is zero; a conic gradient's from angle.
	Angle float32
	// Corner is a linear gradient's "to" corner, -1 or 1 across and down,
	// whose angle follows the box's shape.
	Corner [2]int8
	// At is a conic or radial gradient's centre, 50% 50% by default.
	At BackgroundPosition
	// Circle and Size are a radial gradient's shape (an ellipse unless
	// Circle) and its extent.
	Circle bool
	Size   RadialSize
	// Stops are the colour stops; a conic gradient's positions are
	// percentages of a turn, its degrees made so.
	Stops []Stop
}

// GradientKind is which of the gradient functions a Gradient is.
type GradientKind uint8

const (
	LinearGradient GradientKind = iota
	ConicGradient
	RadialGradient
)

// RadialSize is a radial gradient's extent keyword.
type RadialSize uint8

const (
	FarthestCorner RadialSize = iota
	ClosestSide
	ClosestCorner
	FarthestSide
)

// Stop is a colour stop: its colour and, when Placed, where it stands on
// the gradient line.
type Stop struct {
	Color  color.Color
	At     Length
	Placed bool
}

// Transition is one transition: a property or "all", in seconds.
type Transition struct {
	Property        string
	Duration, Delay float32
	Easing          string
}

type Look struct {
	Background      color.Color
	BackgroundImage Image
	// BackgroundSize is background-size in points, across and down; 0 is
	// auto: the image's own size, or its width's or height's share of it.
	BackgroundSize [2]float32
	// BackgroundNoRepeat is background-repeat: no-repeat; the image tiles
	// otherwise.
	BackgroundNoRepeat bool
	// BackgroundPosition is where the image's top-left sits in the padding
	// box.
	BackgroundPosition BackgroundPosition
	// ObjectFit and ObjectPosition place a replaced element's picture
	// (an img's) in its content box.
	// ImageRendering is image-rendering: how pictures scale.
	ImageRendering ImageRendering
	ObjectFit      ObjectFit
	ObjectPosition BackgroundPosition
	BorderColor    Sides[color.Color]
	BorderStyle    Sides[BorderStyle]
	BorderImage    Image
	// BorderImageRepeat is border-image-repeat across and down.
	BorderImageRepeat [2]ImageRepeat
	// Radius is each corner's horizontal and vertical radius.
	Radius        Corners[Size[Length]]
	Shadows       []Shadow
	OutlineWidth  float32
	OutlineStyle  BorderStyle
	OutlineColor  color.Color
	OutlineOffset float32
	Opacity       float32
	// Visible is visibility: visible.
	Visible bool
	// PointerNone is pointer-events: none: hit testing passes through the
	// box to what is under it.
	PointerNone bool
	// Cursor is the cursor keyword, "" for auto.
	Cursor string
	// Transform is transform's functions in order, about
	// TransformOrigin (percentages of the border box).
	Transform       []TransformOp
	TransformOrigin Size[Length]
	Transitions     []Transition
}

type TransformKind uint8

const (
	Translate TransformKind = iota
	ScaleOp
	Rotate
	Skew
	Matrix
)

// TransformOp is one transform function: translate(X, Y) with lengths of
// the border box, scale(A[0], A[1]), rotate(A[0] radians), skew(A[0],
// A[1] radians) or matrix(A…).
type TransformOp struct {
	Kind TransformKind
	X, Y Length
	A    [6]float32
}

// CurrentColorFields is a set of colours given as currentColor.
type CurrentColorFields uint8

const (
	BorderTopCurrent CurrentColorFields = 1 << iota
	BorderRightCurrent
	BorderBottomCurrent
	BorderLeftCurrent
	OutlineCurrent
)

// FollowColor sets every colour marked currentColor to c.
func (s *Computed) FollowColor(c color.Color) {
	for f, p := range map[CurrentColorFields]*color.Color{
		BorderTopCurrent: &s.BorderColor.Top, BorderRightCurrent: &s.BorderColor.Right,
		BorderBottomCurrent: &s.BorderColor.Bottom, BorderLeftCurrent: &s.BorderColor.Left,
		OutlineCurrent: &s.OutlineColor,
	} {
		if s.CurrentColor&f != 0 {
			*p = c
		}
	}
}

// ListStyle is a list-style-type: ui draws only details' disclosure
// triangles. ListDisc is CSS's initial value, which draws nothing in ui.
type ListStyle uint8

const (
	ListDisc ListStyle = iota
	ListNone
	ListDisclosureClosed
	ListDisclosureOpen
)

// BackgroundPosition is background-position: X from the left edge, or from
// the right one when FromRight, and Y from the top, or the bottom when
// FromBottom. A percentage places the image's point at the box's point
// (center is 50%), as CSS does.
type BackgroundPosition struct {
	X, Y                  Length
	FromRight, FromBottom bool
}

// ObjectFit is object-fit: how an img's picture fills its content box.
type ObjectFit uint8

const (
	// FitFill stretches the picture to the box.
	FitFill ObjectFit = iota
	// FitContain scales it to fit inside, keeping its aspect.
	FitContain
	// FitCover scales it to cover the box, keeping its aspect, cut to it.
	FitCover
	// FitNone keeps its size.
	FitNone
	// FitScaleDown is FitNone or FitContain, whichever is smaller.
	FitScaleDown
)

// ImageRepeat is border-image-repeat along one axis: how a border image's
// edges and centre fill their length.
type ImageRepeat uint8

const (
	// RepeatStretch stretches the slice over the whole length.
	RepeatStretch ImageRepeat = iota
	// RepeatTile tiles it at its size, centred, cut at both ends.
	RepeatTile
	// RepeatRound tiles it a whole number of times, scaled to fit.
	RepeatRound
	// RepeatSpace tiles it a whole number of times at its size, the
	// leftover shared out as equal gaps.
	RepeatSpace
)

// ImageRendering is image-rendering.
type ImageRendering uint8

const (
	// RenderAuto smooths a scaled picture.
	RenderAuto ImageRendering = iota
	// RenderPixelated scales it by its nearest pixels.
	RenderPixelated
)
