package css

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/lantervale/ember/ui/internal/dom"
)

//nolint:gochecknoglobals // lookup table
var stateNames = map[string]dom.State{
	"hover": dom.Hover, "focus": dom.Focus, "focus-visible": dom.FocusVisible, "active": dom.Active,
	"disabled": dom.Disabled, "checked": dom.Checked, "open": dom.Open, "modal": dom.Modal,
}

// pseudoElements are the subset's pseudo-elements and the one property
// each takes.
//
//nolint:gochecknoglobals,goconst // lookup table
var pseudoElements = map[string]string{"placeholder": "color", "selection": "background-color",
	"backdrop": "background-color"}

// Specificity is a selector's (ids, classes, types) weight, compared in
// that order.
type Specificity [3]int

// Less reports whether s weighs less than o.
func (s Specificity) Less(o Specificity) bool {
	for i := range s {
		if s[i] != o[i] {
			return s[i] < o[i]
		}
	}
	return false
}

// Combinator joins two compound selectors.
type Combinator uint8

const (
	Descendant Combinator = iota
	Child
)

type pseudoKind uint8

const (
	pseudoState pseudoKind = iota
	pseudoFirstChild
	pseudoLastChild
	pseudoRoot
	pseudoNot
	pseudoNthChild
)

type pseudo struct {
	kind  pseudoKind
	state dom.State
	not   []Selector
	// a and b are :nth-child(An+B)'s.
	a, b int
}

// Attr is an attribute selector: [Name], or [Name="Value"] when HasValue.
type Attr struct {
	Name, Value string
	HasValue    bool
}

// Compound is a compound selector: a type (TextTag for any) with ids,
// classes, attributes and pseudo-classes, all of which must match.
type Compound struct {
	Tag     dom.Tag
	IDs     []string
	Classes []string
	Attrs   []Attr
	pseudos []pseudo
	text    string
}

// Selector is a complex selector: compounds joined by combinators, read
// left to right; Combinators[i] joins Compounds[i] and Compounds[i+1].
type Selector struct {
	Compounds   []Compound
	Combinators []Combinator
	Specificity Specificity
	// PseudoElement is "placeholder" or "selection" for a ::placeholder or
	// ::selection selector, the subset's pseudo-elements; Matches then tests
	// the element it belongs to.
	PseudoElement string
}

func (s Selector) String() string {
	var b strings.Builder
	for i, c := range s.Compounds {
		if i > 0 {
			if s.Combinators[i-1] == Child {
				b.WriteString(" > ")
			} else {
				b.WriteString(" ")
			}
		}
		b.WriteString(c.text)
	}
	if s.PseudoElement != "" {
		b.WriteString("::" + s.PseudoElement)
	}
	return b.String()
}

// Matches reports whether the element e matches s.
func (s Selector) Matches(e *dom.Node) bool {
	return s.matchFrom(len(s.Compounds)-1, e)
}

func (s Selector) matchFrom(i int, e *dom.Node) bool {
	if !s.Compounds[i].matches(e) {
		return false
	}
	if i == 0 {
		return true
	}
	if s.Combinators[i-1] == Child {
		p := e.Parent()
		return p != nil && s.matchFrom(i-1, p)
	}
	for p := e.Parent(); p != nil; p = p.Parent() {
		if s.matchFrom(i-1, p) {
			return true
		}
	}
	return false
}

func (c *Compound) matches(e *dom.Node) bool {
	if c.Tag != dom.TextTag && c.Tag != e.Tag() {
		return false
	}
	for _, id := range c.IDs {
		if e.ID() != id {
			return false
		}
	}
	for _, cl := range c.Classes {
		if !e.HasClass(cl) {
			return false
		}
	}
	for _, a := range c.Attrs {
		v, ok := e.Attr(a.Name)
		if !ok || a.HasValue && v != a.Value {
			return false
		}
	}
	for _, p := range c.pseudos {
		switch p.kind {
		case pseudoState:
			if !e.State().Has(p.state) {
				return false
			}
		case pseudoFirstChild:
			if !edgeChild(e, 1) {
				return false
			}
		case pseudoLastChild:
			if !edgeChild(e, -1) {
				return false
			}
		case pseudoRoot:
			if e.Parent() != nil {
				return false
			}
		case pseudoNthChild:
			if !nth(p.a, p.b, childIndex(e)) {
				return false
			}
		case pseudoNot:
			for _, s := range p.not {
				if s.Matches(e) {
					return false
				}
			}
		}
	}
	return true
}

