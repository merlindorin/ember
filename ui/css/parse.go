package css

import (
	"fmt"
	"io/fs"
	"path"
	"strings"
	"sync"
)

// Component is a component value: a token, or a function or a block with
// what it holds. A block has the Kind of its opening token (LParen,
// LBracket or LBrace).
type Component struct {
	Token
	Args []Component
}

func (c Component) String() string {
	switch c.Kind {
	case Function:
		return c.Text + "(" + join(c.Args) + ")"
	case LParen:
		return "(" + join(c.Args) + ")"
	case LBracket:
		return "[" + join(c.Args) + "]"
	case LBrace:
		return "{" + join(c.Args) + "}"
	case EOF, Ident, AtKeyword, Hash, String, BadString, URL, BadURL, Delim, Number, Percentage, Dimension, Whitespace,
		CDO, CDC, Colon, Semicolon, Comma, RBracket, RParen, RBrace:
		// nothing special for these tokens
	}
	return c.Token.String()
}

func join(cs []Component) string {
	var b strings.Builder
	for _, c := range cs {
		b.WriteString(c.String())
	}
	return b.String()
}

// Declaration is one property and its value. Name is lowercase, except a
// custom property's, which keeps its case. Value has no surrounding
// whitespace and no !important.
type Declaration struct {
	Name      string
	Value     []Component
	Important bool
	Pos       Pos
	// longhands is a shorthand's expansion, and vals a longhand's value
	// without whitespace, when it has no var().
	longhands map[string][]Component
	vals      []Component
	// base is the directory of the sheet it came from, for relative urls.
	base string
}

func (d Declaration) String() string {
	s := d.Name + ": " + join(d.Value)
	if d.Important {
		s += " !important"
	}
	return s
}

// Rule is a style rule: its selectors and its declarations.
type Rule struct {
	Selectors []Selector
	Decls     []Declaration
	Pos       Pos
}

// Sheet is a parsed stylesheet. Name is the file it came from.
type Sheet struct {
	Name  string
	Rules []Rule
	faces []FontFace
	once  sync.Once
	index *index
}

// String writes the sheet's rules back as CSS, without what was dropped
// or its @font-face rules.
func (s *Sheet) String() string {
	var b strings.Builder
	for _, r := range s.Rules {
		for i, sel := range r.Selectors {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(sel.String())
		}
		b.WriteString(" {")
		for _, d := range r.Decls {
			b.WriteString(" ")
			b.WriteString(d.String())
			b.WriteString(";")
		}
		b.WriteString(" }\n")
	}
	return b.String()
}

// Warning is something a sheet says that ui drops: a syntax error, an
// unknown property, a value or selector outside the subset.
type Warning struct { //nolint:errname // a dropped declaration, reported, never returned as a failure
	File string
	Pos  Pos
	Msg  string
}

func (w Warning) Error() string {
	file := w.File
	if file == "" {
		file = "style"
	}
	return fmt.Sprintf("%s:%d:%d: %s", file, w.Pos.Line, w.Pos.Col, w.Msg)
}

// Load reads and parses the stylesheet name in fsys. The error is only for
// reading; what the sheet gets wrong comes back as warnings.
func Load(fsys fs.FS, name string) (*Sheet, []Warning, error) {
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return nil, nil, fmt.Errorf("css: %w", err)
	}
	s, warns := Parse(name, string(data))
	return s, warns, nil
}

// Parse reads a stylesheet. name only labels warnings. What it can't use
// is dropped with a warning, as browsers drop it, and the rest is kept.
func Parse(name, src string) (*Sheet, []Warning) {
	p := parser{file: name, toks: Tokenize(src)}
	sheet := &Sheet{Name: name}
	for {
		t := p.peek()
		switch t.Kind {
		case EOF:
			return sheet, p.warns
		case Whitespace, CDO, CDC:
			p.next()
		case AtKeyword:
			p.next()
			if strings.EqualFold(t.Text, "font-face") {
				if f, ok := p.fontFace(t.Pos); ok {
					sheet.faces = append(sheet.faces, f)
				}
				continue
			}
			p.warnf(t.Pos, "@%s is not supported", t.Text)
			p.skipAtRule()
		case Ident, Function, Hash, String, BadString, URL, BadURL, Delim, Number, Percentage, Dimension, Colon, Semicolon,
			Comma, LBracket, RBracket, LParen, RParen, LBrace, RBrace:
			if r, ok := p.rule(); ok {
				sheet.Rules = append(sheet.Rules, r)
			}
		}
	}
}

