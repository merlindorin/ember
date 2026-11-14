// Lightpanels renders the Light pages' option pictures: one fixed crypt
// scene, a lamp at the yellow dot, a column, a crate and a low wall of
// crates, with one call changed per panel and written under it.
//
//	EMBER_HIDDEN=1 EMBER_CAPTURE=panel.png EMBER_CAPTURE_FRAME=10 \
//		go run ./examples/internal/lightpanels -panel falloff-0.5
package main

import (
	"flag"
	"fmt"
	"image"
	"log"
	"math"
	"sort"
	"strings"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/collide/tiles"
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/examples/internal/crypt"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/twod/light"
)

// The panel's size, in points.
const width, height = 320, 240

// lampAt is where a panel's lamp stands unless it says otherwise.
func lampAt() geom.Vec2 { return geom.V2(104, 150) }

// behind is right beside the column, so its shadow covers half the panel.
func behind() geom.Vec2 { return geom.V2(250, 112) }

// night is the ambient light of a panel that sets none.
func night() color.Color { return color.RGB(0.05, 0.05, 0.06) }

// warm is the lamps' colour.
func warm() color.Color { return color.RGB(1, 0.8, 0.55) }

// panel is one picture: the call it shows, where its lamp stands, the
// lights it makes, and what else of the scene it changes.
type panel struct {
	call   string
	at     geom.Vec2
	lights func(at geom.Vec2) []*light.Light
	// plot draws the falloff curve under the panel.
	plot    bool
	falloff float32
	// ambient replaces the night when not nil; ambientMap paints one.
	ambient    *color.Color
	ambientMap bool
	// floorUnlit leaves the floor's layer unlit; things are options on
	// the map of the wall, the column and the crate.
	floorUnlit bool
	things     []twod.TileMapOption
	// tiles are options on the crate's and the column's tileset.
	tiles []twod.TilesetOption
	// noOccluders leaves the crates and columns not solid: they cast
	// nothing.
	noOccluders bool
	sprites     func() []*twod.Sprite
	debug       light.DebugView
	outlines    bool
	// noDot leaves out the lamp's marker, for lights without one spot;
	// dots marks every light of the panel instead.
	noDot bool
	dots  bool
	// crowd, when not nil, adds a lit layer of monsters, each casting
	// from a box at its feet in occluder group 2, with these options.
	crowd []twod.Option
	// camera are more options of the camera.
	camera []twod.CameraOption
	// floorOnly makes the lights light the floor's layer alone.
	floorOnly bool
}

func point(radius float32, c color.Color, opts ...light.Option) func(geom.Vec2) []*light.Light {
	return func(at geom.Vec2) []*light.Light {
		return []*light.Light{light.NewPoint(radius, c, append([]light.Option{light.At(at.X, at.Y)}, opts...)...)}
	}
}

func rgb(r, g, b float32) *color.Color { c := color.RGB(r, g, b); return &c }

func main() {
	name := flag.String("panel", "falloff-0.5", "the panel to show: "+names())
	flag.Parse()
	p, ok := panels()[*name]
	if !ok {
		log.Fatalf("no panel %q; try one of %s", *name, names())
	}
	win := scene(p)
	w := ember.New().AddEntities(win)
	light.SetDebug(win, p.debug)
	light.SetOutlines(win, p.outlines)
	light.SetDebugArea(win, geom.Rect{W: width, H: height - 22})
	w.MustRun()
}

// stoneFloor is the panel's floor: 10×8 stones, their eight variants in
// a fixed pattern.
func stoneFloor() *twod.TileMap {
	var rows []string
	for y := range 8 {
		row := ""
		for x := range 10 {
			row += fmt.Sprint(1 + (x+y*3)%8)
		}
		rows = append(rows, row)
	}
	digits := map[rune]int{}
	for i := 1; i <= 8; i++ {
		digits[rune('0'+i)] = i
	}
	return twod.NewTileMap(crypt.Floor(), twod.ASCII(strings.Join(rows, "\n"), digits), twod.CellSize(32, 32))
}

// scene is the fixed crypt with p's changes: a low wall of crates along
// the top, a column and a crate right of the lamp, the call written along
// the bottom.
func scene(p panel) *platform.Window {
	at := p.at
	if at == (geom.Vec2{}) {
		at = lampAt()
	}
	win := platform.NewWindow(platform.Title("ember · light"), platform.Size(width, height), platform.FixedSize())

	floor := stoneFloor()
	ground := twod.NewLayer(-1, light.Lit())
	if p.floorUnlit {
		ground = twod.NewLayer(-1)
	}
	ground.Add(floor)
	room := twod.NewLayer(0, twod.SortByY(), light.Lit())
	tileOpts := []twod.TilesetOption{
		twod.Tall(crypt.Crate, crypt.Column), light.TileHeight(24, crypt.Crate), light.TileHeight(48, crypt.Column),
	}
	if !p.noOccluders {
		tileOpts = append(tileOpts, tiles.Solid(crypt.Crate, crypt.Column))
	}
	tileOpts = append(tileOpts, p.tiles...)
	set := twod.NewTileset(crypt.Things().Texture(), twod.Cell(16, 48), tileOpts...)
	layout := twod.ASCII(
		"..........\n"+
			".cccc.....\n"+
			"..........\n"+
			"......C...\n"+
			".....c....\n"+
			"..........\n"+
			"..........\n"+
			"..........", map[rune]int{'c': crypt.Crate, 'C': crypt.Column})
	thingOpts := append([]twod.TileMapOption{layout, twod.CellSize(32, 32), twod.TallIn(room)}, p.things...)
	things := twod.NewTileMap(set, thingOpts...)
	if p.sprites != nil {
		for _, s := range p.sprites() {
			room.Add(s)
		}
	}

	overlay := twod.NewLayer(1, twod.Fixed()).Add(
		twod.NewPolygon(box(width, 22), twod.At(0, height-22), twod.Color(color.RGB(0.05, 0.05, 0.08).Alpha(0.85))),
		twod.NewText(p.call, twod.At(8, height-18), twod.Pivot(0, 0), twod.FontSize(12)),
	)
	if p.lights != nil && p.dots {
		for _, l := range p.lights(at) {
			overlay.Add(
				twod.NewPolygon(dot(4), twod.At(l.Position().X, l.Position().Y), twod.Color(color.RGB(1, 0.85, 0.3))),
			)
		}
	}
	if p.lights != nil && !p.noDot {
		overlay.Add(twod.NewPolygon(dot(4), twod.At(at.X, at.Y), twod.Color(color.RGB(1, 0.85, 0.3))))
	}
	if p.plot {
		overlay.Add(plot(p.falloff))
	}

	ambient := night()
	if p.ambient != nil {
		ambient = *p.ambient
	}
	opts := []twod.CameraOption{twod.At(width/2, height/2), light.Ambient(ambient)}
	if p.ambientMap {
		opts = append(opts, light.AmbientMap(moonlit()))
	}
	opts = append(opts, p.camera...)
	win.Add(ground, room, things, overlay, twod.NewCamera(opts...))
	if p.crowd != nil {
		win.Add(crowd(p.crowd))
	}
	if p.lights != nil {
		for _, l := range p.lights(at) {
			if p.floorOnly {
				l.SetAffects(ground)
			}
			win.Add(l)
		}
	}
	return win
}

