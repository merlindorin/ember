package main

import (
	"strconv"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/ui"
)

// Orb is an element of the game's own: a round flask filled to the hero's
// health, drawn by the game on a ui Canvas, so the layout places it
// like any element. Its role and value tell automation what it shows.
type Orb struct {
	doc         *ui.Document
	node        *ui.Node
	health, max float32
}

// NewOrb returns a full orb, with any of the usual ui arguments.
func NewOrb(doc *ui.Document, full float32, args ...ui.Arg) *Orb {
	o := &Orb{doc: doc, health: full, max: full}
	o.node = ui.Canvas(o.draw, append([]ui.Arg{ui.Attr("role", "meter")}, args...)...)
	o.show()
	return o
}

func (o *Orb) Health() float32 { return o.health }

// SetHealth fills the orb to h, within 0 and its maximum, and has it
// drawn again.
func (o *Orb) SetHealth(h float32) {
	o.health = geom.Clamp(h, 0, o.max)
	o.show()
	o.doc.Invalidate(o.node)
}

func (o *Orb) show() {
	o.node.SetAttr("aria-valuenow", strconv.FormatFloat(float64(o.health), 'f', -1, 32))
}

// draw fills the orb, in its own space: c.Size() is the size the layout
// gave it.
func (o *Orb) draw(c *twod.Canvas) {
	size := c.Size()
	c.Orb(geom.V2(size.X/2, size.Y/2), min(size.X, size.Y)/2, o.health/o.max, color.Hex("#c0392b"))
}