// ParseInline reads a style attribute's declarations.
func ParseInline(src string) ([]Declaration, []Warning) {
	p := parser{toks: Tokenize(src)}
	var block []Component
	for p.peek().Kind != EOF {
		block = append(block, p.component())
	}
	return p.declarations(block), p.warns
}

type parser struct {
	file  string
	toks  []Token
	i     int
	warns []Warning
}

func (p *parser) warnf(pos Pos, format string, args ...any) {
	p.warns = append(p.warns, Warning{File: p.file, Pos: pos, Msg: fmt.Sprintf(format, args...)})
}

func (p *parser) peek() Token { return p.toks[p.i] }

func (p *parser) next() Token {
	t := p.toks[p.i]
	if t.Kind != EOF {
		p.i++
	}
	return t
}

func (p *parser) skipAtRule() {
	for {
		switch p.peek().Kind {
		case EOF:
			return
		case Semicolon:
			p.next()
			return
		case LBrace:
			p.component()
			return
		case Ident, Function, AtKeyword, Hash, String, BadString, URL, BadURL, Delim, Number, Percentage, Dimension,
			Whitespace, CDO, CDC, Colon, Comma, LBracket, RBracket, LParen, RParen, RBrace:
			// nothing special for these tokens
		}
		p.component()
	}
}

func (p *parser) rule() (Rule, bool) {
	pos := p.peek().Pos
	var prelude []Component
	for {
		switch p.peek().Kind {
		case EOF:
			p.warnf(pos, "rule has no { block }")
			return Rule{}, false
		case LBrace:
			block := p.component()
			sels, err := parseSelectors(prelude)
			if err != nil {
				p.warnf(err.Pos, "%s; rule dropped", err.Msg)
				return Rule{}, false
			}
			decls := p.declarations(block.Args)
			if pseudo := sels[0].PseudoElement; pseudo != "" || pseudoCount(sels) > 0 {
				if pseudoCount(sels) < len(sels) || !samePseudo(sels) {
					p.warnf(pos, "::%s can't share a rule with other selectors; rule dropped", firstPseudo(sels))
					return Rule{}, false
				}
				takes := pseudoElements[pseudo]
				kept := decls[:0]
				for _, d := range decls {
					if d.Name != takes && !strings.HasPrefix(d.Name, "--") {
						p.warnf(d.Pos, "::%s takes only %s, not %s", pseudo, takes, d.Name)
						continue
					}
					kept = append(kept, d)
				}
				decls = kept
			}
			return Rule{Selectors: sels, Decls: decls, Pos: pos}, true
		case Ident, Function, AtKeyword, Hash, String, BadString, URL, BadURL, Delim, Number, Percentage, Dimension,
			Whitespace, CDO, CDC, Colon, Semicolon, Comma, LBracket, RBracket, LParen, RParen, RBrace:
			// nothing special for these tokens
		}
		prelude = append(prelude, p.component())
	}
}

func (p *parser) component() Component {
	t := p.next()
	var closer Kind
	switch t.Kind {
	case Function, LParen:
		closer = RParen
	case LBracket:
		closer = RBracket
	case LBrace:
		closer = RBrace
	case EOF, Ident, AtKeyword, Hash, String, BadString, URL, BadURL, Delim, Number, Percentage, Dimension, Whitespace,
		CDO, CDC, Colon, Semicolon, Comma, RBracket, RParen, RBrace:
		return Component{Token: t}
	}
	c := Component{Token: t}
	for {
		switch p.peek().Kind {
		case closer:
			p.next()
			return c
		case EOF:
			return c
		case Ident, Function, AtKeyword, Hash, String, BadString, URL, BadURL, Delim, Number, Percentage, Dimension,
			Whitespace, CDO, CDC, Colon, Semicolon, Comma, LBracket, RBracket, LParen, RParen, LBrace, RBrace:
			// nothing special for these tokens
		}
		c.Args = append(c.Args, p.component())
	}
}

