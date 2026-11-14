package twod

import (
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/render"
)

// Every option reads back under its own name, and what a game changes
// while it plays has a setter taking the option's arguments.

func (l *Look) Overlay() bool { return l.overlay }

func (s *Sprite) Still() Frame           { return Frame{Texture: s.texture, Region: s.region} }
func (s *Sprite) Flip() (bool, bool)     { return s.flipX, s.flipY }
func (s *Sprite) Pivot() geom.Vec2       { return s.pivot }
func (s *Sprite) SetPivot(x, y float32)  { s.pivot = geom.V2(x, y); s.refile() }
func (s *Sprite) Speed() float32         { return s.speed }
func (s *Sprite) SetSpeed(speed float32) { Speed(speed).sprite(s) }

func (l *Line) Closed() bool           { return l.closed }
func (l *Line) SetClosed(closed bool)  { l.closed = closed }
func (l *Line) Joints() Joint          { return l.joint }
func (l *Line) SetJoints(j Joint)      { l.joint = j }
func (l *Line) Caps() (Cap, Cap)       { return l.start, l.end }
func (l *Line) SetCaps(start, end Cap) { l.start, l.end = start, end }

func (t *Text) Typeface() *Face          { return t.face }
func (t *Text) SetTypeface(face *Face)   { Typeface(face).text(t) }
func (t *Text) FontSize() float32        { return t.size }
func (t *Text) SetFontSize(size float32) { FontSize(size).text(t) }
func (t *Text) Wrap() float32            { return t.wrap }
func (t *Text) SetWrap(width float32)    { t.wrap = width }
func (t *Text) TextShadow() color.Color  { return t.shadow }
func (t *Text) SetShadow(c color.Color)  { t.shadow = c }
func (t *Text) Pivot() geom.Vec2         { return t.pivot }
func (t *Text) SetPivot(x, y float32)    { t.pivot = geom.V2(x, y) }

func (t *BitmapText) Glyphs() *BitmapFont     { return t.font }
func (t *BitmapText) SetGlyphs(f *BitmapFont) { Glyphs(f).bitmapText(t) }
func (t *BitmapText) TextShadow() color.Color { return t.shadow }
func (t *BitmapText) SetShadow(c color.Color) { t.shadow = c }
func (t *BitmapText) Pivot() geom.Vec2        { return t.pivot }
func (t *BitmapText) SetPivot(x, y float32)   { t.pivot = geom.V2(x, y) }

func (p *NinePatch) Pivot() geom.Vec2      { return p.pivot }
func (p *NinePatch) SetPivot(x, y float32) { p.pivot = geom.V2(x, y) }

func (l *Layer) SetNumber(n int)            { l.number = n }
func (l *Layer) SetFixed(fixed bool)        { l.fixed = fixed }
func (l *Layer) Parallax() float32          { return l.parallax }
func (l *Layer) Anchor() (float32, float32) { return l.anchor.X, l.anchor.Y }
func (l *Layer) SetAnchor(x, y float32)     { l.anchor = geom.V2(x, y) }
func (l *Layer) SetParallax(f float32)      { l.parallax = f }
func (l *Layer) ClipTo() geom.Rect          { return l.clip }
func (l *Layer) SetClipTo(r geom.Rect)      { l.clip = r }
func (l *Layer) AsMask() bool               { return l.mask }
func (l *Layer) MaskedBy() *Layer           { return l.maskedBy }
func (l *Layer) SetMaskedBy(mask *Layer)    { l.maskedBy = mask }
func (l *Layer) Effects() []*render.Effect  { return l.effects }
func (l *Layer) PassScale() float32         { return l.passScale }
func (l *Layer) SetPassScale(f float32)     { l.passScale = geom.Clamp(f, 0.01, 1) }

func (c *Camera) In() *platform.Window                            { return c.window }
func (c *Camera) Follow() interface{ GlobalPosition() geom.Vec2 } { return c.follow }
func (c *Camera) Smoothing() float32                              { return c.smoothing }
func (c *Camera) SetSmoothing(speed float32)                      { c.smoothing = speed }
func (c *Camera) Bounds() geom.Rect                               { return c.bounds }
func (c *Camera) SetBounds(r geom.Rect)                           { c.bounds = r }
func (c *Camera) SnapToPixels() bool                              { return c.snap }
func (c *Camera) SetSnapToPixels(on bool)                         { c.snap = on }
func (c *Camera) Priority() int                                   { return c.priority }
func (c *Camera) SetPriority(n int)                               { c.priority = n }
func (c *Camera) Resolution() (int, int)                          { return int(c.resolution.X), int(c.resolution.Y) }
func (c *Camera) SetResolution(w, h int)                          { Resolution(w, h).camera(c) }
func (c *Camera) Upscale() Filter                                 { return c.upscale }
func (c *Camera) SetUpscale(f Filter)                             { c.upscale = f }
func (c *Camera) FitWindow() bool                                 { return c.fit }
func (c *Camera) ViewSize() (float32, float32)                    { return c.viewSize.X, c.viewSize.Y }
func (c *Camera) SetViewSize(w, h float32)                        { ViewSize(w, h).camera(c) }
func (c *Camera) SetFitWindow(on bool)                            { c.fit = on }
func (c *Camera) Into() *Texture                                  { return c.into }

func (m *TileMap) CellSize() geom.Vec2 { return m.tileSize }
func (m *TileMap) Isometric() bool     { return m.isometric }
