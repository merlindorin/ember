package ui

import (
	"strings"

	"github.com/lantervale/ember/ui/layout"
)

// Translator says a message in the player's language: each text node's
// text, a placeholder or a select's label, never what a field holds.
type Translator interface {
	Tr(message string) string
}

// WithTranslator shows the document's text as t says it. An element
// whose translate attribute is "no" keeps its own text, and so does
// what's under it; automation IDs stay on the text as written.
func WithTranslator(t Translator) DocumentOption {
	return func(d *Document) { d.SetTranslator(t) }
}

// Translator is what the document's text is shown through, nil for none.
func (d *Document) Translator() Translator { return d.translator }

// SetTranslator shows the document's text through t from the next frame,
// laying every text out again; nil shows it as written. The document
// remembers what each message said, so a game switching language calls
// it again, even with the same t.
func (d *Document) SetTranslator(t Translator) {
	d.translator, d.said, d.relayout = t, nil, true
}

// tr is the document's translator remembering what each message said,
// so a message is asked once; nil without a translator.
func (d *Document) tr() func(string) string {
	if d.translator == nil {
		return nil
	}
	return func(m string) string {
		if s, ok := d.said[m]; ok {
			return s
		}
		if d.said == nil || len(d.said) >= maxSaid {
			d.said = map[string]string{}
		}
		s := d.translator.Tr(m)
		d.said[m] = s
		return s
	}
}

// maxSaid bounds the remembered messages: text that keeps changing, a
// clock's, starts the memory afresh rather than growing it.
const maxSaid = 4096

// shown is a text node's text as the player reads it.
func (d *Document) shown(n *Node) string { return layout.Translated(n, d.tr()) }

// written is a text node's text as the game wrote it.
func written(n *Node) string { return n.Text() }

// say is the text of n's subtree, each text node read by text.
func say(n *Node, text func(*Node) string) string {
	if n.IsText() {
		return text(n)
	}
	var b strings.Builder
	n.Walk(func(c *Node) bool {
		if c.IsText() {
			b.WriteString(text(c))
		}
		return true
	})
	return b.String()
}
