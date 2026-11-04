package collide

import "github.com/merlindorin/ember/geom"

// hash is a spatial hash of bodies: a cell's entries are in the bucket its
// coordinates hash to, beside other cells' at times. Each entry carries
// its body's bounds as filed, so a query rejects it without touching the
// body. filed counts the entries, to know when to grow.
type hash struct {
	buckets [][]entry
	filed   int
}

// entry is a body filed in cell x, y, whose cells start at x0, y0.
type entry struct {
	b            *Body
	box          geom.Rect
	x, y, x0, y0 int32
}

func newHash() hash { return hash{buckets: make([][]entry, 1024)} }

func (h *hash) bucket(x, y int32) *[]entry {
	//nolint:gosec // a hash: the coordinates' bits are what count
	return &h.buckets[(uint32(x)*73856093^uint32(y)*19349663)&uint32(len(h.buckets)-1)]
}

func (h *hash) add(b *Body, box geom.Rect, r cellRange) {
	for y := r.y0; y <= r.y1; y++ {
		for x := r.x0; x <= r.x1; x++ {
			in := h.bucket(x, y)
			*in = append(*in, entry{b: b, box: box, x: x, y: y, x0: r.x0, y0: r.y0})
			h.filed++
		}
	}
}

func (h *hash) remove(b *Body, r cellRange) {
	for y := r.y0; y <= r.y1; y++ {
		for x := r.x0; x <= r.x1; x++ {
			in := h.bucket(x, y)
			for i := range *in {
				if (*in)[i].b == b {
					last := len(*in) - 1
					(*in)[i], (*in)[last] = (*in)[last], entry{}
					*in = (*in)[:last]
					h.filed--
					break
				}
			}
		}
	}
}

// update sets the bounds b is filed with in cells r to box.
func (h *hash) update(b *Body, box geom.Rect, r cellRange) {
	for y := r.y0; y <= r.y1; y++ {
		for x := r.x0; x <= r.x1; x++ {
			in := *h.bucket(x, y)
			for i := range in {
				if in[i].b == b {
					in[i].box = box
					break
				}
			}
		}
	}
}

// collect appends to out the bodies whose bounds touch area, which covers
// cells r, each once: from the first cell of r it shares with the body.
func (h *hash) collect(out []*Body, area geom.Rect, r cellRange) []*Body {
	for y := r.y0; y <= r.y1; y++ {
		for x := r.x0; x <= r.x1; x++ {
			for _, e := range *h.bucket(x, y) {
				if e.x == x && e.y == y && max(r.x0, e.x0) == x && max(r.y0, e.y0) == y && touches(area, e.box) {
					out = append(out, e.b)
				}
			}
		}
	}
	return out
}
