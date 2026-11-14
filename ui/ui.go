// Package ui builds UI as modern HTML elements styled by a CSS subset,
// written from Go and never parsed (decision 0049, docs/design/ui.md).
// A Document holds one tree per window; its nodes are laid out the way
// Chrome lays out the same page, which ui/export writes.
package ui

import (
	"strconv"
	"strings"

	"github.com/lantervale/ember/ui/internal/dom"
)

// Node is an element or a text node of a Document's tree.
type Node = dom.Node

// Arg is what an element's constructor takes: an attribute, or a child,
// since a *Node is one.
type Arg interface{ Apply(*Node) }

type option func(*Node)

func (o option) Apply(n *Node) { o(n) }

func element(tag dom.Tag, args []Arg) *Node {
	n := dom.NewElement(tag)
	for _, a := range args {
		a.Apply(n)
	}
	return n
}

func Div(args ...Arg) *Node     { return element(dom.Div, args) }
func Section(args ...Arg) *Node { return element(dom.Section, args) }
func Header(args ...Arg) *Node  { return element(dom.Header, args) }
func Footer(args ...Arg) *Node  { return element(dom.Footer, args) }
func Nav(args ...Arg) *Node     { return element(dom.Nav, args) }
func Main(args ...Arg) *Node    { return element(dom.Main, args) }
func Aside(args ...Arg) *Node   { return element(dom.Aside, args) }
func Article(args ...Arg) *Node { return element(dom.Article, args) }
func P(args ...Arg) *Node       { return element(dom.P, args) }
func H1(args ...Arg) *Node      { return element(dom.H1, args) }
func H2(args ...Arg) *Node      { return element(dom.H2, args) }
func H3(args ...Arg) *Node      { return element(dom.H3, args) }
func H4(args ...Arg) *Node      { return element(dom.H4, args) }
func Span(args ...Arg) *Node    { return element(dom.Span, args) }
func Strong(args ...Arg) *Node  { return element(dom.Strong, args) }
func Em(args ...Arg) *Node      { return element(dom.Em, args) }
func Code(args ...Arg) *Node    { return element(dom.Code, args) }
func Kbd(args ...Arg) *Node     { return element(dom.Kbd, args) }
func Samp(args ...Arg) *Node    { return element(dom.Samp, args) }
func Pre(args ...Arg) *Node     { return element(dom.Pre, args) }

// Label is a label element. As in HTML, a click on it focuses and clicks
// the control it names (For's id, else the first control inside it), and
// its text names that control for automation.
func Label(args ...Arg) *Node    { return element(dom.Label, args) }
func Button(args ...Arg) *Node   { return element(dom.Button, args) }
func Select(args ...Arg) *Node   { return element(dom.Select, args) }
func Textarea(args ...Arg) *Node { return element(dom.Textarea, args) }
func Details(args ...Arg) *Node  { return element(dom.Details, args) }
func Summary(args ...Arg) *Node  { return element(dom.Summary, args) }
func Dialog(args ...Arg) *Node   { return element(dom.Dialog, args) }
func Table(args ...Arg) *Node    { return element(dom.Table, args) }
func Thead(args ...Arg) *Node    { return element(dom.Thead, args) }
func Tbody(args ...Arg) *Node    { return element(dom.Tbody, args) }
func Tfoot(args ...Arg) *Node    { return element(dom.Tfoot, args) }
func Caption(args ...Arg) *Node  { return element(dom.Caption, args) }
func Tr(args ...Arg) *Node       { return element(dom.Tr, args) }
func Th(args ...Arg) *Node       { return element(dom.Th, args) }
func Td(args ...Arg) *Node       { return element(dom.Td, args) }
func Progress(args ...Arg) *Node { return element(dom.Progress, args) }
func Meter(args ...Arg) *Node    { return element(dom.Meter, args) }
func Img(args ...Arg) *Node      { return element(dom.Img, args) }
func Hr(args ...Arg) *Node       { return element(dom.Hr, args) }

// Option is an option element of a select.
func Option(args ...Arg) *Node { return element(dom.Option, args) }

// Input is an input, of Type "text" unless set.
func Input(args ...Arg) *Node { return element(dom.Input, args) }

// ID sets the id attribute, read back with Node.ID.
func ID(id string) Arg { return option(func(n *Node) { n.SetID(id) }) }

// Class adds classes, read back with Node.Classes.
func Class(names ...string) Arg {
	return option(func(n *Node) {
		for _, c := range names {
			n.AddClass(c)
		}
	})
}

// Text adds a text node, read back with Node.Text.
func Text(s string) Arg { return option(func(n *Node) { n.Append(dom.NewText(s)) }) }

// Style adds declarations to the style attribute, after any an earlier
// argument gave (kit.Padding, kit.Gap…), so a property set twice takes the
// later value; read back with Node.Style. Node.SetStyle replaces them all.
func Style(css string) Arg {
	return option(func(n *Node) {
		if old := strings.TrimRight(strings.TrimSpace(n.Style()), ";"); old != "" {
			css = old + "; " + css
		}
		n.SetStyle(css)
	})
}

// Attr sets any attribute, read back with Node.Attr.
func Attr(name, value string) Arg { return option(func(n *Node) { n.SetAttr(name, value) }) }

// Type sets an input's type, read back with Node.Attr("type").
func Type(t string) Arg { return Attr("type", t) }

// Value sets a control's value, read back with Node.Attr("value").
func Value(v string) Arg { return Attr("value", v) }

// For names the element a label is for, read back with Node.Attr("for").
func For(id string) Arg { return Attr("for", id) }

// Placeholder is a text field's or select's text while empty, read back
// with Node.Attr("placeholder").
func Placeholder(text string) Arg { return Attr("placeholder", text) }

func flag(name string, on bool) Arg {
	return option(func(n *Node) {
		if on {
			n.SetAttr(name, "")
		} else {
			n.RemoveAttr(name)
		}
	})
}

// Checked starts a checkbox checked, read back with Document.Checked.
func Checked(on bool) Arg { return flag("checked", on) }

// Open starts a details open, read back with Node.Attr("open").
func Open() Arg { return flag("open", true) }

// Disabled disables a control, read back with Node.Attr("disabled").
func Disabled() Arg { return flag("disabled", true) }

// Draggable lets the element be dragged onto a drop target, as the DOM's
// draggable="true" (decision 0062).
func Draggable() Arg { return Attr("draggable", attrTrue) }

// Selected starts an option chosen, read back with Document.Selected.
func Selected() Arg { return flag("selected", true) }

// TabIndex makes any element focusable in tab order i (-1 for click
// only), read back with Node.Attr("tabindex").
func TabIndex(i int) Arg { return Attr("tabindex", strconv.Itoa(i)) }
