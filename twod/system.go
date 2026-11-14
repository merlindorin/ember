package twod

import (
	"cmp"
	"time"

	"slices"

	"github.com/lantervale/ember/internal/cursorhint"
	"github.com/lantervale/ember/internal/draw"
	lt "github.com/lantervale/ember/internal/lit"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/internal/precollect"
	"github.com/lantervale/ember/internal/slot"
	"github.com/lantervale/ember/internal/wm"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/world"
)

// Plugin installs the 2D system. ember.New adds it.
type Plugin struct{}

func (Plugin) Build(w *world.World) {
	w.AddPlugins(canvasPlugin{})
	clock := &stepClock{}
	w.AddSystems(&System{}, &stepStart{clock: clock}, &stepEnd{clock: clock})
}

// System plays animations, moves cameras and draws the 2D entities added
// to the World, each in the window it is a child of, late in Update, before
// the UI.
type System struct {
	world.Group[Entity]
	world *world.World
	// entries is each window's draw list of the last frame, its memory
	// reused, and roots the trees it was collected from.
	entries map[*platform.Window][]entry
	roots   []rootOf
	players players
	// scene and culls are the frame's, their memory reused.
	scene scene
	culls map[*platform.Window]*cull
}

func (s *System) New(w *world.World) { s.world = w }

// AddByInterface gives a sprite joining the World the list of playing
// sprites, and lists it if it plays already.
func (s *System) AddByInterface(e world.Identifier) {
	s.Group.AddByInterface(e)
	if en, ok := e.(Entity); ok {
		if p := en.GetSceneNode().ParentNode(); p != nil {
			if sp, isSprite := p.Entity().(*Sprite); isSprite && sp.filed != nil {
				sp.filed.keepVisiting()
			}
		}
	}
	if sp, ok := e.(*Sprite); ok {
		sp.players = &s.players
		sp.join()
	}
}

// Remove takes a sprite leaving the World off the list.
func (s *System) Remove(e world.Identifier) {
	if sp, ok := e.(*Sprite); ok {
		sp.listed, sp.players = false, nil
		sp.turn++
	}
	s.Group.Remove(e)
}
func (s *System) Priority() int { return -80 }
func (s *System) Name() string  { return "twod" }

type entry struct {
	e        Entity
	global   geom.Transform2D
	tint     color.Color
	z, layer int
	fixed    bool
	parallax float32
	// anchor is a fixed layer's origin, as fractions of its window.
	anchor geom.Vec2
	// owner is the innermost layer holding e, nil for none.
	owner *Layer
	blend BlendMode
	// clip, when set, is the layer's ClipTo in its space, placed by space.
	clip  *geom.Rect
	space geom.Transform2D
	// light is how it reads the light, anchor2 where its LitAt anchor
	// stands in its layer, and around, when not empty, the box in its
	// layer it takes the light round instead.
	light   lt.Mode
	anchor2 geom.Vec2
	around  geom.Rect
	// left is the occluder groups whose shadows don't fall on it.
	left uint8
	// named is the chain of layers some light Affects that it is under,
	// innermost first, nil for none.
	named *namedLayer
}

// namedLayer is a layer some light Affects, in a chain up to the
// outermost such layer holding it.
type namedLayer struct {
	layer any
	up    *namedLayer
}

func (s *System) Update(f world.Frame) {
	pu := slot.Get[windows](f.World)
	if pu == nil {
		return
	}
	begin := time.Now()
	precollect.Run(f.World)
	sc := s.gather(f)
	sc.frame(f, pu)
	sc.prepare(f, pu)
	byWindow := s.collectAll(f, pu, sc)
	collected := time.Now()
	f.World.Stats().Record("twod collect", collected.Sub(begin))
	for win, feeding := range sc.feeds {
		u := pu.primary.For(win)
		for _, cam := range feeding {
			drawInto(u, byWindow[win], cam)
		}
	}
	for win, entries := range byWindow {
		drawEntries(pu.primary.For(win), entries, sc.cameras[win])
	}
	f.World.Stats().Record("twod draw", time.Since(collected))
	for win, c := range pu.byWin {
		if win != nil {
			cursorhint.Show(win, int(c.cursor))
		}
	}
	v := slot.Get[views](f.World)
	if v == nil {
		v = &views{}
		slot.Set(f.World, v)
	}
	v.cameras, v.alpha = sc.cameras, f.Time.Alpha
}

