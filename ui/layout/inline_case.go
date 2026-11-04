package layout

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/rivo/uniseg"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/merlindorin/ember/ui/internal/dom"
	"github.com/merlindorin/ember/ui/internal/style"
)

// Translate lays text out as tr says it: each text node's text is one message, and the
// node keeps its own.
func Translate(tr func(message string) string) Option {
	return func(t *tree) { t.translate = tr }
}

// Translated is a text node's text as tr says it (see TranslatedAt), but as it is
// when the node is what an edit field holds; an edit field's placeholder and a
// select's label are translated.
func Translated(text *dom.Node, tr func(message string) string) string {
	owner := text.Parent()
	if owner != nil && dom.Data(owner).Shadow == text && !dom.Data(owner).Placeholder && owner.Tag() != dom.Select {
		return text.Text()
	}
	return TranslatedAt(owner, text.Text(), tr)
}

// TranslatedAt is s, a text shown in element e, as tr says it, or as it is when tr is
// nil or when HTML's translate attribute of the nearest element from e up holding one
// is "no". Text only of white space is never a message.
func TranslatedAt(e *dom.Node, s string, tr func(message string) string) string {
	if tr == nil || strings.TrimSpace(s) == "" {
		return s
	}
	for ; e != nil; e = e.Parent() {
		if v, ok := e.Attr("translate"); ok {
			if strings.EqualFold(strings.TrimSpace(v), "no") {
				return s
			}
			break
		}
	}
	return tr(s)
}

// shownText is a run's text as text-transform shows it, which layout
// measures and paint draws; the node keeps its own. inWord carries, from
// one run to the next, whether the text so far ends inside a word, for
// capitalize. An edit field shows what was typed.
func shownText(text string, st *style.Computed, owner *dom.Node, inWord *bool) string {
	if owner != nil && (owner.Tag() == dom.Input || owner.Tag() == dom.Textarea) {
		*inWord = endsInWord(text, *inWord)
		return text
	}
	switch st.TextTransform {
	case style.Uppercase:
		text = cases.Upper(language.Und).String(text)
	case style.Lowercase:
		text = cases.Lower(language.Und).String(text)
	case style.Capitalize:
		return capitalize(text, inWord)
	case style.NoTransform:
	}
	*inWord = endsInWord(text, *inWord)
	return text
}

// capitalize titles the first letter of each word, as Chrome does: words
// as Unicode segments them (well-Known, O'neil's, 3rd), a word that a
// run continues from the one before left as it is.
func capitalize(text string, inWord *bool) string {
	var b strings.Builder
	title := cases.Title(language.Und, cases.NoLower)
	rest, state, first := text, -1, true
	var w string
	for len(rest) > 0 {
		w, rest, state = uniseg.FirstWordInString(rest, state)
		r, _ := utf8.DecodeRuneInString(w)
		if unicode.IsLetter(r) && (!first || !*inWord) {
			g, _, _, _ := uniseg.FirstGraphemeClusterInString(w, -1)
			w = title.String(g) + w[len(g):]
		}
		*inWord = endsInWord(w, *inWord)
		first = false
		b.WriteString(w)
	}
	return b.String()
}

// endsInWord is whether text leaves the line inside a word, inWord
// before it.
func endsInWord(text string, inWord bool) bool {
	for _, r := range text {
		switch {
		case unicode.IsSpace(r):
			inWord = false
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			inWord = true
		}
	}
	return inWord
}
