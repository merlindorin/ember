package light

import (
	"fmt"
	"log"
	"maps"
	"math"
	"slices"
	"strings"

	"github.com/lantervale/ember/collide"
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/internal/draw"
	"github.com/lantervale/ember/internal/lit"
	"github.com/lantervale/ember/internal/wm"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/render"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/world"
)

// Plugin installs the lighting: each frame it lists, for each window, the
// lights and what blocks them in reach of its view. gpu.Plugin adds it. A
// window lights its Lit layers once a light shines in it or its camera
// has an Ambient; until then they show as they are.
type Plugin struct{}

func (Plugin) Build(w *world.World) {
	s := &system{byWin: map[*platform.Window]*draw.Lighting{}, bakes: map[any]*bake{}}
	w.AddSystems(s)
	draw.InstallLighting(w, s)
}

// occluders are the 2D entities made with Occluder.
type occluders struct{ world.Group[twod.Entity] }

// emissives are the sprites made with Emissive.
type emissives struct{ world.Group[*twod.Sprite] }

func (g *emissives) Matches(o world.Identifier) bool {
	s, ok := o.(*twod.Sprite)
	return ok && lit.Of(s).Emissive > 0
}

func (g *emissives) AddByInterface(o world.Identifier) {
	if g.Matches(o) {
		g.Group.AddByInterface(o)
	}
}

// glowKey is one glow texture: a picture's region blurred over a reach.
type glowKey struct {
	source *render.Texture
	region geom.Rect
	pad    float32
}

func (g *occluders) Matches(o world.Identifier) bool {
	e, ok := o.(twod.Entity)
	return ok && lit.Of(e).Occluder
}

func (g *occluders) AddByInterface(o world.Identifier) {
	if g.Matches(o) {
		g.Group.AddByInterface(o)
	}
}

type system struct {
	windows   world.Group[*platform.Window]
	lights    world.Group[*Light]
	occluders occluders
	emissives emissives
	// silhouettes are each window's Silhouette sprites this frame, and
	// lists the lists their shadows are drawn into.
	silhouettes map[*platform.Window][]silhouette
	frame       silhouetteFrame
	// blocked is each window's grid of where its occluders lie, reset
	// the first time a frame marks it, in fresh.
	blocked map[*platform.Window]*blocked
	fresh   map[*platform.Window]bool
	// candidates are each window's lights this frame, before its camera's
	// Budget; budgets fade them in and out of it.
	candidates map[*platform.Window][]candidate
	budgets    map[*platform.Window]*budget
	// fallen is, by window, the layers last told to read the window's
	// light picture past draw.MaxPictures.
	fallen map[*platform.Window]string
	probes world.Group[*Probe]
	// maps are the World's tile maps, and caches what each blocks light
	// with, placed.
	maps   world.Group[tileSource]
	caches map[tileSource]*mapCache
	// glows are the glow textures made so far, by what they glow.
	glows map[glowKey]*render.Texture
	byWin map[*platform.Window]*draw.Lighting
	// bakes are the static lights of each camera.
	bakes map[any]*bake
	// used are the windows lit this frame.
	used map[*platform.Window]bool
	// scratch holds a map's shadows while they're drawn.
	scratch []shadow
}

// Priority runs it after twod, whose cameras and drawn places it reads.
func (s *system) Priority() int { return -95 }
func (s *system) Name() string  { return "light" }

func (s *system) Lighting(win *platform.Window) *draw.Lighting { return s.byWin[win] }

func (s *system) Update(f world.Frame) {
	s.reset()
	statics := s.gather(f.World)
	s.applyBudgets(f)
	cams := map[any]bool{}
	for win := range s.windows.All() {
		if cam, baked := s.lightWindow(f.World, win, statics[win]); baked {
			cams[cam] = true
		}
	}
	for cam := range s.bakes {
		if !cams[cam] {
			delete(s.bakes, cam)
		}
	}
	s.addOccluders(f.World)
	for win, sprites := range s.silhouettes {
		px := win.ContentScale() * draw.LightScale
		if px <= 0 {
			px = draw.LightScale
		}
		silhouettes(s.byWin[win], sprites, draw.LightRect(win.Size()), px, &s.frame)
	}
	for win, l := range s.byWin {
		if s.used[win] {
			s.blockedOf(win, draw.LightRect(win.Size())).unblock(l)
			s.plan(f.World, win, l)
		}
	}
	s.addEmissives(f.World)
	s.placeProbes(f.World)
	s.forgetUnused()
	s.drawDebug(f.World)
}

