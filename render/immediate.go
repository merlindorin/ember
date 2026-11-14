package render

import (
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/scene"
	"github.com/lantervale/ember/world"
)

// Gizmos is an entity collecting debug lines for the current frame. Anything
// drawn is cleared at the start of the next frame, so draw every frame you
// want it visible. Lines are presented while Enabled.
//
//	g := render.NewGizmos()
//	w.AddEntity(g)
//	g.Line(a, b, color.Red)
//	g.Circle(player.Position, attackRange, color.Yellow)
type Gizmos struct {
	*world.BasicEntity
	Lines   []GizmoLine
	Enabled bool
}

func NewGizmos() *Gizmos { return &Gizmos{BasicEntity: world.NewBasic(), Enabled: true} }

func (g *Gizmos) GetGizmos() *Gizmos { return g }

// GizmoSet is any entity holding gizmos.
type GizmoSet interface {
	world.BasicFace
	GetGizmos() *Gizmos
}

type GizmoLine struct {
	A, B  geom.Vec3
	Color color.Color
}

func (g *Gizmos) Line(a, b geom.Vec3, c color.Color) {
	g.Lines = append(g.Lines, GizmoLine{a, b, c})
}

// Circle draws a horizontal circle.
func (g *Gizmos) Circle(center geom.Vec3, radius float32, c color.Color) {
	const n = 32
	prev := center.Add(geom.V3(radius, 0, 0))
	for i := 1; i <= n; i++ {
		a := geom.Tau * float32(i) / n
		p := center.Add(geom.V3(geom.Cos(a)*radius, 0, geom.Sin(a)*radius))
		g.Line(prev, p, c)
		prev = p
	}
}

func (g *Gizmos) Box(b geom.AABB, c color.Color) {
	p := [8]geom.Vec3{}
	for i := range p {
		p[i] = geom.V3(pick(i&1, b.Min.X, b.Max.X), pick(i&2, b.Min.Y, b.Max.Y), pick(i&4, b.Min.Z, b.Max.Z))
	}
	for _, e := range [12][2]int{
		{0, 1}, {2, 3}, {4, 5}, {6, 7},
		{0, 2}, {1, 3}, {4, 6}, {5, 7},
		{0, 4}, {1, 5}, {2, 6}, {3, 7},
	} {
		g.Line(p[e[0]], p[e[1]], c)
	}
}

// Axes draws the local axes of a transform.
func (g *Gizmos) Axes(t *scene.Transform, size float32) {
	m := t.World()
	o := m.Translation()
	g.Line(o, m.MulPoint(geom.V3(size, 0, 0)), color.Red)
	g.Line(o, m.MulPoint(geom.V3(0, size, 0)), color.Green)
	g.Line(o, m.MulPoint(geom.V3(0, 0, size)), color.Blue)
}

func pick(bit int, a, b float32) float32 {
	if bit != 0 {
		return b
	}
	return a
}

// Billboards is an entity collecting camera-facing quads for the current
// frame: particles, glows, health bars in the world. Additive billboards
// glow and bloom. They are cleared at the start of the next frame.
type Billboards struct {
	*world.BasicEntity
	Items []Billboard
}

func NewBillboards() *Billboards { return &Billboards{BasicEntity: world.NewBasic()} }

func (b *Billboards) GetBillboards() *Billboards { return b }

// BillboardSet is any entity holding billboards.
type BillboardSet interface {
	world.BasicFace
	GetBillboards() *Billboards
}

type Billboard struct {
	Position geom.Vec3
	Size     float32
	Color    color.Color
	Additive bool
	// Rotation spins the quad in screen space (radians).
	Rotation float32
}

func (b *Billboards) Add(bb Billboard) { b.Items = append(b.Items, bb) }

// Glow is a soft additive light blob, the building block of fire and magic.
func (b *Billboards) Glow(p geom.Vec3, size float32, c color.Color) {
	b.Items = append(b.Items, Billboard{Position: p, Size: size, Color: c, Additive: true})
}

// Plugin resolves the cameras' views, clears the gizmos and billboards each
// frame and keeps transforms up to date. It is pure Go: servers can install
// it to get the same View math without a GPU.
type Plugin struct{}

func (Plugin) Build(a *world.World) {
	a.AddSystems(&scene.TransformSystem{}, &ViewSystem{}, &clearSystem{})
}

type clearSystem struct {
	world.Group[GizmoSet]
	billboards world.Group[BillboardSet]
}

func (c *clearSystem) New(w *world.World) { w.AddView(&c.billboards) }

func (c *clearSystem) Update(world.Frame) {
	for g := range c.All() {
		g.GetGizmos().Lines = g.GetGizmos().Lines[:0]
	}
	for b := range c.billboards.All() {
		b.GetBillboards().Items = b.GetBillboards().Items[:0]
	}
}

func (c *clearSystem) Phase() world.Phase { return world.PreUpdate }
func (c *clearSystem) Priority() int      { return 1 << 30 }
func (c *clearSystem) Name() string       { return "render.clear-immediate" }
