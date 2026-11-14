package layout

import (
	"slices"
	"strings"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// Layout lays the styled tree under root out in a viewport, measuring text with fonts.
func Layout(root *dom.Node, viewport geom.Vec2, fonts box.Fonts, opts ...Option) *box.Tree {
	return layoutIn(root, spaces{def(viewport.X), def(viewport.Y)}, fonts, false, opts...)
}

// Option configures a Layout.
type Option func(*tree)

// Images gives layout the natural sizes of replaced elements' pictures, such as an img's
// src; without it an img is empty.
func Images(i box.Images) Option { return func(t *tree) { t.images = i } }

// Retained keeps a layout's nodes for the next layout: an element whose subtree did not
// change since (no DirtyLayout or DirtyDescendant, the same style and children) keeps its
// nodes, caches and boxes, so only the path to what changed is laid out again; an element
// on that path, unchanged itself, is built again in its own node and box. The tree's Of is
// one map updated in place, its Dropped what each layout took out of it.
type Retained struct {
	nodes map[*dom.Node]*node
	of    map[*dom.Node]*box.Box
	gen   uint32
	unit  float32
	// escaped are the boxes the last layout marked Escapes, cleared before
	// the next marks its own, as kept boxes aren't written again.
	escaped []*box.Box
	// replaced and gone are the sweep's lists, kept for their room.
	replaced []keyed
	gone     []*node
}

// Retain lays out reusing r's nodes, and keeps this layout's in r.
func Retain(r *Retained) Option { return func(t *tree) { t.keep = r } }

// layoutIn lays root out; a retained layout in which a scroll container's
// gutter came or went lays out once more, the elements whose gutter changed
// built again with it, so the boxes that shrink to fit around them measure
// it, as Chrome's layout takes the scrollbar in at once.
func layoutIn(root *dom.Node, avail spaces, fonts box.Fonts, rounding bool, opts ...Option) *box.Tree {
	out, regutter := layoutOnce(root, avail, fonts, rounding, opts...)
	if len(regutter) == 0 {
		return out
	}
	for _, e := range regutter {
		dom.Data(e).Dirty |= dom.DirtyLayout
		for p := e.Parent(); p != nil; p = p.Parent() {
			dom.Data(p).Dirty |= dom.DirtyDescendant
		}
	}
	again, _ := layoutOnce(root, avail, fonts, rounding, opts...)
	written, dropped := again.Written, again.Dropped
	again.Written, again.Dropped = nil, nil
	for _, k := range slices.Concat(out.Written, written) {
		if _, ok := again.Of[k]; ok {
			again.Written = append(again.Written, k)
		}
	}
	for _, k := range slices.Concat(out.Dropped, dropped) {
		if _, ok := again.Of[k]; !ok {
			again.Dropped = append(again.Dropped, k)
		}
	}
	return again
}

// layoutOnce lays root out once, and lists the elements whose scrollbar
// gutter changed in a retained layout.
func layoutOnce(root *dom.Node, avail spaces, fonts box.Fonts, rounding bool, opts ...Option) (*box.Tree, []*dom.Node) {
	t := &tree{fonts: fonts, out: &box.Tree{Of: map[*dom.Node]*box.Box{}}}
	for _, o := range opts {
		o(t)
	}
	if k := t.keep; k != nil {
		if k.unit == unitsPerPoint(fonts) && k.nodes != nil {
			t.prevNodes, t.nodes, t.out.Of = k.nodes, k.nodes, k.of
		} else {
			t.nodes, t.lost = map[*dom.Node]*node{}, k.of
		}
		t.gen = k.gen + 1
		t.replaced, t.gone = k.replaced[:0], k.gone[:0]
		k.nodes, k.of, k.gen, k.unit = t.nodes, t.out.Of, t.gen, unitsPerPoint(fonts)
	}
	r := t.build(root, nil)
	defer t.sweep()
	if r == nil {
		return t.out, nil
	}
	t.root(r, avail)
	if rounding {
		roundLayout(r, point{})
	} else {
		keepLayout(r, point{})
	}
	t.out.Root = r.box
	t.write(r, nil)
	t.out.Live, t.out.Applied = t.out.Live[:0], t.out.Applied[:0]
	collectLive(t.out.Root, t.out)
	t.markEscapes(r)
	clean(root)
	return t.out, t.regutter
}

// clean clears n's layout flags and those below it, down the paths DirtyDescendant marks.
func clean(n *dom.Node) {
	d := dom.Data(n)
	below := d.Dirty&dom.DirtyDescendant != 0
	d.Dirty &^= dom.DirtyLayout | dom.DirtyDescendant
	if below {
		for _, c := range n.Children() {
			clean(c)
		}
	}
}

type tree struct {
	fonts  box.Fonts
	out    *box.Tree
	images box.Images
	// keep, when set, holds the nodes kept between layouts: prevNodes are the last
	// layout's, nodes this one's, the same map unless the unit changed, and gen counts
	// the layouts.
	keep      *Retained
	prevNodes map[*dom.Node]*node
	nodes     map[*dom.Node]*node
	gen       uint32
	// replaced are the keys rebuilt nodes held besides their elements, dropped unless
	// written again; gone the nodes rebuilt nodes held, dropped unless reached again;
	// lost the last layout's Of when this one started anew.
	replaced []keyed
	gone     []*node
	lost     map[*dom.Node]*box.Box
	// anonymousStyles is the style of the anonymous blocks of each parent style.
	anonymousStyles map[*style.Computed]*style.Computed
	// regutter are the elements whose scrollbar gutter changed in this
	// retained layout.
	regutter []*dom.Node
	// lookups, when set, counts the cache lookups of a layout, for tests.
	lookups *int
	// translate, when set, is what each text node says, from its own text.
	translate func(message string) string
}

type context uint8

const (
	contextLeaf context = iota
	contextNone
	contextBlock
	contextFlex
	contextGrid
	contextTable
)

// node is one box as the algorithms see it: an element, or an anonymous block around
// inline content.
type node struct {
	box  *box.Box
	st   *style.Computed
	kids []*node
	// inline is the text the node lays out in lines, when its content is inline.
	inline *inlineContent

	cache     cache
	unrounded placed
	final     placed
	absolute  point
	hoisted   []*node
	parent    *node
	grid      *gridInfo
	// row is a table row's cells' widths and heights, kept with the row;
	// captionRoom a table's captions' room above and below its grid.
	row         *tableRow
	captionRoom [2]float32
	// atomic marks an atomic inline, placed by its owner's lines; pendingOOF is what its
	// layout surfaced for an ancestor to place.
	atomic     bool
	pendingOOF []oofCandidate
	control    controlKind
	role       tableRole
	// replaced is an img, natural its picture's size (zero without one).
	replaced bool
	natural  size
	// gutter is the scrollbar room layout chose beyond the style's, overflow auto's once the
	// content overflows.
	gutter size
	// children are the element's children it was built from; fresh marks a node built in
	// this layout, performed the layout in which its own layout last ran, and reached the
	// last layout that built or kept it.
	children  []*dom.Node
	fresh     bool
	performed uint32
	reached   uint32
	// contents are the display: contents elements it flattened, with their boxes;
	// anonymous is set when it wraps inline content in anonymous blocks among its kids.
	contents  []keyed
	anonymous bool
	// readsParent is set when the node's style resolves against its parent's size, which
	// its cache then keys on.
	readsParent bool
}

func (n *node) context() context {
	switch {
	case n.st.Display == style.DisplayNone:
		return contextNone
	case n.role == roleTable && len(n.kids) > 0:
		return contextTable
	case len(n.kids) == 0 || n.inline != nil:
		return contextLeaf
	}
	switch n.st.Display {
	case style.FlexDisplay, style.InlineFlex:
		return contextFlex
	case style.GridDisplay, style.InlineGrid:
		return contextGrid
	case style.Inline, style.Block, style.InlineBlock, style.DisplayNone, style.Contents:
	}
	return contextBlock
}

// isBlockLeaf reports a childless node with block display, which Taffy treats as a block.
func (n *node) isBlockLeaf() bool {
	return len(n.kids) == 0 && isBlockDisplay(n.st.Display)
}

func isBlockDisplay(d style.Display) bool {
	return d == style.Block || d == style.InlineBlock || d == style.Inline
}

func styleOf(n *dom.Node) *style.Computed {
	if s := dom.Data(n).Style; s != nil {
		return s
	}
	s := style.Initial()
	return &s
}

// build makes the node of an element and its subtree, nil for display: contents (whose
// children the parent takes).
func (t *tree) build(e *dom.Node, parent *node) *node {
	if n := t.kept(e); n != nil {
		t.adopt(n, parent)
		return n
	}
	n := t.element(e, parent)
	if k := controlOf(e); k != noControl {
		t.buildControl(n, k)
		return n
	}
	if isTable(e) {
		t.buildTable(n, e)
		return n
	}
	if e.Tag() == dom.Img {
		t.buildImage(n, e)
		return n
	}
	if e.Tag() == dom.Canvas {
		buildCanvas(n, e)
		return n
	}
	t.buildChildren(n, e.Children())
	return n
}

// kept is e's node from the last layout when nothing under e changed since.
func (t *tree) kept(e *dom.Node) *node {
	n := t.prevNodes[e]
	if n == nil {
		return nil
	}
	d := dom.Data(e)
	if d.Dirty&(dom.DirtyLayout|dom.DirtyDescendant) != 0 || n.st != d.Style || !slices.Equal(n.children, e.Children()) {
		return nil
	}
	return n
}

// element is a node for e to build its subtree in: e's node from the last layout, taken
// again with its box and emptied, when e itself is unchanged and only something under it
// did, else a new one.
func (t *tree) element(e *dom.Node, parent *node) *node {
	st := styleOf(e)
	old := t.prevNodes[e]
	if n := old; n != nil && n.st == st && dom.Data(e).Dirty&dom.DirtyLayout == 0 &&
		slices.Equal(n.children, e.Children()) {
		t.retire(n, false)
		b := n.box
		*b = box.Box{Kind: box.Element, Node: e, Style: st, Children: b.Children[:0]}
		*n = node{box: b, st: st, parent: parent, fresh: true, readsParent: n.readsParent,
			kids: n.kids[:0], children: n.children, gutter: n.gutter, reached: t.gen}
		t.place(e, b)
		t.nodes[e] = n
		return n
	}
	b := &box.Box{Kind: box.Element, Node: e, Style: st}
	t.place(e, b)
	n := &node{box: b, st: st, parent: parent, fresh: true, readsParent: readsParent(st), reached: t.gen}
	if t.nodes != nil {
		if old != nil {
			t.retire(old, true)
			n.gutter = old.gutter
		}
		t.nodes[e] = n
		n.children = slices.Clone(e.Children())
	}
	return n
}

// place gives k the box b in the tree's Of.
func (t *tree) place(k *dom.Node, b *box.Box) {
	t.out.Of[k] = b
	t.out.Written = append(t.out.Written, k)
}

// keyed is a key of the tree's Of with the box a node wrote for it.
type keyed struct {
	key *dom.Node
	box *box.Box
}

// retire notes what old, the last layout's node of an element built again, held besides
// its element for sweep: its text runs, spans and display: contents boxes, its atomic
// inlines, and all its anonymous blocks held, as they are built anew each time; its kids
// too when
// they may be gone, which they can't be from an element of the same children holding no
// inline content or display: contents.
func (t *tree) retire(old *node, kids bool) {
	t.retireInline(old)
	if old.anonymous {
		for _, k := range old.kids {
			if k.box.Kind == box.AnonymousBlock {
				t.retireInline(k)
				t.gone = append(t.gone, k.kids...)
			}
		}
	}
	t.replaced = append(t.replaced, old.contents...)
	if kids || old.inline != nil || len(old.contents) > 0 {
		t.gone = append(t.gone, old.kids...)
	}
}

// retireInline notes old's text runs and spans, and its atomic inlines.
func (t *tree) retireInline(old *node) {
	c := old.inline
	if c == nil {
		return
	}
	for _, r := range c.runs {
		if r.atomic != nil {
			t.gone = append(t.gone, r.atomic.node)
		} else if r.box.Node != nil {
			t.replaced = append(t.replaced, keyed{r.box.Node, r.box})
		}
	}
	for _, s := range c.spans {
		t.replaced = append(t.replaced, keyed{s.box.Node, s.box})
	}
}

// sweep takes out of the tree's Of and nodes, once the layout is written, what it no
// longer reaches: a retired key nothing wrote again, a retired node's element no build or
// adoption reached, with all it held. With a new unit, every key of the last layout's
// not written again is lost.
func (t *tree) sweep() {
	keep := t.keep
	if keep == nil {
		return
	}
	defer func() { keep.replaced, keep.gone = t.replaced[:0], t.gone[:0] }()
	for _, k := range t.replaced {
		if t.out.Of[k.key] == k.box {
			t.drop(k.key)
		}
	}
	for _, n := range t.gone {
		t.dropUnreached(n)
	}
	for k := range t.lost {
		if _, ok := t.out.Of[k]; !ok {
			t.out.Dropped = append(t.out.Dropped, k)
		}
	}
}

func (t *tree) drop(k *dom.Node) {
	delete(t.out.Of, k)
	delete(t.nodes, k)
	t.out.Dropped = append(t.out.Dropped, k)
}

// dropUnreached drops n's element unless this layout reached it, and then everything n
// held.
func (t *tree) dropUnreached(n *node) {
	if e := n.box.Node; e != nil && n.box.Kind == box.Element {
		if now := t.nodes[e]; now != nil && now.reached == t.gen {
			return
		}
		if t.out.Of[e] == n.box {
			t.drop(e)
		} else if t.nodes[e] == n {
			delete(t.nodes, e)
		}
	}
	if c := n.inline; c != nil {
		for _, r := range c.runs {
			if r.atomic != nil {
				t.dropUnreached(r.atomic.node)
			} else if r.box.Node != nil && t.out.Of[r.box.Node] == r.box {
				t.drop(r.box.Node)
			}
		}
		for _, s := range c.spans {
			if t.out.Of[s.box.Node] == s.box {
				t.drop(s.box.Node)
			}
		}
	}
	for _, k := range n.contents {
		if t.out.Of[k.key] == k.box {
			t.drop(k.key)
		}
	}
	for _, k := range n.kids {
		t.dropUnreached(k)
	}
}

// adopt takes the kept node n, with its subtree, into this layout under parent.
func (t *tree) adopt(n *node, parent *node) {
	n.parent, n.reached = parent, t.gen
	if e := n.box.Node; dom.Data(e).Placed != n.box {
		t.out.Written = append(t.out.Written, e)
	}
}

// buildChildren gives n its in-flow children: elements, and the text between them either
// as n's own inline content or wrapped in anonymous blocks.
func (t *tree) buildChildren(n *node, children []*dom.Node) {
	items := t.flatten(n, children)
	container := n.st.Display
	inFlexOrGrid := container == style.FlexDisplay || container == style.InlineFlex ||
		container == style.GridDisplay || container == style.InlineGrid
	if !inFlexOrGrid && !hasBlockChild(items) {
		for _, it := range items {
			if it.element == nil {
				continue
			}
			if st := styleOf(it.element); outOfFlow(st.Position) || st.Display == style.DisplayNone {
				n.kids = append(n.kids, t.build(it.element, n))
			}
		}
		n.inline = t.inlineOf(n, items)
		return
	}
	var run []item
	flush := func() {
		if onlyCollapsibleSpace(run) {
			run = nil
			return
		}
		anon := t.anonymous(n)
		if content := t.inlineOf(anon, run); content != nil {
			anon.inline = content
			n.kids = append(n.kids, anon)
			n.anonymous = true
		}
		run = nil
	}
	for _, it := range items {
		if it.text != nil {
			run = append(run, it)
			continue
		}
		if !inFlexOrGrid && !elementBlockLevel(it.element) {
			run = append(run, it)
			continue
		}
		flush()
		n.kids = append(n.kids, t.build(it.element, n))
	}
	flush()
}

// onlyCollapsibleSpace reports a run of text nodes that white-space collapses away.
func onlyCollapsibleSpace(run []item) bool {
	for _, it := range run {
		if it.text == nil {
			return false
		}
		st := styleOf(it.text)
		if p := it.text.Parent(); p != nil {
			st = styleOf(p)
		}
		if !collapses(st.WhiteSpace) || !isCollapsibleSpace(it.text.Text()) {
			return false
		}
	}
	return true
}

type item struct {
	element *dom.Node
	text    *dom.Node
}

// flatten lists children with display: contents replaced by theirs, noted on owner.
func (t *tree) flatten(owner *node, children []*dom.Node) []item {
	var items []item
	for _, c := range children {
		if c.IsText() {
			items = append(items, item{text: c})
			continue
		}
		if styleOf(c).Display == style.Contents {
			b := &box.Box{Kind: box.Element, Node: c, Style: styleOf(c)}
			t.place(c, b)
			owner.contents = append(owner.contents, keyed{c, b})
			items = append(items, t.flatten(owner, c.Children())...)
			continue
		}
		items = append(items, item{element: c})
	}
	return items
}

func isBlockLevel(st *style.Computed) bool {
	switch st.Display {
	case style.Inline, style.InlineBlock, style.InlineFlex, style.InlineGrid:
		return outOfFlow(st.Position)
	case style.Block, style.FlexDisplay, style.GridDisplay, style.DisplayNone, style.Contents:
	}
	return true
}

// elementBlockLevel is isBlockLevel for an element, which a table always is.
func elementBlockLevel(e *dom.Node) bool { return isTable(e) || isBlockLevel(styleOf(e)) }

func hasBlockChild(items []item) bool {
	for _, it := range items {
		if it.element != nil {
			st := styleOf(it.element)
			if st.Display != style.DisplayNone && elementBlockLevel(it.element) && !outOfFlow(st.Position) {
				return true
			}
		}
	}
	return false
}

// anonymous is the block wrapping inline content in a flex, grid or block container: it
// inherits the text properties and nothing else.
func (t *tree) anonymous(parent *node) *node {
	st := t.anonymousStyles[parent.st]
	if st == nil {
		s := style.Initial()
		s.Display = style.Block
		s.Text = parent.st.Text
		st = &s
		if t.anonymousStyles == nil {
			t.anonymousStyles = map[*style.Computed]*style.Computed{}
		}
		t.anonymousStyles[parent.st] = st
	}
	b := &box.Box{Kind: box.AnonymousBlock, Style: st}
	return &node{box: b, st: st, parent: parent, fresh: true}
}

// write fills the box tree from the laid-out nodes.
func (t *tree) write(n *node, parent *node) {
	t.writeIn(n, parent, true)
}

// writeIn writes n's box and, when n is new or its layout ran in this one, rebuilds its
// children and text; a kept node laid out from its cache keeps them. placed says n's
// parent placed it this time: an atomic inline keeps its place in a kept span otherwise.
func (t *tree) writeIn(n *node, parent *node, placed bool) {
	b := n.box
	f := n.final
	rewrite := t.keep == nil || n.fresh || n.performed == t.gen
	n.fresh = false
	b.Size = geom.Vec2{X: f.size.w, Y: f.size.h}
	if n.st.Display != style.DisplayNone && (placed || !n.atomic) {
		origin := point{}
		if parent != nil {
			origin = parent.absolute
		}
		at := n.absolute
		b.Pos = geom.Vec2{X: at.x - origin.x, Y: at.y - origin.y}
		b.Sticky = geom.Vec2{}
	}
	b.Margin = box.Edges{Top: f.margin.t, Right: f.margin.r, Bottom: f.margin.b, Left: f.margin.l}
	b.Border = box.Edges{Top: f.border.t, Right: f.border.r, Bottom: f.border.b, Left: f.border.l}
	b.Padding = box.Edges{Top: f.padding.t, Right: f.padding.r, Bottom: f.padding.b, Left: f.padding.l}
	b.Content = geom.Vec2{X: f.overflow.rect.r - f.overflow.rect.l, Y: f.overflow.rect.b - f.overflow.rect.t}
	b.Scrollbar = geom.Vec2{X: f.scrollbar.w, Y: f.scrollbar.h}
	b.Wrap = box.Edges{Top: n.captionRoom[0], Bottom: n.captionRoom[1]}
	if !rewrite {
		for _, c := range n.kids {
			t.writeIn(c, n, false)
		}
		return
	}
	b.Children = b.Children[:0]
	if n.inline != nil {
		t.writeText(n)
	}
	for _, c := range n.kids {
		if c.st.Display != style.DisplayNone {
			b.Append(c.box)
		}
		t.writeIn(c, n, true)
	}
	t.atomsIntoSpans(n)
}

// keepLayout places every box without rounding, for fixtures that turn it off.
func keepLayout(n *node, cumulative point) { keepIn(n, cumulative, false) }

// keepIn is keepLayout under a display: none ancestor when hidden, where out-of-flow
// boxes take their parent's place too, as no containing block places them.
func keepIn(n *node, cumulative point, hidden bool) {
	n.final = n.unrounded
	n.final.scrollbar = n.final.scrollbar.add(n.gutter)
	n.absolute = point{cumulative.x + n.unrounded.location.x, cumulative.y + n.unrounded.location.y}
	hidden = hidden || n.st.Display == style.DisplayNone
	for _, c := range n.kids {
		if hidden || !outOfFlow(c.st.Position) || c.st.Display == style.DisplayNone {
			keepIn(c, n.absolute, hidden)
		}
	}
	for _, c := range n.hoisted {
		keepIn(c, n.absolute, hidden)
	}
}

func isCollapsibleSpace(s string) bool { return strings.TrimLeft(s, " \t\n\r\f") == "" }

// markEscapes sets Escapes on each box a containing block placed from
// above and the boxes between it and that block, the last layout's marks
// cleared first when the tree is kept.
func (t *tree) markEscapes(root *node) {
	var marked []*box.Box
	if k := t.keep; k != nil {
		for _, b := range k.escaped {
			b.Escapes = false
		}
		marked = k.escaped[:0]
	}
	var walk func(n *node)
	walk = func(n *node) {
		for _, h := range n.hoisted {
			for b := h.box; b != nil && b != n.box; b = b.Parent {
				b.Escapes = true
				marked = append(marked, b)
			}
		}
		for _, k := range n.kids {
			walk(k)
		}
	}
	walk(root)
	if t.keep != nil {
		t.keep.escaped = marked
	}
}