// reset empties every window's lighting for a new frame, keeping its lists.
func (s *system) reset() {
	for win, l := range s.byWin {
		l.Lights, l.Budget, l.Fading = l.Lights[:0], 0, 0
		l.Occluders.Reset(0)
		l.Occluders.SetBlend(draw.Occlude)
		l.Emissive.Reset(0)
		l.Emissive.SetBlend(draw.Add)
		clear(l.Glows)
		l.Probes = l.Probes[:0]
		l.AmbientMap.Reset(0)
		l.Debug.Reset(0)
		l.Static = nil
		l.Silhouettes, l.SilhouetteQuads, l.SilhouetteSide = l.Silhouettes[:0], l.SilhouetteQuads[:0], 0
		s.byWin[win] = l
	}
	s.frame.lists.reset()
	clear(s.silhouettes)
	clear(s.fresh)
	if s.used == nil {
		s.used = map[*platform.Window]bool{}
	}
	clear(s.used)
}

// of is win's lighting this frame, made the first time, and marks win used.
func (s *system) of(win *platform.Window) *draw.Lighting {
	l := s.byWin[win]
	if l == nil {
		l = &draw.Lighting{
			Occluders:  &draw.List{},
			Emissive:   &draw.List{},
			AmbientMap: &draw.List{},
			Debug:      &draw.List{},
			Glows:      map[*render.Texture]draw.Glow{},
		}
		l.Occluders.SetBlend(draw.Occlude)
		l.Emissive.SetBlend(draw.Add)
		s.byWin[win] = l
	}
	s.used[win] = true
	return l
}

// gather sorts the shown lights: directional ones straight into their
// window's lighting, the others into candidates for its budget, and static
// ones into what it returns, by window, for the bakes.
func (s *system) gather(w *world.World) map[*platform.Window]map[*Light]draw.Light {
	statics := map[*platform.Window]map[*Light]draw.Light{}
	for l := range s.lights.All() {
		if l.Hidden() || l.intensity <= 0 || l.kind != draw.DirectionalLight && l.radius <= 0 {
			continue
		}
		t, win, ok := lit.Screen(w, l)
		win = wm.Resolve(w, win)
		if !ok || win == nil {
			continue
		}
		if l.static && (l.kind == draw.PointLight || l.kind == draw.ConeLight) {
			dl, _ := l.screen(t, draw.LightRect(win.Size()))
			if statics[win] == nil {
				statics[win] = map[*Light]draw.Light{}
				s.of(win)
			}
			statics[win][l] = dl
			continue
		}
		if dl, shown := l.screen(t, draw.LightRect(win.Size())); shown {
			if dl.Kind == draw.DirectionalLight {
				s.of(win).Lights = append(s.of(win).Lights, dl)
				continue
			}
			s.of(win)
			if s.candidates == nil {
				s.candidates, s.budgets = map[*platform.Window][]candidate{}, map[*platform.Window]*budget{}
			}
			s.candidates[win] = append(s.candidates[win], candidate{light: l, drawn: dl})
		}
	}
	return statics
}

// applyBudgets keeps each window's candidates within its camera's Budget.
func (s *system) applyBudgets(f world.Frame) {
	for win, cands := range s.candidates {
		n, fade := 0, float32(defaultBudgetFade)
		if cam, _, ok := lit.View(f.World, win); ok {
			if st := cameraState(cam); st != nil {
				n = st.Budget
				if st.HasBudgetFade {
					fade = st.BudgetFade
				}
			}
		}
		b := s.budgets[win]
		if b == nil {
			b = &budget{}
			s.budgets[win] = b
		}
		b.apply(s.byWin[win], cands, win, n, fade, f.DT)
		s.candidates[win] = cands[:0]
	}
}

