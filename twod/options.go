package twod

import (
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
)

// The options shared by every 2D entity are Options; each kind of entity
// also takes options of its own, typed so that one meant for another kind
// does not compile.
type (
	SpriteOption     interface{ sprite(*Sprite) }
	LineOption       interface{ line(*Line) }
	PolygonOption    interface{ polygon(*Polygon) }
	TextOption       interface{ text(*Text) }
	NinePatchOption  interface{ ninePatch(*NinePatch) }
	LayerOption      interface{ layer(*Layer) }
	CameraOption     interface{ camera(*Camera) }
	BitmapTextOption interface{ bitmapText(*BitmapText) }
)

func (o Option) sprite(s *Sprite)         { o(s) }
func (o Option) line(l *Line)             { o(l) }
func (o Option) polygon(p *Polygon)       { o(p) }
func (o Option) text(t *Text)             { o(t) }
func (o Option) ninePatch(p *NinePatch)   { o(p) }
func (o Option) layer(l *Layer)           { o(l) }
func (o Option) camera(c *Camera)         { o(c) }
func (o Option) bitmapText(t *BitmapText) { o(t) }

type (
	spriteOption     func(*Sprite)
	lineOption       func(*Line)
	textOption       func(*Text)
	ninePatchOption  func(*NinePatch)
	layerOption      func(*Layer)
	cameraOption     func(*Camera)
	bitmapTextOption func(*BitmapText)
)

func (o spriteOption) sprite(s *Sprite)             { o(s) }
func (o lineOption) line(l *Line)                   { o(l) }
func (o textOption) text(t *Text)                   { o(t) }
func (o ninePatchOption) ninePatch(p *NinePatch)    { o(p) }
func (o layerOption) layer(l *Layer)                { o(l) }
func (o cameraOption) camera(c *Camera)             { o(c) }
func (o bitmapTextOption) bitmapText(t *BitmapText) { o(t) }

// PivotOption sets which point of a sprite, a nine-patch, a text or a
// bitmap text sits on its position.
type PivotOption interface {
	SpriteOption
	NinePatchOption
	TextOption
	BitmapTextOption
}

type pivotOption geom.Vec2

func (o pivotOption) sprite(s *Sprite)         { s.pivot = geom.Vec2(o) }
func (o pivotOption) ninePatch(p *NinePatch)   { p.pivot = geom.Vec2(o) }
func (o pivotOption) text(t *Text)             { t.pivot = geom.Vec2(o) }
func (o pivotOption) bitmapText(t *BitmapText) { t.pivot = geom.Vec2(o) }

// Pivot is the point of a sprite, a nine-patch or a text that At places,
// as a fraction of its size: 0.5, 0.5 (the default) is its centre, 0, 0 its
// top-left corner. Lines and polygons have no size: At is the origin of
// their points.
func Pivot(x, y float32) PivotOption { return pivotOption(geom.V2(x, y)) }

// SizeOption sets the box a sprite or a nine-patch fills, in points.
type SizeOption interface {
	SpriteOption
	NinePatchOption
}

type sizeOption geom.Vec2

func (o sizeOption) sprite(s *Sprite)       { s.size = geom.Vec2(o) }
func (o sizeOption) ninePatch(p *NinePatch) { p.size = geom.Vec2(o) }

// Size is the box, in points, a sprite fits its picture to or a
// nine-patch stretches to. Scale still multiplies it.
func Size(w, h float32) SizeOption { return sizeOption(geom.V2(w, h)) }

// ColorOption sets the colour of a line, a polygon, a text or a bitmap
// text.
type ColorOption interface {
	LineOption
	PolygonOption
	TextOption
	BitmapTextOption
}

type colorOption color.Color

func (o colorOption) line(l *Line)             { l.color = color.Color(o) }
func (o colorOption) polygon(p *Polygon)       { p.color = color.Color(o) }
func (o colorOption) text(t *Text)             { t.color = color.Color(o) }
func (o colorOption) bitmapText(t *BitmapText) { t.color = color.Color(o) }

// Color sets what a line, a polygon or a text draws with. Tint, on any
// entity, multiplies that and its children's colours.
func Color(c color.Color) ColorOption { return colorOption(c) }
