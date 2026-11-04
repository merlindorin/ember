package dom //nolint:testpackage // tests unexported internals

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func clean(nodes ...*Node) {
	for _, n := range nodes {
		n.Walk(func(d *Node) bool { Data(d).Dirty = 0; return true })
	}
}

func TestAppendReparentsAndMarksTheAncestors(t *testing.T) {
	a, b, c := NewElement(Div), NewElement(Section), NewElement(Span)
	a.Append(b)
	a.Append(c)
	b.Append(c)
	clean(a)
	assert.True(t, slices.Equal([]*Node{b}, a.Children()))
	assert.Same(t, b, c.Parent())

	c.AddClass("x")
	assert.Equal(t, DirtyStyle|DirtyTree, Data(c).Dirty)
	assert.Equal(t, DirtyDescendant, Data(b).Dirty)
	assert.Equal(t, DirtyDescendant, Data(a).Dirty)
}

func TestSettersMarkOnlyRealChanges(t *testing.T) {
	n := NewElement(Div)
	n.SetAttr("type", "text")
	n.AddClass("a")
	n.SetState(Hover, true)
	clean(n)
	n.SetAttr("type", "text")
	n.AddClass("a")
	n.SetState(Hover, true)
	n.SetStyle("")
	assert.Zero(t, Data(n).Dirty)
}

func TestAttrReadsIDClassAndStyleToo(t *testing.T) {
	n := NewElement(Input)
	n.SetAttr("id", "solid")
	n.SetAttr("class", "a  b")
	n.SetAttr("style", "width: 4px")
	n.SetAttr("type", "checkbox")
	for name, want := range map[string]string{"id": "solid", "class": "a b", "style": "width: 4px", "type": "checkbox"} {
		v, ok := n.Attr(name)
		assert.True(t, ok, name)
		assert.Equal(t, want, v, name)
	}
	assert.True(t, n.HasClass("b"))
	n.RemoveClass("a")
	assert.Equal(t, []string{"b"}, n.Classes())
}

func TestTextIsTheSubtreesText(t *testing.T) {
	p := NewElement(P)
	p.Append(NewText("a "), NewElement(Strong))
	p.Children()[1].SetText("b")
	assert.Equal(t, "a b", p.Text())
	p.SetText("c")
	assert.Equal(t, "c", p.Text())
	assert.Len(t, p.Children(), 1)
}

func TestTagNames(t *testing.T) {
	for tag := HTML; tag < tagCount; tag++ {
		got, ok := TagNamed(tag.String())
		assert.True(t, ok)
		assert.Equal(t, tag, got)
	}
	_, ok := TagNamed("center")
	assert.False(t, ok)
	assert.True(t, Input.Void())
}

func TestAShadowIsOutsideTheChildren(t *testing.T) {
	n := NewElement(Input)
	clean(n)
	n.SetShadow("Heron", false)
	s := Data(n).Shadow
	assert.Equal(t, "Heron", s.Text())
	assert.Same(t, n, s.Parent())
	assert.Empty(t, n.Children())
	assert.Equal(t, DirtyLayout, Data(n).Dirty)

	clean(n)
	n.SetShadow("Heron", false)
	assert.Zero(t, Data(n).Dirty)
	n.SetShadow("who", true)
	assert.True(t, Data(n).Placeholder)
	assert.Same(t, s, Data(n).Shadow)
}

func TestRowsGivenToATableGoInATbody(t *testing.T) {
	table, head, a, b := NewElement(Table), NewElement(Thead), NewElement(Tr), NewElement(Tr)
	table.Append(head, a)
	table.Append(b)
	require.Len(t, table.Children(), 2)
	body := table.Children()[1]
	assert.Equal(t, Tbody, body.Tag())
	assert.True(t, slices.Equal([]*Node{a, b}, body.Children()), "one implied tbody for consecutive rows")

	c := NewElement(Tr)
	head.Append(c)
	assert.Same(t, head, c.Parent(), "rows in a section stay there")
}

func TestAMarkReachesTheRootPastAFlaggedAncestor(t *testing.T) {
	a, b, c := NewElement(Div), NewElement(Section), NewElement(Span)
	a.Append(b)
	b.Append(c)
	clean(a)
	Data(b).Dirty = DirtyDescendant

	c.AddClass("x")
	assert.Equal(t, DirtyDescendant, Data(a).Dirty, "the root tells whether anything below changed")
}

func TestSettingAnElementsTextToWhatItHoldsChangesNothing(t *testing.T) {
	root := NewElement(Div)
	span := NewElement(Span)
	root.Append(span)
	span.SetText("HP 40")
	text := span.Children()[0]
	clean := func() {
		for _, n := range []*Node{root, span, text} {
			Data(n).Dirty = 0
		}
	}
	clean()
	span.SetText("HP 40")
	assert.Zero(t, Data(root).Dirty, "nothing to restyle or lay out")
	assert.Same(t, text, span.Children()[0], "the same text node kept")

	span.SetText("HP 39")
	assert.Same(t, text, span.Children()[0], "a new string changes the text node in place")
	assert.Equal(t, "HP 39", span.Text())
	assert.NotZero(t, Data(text).Dirty&DirtyLayout, "which lays out again")
	assert.NotZero(t, Data(root).Dirty&DirtyDescendant)
}