// lightWindow sets win's ambient, ambient map, the occluders of its
// camera's spaces and its static lights; it returns the camera and whether
// that camera has a bake.
func (s *system) lightWindow(w *world.World, win *platform.Window, statics map[*Light]draw.Light) (any, bool) {
	cam, view, ok := lit.View(w, win)
	st := cameraState(cam)
	if !ok || st == nil || !s.used[win] && !st.HasAmbient && st.AmbientMap == nil {
		return cam, false
	}
	lighting := s.of(win)
	lighting.Ambient, lighting.Exposure = color.Black, 1
	if st.HasAmbient {
		lighting.Ambient = st.Ambient
	}
	if st.HasExposure {
		lighting.Exposure = st.Exposure
	}
	r := draw.LightRect(win.Size())
	area := draw.Bounds(view.Inverse(), r)
	if st.AmbientMap != nil {
		toLight := geom.Transform2D{X: view.X, Y: view.Y, Origin: view.Origin.Sub(geom.V2(r.X, r.Y))}
		lit.Tiles(st.AmbientMap, lighting.AmbientMap, area, toLight)
	}
	var spaces []*collide.Space
	for _, sp := range st.Spaces {
		if space, isSpace := sp.(*collide.Space); isSpace {
			spaces = append(spaces, space)
		}
	}
	for _, space := range spaces {
		s.spaceOccluders(lighting, space, win, view, area, r)
	}
	maps := s.mapsIn(w, win)
	s.mapOccluders(lighting, maps, win, view, area, r)
	b := s.bakes[cam]
	if b == nil && len(statics) == 0 {
		return cam, false
	}
	if b == nil {
		b = newBake()
		s.bakes[cam] = b
	}
	placed := map[*Light]draw.Light{}
	for l, dl := range statics {
		placed[l] = worldLight(dl, view)
	}
	b.update(placed, spaces, maps, s.cacheOf)
	lighting.Static = b.static(cam, area, view)
	return cam, true
}

// spaceOccluders adds the solids and solid bodies of space within area to
// lighting's occluders.
func (s *system) spaceOccluders(
	lighting *draw.Lighting,
	space *collide.Space,
	win *platform.Window,
	view geom.Transform2D,
	area, r geom.Rect,
) {
	for _, src := range space.SolidSources() {
		if fromMap(src) {
			continue
		}
		for _, solid := range src.SolidIn(area) {
			quad(lighting.Occluders, s.blockedOf(win, r), view, solid, r, 1, 0)
		}
	}
	for _, b := range space.Bodies(area) {
		if !b.Trigger() && b.Soft() == 0 {
			quad(lighting.Occluders, s.blockedOf(win, r), view, b.Bounds(), r, 1, 0)
		}
	}
}

// mapsIn are the tile maps in win, hidden or not.
func (s *system) mapsIn(w *world.World, win *platform.Window) []tileSource {
	var out []tileSource
	for m := range s.maps.All() {
		if state(m).NoShadows {
			continue
		}
		if _, in, ok := lit.Screen(w, m); ok && in == win {
			out = append(out, m)
		}
	}
	return out
}

// cacheOf is what m blocks light with, placed, kept while m is in the
// World.
func (s *system) cacheOf(m tileSource) *mapCache {
	c := s.caches[m]
	if c == nil {
		if s.caches == nil {
			s.caches = map[tileSource]*mapCache{}
		}
		if len(s.caches) > s.maps.Len() {
			live := map[tileSource]bool{}
			for m := range s.maps.All() {
				live[m] = true
			}
			maps.DeleteFunc(s.caches, func(m tileSource, _ *mapCache) bool { return !live[m] })
		}
		c = &mapCache{}
		s.caches[m] = c
	}
	return c
}

// mapOccluders adds what maps' tiles block light with within area to
// lighting's occluders.
func (s *system) mapOccluders(
	lighting *draw.Lighting, maps []tileSource, win *platform.Window, view geom.Transform2D, area, r geom.Rect,
) {
	scale := view.X.Len()
	for _, m := range maps {
		s.scratch = s.cacheOf(m).shadowsIn(s.scratch[:0], m, area)
		for _, sh := range s.scratch {
			quad(lighting.Occluders, s.blockedOf(win, r), view, sh.rect, r, 1<<sh.group, sh.height*scale)
		}
	}
}