// scene is what a frame draws from: each window's camera, the cameras
// feeding a texture in each, and the endless maps.
type scene struct {
	cameras map[*platform.Window]*Camera
	feeds   map[*platform.Window][]*Camera
	maps    []rootOf
	// views and areas are prepare's scratch.
	views []*Camera
	areas []geom.Rect
}

// gather steps the playing sprites' clips and the particles, emitting
// what they finished, and finds the frame's cameras, maps and roots.
func (s *System) gather(f world.Frame) *scene {
	s.animate(f)
	sc := &s.scene
	if sc.cameras == nil {
		sc.cameras, sc.feeds = map[*platform.Window]*Camera{}, map[*platform.Window][]*Camera{}
	}
	clear(sc.cameras)
	for win, feeding := range sc.feeds {
		clear(feeding)
		sc.feeds[win] = feeding[:0]
	}
	clear(sc.maps)
	sc.maps = sc.maps[:0]
	roots := s.roots[:0]
	for e := range s.All() {
		if m, ok := e.(*InfiniteMap); ok {
			sc.maps = append(sc.maps, rootOf{m, wm.Resolve(f.World, windowOf(f.World, m))})
		}
		if p, ok := e.(*Particles); ok && p.step(f.DT) {
			emitDone(f.World, p)
		}
		if c, ok := e.(*Camera); ok {
			win := wm.Home(f.World, c, c.window)
			if c.into != nil {
				sc.feeds[win] = append(sc.feeds[win], c)
			} else if best, has := sc.cameras[win]; !has || c.priority > best.priority {
				sc.cameras[win] = c
			}
		}
		if parent := e.GetSceneNode().ParentNode(); parent != nil {
			if _, ok := parent.Entity().(Entity); ok {
				continue
			}
		}
		roots = append(roots, rootOf{e, wm.Resolve(f.World, windowOf(f.World, e))})
	}
	clear(roots[len(roots):cap(roots)])
	s.roots = roots
	for win, feeding := range sc.feeds {
		if len(feeding) == 0 {
			delete(sc.feeds, win)
		}
	}
	return sc
}

// frame sets every camera up for the frame in its window.
func (sc *scene) frame(f world.Frame, pu *windows) {
	for win, cam := range sc.cameras {
		u := pu.primary.For(win)
		cam.alpha = f.Time.Alpha
		cam.frame(f.DT, sizeOf(u.Screen()), u.framebuffer())
	}
	for win, feeding := range sc.feeds {
		u := pu.primary.For(win)
		for _, cam := range feeding {
			cam.alpha = f.Time.Alpha
			cam.frame(f.DT, sizeOf(u.Screen()), u.framebuffer())
		}
	}
}

// prepare makes the chunks of each endless map its cameras see.
func (sc *scene) prepare(f world.Frame, pu *windows) {
	for _, m := range sc.maps {
		sc.views = append(append(sc.views[:0], sc.cameras[m.win]), sc.feeds[m.win]...)
		sc.areas = mapViews(sc.areas[:0], m.e, sc.views, sizeOf(pu.primary.For(m.win).Screen()), f.Time.Alpha)
		if im, ok := m.e.(*InfiniteMap); ok {
			im.prepare(sc.areas)
		}
	}
	clear(sc.views)
}

