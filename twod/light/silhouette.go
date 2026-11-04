package light

import (
	"cmp"
	"math"
	"slices"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/internal/draw"
	"github.com/merlindorin/ember/internal/lit"
	"github.com/merlindorin/ember/twod"
)

// silhouetteLights is how many of the lights reaching a Silhouette sprite,
// the strongest at its foot, cast it; directional lights always do.
const silhouetteLights = 3

// silhouette is a Silhouette sprite as its window shows it: t maps its
// space to the window's points.
type silhouette struct {
	sprite *twod.Sprite
	t      geom.Transform2D
}

// silhouetteFrame is what silhouettes reuses from one frame to the next.
type silhouetteFrame struct {
	sizes   []geom.Vec2
	shelves []draw.Shelf
	casting []int
	nears   []near
	setOf   []int
	tiled   []int
	lists   listPool
}

// silhouettes fills l's Silhouettes with the shadows of sprites, in the
// points of the light rectangle r: a directional light's drawn into its
// own list, the others' handed to the GPU as quads, each light given a
// tile of the mask atlas for its reach, on a grid of the light picture's
// pixels, px a point, so a tile stands a whole number of them from where
// it shadows.
func silhouettes(l *draw.Lighting, sprites []silhouette, r geom.Rect, px float32, fr *silhouetteFrame) {
	fr.setOf = slices.Grow(fr.setOf[:0], len(l.Lights))[:len(l.Lights)]
	for i := range fr.setOf {
		fr.setOf[i] = -1
	}
	origin := geom.V2(r.X, r.Y)
	for _, s := range sprites {
		st := lit.Of(s.sprite)
		tex, region := s.sprite.Picture()
		rect := s.sprite.Rect()
		if tex == nil || region.W <= 0 || region.H <= 0 || rect.W <= 0 || rect.H <= 0 {
			continue
		}
		foot := s.t.Xform(geom.V2(rect.X+rect.W/2, rect.Y+rect.H))
		height := rect.H
		if st.Height > 0 {
			height = st.Height
		}
		height *= s.t.Y.Len()
		uv := corners(tex, region, s.sprite)
		fr.casting = casting(l.Lights, foot, st.Groups, fr.casting[:0], &fr.nears)
		for _, i := range fr.casting {
			if l.Lights[i].Kind == draw.DirectionalLight {
				t, ok := projection(l.Lights[i], foot, rect, height, s.t)
				if !ok {
					continue
				}
				n := sunSet(l, i, st.Groups, &fr.lists)
				l.Silhouettes[n].List.SetTransform(geom.Transform2D{X: t.X, Y: t.Y, Origin: t.Origin.Sub(origin)})
				l.Silhouettes[n].List.ImageCorners(rect, &tex.Texture, uv, color.White)
				l.Silhouettes[n].List.SetTransform(geom.Identity2D())
				continue
			}
			b, ok := shadowBounds(l.Lights[i], foot, height, rect.W*s.t.X.Len())
			if !ok {
				continue
			}
			b.X, b.Y = b.X-origin.X, b.Y-origin.Y
			n := fr.setOf[i]
			if n < 0 {
				n = len(l.Silhouettes)
				fr.setOf[i] = n
				l.Silhouettes = append(l.Silhouettes, draw.Silhouette{Light: i, Area: b})
			}
			l.Silhouettes[n].Groups |= st.Groups
			l.Silhouettes[n].Area = union(l.Silhouettes[n].Area, b)
			l.SilhouetteQuads = append(l.SilhouetteQuads, draw.SilhouetteQuad{
				Set: n, Groups: st.Groups, Texture: &tex.Texture,
				UV:   [4]float32{uv[0].X, uv[0].Y, uv[2].X, uv[2].Y},
				Rect: rect, Foot: foot.Sub(origin), Across: s.t.X.Len(), Height: height,
			})
		}
	}
	pic := geom.Rect{W: r.W, H: r.H}
	for n := range l.Silhouettes {
		if s := &l.Silhouettes[n]; s.List == nil {
			lit := reach(l.Lights[s.Light])
			lit.X, lit.Y = lit.X-origin.X, lit.Y-origin.Y
			s.Area = snap(clip(s.Area, lit, pic), 1/px)
		}
	}
	pack(l, 1/px, fr)
}

