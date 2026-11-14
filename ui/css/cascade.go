package css

import (
	"slices"
	"strings"

	"github.com/lantervale/ember/ui/internal/dom"
)

// index buckets a sheet's selectors by their rightmost compound's id,
// else a class, else its type, so an element only tries selectors that
// can match it.
type index struct {
	byID, byClass map[string][]entry
	byTag         map[dom.Tag][]entry
	any           []entry
	// pseudos are the pseudo-elements its selectors name.
	pseudos map[string]bool
	// inherits is set when a declaration sets a property that isn't
	// inherited to inherit.
	inherits bool
	// above are the classes, ids and attributes some selector reads in a
	// compound above its subject, keyed as dom.Pipeline.Changed.
	above map[string]bool
}

type entry struct {
	rule, sel int
}

func (s *Sheet) indexed() *index {
	s.once.Do(func() {
		x := &index{byID: map[string][]entry{}, byClass: map[string][]entry{}, byTag: make(map[dom.Tag][]entry),
			pseudos: map[string]bool{}, above: map[string]bool{}}
		for ri, r := range s.Rules {
			for si, sel := range r.Selectors {
				e := entry{ri, si}
				sel.aboveKeys(func(k string) { x.above[k] = true })
				if sel.PseudoElement != "" {
					x.pseudos[sel.PseudoElement] = true
				}
				last := sel.Compounds[len(sel.Compounds)-1]
				switch {
				case len(last.IDs) > 0:
					x.byID[last.IDs[0]] = append(x.byID[last.IDs[0]], e)
				case len(last.Classes) > 0:
					x.byClass[last.Classes[0]] = append(x.byClass[last.Classes[0]], e)
				case last.Tag != dom.TextTag:
					x.byTag[last.Tag] = append(x.byTag[last.Tag], e)
				default:
					x.any = append(x.any, e)
				}
			}
			for _, d := range r.Decls {
				if p, ok := properties[d.Name]; (!ok || !p.inherited) &&
					slices.ContainsFunc(items(d.Value), func(c Component) bool { return isWord(c, "inherit") }) {
					x.inherits = true
				}
			}
		}
		s.index = x
	})
	return s.index
}

// level is a declaration's place in the cascade by origin and importance.
type level uint8

const (
	agentNormal level = iota
	authorNormal
	authorImportant
	agentImportant
)

// applied is one declaration that applies to an element, with what ranks
// it against the others.
type applied struct {
	decl   *Declaration
	level  level
	inline bool
	spec   Specificity
	order  int
}

func (a applied) less(b applied) bool {
	if a.level != b.level {
		return a.level < b.level
	}
	if a.inline != b.inline {
		return b.inline
	}
	if a.spec != b.spec {
		return a.spec.Less(b.spec)
	}
	return a.order < b.order
}

// matcher keeps the buffers matching and cascading use from one element
// to the next, so a restyle allocates per changed style, not per step.
type matcher struct {
	best   map[int]Specificity
	rules  []int
	out    []applied
	long   []*Declaration
	custom []*Declaration
}

// matched lists the declarations that apply to e, least important first.
// sheets[0] is the user agent's. The list lives until the next call.
func (m *matcher) matched(e *dom.Node, sheets []*Sheet, inline []Declaration) []applied {
	return m.matchedFor(e, sheets, inline, "")
}

// matchedFor is matched for the element's pseudo-element pseudo ("" for the element
// itself); a pseudo-element takes no inline declarations.
func (m *matcher) matchedFor(e *dom.Node, sheets []*Sheet, inline []Declaration, pseudo string) []applied {
	if pseudo != "" {
		inline = nil
	}
	if m.best == nil {
		m.best = map[int]Specificity{}
	}
	m.out = m.out[:0]
	order := 0
	for si, s := range sheets {
		x := s.indexed()
		clear(m.best)
		if id := e.ID(); id != "" {
			m.try(s, x.byID[id], e, pseudo)
		}
		for _, c := range e.Classes() {
			m.try(s, x.byClass[c], e, pseudo)
		}
		m.try(s, x.byTag[e.Tag()], e, pseudo)
		m.try(s, x.any, e, pseudo)
		m.rules = m.rules[:0]
		for r := range m.best {
			m.rules = append(m.rules, r)
		}
		slices.Sort(m.rules)
		for _, r := range m.rules {
			for di := range s.Rules[r].Decls {
				d := &s.Rules[r].Decls[di]
				m.out = append(m.out, applied{decl: d, level: levelOf(si == 0, d.Important), spec: m.best[r], order: order})
				order++
			}
		}
	}
	for i := range inline {
		d := &inline[i]
		m.out = append(m.out, applied{decl: d, level: levelOf(false, d.Important), inline: true, order: order})
		order++
	}
	slices.SortStableFunc(m.out, func(a, b applied) int {
		switch {
		case a.less(b):
			return -1
		case b.less(a):
			return 1
		}
		return 0
	})
	return m.out
}

// try keeps, for each rule of s with a selector among es matching e's
// pseudo, its most specific matching selector.
func (m *matcher) try(s *Sheet, es []entry, e *dom.Node, pseudo string) {
	for _, en := range es {
		sel := s.Rules[en.rule].Selectors[en.sel]
		if sel.PseudoElement != pseudo || !sel.Matches(e) {
			continue
		}
		if sp, ok := m.best[en.rule]; !ok || sp.Less(sel.Specificity) {
			m.best[en.rule] = sel.Specificity
		}
	}
}

func levelOf(agent, important bool) level {
	switch {
	case agent && important:
		return agentImportant
	case agent:
		return agentNormal
	case important:
		return authorImportant
	}
	return authorNormal
}

// cascaded is each longhand's winning declaration, and the custom
// properties set on the element, in cascade order.
type cascaded struct {
	longhands []*Declaration
	custom    []*Declaration
}

// cascade puts each longhand's winner at its place in order. The result
// lives until the next call.
func (m *matcher) cascade(list []applied) cascaded {
	if m.long == nil {
		m.long = make([]*Declaration, len(order))
	}
	clear(m.long)
	m.custom = m.custom[:0]
	for _, a := range list {
		d := a.decl
		if strings.HasPrefix(d.Name, "--") {
			m.custom = append(m.custom, d)
			continue
		}
		if long, ok := shorthands[d.Name]; ok {
			for _, l := range long {
				m.long[orderIndex[l]] = d
			}
			continue
		}
		m.long[orderIndex[d.Name]] = d
	}
	return cascaded{longhands: m.long, custom: m.custom}
}