// parseNth reads :nth-child()'s argument, odd, even or An+B, as a and b.
func parseNth(src string) (int, int, bool) {
	s := strings.ToLower(strings.Join(strings.Fields(src), ""))
	switch s {
	case "odd":
		return 2, 1, true
	case "even":
		return 2, 0, true
	}
	i := strings.IndexByte(s, 'n')
	if i < 0 {
		n, err := strconv.Atoi(s)
		return 0, n, err == nil
	}
	var a, b int
	switch an := s[:i]; an {
	case "", "+":
		a = 1
	case "-":
		a = -1
	default:
		v, err := strconv.Atoi(an)
		if err != nil {
			return 0, 0, false
		}
		a = v
	}
	if rest := s[i+1:]; rest != "" {
		if rest[0] != '+' && rest[0] != '-' {
			return 0, 0, false
		}
		v, err := strconv.Atoi(rest)
		if err != nil {
			return 0, 0, false
		}
		b = v
	}
	return a, b, true
}

// nth reports whether the element child at place i, from 1, is a·n+b for
// some n ≥ 0.
func nth(a, b, i int) bool {
	if i < 1 {
		return false
	}
	if a == 0 {
		return i == b
	}
	return (i-b)%a == 0 && (i-b)/a >= 0
}

// childIndex is e's place among its parent's element children, from 1; 0
// without a parent.
func childIndex(e *dom.Node) int {
	p := e.Parent()
	if p == nil {
		return 0
	}
	i := 0
	for _, c := range p.Children() {
		if c.IsText() {
			continue
		}
		i++
		if c == e {
			return i
		}
	}
	return 0
}

// edgeChild reports whether e is its parent's first (dir 1) or last (dir
// -1) element child; text nodes don't count.
func edgeChild(e *dom.Node, dir int) bool {
	p := e.Parent()
	if p == nil {
		return false
	}
	kids := p.Children()
	i, end := 0, len(kids)
	if dir < 0 {
		i, end = len(kids)-1, -1
	}
	for ; i != end; i += dir {
		if !kids[i].IsText() {
			return kids[i] == e
		}
	}
	return false
}

// ParseSelectors reads a selector list such as "nav > .item:hover, #main".
func ParseSelectors(src string) ([]Selector, error) {
	p := parser{toks: Tokenize(src)}
	var cs []Component
	for p.peek().Kind != EOF {
		cs = append(cs, p.component())
	}
	sels, err := parseSelectors(cs)
	if err != nil {
		return nil, err
	}
	return sels, nil
}

// SelectorError is a selector outside the subset or badly written.
type SelectorError struct {
	Pos Pos
	Msg string
}

func (e *SelectorError) Error() string {
	return fmt.Sprintf("%d:%d: %s", e.Pos.Line, e.Pos.Col, e.Msg)
}

func selErr(c Component, format string, args ...any) *SelectorError {
	return &SelectorError{Pos: c.Pos, Msg: fmt.Sprintf(format, args...)}
}

func parseSelectors(cs []Component) ([]Selector, *SelectorError) {
	var out []Selector
	start := 0
	for i := 0; i <= len(cs); i++ {
		if i < len(cs) && cs[i].Kind != Comma {
			continue
		}
		part := trim(cs[start:i])
		if len(part) == 0 {
			at := Component{Token: Token{Pos: Pos{1, 1}}}
			switch {
			case i < len(cs):
				at = cs[i]
			case start > 0:
				at = cs[start-1]
			}
			return nil, selErr(at, "empty selector")
		}
		s, err := parseComplex(part)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
		start = i + 1
	}
	return out, nil
}