// shadowBounds is the rectangle round where a sprite standing at foot,
// height points tall and width wide, casts its shadow from light l, which
// the GPU projects: from its foot to its shadow's end, away from the
// light, half its width more each side.
func shadowBounds(l draw.Light, foot geom.Vec2, height, width float32) (geom.Rect, bool) {
	away := foot.Sub(l.At)
	d := away.Len()
	if d < 1e-3 {
		return geom.Rect{}, false
	}
	long := height
	if l.Elevation > height {
		long = d * height / (l.Elevation - height)
	}
	long = min(long, l.Radius-d)
	if long <= 0 {
		return geom.Rect{}, false
	}
	end := foot.Add(away.Scale(long / d))
	half := width/2 + 1
	x0, y0 := min(foot.X, end.X)-half, min(foot.Y, end.Y)-half
	x1, y1 := max(foot.X, end.X)+half, max(foot.Y, end.Y)+half
	return geom.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}, true
}

// union is the rectangle round a and b.
func union(a, b geom.Rect) geom.Rect {
	x0, y0 := min(a.X, b.X), min(a.Y, b.Y)
	x1, y1 := max(a.X+a.W, b.X+b.W), max(a.Y+a.H, b.Y+b.H)
	return geom.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}

// reach is the rectangle round what light l shines on, in its window's
// points.
func reach(l draw.Light) geom.Rect {
	if l.Kind == draw.ShapeLight {
		b := l.ShapeBounds()
		return geom.Rect{X: b.X - l.Radius, Y: b.Y - l.Radius, W: b.W + 2*l.Radius, H: b.H + 2*l.Radius}
	}
	return geom.Rect{X: l.At.X - l.Radius, Y: l.At.Y - l.Radius, W: 2 * l.Radius, H: 2 * l.Radius}
}

// clip is a within b and c, empty when they don't meet.
func clip(a, b, c geom.Rect) geom.Rect {
	x0, y0 := max(a.X, b.X, c.X), max(a.Y, b.Y, c.Y)
	x1, y1 := min(a.X+a.W, b.X+b.W, c.X+c.W), min(a.Y+a.H, b.Y+b.H, c.Y+c.H)
	return geom.Rect{X: x0, Y: y0, W: max(x1-x0, 0), H: max(y1-y0, 0)}
}

// snap grows a to the grid of cells g points square.
func snap(a geom.Rect, g float32) geom.Rect {
	if a.W <= 0 || a.H <= 0 {
		return a
	}
	down := func(v float32) float32 { return float32(math.Floor(float64(v/g))) * g }
	up := func(v float32) float32 { return float32(math.Ceil(float64(v/g))) * g }
	x0, y0 := down(a.X), down(a.Y)
	return geom.Rect{X: x0, Y: y0, W: up(a.X+a.W) - x0, H: up(a.Y+a.H) - y0}
}

// pack gives each light's silhouettes a tile of the mask atlas as large as
// its area, a page growing to hold the largest; g is the grid's cell.
func pack(l *draw.Lighting, g float32, fr *silhouetteFrame) {
	fr.sizes = fr.sizes[:0]
	side := float32(math.Ceil(float64(draw.SilhouetteSide/g))) * g
	tiled := fr.tiled[:0]
	defer func() { fr.tiled = tiled }()
	for n, s := range l.Silhouettes {
		if s.List != nil || s.Area.W <= 0 || s.Area.H <= 0 {
			continue
		}
		side = max(side, s.Area.W, s.Area.H)
		fr.sizes = append(fr.sizes, geom.V2(s.Area.W, s.Area.H))
		tiled = append(tiled, n)
	}
	if len(tiled) == 0 {
		return
	}
	l.SilhouetteSide = side
	fr.shelves = draw.Shelves(fr.sizes, side, fr.shelves[:0])
	for j, n := range tiled {
		l.Silhouettes[n].Page, l.Silhouettes[n].Tile = fr.shelves[j].Page, fr.shelves[j].At
	}
}

// sunSet is the index in l's Silhouettes of directional light i's set for
// groups, made with a list the first time.
func sunSet(l *draw.Lighting, i int, groups uint8, lists *listPool) int {
	for n, s := range l.Silhouettes {
		if s.Light == i && s.Groups == groups && s.List != nil {
			return n
		}
	}
	l.Silhouettes = append(l.Silhouettes, draw.Silhouette{Light: i, Groups: groups, List: lists.get()})
	return len(l.Silhouettes) - 1
}

