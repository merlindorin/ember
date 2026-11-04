package main

import (
	"maps"
	"math"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/ui"
)

// element builds one element's page, shown in a window w points wide and
// as tall as its content.
type element struct {
	build func(d *ui.Document) *ui.Node
	w     int
}

// size is the window doc needs: e's width, and the page's height, its
// padding included.
func (e element) size(doc *ui.Document) (int, int) {
	doc.Layout(geom.V2(float32(e.w), 4000))
	return e.w, int(math.Ceil(float64(bottom(doc.Body()))))
}

// bottom is the lowest edge of n and everything under it.
func bottom(n *ui.Node) float32 {
	b := n.Box()
	low := b.Y + b.H
	if b.H == 0 {
		low = 0
	}
	for _, c := range n.Children() {
		low = max(low, bottom(c))
	}
	return low
}

// elements are ui's elements by tag, each built in the file of its group.
func elements() map[string]element {
	all := map[string]element{}
	for _, group := range []map[string]element{blocks(), texts(), controls(), structure()} {
		maps.Copy(all, group)
	}
	return all
}
