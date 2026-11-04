// Package dom is ui's node tree, shared by css, layout, paint and
// export. The root package aliases Node, so games build and change trees
// through it; the pipeline's per-node data is reached only through Data, which
// games cannot import.
package dom

import (
	"slices"
	"strings"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/ui/internal/style"
)

// Attr is one attribute, kept in the order it was set.
type Attr struct{ Name, Value string }

// Node is an element or a text node.
type Node struct {
	tag      Tag
	text     string
	id       string
	classes  []string
	attrs    []Attr
	style    string
	state    State
	parent   *Node
	children []*Node
	pipe     Pipeline
}

// NewElement is an element of tag with no attributes or children.
func NewElement(tag Tag) *Node {
	return &Node{tag: tag, pipe: Pipeline{Dirty: DirtyStyle}}
}

// NewText is a text node.
func NewText(s string) *Node {
	return &Node{tag: TextTag, text: s, pipe: Pipeline{Dirty: DirtyStyle}}
}

func (n *Node) Tag() Tag          { return n.tag }
func (n *Node) IsText() bool      { return n.tag == TextTag }
func (n *Node) ID() string        { return n.id }
func (n *Node) Classes() []string { return n.classes }
func (n *Node) Attrs() []Attr     { return n.attrs }
func (n *Node) Style() string     { return n.style }
func (n *Node) State() State      { return n.state }
func (n *Node) Parent() *Node     { return n.parent }
func (n *Node) Children() []*Node { return n.children }

// Box is the element's border box after the last layout, in the
// document's space, the rectangle Chrome's getBoundingClientRect gives:
// none under a display: none element.
func (n *Node) Box() geom.Rect {
	if n.pipe.Placed == nil {
		return geom.Rect{}
	}
	for p := n; p != nil; p = p.parent {
		if s := p.pipe.Style; s != nil && s.Display == style.DisplayNone {
			return geom.Rect{}
		}
	}
	return n.pipe.Placed.Rect()
}

// Apply makes the node a child of parent, so nodes go among element
// options.
func (n *Node) Apply(parent *Node) { parent.Append(n) }

func (n *Node) HasClass(c string) bool { return slices.Contains(n.classes, c) }

// Text is a text node's text, or the text of an element's subtree.
func (n *Node) Text() string {
	if n.tag == TextTag {
		return n.text
	}
	var b strings.Builder
	n.Walk(func(d *Node) bool {
		if d.tag == TextTag {
			b.WriteString(d.text)
		}
		return true
	})
	return b.String()
}

// Attr is an attribute's value, with id, class and style among them.
func (n *Node) Attr(name string) (string, bool) {
	switch name {
	case "id":
		return n.id, n.id != ""
	case "class":
		return strings.Join(n.classes, " "), len(n.classes) > 0
	case "style":
		return n.style, n.style != ""
	}
	for _, a := range n.attrs {
		if a.Name == name {
			return a.Value, true
		}
	}
	return "", false
}

// Index is the node's place among its parent's children, -1 without one.
func (n *Node) Index() int {
	if n.parent == nil {
		return -1
	}
	return slices.Index(n.parent.children, n)
}

// Walk visits n and its descendants depth first, in document order,
// skipping a node's subtree when fn returns false.
func (n *Node) Walk(fn func(*Node) bool) {
	if !fn(n) {
		return
	}
	for _, c := range n.children {
		c.Walk(fn)
	}
}

// SetText replaces a text node's text, or an element's children with one
// text node; an element holding only a text node has that node's text set,
// so the same string again changes nothing.
func (n *Node) SetText(s string) {
	if n.tag == TextTag {
		if n.text != s {
			n.text = s
			n.mark(DirtyLayout)
		}
		return
	}
	if len(n.children) == 1 && n.children[0].tag == TextTag {
		n.children[0].SetText(s)
		return
	}
	for _, c := range n.children {
		c.parent, c.pipe.Placed = nil, nil
	}
	n.children = nil
	n.Append(NewText(s))
}

func (n *Node) SetID(id string) {
	if n.id != id {
		n.changed("#"+n.id, "#"+id)
		n.id = id
		n.mark(DirtyStyle | DirtyTree)
	}
}