// crowd is three monsters right of the lamp on a lit layer of opts, each
// casting from a low box at its feet in occluder group 2, as the demo's
// horde does.
func crowd(opts []twod.Option) *twod.Layer {
	layer := twod.NewLayer(0, twod.SortByY(), light.Lit())
	for _, o := range opts {
		o(layer)
	}
	feet := twod.NewTexture(image.NewRGBA(image.Rect(0, 0, 10, 4)))
	for _, at := range []geom.Vec2{{X: 150, Y: 136}, {X: 186, Y: 166}, {X: 222, Y: 128}} {
		monster := twod.NewSprite(crypt.Sprite("orc_warrior_idle_anim_f0"), twod.At(at.X, at.Y), twod.Scale(2, 2))
		monster.Add(twod.NewSprite(feet, twod.At(0, 9.5), light.Occluder(2)))
		layer.Add(monster)
	}
	return layer
}

// moonlit is an ambient map: moonlight through a window over the left of
// the room, fading to dark at the right.
func moonlit() *twod.TiledPicture {
	return twod.NewTiledPicture(func(t twod.PictureTile) {
		for y := t.Area.Min.Y; y < t.Area.Max.Y; y++ {
			for x := t.Area.Min.X; x < t.Area.Max.X; x++ {
				cx, _ := t.Cell(x, y)
				v := float32(max(0, 0.5-0.1*float64(cx)))
				s := color.RGB(0.6*v, 0.7*v, v).SRGB()
				t.Image.Pix[t.Image.PixOffset(x, y)+0] = uint8(s[0] * 255)
				t.Image.Pix[t.Image.PixOffset(x, y)+1] = uint8(s[1] * 255)
				t.Image.Pix[t.Image.PixOffset(x, y)+2] = uint8(s[2] * 255)
				t.Image.Pix[t.Image.PixOffset(x, y)+3] = 255
			}
		}
	}, twod.TexelSize(32, 32))
}

// plot draws, in the panel's bottom right corner, the light's strength
// from the lamp (left) to its radius (right), full at the top.
func plot(falloff float32) *twod.Polygon {
	const w, h = 96, 48
	back := color.RGB(0.05, 0.05, 0.08).Alpha(0.85)
	return twod.NewPolygon(box(w+12, h+12), twod.At(208, height-22-66), twod.Color(back)).
		Add(
			twod.NewLine(
				[]geom.Vec2{{X: 0, Y: 0}, {X: 0, Y: h}, {X: w, Y: h}},
				twod.At(6, 6), twod.Width(1), twod.Color(color.Gray(0.5)),
			),
			twod.NewLine(strength(falloff, w, h), twod.At(6, 6), twod.Width(2), twod.Color(color.RGB(1, 0.85, 0.3))),
		)
}

// strength is the light's curve, w wide and h tall, y down: 1 −
// smoothstep(falloff, 1, d) for d from the lamp to the radius, as the GPU
// lights it.
func strength(falloff, w, h float32) []geom.Vec2 {
	var pts []geom.Vec2
	for i := 0; i <= 48; i++ {
		d := float32(i) / 48
		t := float32(0)
		switch {
		case falloff >= 1:
			if d >= 1 {
				t = 1
			}
		default:
			t = min(max((d-falloff)/(1-falloff), 0), 1)
		}
		pts = append(pts, geom.V2(d*w, h*t*t*(3-2*t)))
	}
	return pts
}

// box is a rectangle w by h from its top left corner.
func box(w, h float32) []geom.Vec2 {
	return []geom.Vec2{{X: 0, Y: 0}, {X: w, Y: 0}, {X: w, Y: h}, {X: 0, Y: h}}
}

// dot is a small round marker, r points from its centre.
func dot(r float32) []geom.Vec2 {
	var pts []geom.Vec2
	for i := range 12 {
		a := 2 * math.Pi * float64(i) / 12
		pts = append(pts, geom.V2(r*float32(math.Cos(a)), r*float32(math.Sin(a))))
	}
	return pts
}

func names() string {
	var all []string
	for n := range panels() {
		all = append(all, n)
	}
	sort.Strings(all)
	return strings.Join(all, ", ")
}