// collectAll lists, sorted by layer and z, what each window draws of the
// roots in view.
func (s *System) collectAll(f world.Frame, pu *windows, sc *scene) map[*platform.Window][]entry {
	if s.entries == nil {
		s.entries = map[*platform.Window][]entry{}
	}
	byWindow := s.entries
	for win, entries := range byWindow {
		byWindow[win] = entries[:0]
	}
	if s.culls == nil {
		s.culls = map[*platform.Window]*cull{}
	}
	for _, c := range s.culls {
		c.set = false
	}
	for _, r := range s.roots {
		c := s.culls[r.win]
		if c == nil {
			c = &cull{}
			s.culls[r.win] = c
		}
		if !c.set {
			c.reset(sizeOf(pu.primary.For(r.win).Screen()), sc.cameras[r.win], sc.feeds[r.win])
		}
		layer := entry{parallax: 1}
		if r.e.GetDrawLook().overlay {
			layer = entry{layer: OverlayLayer, fixed: true, parallax: 1}
		}
		byWindow[r.win] = collect(byWindow[r.win], r.e, geom.Identity2D(), color.White, 0, layer, f.Time.Alpha, c)
	}
	for win, entries := range byWindow {
		if len(entries) == 0 {
			delete(byWindow, win)
		}
	}
	for _, entries := range byWindow {
		slices.SortStableFunc(entries, func(a, b entry) int {
			if a.layer != b.layer {
				return a.layer - b.layer
			}
			return a.z - b.z
		})
	}
	return byWindow
}

// rootOf is an entity at the top of a 2D tree and the window it shows in.
type rootOf struct {
	e   Entity
	win *platform.Window
}

// views are the cameras each window showed in the last frame.
type views struct {
	cameras map[*platform.Window]*Camera
	alpha   float32
}

// toScreen is where e showed in its window in the last frame, in points:
// through the window's camera unless e is on a fixed layer.
func toScreen(w *world.World, e Entity) (geom.Vec2, bool) {
	t, _, ok := screenTransform(w, e)
	return t.Origin, ok
}

// collect lists e and its 2D children in drawing order, with their
// transform in their layer, tint and z.
func collect(
	out []entry,
	e Entity,
	parent geom.Transform2D,
	tint color.Color,
	z int,
	layer entry,
	alpha float32,
	c *cull,
) []entry {
	look := e.GetDrawLook()
	if look.hidden {
		return out
	}
	local := drawnLocal(e, alpha)
	global := parent.Mul(local)
	tint, z = tint.Mul(look.tint), z+look.z
	if l, ok := e.(*Layer); ok {
		layer = entry{
			layer:    l.number,
			fixed:    l.fixed,
			parallax: l.parallax,
			owner:    l,
			anchor:   l.anchor,
			named:    layer.named,
		}
		if l.clip.W > 0 && l.clip.H > 0 {
			clip := l.clip
			layer.clip, layer.space = &clip, local
		}
		global, z = local, look.z
	}
	layer.lighting(e, global)
	if drawable(e) {
		en := entry{
			e:        e,
			global:   global,
			tint:     tint,
			z:        z,
			layer:    layer.layer,
			fixed:    layer.fixed,
			parallax: layer.parallax,
			anchor:   layer.anchor,
			clip:     layer.clip,
			space:    layer.space,
			owner:    layer.owner,
			blend:    look.blend,
			light:    layer.light,
			anchor2:  layer.anchor2,
			around:   layer.around,
			left:     layer.left,
			named:    layer.named,
		}
		if c.visible(en) {
			out = append(out, en)
		}
	}
	if look.childrenHidden {
		return out
	}
	picked, gridded := c.pick(e, local)
	if !look.sortByY {
		if gridded {
			for _, f := range picked {
				out = collect(out, f.e, global, tint, z, layer, alpha, c)
			}
			return out
		}
		for k := range e.GetSceneNode().EachChild() {
			if child, ok := k.Entity().(Entity); ok {
				out = collect(out, child, global, tint, z, layer, alpha, c)
			}
		}
		return out
	}
	// The children are collected in turn, and only those with something
	// in view are sorted by their y: their stretches of entries are put in
	// that order.
	base, first := len(out), len(c.spans)
	if gridded {
		for _, f := range picked {
			out = c.collectY(out, f.e, global, tint, z, layer, alpha)
		}
	} else {
		for k := range e.GetSceneNode().EachChild() {
			if child, ok := k.Entity().(Entity); ok {
				out = c.collectY(out, child, global, tint, z, layer, alpha)
			}
		}
	}
	spans := c.spans[first:]
	if len(spans) >= 2 {
		slices.SortStableFunc(spans, byY)
		c.kept = append(c.kept[:0], out[base:]...)
		out = out[:base]
		for _, sp := range spans {
			out = append(out, c.kept[sp.from-base:sp.to-base]...)
		}
	}
	c.spans = c.spans[:first]
	return out
}