// addOccluders adds the occluding entities shown in a used window, keeping
// Silhouette sprites aside for their own pass.
func (s *system) addOccluders(w *world.World) {
	for e := range s.occluders.All() {
		if len(s.used) == 0 {
			break
		}
		if e.GetDrawLook().Hidden() {
			continue
		}
		t, win, ok := lit.Screen(w, e)
		win = wm.Resolve(w, win)
		rect, sized := lit.Size(e)
		if !ok || !sized || win == nil || !s.used[win] {
			continue
		}
		if sp, isSprite := e.(*twod.Sprite); isSprite && lit.Of(e).Silhouette {
			if s.silhouettes == nil {
				s.silhouettes = map[*platform.Window][]silhouette{}
			}
			s.silhouettes[win] = append(s.silhouettes[win], silhouette{sp, t})
			continue
		}
		quad(
			s.byWin[win].Occluders,
			s.blockedOf(win, draw.LightRect(win.Size())),
			t,
			rect,
			draw.LightRect(win.Size()),
			lit.Of(e).Groups,
			lit.Of(e).Height*t.X.Len(),
		)
	}
}

func (s *system) addEmissives(w *world.World) {
	for sp := range s.emissives.All() {
		if sp.Hidden() {
			continue
		}
		t, win, ok := lit.Screen(w, sp)
		win = wm.Resolve(w, win)
		if !ok || win == nil || !s.used[win] {
			continue
		}
		s.glow(s.byWin[win], sp, t, draw.LightRect(win.Size()))
	}
}

// placeProbes queues each probe on its window's lighting; a probe in an
// unlit window reads white, one off screen or past draw.MaxProbes is stale.
func (s *system) placeProbes(w *world.World) {
	for p := range s.probes.All() {
		t, win, ok := lit.Screen(w, p)
		win = wm.Resolve(w, win)
		if !ok || win == nil {
			continue
		}
		if !s.used[win] {
			p.result = draw.ProbeResult{Light: color.White, Ready: true}
			continue
		}
		size := win.Size()
		l := s.byWin[win]
		p.result.Stale = t.Origin.X < 0 || t.Origin.Y < 0 || t.Origin.X >= size.X || t.Origin.Y >= size.Y ||
			len(l.Probes) >= draw.MaxProbes
		if !p.result.Stale {
			l.Probes = append(l.Probes, draw.Probe{At: t.Origin, Result: &p.result})
		}
	}
}

// forgetUnused drops what the system keeps for windows nothing lit.
func (s *system) forgetUnused() {
	for win := range s.byWin {
		if !s.used[win] {
			delete(s.byWin, win)
			delete(s.blocked, win)
			delete(s.budgets, win)
			delete(s.fallen, win)
			delete(s.candidates, win)
		}
	}
}

func (s *system) drawDebug(w *world.World) {
	for win, l := range s.byWin {
		if ld := draw.LightDebugOf(w, win); ld.Outlines {
			clip := ld.Area.W > 0
			if clip {
				l.Debug.PushClip(ld.Area)
			}
			outline(l.Debug, l.Occluders, draw.LightRect(win.Size()), ld.View == draw.LightViewOccluders)
			if clip {
				l.Debug.PopClip()
			}
		}
	}
}

// glow draws sprite sp's glow, t its transform to its window's points,
// into its window's lighting, whose light rectangle is r.
func (s *system) glow(l *draw.Lighting, sp *twod.Sprite, t geom.Transform2D, r geom.Rect) {
	tex, region := sp.Picture()
	rect := sp.Rect()
	if tex == nil || region.W <= 0 || region.H <= 0 || rect.W <= 0 || rect.H <= 0 {
		return
	}
	reach := lit.Of(sp).Emissive
	pad := reach * region.W / rect.W
	k := glowKey{&tex.Texture, region, pad}
	if s.glows == nil {
		s.glows = map[glowKey]*render.Texture{}
	}
	g := s.glows[k]
	if g == nil {
		g = &render.Texture{}
		s.glows[k] = g
	}
	l.Glows[g] = draw.Glow{Source: k.source, Region: region, Pad: pad}
	grown := geom.Rect{X: rect.X - reach, Y: rect.Y - reach, W: rect.W + 2*reach, H: rect.H + 2*reach}
	origin := geom.V2(r.X, r.Y)
	l.Emissive.SetTransform(geom.Transform2D{X: t.X, Y: t.Y, Origin: t.Origin.Sub(origin)})
	l.Emissive.Image(grown, g, geom.Rect{W: 1, H: 1}, sp.Tint())
	l.Emissive.SetTransform(geom.Identity2D())
}

