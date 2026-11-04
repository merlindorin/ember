package dom

import (
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/ui/internal/style"
)

// Dirty says what a change needs redone.
type Dirty uint8

const (
	// DirtyStyle: the node's own style is to recompute (its id, classes,
	// attributes, style or state changed, or it was added).
	DirtyStyle Dirty = 1 << iota
	// DirtyTree: siblings or descendants may match other selectors now
	// (a child added or removed; an id, class or attribute changed, which
	// Changed names).
	DirtyTree
	// DirtyLayout: the node's box is to lay out again (its text, children
	// or attributes changed, or css found a change layout reads).
	DirtyLayout
	// DirtyDescendant: some node below is dirty; once css has run, some
	// node below needs layout.
	DirtyDescendant
	// DirtyState: the node's pseudo-class state changed; its descendants
	// may match other selectors when some combinator reads that state.
	DirtyState
)

// Children is the Changed key of a child added or removed.
const Children = "+"

// Pipeline is what the stages keep per node. css writes Style and clears
// DirtyStyle, DirtyTree and DirtyState; layout clears DirtyLayout and
// DirtyDescendant.
type Pipeline struct {
	// Changed names what changed with DirtyTree, until css next runs:
	// ".class", "#id" and "[attribute" for the element's own, and Children
	// for its children added or removed.
	Changed []string
	Dirty   Dirty
	// StyledState is the node's state when css last styled it.
	StyledState State
	// Style is the node's computed style; a text node's is its parent's.
	Style *style.Computed
	// Shown is what paint draws: Style itself, or a copy with running
	// transitions' values while any run; a text node's is its parent's.
	Shown *style.Computed
	// Placed is the node's principal box after layout, which Node.Box
	// asks for its border box in the document's space.
	Placed interface{ Rect() geom.Rect }
	// Scroll is a scroll container's offset, kept across layouts, which
	// clamp it to the scrollable overflow and copy it to the Box.
	Scroll geom.Vec2
	// Shadow is the text an input, textarea or select shows: its value,
	// the selected option's label, or its placeholder when Placeholder is
	// set. The Document keeps it; layout lays it out as the element's only
	// content, in place of its children, and export never writes it.
	Shadow      *Node
	Placeholder bool
	// Canvas is a canvas element's machine, made with it since it holds
	// the game's drawing, for the Document to take as the element's.
	Canvas any
	// Handlers are the node's callbacks (decision 0057), kept by ui.
	Handlers any
	// Follow is the entity the node follows on screen, kept by ui.
	Follow any
}

// Data is the pipeline data of n.
func Data(n *Node) *Pipeline { return &n.pipe }