func (p *parser) declarations(block []Component) []Declaration {
	var out []Declaration
	for i := 0; i < len(block); {
		c := block[i]
		switch c.Kind {
		case Whitespace, Semicolon:
			i++
			continue
		case EOF, Ident, Function, AtKeyword, Hash, String, BadString, URL, BadURL, Delim, Number, Percentage, Dimension,
			CDO, CDC, Colon, Comma, LBracket, RBracket, LParen, RParen, LBrace, RBrace:
			// nothing special for these tokens
		}
		end := i
		for end < len(block) && block[end].Kind != Semicolon {
			end++
		}
		part := block[i:end]
		i = end
		switch c.Kind {
		case Ident:
			if d, ok := p.declaration(part); ok {
				out = append(out, d)
			}
		case AtKeyword:
			p.warnf(c.Pos, "@%s is not supported", c.Text)
		case EOF, Function, Hash, String, BadString, URL, BadURL, Delim, Number, Percentage, Dimension, Whitespace, CDO, CDC,
			Colon, Semicolon, Comma, LBracket, RBracket, LParen, RParen, LBrace, RBrace:
			p.warnf(c.Pos, "expected a property name, found %q", c.String())
		}
	}
	return out
}

func (p *parser) declaration(part []Component) (Declaration, bool) {
	name := part[0]
	rest := trim(part[1:])
	if len(rest) == 0 || rest[0].Kind != Colon {
		p.warnf(name.Pos, "expected ':' after %q", name.Text)
		return Declaration{}, false
	}
	d := Declaration{Name: name.Text, Value: trim(rest[1:]), Pos: name.Pos}
	if p.file != "" {
		d.base = path.Dir(p.file)
	}
	if n := len(d.Value); n >= 2 && d.Value[n-1].Kind == Ident && strings.EqualFold(d.Value[n-1].Text, "important") {
		bang := n - 2
		for bang >= 0 && d.Value[bang].Kind == Whitespace {
			bang--
		}
		if bang >= 0 && d.Value[bang].Kind == Delim && d.Value[bang].Text == "!" {
			d.Important = true
			d.Value = trim(d.Value[:bang])
		}
	}
	if !strings.HasPrefix(d.Name, "--") {
		d.Name = strings.ToLower(d.Name)
	}
	if len(d.Value) == 0 && !strings.HasPrefix(d.Name, "--") {
		p.warnf(name.Pos, "%s has no value", d.Name)
		return Declaration{}, false
	}
	if msg := checkProperty(d.Name); msg != "" {
		p.warnf(name.Pos, "%s", msg)
		return Declaration{}, false
	}
	if !strings.HasPrefix(d.Name, "--") && !hasVar(d.Value) {
		if err := validate(d.Name, d.Value); err != nil {
			p.warnf(d.Value[0].Pos, "%s: %v", d.Name, err)
			return Declaration{}, false
		}
		if _, ok := shorthands[d.Name]; ok {
			d.longhands, _ = expand(d.Name, items(d.Value))
		} else {
			d.vals = items(d.Value)
		}
	}
	return d, true
}

// pseudoCount counts the pseudo-element selectors in sels.
func pseudoCount(sels []Selector) int {
	n := 0
	for _, s := range sels {
		if s.PseudoElement != "" {
			n++
		}
	}
	return n
}

func samePseudo(sels []Selector) bool {
	for _, s := range sels {
		if s.PseudoElement != sels[0].PseudoElement {
			return false
		}
	}
	return true
}

func firstPseudo(sels []Selector) string {
	for _, s := range sels {
		if s.PseudoElement != "" {
			return s.PseudoElement
		}
	}
	return ""
}

func trim(cs []Component) []Component {
	for len(cs) > 0 && cs[0].Kind == Whitespace {
		cs = cs[1:]
	}
	for len(cs) > 0 && cs[len(cs)-1].Kind == Whitespace {
		cs = cs[:len(cs)-1]
	}
	return cs
}