// cameraState is the light state of a window's camera, nil without one.
func cameraState(cam any) *lit.State {
	if e, ok := cam.(twod.Entity); ok {
		return lit.Of(e)
	}
	return nil
}

// plan works out the light pictures win draws besides its own, for the
// keys its lists drew, and tells once when some fall back past
// draw.MaxPictures.
func (s *system) plan(w *world.World, win *platform.Window, l *draw.Lighting) {
	var keys draw.Keys
	for _, list := range []*draw.List{draw.FrontFor(w, win), draw.BackFor(w, win)} {
		if list != nil {
			keys = keys.Or(list.Shades)
		}
	}
	for _, p := range draw.PassesFor(w, win) {
		if p.List != nil {
			keys = keys.Or(p.List.Shades)
		}
	}
	classes := draw.ClassesFor(w, win)
	l.Plan.Plan(l.Lights, classes, keys)
	fallen := fallenLayers(l.Plan.Fallen, classes)
	if fallen != s.fallen[win] {
		if s.fallen == nil {
			s.fallen = map[*platform.Window]string{}
		}
		s.fallen[win] = fallen
		if fallen != "" {
			log.Printf(
				"light: window %q needs more than %d light pictures for ShadedBy and Affects; "+
					"%s read the window's own, lit by every light",
				win.Title(),
				draw.MaxPictures,
				fallen,
			)
		}
	}
}

// fallenLayers names the layers whose keys fell back, by the innermost
// layer some light Affects that each is under.
func fallenLayers(keys []int, classes [][]any) string {
	var names []string
	for _, k := range keys {
		name := "drawings under no layer a light Affects"
		if c := k >> draw.OccluderGroups; c > 0 && c <= len(classes) {
			name = fmt.Sprintf("%v", classes[c-1][0])
			if e, ok := classes[c-1][0].(twod.Entity); ok && e.GetSceneNode().Name() != "" {
				name = "layer " + e.GetSceneNode().Name()
			}
		}
		if g := k & draw.AllGroups; g != 0 {
			name += fmt.Sprintf(" ShadedBy leaving out groups %03b", g)
		}
		if !slices.Contains(names, name) {
			names = append(names, name)
		}
	}
	return strings.Join(names, ", ")
}

// blockedOf is win's grid of where its occluders lie, over the light
// rectangle r, emptied the first time a frame asks for it.
func (s *system) blockedOf(win *platform.Window, r geom.Rect) *blocked {
	if s.blocked == nil {
		s.blocked, s.fresh = map[*platform.Window]*blocked{}, map[*platform.Window]bool{}
	}
	b := s.blocked[win]
	if b == nil {
		b = &blocked{}
		s.blocked[win] = b
	}
	if !s.fresh[win] {
		b.reset(r)
		s.fresh[win] = true
	}
	return b
}

// quad draws r, in a space t maps to the window's points, into an
// occluder list whose points start at the light rectangle's top-left, in
// the colour of its groups and height, in window points, and marks where
// in b, unless nil.
func quad(list *draw.List, b *blocked, t geom.Transform2D, r, light geom.Rect, groups uint8, height float32) {
	origin := geom.V2(light.X, light.Y)
	// Half an occluder texel more each side closes the gap a ray finds
	// between two occluders touching only at a corner.
	gx := occluderGrow / max(t.X.Len(), 1e-6)
	gy := occluderGrow / max(t.Y.Len(), 1e-6)
	r = geom.Rect{X: r.X - gx, Y: r.Y - gy, W: r.W + 2*gx, H: r.H + 2*gy}
	points := [4]geom.Vec2{
		t.Xform(geom.V2(r.X, r.Y)).Sub(origin), t.Xform(geom.V2(r.X+r.W, r.Y)).Sub(origin),
		t.Xform(geom.V2(r.X+r.W, r.Y+r.H)).Sub(origin), t.Xform(geom.V2(r.X, r.Y+r.H)).Sub(origin),
	}
	lo, hi := points[0], points[0]
	for _, p := range points[1:] {
		lo, hi = geom.V2(min(lo.X, p.X), min(lo.Y, p.Y)), geom.V2(max(hi.X, p.X), max(hi.Y, p.Y))
	}
	if hi.X < 0 || hi.Y < 0 || lo.X > light.W || lo.Y > light.H {
		return
	}
	list.Triangles(points[:], quadIndices, draw.OccluderColor(groups, height))
	if b != nil {
		b.add(geom.Rect{X: lo.X, Y: lo.Y, W: hi.X - lo.X, H: hi.Y - lo.Y}, groups)
	}
}