func parseComplex(cs []Component) (Selector, *SelectorError) {
	var s Selector
	if n := len(cs); n >= 3 && cs[n-3].Kind == Colon && cs[n-2].Kind == Colon && cs[n-1].Kind == Ident &&
		pseudoElements[strings.ToLower(cs[n-1].Text)] != "" {
		s.PseudoElement = strings.ToLower(cs[n-1].Text)
		cs = cs[:n-3]
		if len(cs) == 0 || cs[len(cs)-1].Kind == Whitespace {
			cs = append(cs, Component{Token: Token{Kind: Delim, Text: "*", Pos: cs0Pos(cs)}})
		}
	}
	i := 0
	for {
		c, n, err := parseCompound(cs[i:])
		if err != nil {
			return Selector{}, err
		}
		s.Compounds = append(s.Compounds, c)
		i += n
		sawSpace := false
		for i < len(cs) && cs[i].Kind == Whitespace {
			sawSpace = true
			i++
		}
		if i == len(cs) {
			break
		}
		comb := Descendant
		if t := cs[i]; t.Kind == Delim {
			switch t.Text {
			case ">":
				comb = Child
				i++
			case "+", "~":
				return Selector{}, selErr(t, "the %q combinator is not supported", t.Text)
			}
		}
		if comb == Descendant && !sawSpace {
			return Selector{}, selErr(cs[i], "unexpected %q in selector", cs[i].String())
		}
		for i < len(cs) && cs[i].Kind == Whitespace {
			i++
		}
		if i == len(cs) {
			return Selector{}, selErr(cs[i-1], "selector ends with a combinator")
		}
		s.Combinators = append(s.Combinators, comb)
	}
	for _, c := range s.Compounds {
		s.Specificity = s.Specificity.add(c.specificity())
	}
	if s.PseudoElement != "" {
		s.Specificity[2]++
	}
	return s, nil
}

func cs0Pos(cs []Component) Pos {
	if len(cs) > 0 {
		return cs[len(cs)-1].Pos
	}
	return Pos{1, 1}
}

func (s Specificity) add(o Specificity) Specificity {
	return Specificity{s[0] + o[0], s[1] + o[1], s[2] + o[2]}
}

func (c *Compound) specificity() Specificity {
	sp := Specificity{len(c.IDs), len(c.Classes) + len(c.Attrs), 0}
	if c.Tag != dom.TextTag {
		sp[2]++
	}
	for _, p := range c.pseudos {
		if p.kind != pseudoNot {
			sp[1]++
			continue
		}
		var most Specificity
		for _, s := range p.not {
			if most.Less(s.Specificity) {
				most = s.Specificity
			}
		}
		sp = sp.add(most)
	}
	return sp
}

func parseCompound(cs []Component) (Compound, int, *SelectorError) {
	var c Compound
	i := 0
	if i < len(cs) {
		switch t := cs[i]; {
		case t.Kind == Ident:
			tag, ok := dom.TagNamed(strings.ToLower(t.Text))
			if !ok {
				return c, 0, selErr(t, "ui has no <%s> element", strings.ToLower(t.Text))
			}
			c.Tag = tag
			i++
		case t.Kind == Delim && t.Text == "*":
			i++
		case t.Kind == Delim && t.Text == "|":
			return c, 0, selErr(t, "namespaces are not supported")
		}
	}
	for i < len(cs) {
		t := cs[i]
		switch {
		case t.Kind == Hash && t.HasID:
			c.IDs = append(c.IDs, t.Text)
			i++
		case t.Kind == Delim && t.Text == ".":
			if i+1 >= len(cs) || cs[i+1].Kind != Ident {
				return c, 0, selErr(t, "expected a class name after '.'")
			}
			c.Classes = append(c.Classes, cs[i+1].Text)
			i += 2
		case t.Kind == Colon:
			p, n, err := parsePseudo(cs[i:])
			if err != nil {
				return c, 0, err
			}
			c.pseudos = append(c.pseudos, p)
			i += n
		case t.Kind == LBracket:
			a, err := parseAttr(t)
			if err != nil {
				return c, 0, err
			}
			c.Attrs = append(c.Attrs, a)
			i++
		case t.Kind == Ident || t.Kind == Delim && t.Text == "*":
			return c, 0, selErr(t, "a type selector must come first in a compound")
		default:
			if i == 0 {
				return c, 0, selErr(t, "unexpected %q in selector", t.String())
			}
			c.text = join(cs[:i])
			return c, i, nil
		}
	}
	c.text = join(cs[:i])
	return c, i, nil
}