// SetAttr sets an attribute; id, class and style go to their own fields.
func (n *Node) SetAttr(name, value string) {
	switch name {
	case "id":
		n.SetID(value)
		return
	case "class":
		now := strings.Fields(value)
		for _, c := range n.classes {
			if !slices.Contains(now, c) {
				n.changed("." + c)
			}
		}
		for _, c := range now {
			if !slices.Contains(n.classes, c) {
				n.changed("." + c)
			}
		}
		n.classes = now
		n.mark(DirtyStyle | DirtyTree)
		return
	case "style":
		n.SetStyle(value)
		return
	}
	for i, a := range n.attrs {
		if a.Name == name {
			if a.Value != value {
				n.attrs[i].Value = value
				n.changed("[" + name)
				n.mark(DirtyStyle | DirtyTree | DirtyLayout)
			}
			return
		}
	}
	n.attrs = append(n.attrs, Attr{name, value})
	n.changed("[" + name)
	n.mark(DirtyStyle | DirtyTree | DirtyLayout)
}

func (n *Node) RemoveAttr(name string) {
	i := slices.IndexFunc(n.attrs, func(a Attr) bool { return a.Name == name })
	if i >= 0 {
		n.attrs = slices.Delete(n.attrs, i, i+1)
		n.changed("[" + name)
		n.mark(DirtyStyle | DirtyTree | DirtyLayout)
	}
}

func (n *Node) SetStyle(css string) {
	if n.style != css {
		n.style = css
		n.mark(DirtyStyle)
	}
}

func (n *Node) AddClass(c string) {
	if !n.HasClass(c) {
		n.classes = append(n.classes, c)
		n.changed("." + c)
		n.mark(DirtyStyle | DirtyTree)
	}
}

func (n *Node) RemoveClass(c string) {
	if i := slices.Index(n.classes, c); i >= 0 {
		n.classes = slices.Delete(n.classes, i, i+1)
		n.changed("." + c)
		n.mark(DirtyStyle | DirtyTree)
	}
}

// SetState turns the pseudo-classes in f on or off.
func (n *Node) SetState(f State, on bool) {
	s := n.state &^ f
	if on {
		s |= f
	}
	if s != n.state {
		n.state = s
		n.mark(DirtyStyle | DirtyState)
	}
}

// Append adds children at the end, taking them from any parent they had.
//
// A row given to a table goes in its last child when that is a tbody,
// else in a new tbody, as HTML's parser places it.
func (n *Node) Append(children ...*Node) {
	for _, c := range children {
		if n.tag == Table && c.tag == Tr {
			body := n.lastChild()
			if body == nil || body.tag != Tbody {
				body = NewElement(Tbody)
				n.Append(body)
			}
			body.Append(c)
			continue
		}
		if c.parent != nil {
			c.parent.Remove(c)
		}
		c.parent = n
		n.children = append(n.children, c)
		c.mark(DirtyStyle)
	}
	n.changed(Children)
	n.mark(DirtyTree | DirtyLayout)
}

func (n *Node) lastChild() *Node {
	if len(n.children) == 0 {
		return nil
	}
	return n.children[len(n.children)-1]
}

// Remove takes a child out.
func (n *Node) Remove(c *Node) {
	if i := slices.Index(n.children, c); i >= 0 {
		n.children = slices.Delete(n.children, i, i+1)
		c.parent, c.pipe.Placed = nil, nil
		n.changed(Children)
		n.mark(DirtyTree | DirtyLayout)
	}
}

// changed notes keys among what changed for css, once each.
func (n *Node) changed(keys ...string) {
	for _, k := range keys {
		if !slices.Contains(n.pipe.Changed, k) {
			n.pipe.Changed = append(n.pipe.Changed, k)
		}
	}
}

// mark flags the node and tells its ancestors a descendant changed.
func (n *Node) mark(d Dirty) {
	n.pipe.Dirty |= d
	for p := n.parent; p != nil; p = p.parent {
		p.pipe.Dirty |= DirtyDescendant
	}
}

// SetShadow sets the text n shows in place of its children, its parent n
// but not among n's children; placeholder marks a placeholder's.
func (n *Node) SetShadow(text string, placeholder bool) {
	s := n.pipe.Shadow
	if s == nil {
		s = NewText(text)
		s.parent = n
		n.pipe.Shadow = s
		n.mark(DirtyLayout)
	} else if s.text != text {
		s.text = text
		n.mark(DirtyLayout)
	}
	if n.pipe.Placeholder != placeholder {
		n.pipe.Placeholder = placeholder
		n.mark(DirtyLayout)
	}
}