// screen is l as its window draws it, t its transform to the window's
// points, and whether it reaches the light rectangle r.
func (l *Light) screen(t geom.Transform2D, r geom.Rect) (draw.Light, bool) {
	scale := float32(math.Hypot(float64(t.X.X), float64(t.X.Y)))
	turn := func(radians float32) geom.Vec2 {
		d := geom.V2(float32(math.Cos(float64(radians))), float32(math.Sin(float64(radians))))
		d = t.X.Scale(d.X).Add(t.Y.Scale(d.Y))
		if n := d.Len(); n > 0 {
			return d.Scale(1 / n)
		}
		return geom.V2(1, 0)
	}
	dl := draw.Light{
		Kind:       l.kind,
		At:         t.Origin,
		Radius:     l.radius * scale,
		Falloff:    l.falloff,
		Color:      l.color.Scale(l.intensity),
		XRay:       l.xray,
		Groups:     l.groups,
		Elevation:  l.elevation * scale,
		ShadowSize: l.size * scale,
		ShadowFade: int(l.fade),
		FadeFrom:   l.fadeFrom,
		ShadowTint: l.shadowTint(),
	}
	for _, layer := range l.affects {
		dl.Affects = append(dl.Affects, layer)
	}
	switch l.kind {
	case draw.ConeLight:
		dl.Dir, dl.Spread = turn(l.angle), l.spread
	case draw.DirectionalLight:
		dl.Dir, dl.Length = turn(l.angle), l.length*scale
		return dl, true
	case draw.ShapeLight:
		dl.Shape, dl.Dir = l.shape.kind, turn(0)
		dl.Size = l.shape.size.Scale(scale)
		for _, p := range l.shape.points {
			dl.Points = append(dl.Points, t.Xform(p))
		}
		if !l.xray {
			for _, p := range l.shape.outline(shadowSamples) {
				dl.Samples = append(dl.Samples, t.Xform(p))
			}
		}
		if !reachesRect(r, dl.ShapeBounds(), dl.Radius) {
			return dl, false
		}
		return dl, true
	case draw.PointLight:
	}
	return dl, reaches(r, dl.At, dl.Radius)
}

// shadowSamples is how many points along its outline a shape light casts
// shadows from.
const shadowSamples = 8

// reachesRect reports whether b, grown by reach, meets r.
func reachesRect(r, b geom.Rect, reach float32) bool {
	return b.X-reach < r.X+r.W && b.X+b.W+reach > r.X && b.Y-reach < r.Y+r.H && b.Y+b.H+reach > r.Y
}

//nolint:gochecknoglobals // a lookup table
var quadIndices = []int{0, 1, 2, 0, 2, 3}

// occluderGrow is how far, in window points, an occluder is drawn past
// its edges: half a texel of the occluder picture.
const occluderGrow = 0.5 / draw.OccluderScale

// reaches reports whether a disc of radius round at reaches r.
func reaches(r geom.Rect, at geom.Vec2, radius float32) bool {
	dx := max(r.X-at.X, 0, at.X-(r.X+r.W))
	dy := max(r.Y-at.Y, 0, at.Y-(r.Y+r.H))
	return dx*dx+dy*dy <= radius*radius
}

// outline draws into c the edges of the occluder quads of list, in the
// light rectangle r's points, in their groups' colours, lightened when
// light is set so they show over the occluder view's fills: F3's F6.
func outline(c, list *draw.List, r geom.Rect, light bool) {
	if list == nil {
		return
	}
	origin := geom.V2(r.X, r.Y)
	vs := list.Vertices
	for i := 0; i+3 < len(vs); i += 4 {
		col := vs[i].Color
		col.A = 1
		if col.R+col.G+col.B == 0 {
			col = color.White
		}
		if light {
			col = color.RGB(0.6+0.4*col.R, 0.6+0.4*col.G, 0.6+0.4*col.B)
		}
		for k := range 4 {
			c.Line(vs[i+k].Pos.Add(origin), vs[i+(k+1)%4].Pos.Add(origin), 1, col)
		}
	}
}
