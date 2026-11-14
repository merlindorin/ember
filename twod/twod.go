// Package twod is ember's 2D: sprites, animated sprites, lines, polygons,
// text, nine-patches, layers and 2D cameras, and the canvas they and the
// UI draw on. Each is an entity built with options, added to a window as
// its child (else it shows in the primary window), and changed with
// setters:
//
//	win.AppendChild(twod.NewSprite(crate, twod.At(320, 180), twod.Rotation(0.4)))
//
// One unit is one point at zoom 1. Without a camera, the origin is the
// window's top-left corner; a camera centres the view on its position.
// Under a camera with a Resolution, one unit is one pixel of that
// resolution instead. They need no node tree, and they still save and load
// as scene nodes.
//
// At places an entity's pivot. Sized entities (sprites, nine-patches,
// texts) are pivoted at their centre unless Pivot says otherwise; a tile
// map's pivot is the top-left corner of its first cell; a line's or a
// polygon's is the origin of its points.
//
// Color is what an entity draws with: a text's, a line's or a polygon's
// own colour. Tint multiplies the colours of an entity and all its
// children, whatever they draw.
package twod

import (
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	lt "github.com/lantervale/ember/internal/lit"
	"github.com/lantervale/ember/node"
	"github.com/lantervale/ember/world"
)

// Placement is where a 2D entity is in its parent: position, rotation and
// scale. An entity moved in FixedUpdate is drawn between where the last two
// fixed steps left it, so it moves smoothly at any frame rate.
type Placement struct {
	position geom.Vec2 `node:"prop"`
	rotation float32   `node:"prop"`
	scale    geom.Vec2 `node:"prop"`
	steps
}

func (p *Placement) GetPlacement() *Placement  { return p }
func (p *Placement) Position() geom.Vec2       { return p.position }
func (p *Placement) SetPosition(at geom.Vec2)  { p.moving(); p.position = at }
func (p *Placement) Translate(delta geom.Vec2) { p.moving(); p.position = p.position.Add(delta) }
func (p *Placement) Rotation() float32         { return p.rotation }
func (p *Placement) SetRotation(r float32)     { p.moving(); p.rotation = r }
func (p *Placement) Scale() geom.Vec2          { return p.scale }
func (p *Placement) SetScale(s geom.Vec2)      { p.moving(); p.scale = s }

// Local is the transform the placement makes in the parent.
func (p *Placement) Local() geom.Transform2D {
	return geom.Transform2DFrom(p.position, p.rotation, 0, p.scale)
}

// Look is how a 2D entity is ordered and coloured among the others.
type Look struct {
	z       int         `node:"prop"`
	tint    color.Color `node:"prop"`
	hidden  bool        `node:"prop"`
	sortByY bool        `node:"prop"`
	blend   BlendMode   `node:"prop"`
	overlay bool
	// childrenHidden draws the entity without its children.
	childrenHidden bool
	// light is how the light package lights it (internal/lit).
	light lt.State
}

func (l *Look) GetDrawLook() *Look    { return l }
func (l *Look) Z() int                { return l.z }
func (l *Look) SetZ(z int)            { l.z = z }
func (l *Look) Tint() color.Color     { return l.tint }
func (l *Look) SetTint(c color.Color) { l.tint = c }
func (l *Look) Hidden() bool          { return l.hidden }
func (l *Look) SetHidden(hidden bool) { l.hidden = hidden }

// ChildrenHidden reports whether the entity draws without its children,
// as a folded ui.Panel does; SetChildrenHidden sets it. The children keep
// their own Hidden.
func (l *Look) ChildrenHidden() bool      { return l.childrenHidden }
func (l *Look) SetChildrenHidden(on bool) { l.childrenHidden = on }
func (l *Look) SortByY() bool             { return l.sortByY }
func (l *Look) SetSortByY(on bool)        { l.sortByY = on }
func (l *Look) Blend() BlendMode          { return l.blend }
func (l *Look) SetBlend(m BlendMode)      { l.blend = m }

// NewLook returns the look of a visible, untinted entity.
func NewLook() Look { return Look{tint: color.White} }

// Item is what a 2D entity is made of: the entity, its scene node, its
// placement and its look. A game's own 2D entity embeds one made by Base.
type Item struct {
	*world.BasicEntity
	node.Node
	Placement
	Look
}