// near is a light reaching a sprite and how strongly.
type near struct {
	i     int
	power float32
}

// casting appends to out the lights, by index, that cast a sprite of
// groups standing at foot: those its groups block with shadows, every
// directional one and the silhouetteLights strongest reaching it.
func casting(lights []draw.Light, foot geom.Vec2, groups uint8, out []int, nears *[]near) []int {
	top := (*nears)[:0]
	for i, l := range lights {
		if l.XRay || l.Groups&groups == 0 {
			continue
		}
		if l.Kind == draw.DirectionalLight {
			out = append(out, i)
			continue
		}
		d2 := foot.Sub(l.At).LenSq()
		if d2 >= l.Radius*l.Radius {
			continue
		}
		n := near{i, (l.Color.R + l.Color.G + l.Color.B) * (1 - float32(math.Sqrt(float64(d2)))/l.Radius)}
		at := len(top)
		for at > 0 && cmp.Compare(top[at-1].power, n.power) < 0 {
			at--
		}
		if at < silhouetteLights {
			if len(top) < silhouetteLights {
				top = append(top, near{})
			}
			copy(top[at+1:], top[at:len(top)-1])
			top[at] = n
		}
	}
	for _, n := range top {
		out = append(out, n.i)
	}
	*nears = top
	return out
}

// projection is the transform from a sprite's space, its picture rect,
// to the window's points that lays its picture on the ground as its
// shadow from l: its foot where it stands, its top away from the light as
// far as its shadow reaches, its width kept across; height is how tall it
// stands in points and t its own transform.
func projection(
	l draw.Light,
	foot geom.Vec2,
	rect geom.Rect,
	height float32,
	t geom.Transform2D,
) (geom.Transform2D, bool) {
	var dir geom.Vec2
	length := height
	switch l.Kind {
	case draw.DirectionalLight:
		dir, length = l.Dir, l.Length
	case draw.PointLight, draw.ConeLight, draw.ShapeLight:
		away := foot.Sub(l.At)
		d := away.Len()
		if d < 1e-3 {
			return geom.Transform2D{}, false
		}
		dir = away.Scale(1 / d)
		if l.Elevation > height {
			length = d * height / (l.Elevation - height)
		}
		length = min(length, l.Radius-d)
	}
	if length <= 0 || dir.Len() < 1e-6 {
		return geom.Transform2D{}, false
	}
	dir = dir.Scale(1 / dir.Len())
	across := geom.V2(dir.Y, -dir.X).Scale(t.X.Len())
	along := dir.Scale(-length / rect.H)
	footLocal := geom.V2(rect.X+rect.W/2, rect.Y+rect.H)
	origin := foot.Sub(across.Scale(footLocal.X)).Sub(along.Scale(footLocal.Y))
	return geom.Transform2D{X: across, Y: along, Origin: origin}, true
}

// corners are the texture points of a sprite's picture, region of tex in
// pixels, at its rect's corners, flipped as it is.
func corners(tex *twod.Texture, region geom.Rect, s *twod.Sprite) [4]geom.Vec2 {
	size := tex.Size()
	u0, v0 := region.X/size.X, region.Y/size.Y
	u1, v1 := (region.X+region.W)/size.X, (region.Y+region.H)/size.Y
	if fx, fy := s.Flip(); fx || fy {
		if fx {
			u0, u1 = u1, u0
		}
		if fy {
			v0, v1 = v1, v0
		}
	}
	return [4]geom.Vec2{{X: u0, Y: v0}, {X: u1, Y: v0}, {X: u1, Y: v1}, {X: u0, Y: v1}}
}

// listPool hands out lists for a frame, reused the next.
type listPool struct {
	lists []*draw.List
	used  int
}

func (p *listPool) reset() { p.used = 0 }

func (p *listPool) get() *draw.List {
	if p.used == len(p.lists) {
		l := &draw.List{}
		p.lists = append(p.lists, l)
	}
	l := p.lists[p.used]
	p.used++
	l.Reset(0)
	l.SetBlend(draw.Occlude)
	return l
}
