package look

import (
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/twod"
)

// The UI pack's large tiles are 32 pixels with an 8-pixel frame, drawn at
// px points a pixel like the HUD's.
const (
	tilePx = 32
	edgePx = 8
	px     = 2
)

// FrameWidth is how far a board's frame reaches outside the board.
const FrameWidth = edgePx * px

// Skin draws the boards in the HUD's look: a wooden tray, parchment pieces
// and Silkscreen for numbers.
type Skin struct {
	wood, parchment, plate *twod.Texture
	// Face is Silkscreen, the HUD's pixel font, for text on a board.
	Face *twod.Face
}

// LoadSkin reads the tiles and the font the HUD's stylesheet uses.
func LoadSkin() *Skin {
	load := func(name string) *twod.Texture {
		tex, err := twod.LoadTexture(skin, "adventure/"+name, twod.Pixelated())
		if err != nil {
			panic(err) // embedded files that always decode
		}
		return tex
	}
	face, err := twod.LoadFace(skin, "adventure/silkscreen.ttf")
	if err != nil {
		panic(err)
	}
	return &Skin{wood: load("wood.png"), parchment: load("parchment.png"), plate: load("plate.png"), Face: face}
}

// Tray draws the wooden tray over r, its frame FrameWidth outside it.
func (s *Skin) Tray(c *twod.Canvas, r geom.Rect) {
	patch(c, s.wood, geom.Rect{X: r.X - FrameWidth, Y: r.Y - FrameWidth, W: r.W + 2*FrameWidth, H: r.H + 2*FrameWidth})
}

// Board draws the tray around r and sinks r into it, the dark surface a
// game draws on.
func (s *Skin) Board(c *twod.Canvas, r geom.Rect) {
	s.Tray(c, r)
	s.Slot(c, r)
}

// Slot draws r sunk into the tray: darker wood, shaded under its top and
// left edges.
func (s *Skin) Slot(c *twod.Canvas, r geom.Rect) {
	c.Rect(r, color.Hex(Cell))
	c.Rect(geom.Rect{X: r.X, Y: r.Y, W: r.W, H: 3 * px}, color.Hex(Surface))
	c.Rect(geom.Rect{X: r.X, Y: r.Y, W: 2 * px, H: r.H}, color.Hex(Surface))
}

// Piece draws the parchment tile in r.
func (s *Skin) Piece(c *twod.Canvas, r geom.Rect) { patch(c, s.parchment, r) }

// Plate draws the parchment tile with metal corners in r, for the best piece.
func (s *Skin) Plate(c *twod.Canvas, r geom.Rect) { patch(c, s.plate, r) }

// patch draws a tile as a nine-patch over r, its frame px points a pixel,
// its middle stretched.
func patch(c *twod.Canvas, tex *twod.Texture, r geom.Rect) {
	e := min(float32(FrameWidth), r.W/2, r.H/2)
	xs := [4]float32{r.X, r.X + e, r.X + r.W - e, r.X + r.W}
	ys := [4]float32{r.Y, r.Y + e, r.Y + r.H - e, r.Y + r.H}
	edge := float32(edgePx) / tilePx
	us := [4]float32{0, edge, 1 - edge, 1}
	for j := range 3 {
		for i := range 3 {
			c.Image(geom.Rect{X: xs[i], Y: ys[j], W: xs[i+1] - xs[i], H: ys[j+1] - ys[j]}, tex,
				geom.Rect{X: us[i], Y: us[j], W: us[i+1] - us[i], H: us[j+1] - us[j]}, color.Hex("#ffffff"))
		}
	}
}
