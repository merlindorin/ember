package ui

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/rivo/uniseg"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/lantervale/ember/ui/internal/style"
)

// caseMap ties a field's value to its text as text-transform draws it,
// rune positions both ways: a cluster whose case changes its length (ß
// drawn SS) maps its edges to each other, and a position inside a longer
// drawn cluster to the nearer edge.
type caseMap struct {
	toDrawn, toValue []int
}

// drawnAt is value position i's place in the drawn text.
func (m *caseMap) drawnAt(i int) int {
	if m == nil {
		return i
	}
	return m.toDrawn[min(max(i, 0), len(m.toDrawn)-1)]
}

// valueAt is drawn position j's place in the value.
func (m *caseMap) valueAt(j int) int {
	if m == nil {
		return j
	}
	return m.toValue[min(max(j, 0), len(m.toValue)-1)]
}

// fieldCase is text as tt draws it in an edit field, cluster by cluster,
// and the map between them; nil when nothing changes.
func fieldCase(text string, tt style.TextTransform) (string, *caseMap) {
	if tt == style.NoTransform || text == "" {
		return text, nil
	}
	var upper, lower, title cases.Caser
	switch tt {
	case style.Uppercase:
		upper = cases.Upper(language.Und)
	case style.Lowercase:
		lower = cases.Lower(language.Und)
	case style.Capitalize:
		title = cases.Title(language.Und, cases.NoLower)
	case style.NoTransform:
	}
	var b strings.Builder
	m := &caseMap{toDrawn: []int{0}, toValue: []int{0}}
	v, d := 0, 0
	add := func(src, shown string) {
		nv, nd := utf8.RuneCountInString(src), utf8.RuneCountInString(shown)
		for k := 1; k <= nv; k++ {
			m.toDrawn = append(m.toDrawn, d+nd*k/nv)
		}
		for k := 1; k <= nd; k++ {
			at := v
			if 2*k >= nd {
				at = v + nv
			}
			if nv == nd {
				at = v + k
			}
			m.toValue = append(m.toValue, at)
		}
		v, d = v+nv, d+nd
		b.WriteString(shown)
	}
	rest, state := text, -1
	for len(rest) > 0 {
		var word string
		word, rest, state = uniseg.FirstWordInString(rest, state)
		r, _ := utf8.DecodeRuneInString(word)
		first := true
		for len(word) > 0 {
			g, tail, _, _ := uniseg.FirstGraphemeClusterInString(word, -1)
			shown := g
			switch tt {
			case style.Uppercase:
				shown = upper.String(g)
			case style.Lowercase:
				shown = lower.String(g)
			case style.Capitalize:
				if first && unicode.IsLetter(r) {
					shown = title.String(g)
				}
			case style.NoTransform:
			}
			add(g, shown)
			word, first = tail, false
		}
	}
	return b.String(), m
}
