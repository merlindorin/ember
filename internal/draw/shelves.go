package draw

import "github.com/merlindorin/ember/geom"

// SilhouetteSide is a silhouette mask atlas page's side, in points, at
// least: a page grows to hold its largest tile. Tests make it small.
//
//nolint:gochecknoglobals // tests make it small, as its doc says
var SilhouetteSide float32 = 2048

// Shelf is a rectangle Shelves placed: its page and where on it.
type Shelf struct {
	Page int
	At   geom.Rect
}

// Shelves packs rectangles of the sizes given, in order, into pages side
// square: along a shelf as tall as its tallest, then a new shelf under
// it, then a new page. A size larger than a page is cut to it.
func Shelves(sizes []geom.Vec2, side float32, out []Shelf) []Shelf {
	page, x, y, shelf := 0, float32(0), float32(0), float32(0)
	for _, s := range sizes {
		w, h := min(s.X, side), min(s.Y, side)
		if x+w > side {
			x, y, shelf = 0, y+shelf, 0
		}
		if y+h > side {
			page, x, y, shelf = page+1, 0, 0, 0
		}
		out = append(out, Shelf{Page: page, At: geom.Rect{X: x, Y: y, W: w, H: h}})
		x += w
		shelf = max(shelf, h)
	}
	return out
}