// Base is the Item a game's own 2D entity embeds before Build.
func Base() Item {
	return Item{BasicEntity: world.NewBasic(), Placement: Placement{scale: geom.V2(1, 1)}, Look: NewLook()}
}

// Build finishes a game's own 2D entity, which embeds an Item made by Base:
// it names it and applies its options.
//
//	ship := twod.Build(&Ship{Item: twod.Base()}, "ship", twod.At(100, 80))
func Build[T Entity](e T, name string, opts ...Option) T {
	e.GetSceneNode().Bind(e)
	e.GetSceneNode().SetName(name)
	for _, o := range opts {
		o(e)
	}
	return e
}

// Entity is any 2D entity: something with a look, placed in its parent by
// a transform. ui elements are entities too, placed by their layout.
type Entity interface {
	node.Noded
	GetDrawLook() *Look
	Local() geom.Transform2D
}

// Drawer is an entity that draws itself. Draw gets a canvas in the
// entity's own space, its size from Canvas.Size.
type Drawer interface {
	Draw(c *Canvas)
}

// Exposer is an entity automation clients read: after Draw, it is recorded
// under its name with the role and value Expose returns.
type Exposer interface {
	Expose() (role string, value any)
}

// Preparer is an entity with more to set on the canvas before Draw, such
// as its size and clips. ui elements are Preparers; the function returned
// undoes it after Draw.
type Preparer interface {
	BeginDraw(c *Canvas) (end func())
}

// OverlayLayer is the fixed layer above every other, where the UI draws.
const OverlayLayer = 1 << 20

// Add makes children of the entity, and returns it: they move, tint and
// leave with it. Any entity can be a child, such as a collide body. The
// ready-made entities return themselves.
func (i *Item) Add(children ...world.BasicFace) *Item {
	add(i, children)
	return i
}

func add(parent node.Noded, children []world.BasicFace) {
	for _, c := range children {
		if n, ok := c.(node.Noded); ok {
			parent.GetSceneNode().AddNode(n)
		}
		parent.GetBasicEntity().AppendChild(c)
	}
}

// Global is the entity's transform in its layer: its parents' and its own.
func (i *Item) Global() geom.Transform2D { return global(i.Entity()) }

// GlobalPosition is where the entity is in its layer.
func (i *Item) GlobalPosition() geom.Vec2 { return i.Global().Origin }

func global(e node.Noded) geom.Transform2D {
	t := geom.Identity2D()
	if self, ok := e.(Entity); ok {
		t = self.Local()
	}
	for p := e.GetSceneNode().ParentNode(); p != nil; p = p.ParentNode() {
		parent, ok := p.Entity().(Entity)
		if !ok {
			break
		}
		t = parent.Local().Mul(t)
	}
	return t
}

// Option sets up a 2D entity.
type Option func(e Entity)

func placed(f func(*Placement)) Option {
	return func(e Entity) {
		if p, ok := e.(interface{ GetPlacement() *Placement }); ok {
			f(p.GetPlacement())
		}
	}
}

// At places the entity's pivot, in points from its parent's origin.
func At(x, y float32) Option { return placed(func(p *Placement) { p.position = geom.V2(x, y) }) }

// Rotation turns the entity by radians, clockwise on screen.
func Rotation(radians float32) Option { return placed(func(p *Placement) { p.rotation = radians }) }

// Scale stretches the entity and its children.
func Scale(x, y float32) Option { return placed(func(p *Placement) { p.scale = geom.V2(x, y) }) }

// Z orders the entity among the others of its layer, higher on top. It
// adds to its parent's.
func Z(z int) Option { return func(e Entity) { e.GetDrawLook().z = z } }

// Tint multiplies the colours of the entity and its children by c, on top
// of what they draw with.
func Tint(c color.Color) Option { return func(e Entity) { e.GetDrawLook().tint = c } }

// Hidden hides the entity and its children.
func Hidden() Option { return func(e Entity) { e.GetDrawLook().hidden = true } }

// Overlay draws an entity that has no 2D parent, and its children, on the
// fixed OverlayLayer, above the 2D world: ui elements are made this way.
func Overlay() Option { return func(e Entity) { e.GetDrawLook().overlay = true } }

// SortByY draws the entity's children by their y, lowest first, so things
// further down the screen show in front: top-down depth.
func SortByY() Option { return func(e Entity) { e.GetDrawLook().sortByY = true } }
