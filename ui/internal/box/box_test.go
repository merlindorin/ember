package box_test

import (
	"testing"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/ui/internal/box"
	"github.com/stretchr/testify/assert"
)

func TestRectAddsTheParentsAndTheirScroll(t *testing.T) {
	root := &box.Box{Size: geom.V2(800, 600)}
	panel := &box.Box{Pos: geom.V2(10, 20), Size: geom.V2(200, 100), Scroll: geom.V2(0, 30)}
	row := &box.Box{Pos: geom.V2(5, 50), Size: geom.V2(190, 20)}
	root.Append(panel)
	panel.Append(row)
	assert.Equal(t, geom.Rect{X: 15, Y: 40, W: 190, H: 20}, row.Rect())
	assert.Same(t, root, panel.Parent)
}

func TestContentRectInsetsBorderAndPadding(t *testing.T) {
	b := &box.Box{Size: geom.V2(100, 50), Border: box.Edges{1, 2, 3, 4}, Padding: box.Edges{10, 10, 10, 10}}
	assert.Equal(t, geom.Rect{X: 14, Y: 11, W: 74, H: 26}, b.ContentRect())
}