func parsePseudo(cs []Component) (pseudo, int, *SelectorError) {
	if len(cs) < 2 {
		return pseudo{}, 0, selErr(cs[0], "expected a pseudo-class after ':'")
	}
	t := cs[1]
	switch t.Kind {
	case Colon:
		return pseudo{}, 0, selErr(t, "pseudo-elements are not supported")
	case Ident:
		name := strings.ToLower(t.Text)
		if st, ok := stateNames[name]; ok {
			return pseudo{kind: pseudoState, state: st}, 2, nil
		}
		switch name {
		case "first-child":
			return pseudo{kind: pseudoFirstChild}, 2, nil
		case "last-child":
			return pseudo{kind: pseudoLastChild}, 2, nil
		case "root":
			return pseudo{kind: pseudoRoot}, 2, nil
		}
		return pseudo{}, 0, selErr(t, "the :%s pseudo-class is not supported", name)
	case Function:
		name := strings.ToLower(t.Text)
		if name == "nth-child" {
			a, b, ok := parseNth(join(t.Args))
			if !ok {
				return pseudo{}, 0, selErr(t, ":nth-child() takes odd, even or An+B")
			}
			return pseudo{kind: pseudoNthChild, a: a, b: b}, 2, nil
		}
		if name != "not" {
			return pseudo{}, 0, selErr(t, "the :%s() pseudo-class is not supported", name)
		}
		if len(trim(t.Args)) == 0 {
			return pseudo{}, 0, selErr(t, ":not() is empty")
		}
		sels, err := parseSelectors(t.Args)
		if err != nil {
			return pseudo{}, 0, err
		}
		return pseudo{kind: pseudoNot, not: sels}, 2, nil
	case EOF, AtKeyword, Hash, String, BadString, URL, BadURL, Delim, Number, Percentage, Dimension, Whitespace, CDO, CDC,
		Semicolon, Comma, LBracket, RBracket, LParen, RParen, LBrace, RBrace:
		// nothing special for these tokens
	}
	return pseudo{}, 0, selErr(t, "expected a pseudo-class after ':'")
}

// parseAttr reads [name] or [name="value"]; other matchers and flags are
// outside the subset.
func parseAttr(b Component) (Attr, *SelectorError) {
	v := items(b.Args)
	if len(v) == 0 || v[0].Kind != Ident {
		return Attr{}, selErr(b, "expected an attribute name in [ ]")
	}
	a := Attr{Name: strings.ToLower(v[0].Text)}
	switch {
	case len(v) == 1:
		return a, nil
	case v[1].Kind == Delim && v[1].Text == "=":
		if len(v) == 3 && (v[2].Kind == Ident || v[2].Kind == String) {
			a.Value, a.HasValue = v[2].Text, true
			return a, nil
		}
		if len(v) == 4 {
			return Attr{}, selErr(v[3], "attribute flags are not supported")
		}
		return Attr{}, selErr(b, "expected [%s=value]", a.Name)
	case v[1].Kind == Delim && strings.ContainsAny(v[1].Text, "^$*|~"):
		return Attr{}, selErr(v[1], "only [attr] and [attr=\"value\"] are supported, not %s=", v[1].Text)
	}
	return Attr{}, selErr(v[1], "unexpected %q in [ ]", v[1].String())
}

// treeStates are the states sheets' selectors read in compounds above
// their subject, :not() included: a change of one may change what an
// element's descendants match. Sibling combinators are outside the
// subset, so a state reaches no sibling.
func treeStates(sheets []*Sheet) dom.State {
	var s dom.State
	for _, sh := range sheets {
		for _, r := range sh.Rules {
			for _, sel := range r.Selectors {
				s |= sel.treeStates()
			}
		}
	}
	return s
}

func (s Selector) treeStates() dom.State {
	var st dom.State
	for i, c := range s.Compounds {
		for _, p := range c.pseudos {
			if p.kind == pseudoState && i < len(s.Compounds)-1 {
				st |= p.state
			}
			for _, n := range p.not {
				st |= n.treeStates()
				if i < len(s.Compounds)-1 {
					st |= n.states()
				}
			}
		}
	}
	return st
}

// states are every state s reads.
func (s Selector) states() dom.State {
	var st dom.State
	for _, c := range s.Compounds {
		for _, p := range c.pseudos {
			if p.kind == pseudoState {
				st |= p.state
			}
			for _, n := range p.not {
				st |= n.states()
			}
		}
	}
	return st
}

// aboveKeys adds the classes, ids and attributes s reads in a compound
// above its subject, keyed as dom.Pipeline.Changed: a change of those
// on an element can restyle its descendants.
func (s Selector) aboveKeys(add func(string)) {
	for i, c := range s.Compounds {
		above := i < len(s.Compounds)-1
		if above {
			c.keys(add)
		}
		for _, p := range c.pseudos {
			for _, n := range p.not {
				if above {
					for _, nc := range n.Compounds {
						nc.keys(add)
					}
				} else {
					n.aboveKeys(add)
				}
			}
		}
	}
}

func (c Compound) keys(add func(string)) {
	for _, id := range c.IDs {
		add("#" + id)
	}
	for _, cl := range c.Classes {
		add("." + cl)
	}
	for _, a := range c.Attrs {
		add("[" + a.Name)
	}
}