func byY(a, b ySpan) int { return cmp.Compare(a.y, b.y) }

// collectY collects child of a SortByY parent, noting the stretch of
// entries it gave, if any, and its y.
func (c *cull) collectY(
	out []entry,
	child Entity,
	global geom.Transform2D,
	tint color.Color,
	z int,
	layer entry,
	alpha float32,
) []entry {
	from := len(out)
	out = collect(out, child, global, tint, z, layer, alpha, c)
	if len(out) > from {
		c.spans = append(c.spans, ySpan{child.Local().Origin.Y, from, len(out)})
	}
	return out
}

// ySpan is the stretch of entries one child of a SortByY parent collected,
// and its y.
type ySpan struct {
	y        float32
	from, to int
}

func drawEntries(u *Canvas, entries []entry, cam *Camera) {
	front := u.list
	window := sizeOf(u.Screen())
	pixels := u.framebuffer()
	if pixels.X <= 0 || pixels.Y <= 0 {
		pixels = window
	}
	viewport := window
	var scaled *draw.Pass
	if cam != nil {
		viewport = cam.viewport
		if cam.resolution != (geom.Vec2{}) {
			scaled = cam.pass
		}
	}
	toLight := draw.ToLight(window)
	first := len(u.passes)
	pictures := u.pictures[:0]
	defer func() { u.endEntries(front, first, pictures) }()
	for _, en := range entries {
		if cam != nil && !en.fixed && !cam.shows(en) {
			continue
		}
		list, screen, fb, lightMap := front, window, pixels, toLight
		if en.layer < 0 {
			list = u.behind()
		}
		u.scaled, u.pass = nil, nil
		if scaled != nil && !en.fixed {
			if u.start(scaled) {
				lo, hi := cam.screen.Xform(geom.Vec2{}), cam.screen.Xform(scaled.Screen)
				list.Picture(geom.Rect{X: lo.X, Y: lo.Y, W: hi.X - lo.X, H: hi.Y - lo.Y}, scaled, draw.Alpha, nil)
			}
			list, screen, fb, u.scaled, u.pass = scaled.List, scaled.Screen, scaled.Screen, cam, scaled
			lightMap = toLight.Mul(cam.screen)
		}
		if l := en.owner; l != nil && l.pictured() {
			p := l.picturePass(screen, fb)
			if u.start(p) && !l.mask {
				var mask *draw.Pass
				if l.maskedBy != nil {
					mask = l.maskedBy.picturePass(screen, fb)
					u.start(mask)
				}
				pictures = append(pictures, picture{pass: p, mask: mask, list: list, at: len(list.Vertices)})
				list.Picture(geom.Rect{W: screen.X, H: screen.Y}, p, draw.Blend(l.blend), mask)
			}
			list = p.List
		}
		u.list = list
		view := geom.Identity2D()
		switch {
		case en.fixed:
			view.Origin = en.anchor.Mul(window)
		case cam != nil:
			view = cam.transform(viewport, en.parallax)
		}
		listLight(u, en, view, lightMap)
		drawEntry(u, en, view, cam != nil && cam.snap)
	}
}

// endEntries crops the layer pictures drawEntries made, puts the canvas back
// on front with every list's state reset, and orders the passes it started
// from first so the GPU draws the innermost first.
func (u *Canvas) endEntries(front *draw.List, first int, pictures []picture) {
	cropPictures(pictures)
	u.pictures = pictures
	u.list, u.scaled, u.pass = front, nil, nil
	for _, l := range []*draw.List{front, u.back} {
		if l != nil {
			l.SetTransform(geom.Identity2D())
			l.SetModulate(color.White)
			l.SetBlend(draw.Alpha)
			l.SetLight(draw.Unlit, geom.Identity2D(), geom.Vec2{})
			l.SnapVertices = false
		}
	}
	slices.Reverse(u.passes[first:])
}

