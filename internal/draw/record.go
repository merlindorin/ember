package draw

import (
	"slices"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
)

// state is what a List applies to everything drawn on it as it is drawn:
// a range recorded under one state replays only under the same.
type state struct {
	transform   geom.Transform2D
	transformed bool
	modulate    color.Color
	modulated   bool
	blend       Blend
	snap        bool
	light       LightMode
	left, class uint8
	clip        geom.Rect
	clipped     bool
	round       Rounded
}

func (c *List) state() state {
	clip, clipped := c.clip()
	return state{
		c.transform,
		c.transformed,
		c.modulate,
		c.modulated,
		c.blend,
		c.SnapVertices,
		c.light,
		c.left,
		c.class,
		clip,
		clipped,
		c.round(),
	}
}

// Mark is a point in a List, and the List's state there, to record what is
// drawn from it.
type Mark struct {
	vertices, indices, batches int
	state                      state
}

// Mark is where the List is now.
func (c *List) Mark() Mark {
	return Mark{len(c.Vertices), len(c.Indices), len(c.Batches), c.state()}
}

// Recording is a range of a List copied to be drawn again, as it was, at
// the same place under the same state: a UI host's last frame, replayed
// when nothing in it changed. It keeps the fonts and textures it drew
// alive; the GPU reads their versions as they are when replayed.
type Recording struct {
	vertices []Vertex
	// indices are as drawn, from base, the range's first vertex in the
	// list it was drawn in; batches' First count from its first index.
	indices []uint32
	base    uint32
	batches []Batch
	state   state
	ok      bool
}

// Ok reports whether r holds a range Replay can draw.
func (r *Recording) Ok() bool { return r.ok }

// Record copies what c drew since m into r, reusing r's room, and reports
// whether it can be replayed. It can't when c's state moved from m's, or
// when the range has lit vertices, a masked batch or one drawn alone (a
// pass's picture, a render texture): r then replays nothing.
func (c *List) Record(m Mark, r *Recording) bool {
	r.ok = false
	r.vertices, r.indices, r.batches = r.vertices[:0], r.indices[:0], r.batches[:0]
	if c.state() != m.state {
		return false
	}
	for _, v := range c.Vertices[m.vertices:] {
		if v.LightSpread != (geom.Vec2{}) {
			return false
		}
	}
	first := max(m.batches-1, 0)
	for _, b := range c.Batches[first:] {
		end := b.First + b.Count
		if end <= m.indices {
			continue
		}
		if b.Mask != nil || b.alone {
			return false
		}
		start := max(b.First, m.indices)
		b.First, b.Count = start-m.indices, end-start
		r.batches = append(r.batches, b)
	}
	r.vertices = append(r.vertices, c.Vertices[m.vertices:]...)
	r.indices = append(r.indices, c.Indices[m.indices:]...)
	r.base = vertexIndex(m.vertices)
	r.state, r.ok = m.state, true
	return true
}

// Replay draws r again at the end of c, merging its first batch into c's
// open one where drawing it would have, and reports whether it did. At
// the vertex it was recorded at, as a still frame is, its indices are
// copied as they are; elsewhere they are moved to c's end. It
// draws nothing, reporting false, when r holds nothing replayable or c's
// state is not the one r was recorded under.
func (c *List) Replay(r *Recording) bool {
	if !r.ok || c.state() != r.state {
		return false
	}
	base, at := vertexIndex(len(c.Vertices)), len(c.Indices)
	c.Vertices = append(c.Vertices, r.vertices...)
	if base == r.base {
		c.Indices = append(c.Indices, r.indices...)
	} else {
		for _, i := range r.indices {
			c.Indices = append(c.Indices, i-r.base+base)
		}
	}
	for k, b := range r.batches {
		b.First += at
		if k == 0 && c.merges(b) {
			open := &c.Batches[len(c.Batches)-1]
			if open.Atlas == nil {
				open.Atlas = b.Atlas
			}
			if open.ImageCount == 0 {
				open.Images, open.ImageCount = b.Images, b.ImageCount
			}
			open.Count += b.Count
			continue
		}
		c.Batches = append(c.Batches, b)
	}
	return true
}

// merges reports whether b can extend c's open batch: drawn right after
// it, with a font it can take, the same images in the same slots (or
// either without any), and the same blend, clip and mask.
func (c *List) merges(b Batch) bool {
	n := len(c.Batches)
	if n == 0 {
		return false
	}
	open := c.Batches[n-1]
	images := open.ImageCount == 0 || b.ImageCount == 0 ||
		open.ImageCount == b.ImageCount && slices.Equal(open.Images[:open.ImageCount], b.Images[:b.ImageCount])
	return !open.alone && open.Mask == nil && open.First+open.Count == b.First &&
		(open.Atlas == nil || b.Atlas == nil || open.Atlas == b.Atlas) && images &&
		open.Blend == b.Blend && open.Clipped == b.Clipped && open.Clip == b.Clip && open.Round == b.Round
}
