package kit

import (
	"slices"
	"strings"

	"github.com/merlindorin/ember/internal/emojinames"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/css"
	"github.com/merlindorin/ember/ui/internal/dom"
)

// RichText is a Label whose text reads [b]…[/b],
// [i]…[/i] and [color=…]…[/color] as strong, em and coloured spans. A
// color is #rrggbb, #rrggbbaa or a CSS colour name; a tag it doesn't know
// stays as typed. It takes Label's classes (wrap, truncate, center,
// right) and ExpandShortcodes; SetRich changes its text.
func RichText(text string, args ...ui.Arg) *ui.Node {
	n := ui.Span(ui.Class("kit-label", "kit-rich-text"))
	appendBBCode(n, text)
	for _, a := range args {
		a.Apply(n)
	}
	return n
}

// ExpandShortcodes shows the :name: shortcodes in an element's text as
// their emoji, as the old ember/ui's ExpandShortcodes does. ui draws emoji from
// the faces the Document has.
func ExpandShortcodes() ui.Arg { return expandShortcodes{} }

type expandShortcodes struct{}

func (expandShortcodes) Apply(n *ui.Node) {
	n.SetAttr("data-shortcodes", "")
	expand(n)
}

func expand(n *ui.Node) {
	n.Walk(func(d *dom.Node) bool {
		if d.IsText() {
			d.SetText(emojinames.Expand(d.Text()))
		}
		return true
	})
}

type bbTag struct {
	name  string
	color string
}

func (t bbTag) open() *ui.Node {
	switch t.name {
	case "b":
		return ui.Strong()
	case "i":
		return ui.Em()
	}
	return ui.Span(ui.Style("color: " + t.color))
}

// appendBBCode adds text's runs under n, nesting the tags as elements. A
// tag closed out of order closes the tags opened after it and opens them
// again, so [b]a[i]b[/b]c[/i] reads as it does in the old ember/ui.
func appendBBCode(n *ui.Node, s string) {
	p := bbcode{nodes: []*ui.Node{n}}
	for len(s) > 0 {
		end := strings.IndexByte(s, ']')
		if s[0] != '[' || end < 0 {
			next := strings.IndexByte(s[1:], '[')
			if next < 0 {
				p.text.WriteString(s)
				break
			}
			p.text.WriteString(s[:next+1])
			s = s[next+1:]
			continue
		}
		if !p.tag(s[1:end]) {
			p.text.WriteString(s[:end+1])
		}
		s = s[end+1:]
	}
	p.flush()
}

// bbcode is appendBBCode's state: the tags open, the elements they opened
// under the node, and the text not yet added.
type bbcode struct {
	open  []bbTag
	nodes []*ui.Node
	text  strings.Builder
}

// tag acts on one [tag], reporting whether it is one RichText reads.
func (p *bbcode) tag(tag string) bool {
	switch {
	case tag == "b" || tag == "i":
		p.flush()
		p.push(bbTag{name: tag})
		return true
	case tag == "/b" || tag == "/i" || tag == "/color":
		return p.closeTag(tag[1:])
	case strings.HasPrefix(tag, "color="):
		c := tag[len("color="):]
		if !isColor(c) {
			return false
		}
		p.flush()
		p.push(bbTag{name: "color", color: c})
		return true
	}
	return false
}

// flush adds the text so far to the innermost element.
func (p *bbcode) flush() {
	if p.text.Len() > 0 {
		p.nodes[len(p.nodes)-1].Append(dom.NewText(p.text.String()))
		p.text.Reset()
	}
}

// push opens t inside the innermost element.
func (p *bbcode) push(t bbTag) {
	e := t.open()
	p.nodes[len(p.nodes)-1].Append(e)
	p.open = append(p.open, t)
	p.nodes = append(p.nodes, e)
}

// closeTag closes the innermost open tag called name, and opens again the
// tags opened after it; false when none is open.
func (p *bbcode) closeTag(name string) bool {
	at := -1
	for i := len(p.open) - 1; i >= 0; i-- {
		if p.open[i].name == name {
			at = i
			break
		}
	}
	if at < 0 {
		return false
	}
	p.flush()
	reopen := append([]bbTag(nil), p.open[at+1:]...)
	p.open, p.nodes = p.open[:at], p.nodes[:at+1]
	for _, t := range reopen {
		p.push(t)
	}
	return true
}

// isColor reports whether c is a colour a [color=…] tag takes.
func isColor(c string) bool {
	if c == "" || strings.ContainsAny(c, ";:{}()\"'") {
		return false
	}
	_, warns := css.ParseInline("color: " + c)
	return len(warns) == 0
}

// SetRich makes the RichText n read text, its tags read as RichText's,
// and its shortcodes expanded if it expands them.
func SetRich(n *ui.Node, text string) {
	for _, c := range slices.Clone(n.Children()) {
		n.Remove(c)
	}
	appendBBCode(n, text)
	if _, ok := n.Attr("data-shortcodes"); ok {
		expand(n)
	}
}