// drawInto draws the entries of cam's window, but its fixed layers, into
// cam's render texture. A layer drawn as a picture (a blend, an effect, a
// mask, MaskedBy) is one there too, at the texture's size, through
// passes of the camera's own: a disc AsMask layer leaves the texture's
// corners clear.
func drawInto(u *Canvas, entries []entry, cam *Camera) {
	front := u.list
	first := len(u.passes)
	defer func() {
		u.list, u.pass = front, nil
		slices.Reverse(u.passes[first:])
	}()
	u.start(cam.pass)
	size := cam.pass.Screen
	for _, en := range entries {
		if en.fixed || !cam.shows(en) {
			continue
		}
		list := cam.pass.List
		if l := en.owner; l != nil && l.pictured() {
			p := l.pictureIn(cam, size)
			if u.start(p) && !l.mask {
				var mask *draw.Pass
				if l.maskedBy != nil {
					mask = l.maskedBy.pictureIn(cam, size)
					u.start(mask)
				}
				list.Picture(geom.Rect{W: size.X, H: size.Y}, p, draw.Blend(l.blend), mask)
			}
			list = p.List
		}
		u.list, u.pass = list, cam.pass
		u.list.SetLight(draw.Unlit, geom.Identity2D(), geom.Vec2{})
		drawEntry(u, en, cam.transform(cam.viewport, en.parallax), cam.snap)
	}
}

// drawEntry draws en on u's current list, through view.
func drawEntry(u *Canvas, en entry, view geom.Transform2D, snap bool) {
	u.list.SnapVertices = snap
	if en.clip != nil {
		u.SetTransform(view.Mul(en.space))
		u.PushClip(*en.clip)
	}
	u.SetTransform(view.Mul(en.global))
	u.list.SetModulate(en.tint)
	u.list.SetBlend(draw.Blend(en.blend))
	drawOne(u, en.e)
	if en.clip != nil {
		u.PopClip()
	}
}

// start begins pass for this frame, the first time it is met, and reports
// whether it did.
func (u *Canvas) start(p *draw.Pass) bool {
	if slices.Contains(u.passes, p) {
		return false
	}
	p.List.Reset(u.list.Time())
	u.passes = append(u.passes, p)
	return true
}

// drawOne draws e, scoped by the names of the nodes above it, and records it
// when it exposes itself.
func drawOne(u *Canvas, e Entity) {
	u.path()
	prev := u.scope
	u.pending = e
	drawing := u.drawing
	u.drawing = e
	defer func() { u.drawing = drawing }()
	defer u.RestoreScope(prev)
	u.size = geom.Vec2{}
	if p, ok := e.(Preparer); ok {
		defer p.BeginDraw(u)()
	}
	if d, ok := e.(Drawer); ok {
		d.Draw(u)
	}
	if x, ok := e.(Exposer); ok {
		role, value := x.Expose()
		name := e.GetSceneNode().Name()
		u.Record(name, role, name, value, true, geom.Rect{W: u.size.X, H: u.size.Y})
	}
}

func drawable(e Entity) bool {
	switch e.(type) {
	case Drawer, Exposer:
		return true
	}
	return false
}

// windowOf is the window e is under, nil for the primary window.
func windowOf(w *world.World, e Entity) *platform.Window { return wm.Of(w, e) }

func sizeOf(r geom.Rect) geom.Vec2 { return geom.V2(r.W, r.H) }

// animate steps the playing sprites' clips, emitting what they finished or
// looped, and drops from the list those that stopped playing.
func (s *System) animate(f world.Frame) {
	list := s.players.list[:0]
	for _, p := range s.players.list {
		a := p.sprite
		if !a.listed || p.turn != a.turn {
			continue
		}
		a.refile()
		for _, ev := range a.step(f.DT) {
			switch ev := ev.(type) {
			case AnimationFinished:
				world.Emit(f.World, ev)
			case AnimationLooped:
				world.Emit(f.World, ev)
			}
		}
		if a.listed = a.playing; a.listed {
			list = append(list, p)
		}
	}
	clear(s.players.list[len(list):])
	s.players.list = list
}
