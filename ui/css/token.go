// Package css reads ui's CSS subset: tokens and rules as CSS Syntax
// Level 3 defines them, selectors, the cascade and computed values.
package css

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Kind is the kind of a Token.
type Kind uint8

const (
	EOF Kind = iota
	Ident
	Function
	AtKeyword
	Hash
	String
	BadString
	URL
	BadURL
	Delim
	Number
	Percentage
	Dimension
	Whitespace
	CDO
	CDC
	Colon
	Semicolon
	Comma
	LBracket
	RBracket
	LParen
	RParen
	LBrace
	RBrace
)

//nolint:gochecknoglobals // lookup table
var kindNames = [...]string{
	EOF: "end of file", Ident: "ident", Function: "function", AtKeyword: "at-keyword",
	Hash: "hash", String: "string", BadString: "bad string", URL: "url", BadURL: "bad url",
	Delim: "delim", Number: "number", Percentage: "percentage", Dimension: "dimension",
	Whitespace: "whitespace", CDO: "<!--", CDC: "-->", Colon: ":", Semicolon: ";",
	Comma: ",", LBracket: "[", RBracket: "]", LParen: "(", RParen: ")", LBrace: "{", RBrace: "}",
}

func (k Kind) String() string {
	if int(k) < len(kindNames) {
		return kindNames[k]
	}
	return fmt.Sprintf("Kind(%d)", k)
}

// Pos is a place in a source: 1-based line and column, counted in runes.
type Pos struct {
	Line, Col int
}

// Token is one CSS token. Text is the ident, function or at-keyword name,
// the hash name, the string or url value, the delim, or a number's source;
// Unit is a dimension's unit as written.
type Token struct {
	Kind  Kind
	Text  string
	Num   float64
	Int   bool
	Unit  string
	HasID bool
	Pos   Pos
}

func (t Token) String() string {
	switch t.Kind {
	case Ident, Delim:
		return t.Text
	case Function:
		return t.Text + "("
	case AtKeyword:
		return "@" + t.Text
	case Hash:
		return "#" + t.Text
	case String:
		return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\a `).Replace(t.Text) + `"`
	case URL:
		return "url(" + t.Text + ")"
	case Number:
		return t.Text
	case Percentage:
		return t.Text + "%"
	case Dimension:
		return t.Text + t.Unit
	case Whitespace:
		return " "
	case EOF, BadString, BadURL, CDO, CDC, Colon, Semicolon, Comma, LBracket, RBracket, LParen, RParen, LBrace, RBrace:
		// nothing special for these tokens
	}
	return t.Kind.String()
}

// Tokenize splits src into tokens, ending with EOF. Comments are dropped.
func Tokenize(src string) []Token {
	z := tokenizer{src: preprocess(src), line: 1, col: 1}
	var out []Token
	for {
		t := z.next()
		out = append(out, t)
		if t.Kind == EOF {
			return out
		}
	}
}

func preprocess(s string) string {
	if !strings.ContainsAny(s, "\r\f\x00") {
		return s
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.NewReplacer("\r", "\n", "\f", "\n", "\x00", "\uFFFD").Replace(s)
}

type tokenizer struct {
	src       string
	i         int
	line, col int
}

const eof = -1

func (z *tokenizer) peekAt(n int) rune {
	i := z.i
	for ; n > 0; n-- {
		if i >= len(z.src) {
			return eof
		}
		_, w := utf8.DecodeRuneInString(z.src[i:])
		i += w
	}
	if i >= len(z.src) {
		return eof
	}
	r, _ := utf8.DecodeRuneInString(z.src[i:])
	return r
}

func (z *tokenizer) peek() rune { return z.peekAt(0) }

func (z *tokenizer) read() rune {
	if z.i >= len(z.src) {
		return eof
	}
	r, w := utf8.DecodeRuneInString(z.src[z.i:])
	z.i += w
	if r == '\n' {
		z.line++
		z.col = 1
	} else {
		z.col++
	}
	return r
}

func (z *tokenizer) next() Token {
	for z.peek() == '/' && z.peekAt(1) == '*' {
		z.read()
		z.read()
		for {
			r := z.read()
			if r == eof || r == '*' && z.peek() == '/' {
				z.read()
				break
			}
		}
	}
	pos := Pos{z.line, z.col}
	tok := z.consume()
	tok.Pos = pos
	return tok
}

func (z *tokenizer) consume() Token {
	r := z.peek()
	if k := punctuation(r); k != Delim {
		z.read()
		return Token{Kind: k}
	}
	switch {
	case r == eof:
		return Token{Kind: EOF}
	case isSpace(r):
		for isSpace(z.peek()) {
			z.read()
		}
		return Token{Kind: Whitespace}
	case r == '"' || r == '\'':
		z.read()
		return z.string(r)
	case r == '#':
		z.read()
		if isName(z.peek()) || validEscape(z.peek(), z.peekAt(1)) {
			id := z.startsIdent(z.peek(), z.peekAt(1), z.peekAt(2))
			return Token{Kind: Hash, Text: z.name(), HasID: id}
		}
		return Token{Kind: Delim, Text: "#"}
	case r == '+' || r == '.':
		if z.startsNumber(r, z.peekAt(1), z.peekAt(2)) {
			return z.numeric()
		}
	case r == '-':
		if z.startsNumber(r, z.peekAt(1), z.peekAt(2)) {
			return z.numeric()
		}
		if z.peekAt(1) == '-' && z.peekAt(2) == '>' {
			for range 3 {
				z.read()
			}
			return Token{Kind: CDC}
		}
		if z.startsIdent(r, z.peekAt(1), z.peekAt(2)) {
			return z.identLike()
		}
	case r == '<':
		if z.peekAt(1) == '!' && z.peekAt(2) == '-' && z.peekAt(3) == '-' {
			for range 4 {
				z.read()
			}
			return Token{Kind: CDO}
		}
	case r == '@':
		if z.startsIdent(z.peekAt(1), z.peekAt(2), z.peekAt(3)) {
			z.read()
			return Token{Kind: AtKeyword, Text: z.name()}
		}
	case r == '\\':
		if validEscape(r, z.peekAt(1)) {
			return z.identLike()
		}
	case isDigit(r):
		return z.numeric()
	case isNameStart(r):
		return z.identLike()
	}
	z.read()
	return Token{Kind: Delim, Text: string(r)}
}

func (z *tokenizer) string(quote rune) Token {
	var b strings.Builder
	for {
		switch z.peek() {
		case eof:
			return Token{Kind: String, Text: b.String()}
		case quote:
			z.read()
			return Token{Kind: String, Text: b.String()}
		case '\n':
			return Token{Kind: BadString, Text: b.String()}
		case '\\':
			next := z.peekAt(1)
			if next == eof {
				z.read()
				continue
			}
			if next == '\n' {
				z.read()
				z.read()
				continue
			}
			z.read()
			b.WriteRune(z.escape())
		default:
			b.WriteRune(z.read())
		}
	}
}

func (z *tokenizer) escape() rune {
	r := z.read()
	if r == eof {
		return utf8.RuneError
	}
	if !isHex(r) {
		return r
	}
	v := hexVal(r)
	for n := 1; n < 6 && isHex(z.peek()); n++ {
		v = v*16 + hexVal(z.read())
	}
	if isSpace(z.peek()) {
		z.read()
	}
	if v == 0 || v > utf8.MaxRune || v >= 0xD800 && v <= 0xDFFF {
		return utf8.RuneError
	}
	return v
}

func (z *tokenizer) name() string {
	var b strings.Builder
	for {
		r := z.peek()
		switch {
		case isName(r):
			b.WriteRune(z.read())
		case validEscape(r, z.peekAt(1)):
			z.read()
			b.WriteRune(z.escape())
		default:
			return b.String()
		}
	}
}

func (z *tokenizer) identLike() Token {
	name := z.name()
	if z.peek() != '(' {
		return Token{Kind: Ident, Text: name}
	}
	z.read()
	if !strings.EqualFold(name, "url") {
		return Token{Kind: Function, Text: name}
	}
	for isSpace(z.peek()) && isSpace(z.peekAt(1)) {
		z.read()
	}
	r := z.peek()
	if isSpace(r) {
		r = z.peekAt(1)
	}
	if r == '"' || r == '\'' {
		return Token{Kind: Function, Text: name}
	}
	return z.url()
}

func (z *tokenizer) url() Token {
	for isSpace(z.peek()) {
		z.read()
	}
	var b strings.Builder
	for {
		r := z.read()
		switch {
		case r == ')' || r == eof:
			return Token{Kind: URL, Text: b.String()}
		case isSpace(r):
			for isSpace(z.peek()) {
				z.read()
			}
			if z.peek() == ')' || z.peek() == eof {
				z.read()
				return Token{Kind: URL, Text: b.String()}
			}
			z.badURL()
			return Token{Kind: BadURL}
		case r == '"' || r == '\'' || r == '(' || isNonPrintable(r):
			z.badURL()
			return Token{Kind: BadURL}
		case r == '\\':
			if validEscape(r, z.peek()) {
				b.WriteRune(z.escape())
				continue
			}
			z.badURL()
			return Token{Kind: BadURL}
		default:
			b.WriteRune(r)
		}
	}
}

func (z *tokenizer) badURL() {
	for {
		r := z.peek()
		if r == eof {
			return
		}
		if r == ')' {
			z.read()
			return
		}
		if validEscape(r, z.peekAt(1)) {
			z.read()
			z.escape()
			continue
		}
		z.read()
	}
}

func (z *tokenizer) numeric() Token {
	start := z.i
	isInt := true
	if r := z.peek(); r == '+' || r == '-' {
		z.read()
	}
	for isDigit(z.peek()) {
		z.read()
	}
	if z.peek() == '.' && isDigit(z.peekAt(1)) {
		isInt = false
		z.read()
		for isDigit(z.peek()) {
			z.read()
		}
	}
	if r := z.peek(); r == 'e' || r == 'E' {
		r1 := z.peekAt(1)
		if isDigit(r1) || (r1 == '+' || r1 == '-') && isDigit(z.peekAt(2)) {
			isInt = false
			z.read()
			z.read()
			for isDigit(z.peek()) {
				z.read()
			}
		}
	}
	text := z.src[start:z.i]
	num, _ := strconv.ParseFloat(text, 64)
	tok := Token{Num: num, Int: isInt, Text: text}
	switch {
	case z.startsIdent(z.peek(), z.peekAt(1), z.peekAt(2)):
		tok.Kind = Dimension
		tok.Unit = z.name()
	case z.peek() == '%':
		z.read()
		tok.Kind = Percentage
	default:
		tok.Kind = Number
	}
	return tok
}

func (z *tokenizer) startsIdent(a, b, c rune) bool {
	switch {
	case a == '-':
		return isNameStart(b) || b == '-' || validEscape(b, c)
	case isNameStart(a):
		return true
	case a == '\\':
		return validEscape(a, b)
	}
	return false
}

func (z *tokenizer) startsNumber(a, b, c rune) bool {
	switch a {
	case '+', '-':
		return isDigit(b) || b == '.' && isDigit(c)
	case '.':
		return isDigit(b)
	}
	return isDigit(a)
}

// punctuation is the kind of the one-character token r, Delim when r
// isn't one.
func punctuation(r rune) Kind {
	switch r {
	case '(':
		return LParen
	case ')':
		return RParen
	case '[':
		return LBracket
	case ']':
		return RBracket
	case '{':
		return LBrace
	case '}':
		return RBrace
	case ',':
		return Comma
	case ':':
		return Colon
	case ';':
		return Semicolon
	}
	return Delim
}

func validEscape(a, b rune) bool { return a == '\\' && b != '\n' && b != eof }

func isSpace(r rune) bool { return r == ' ' || r == '\t' || r == '\n' }
func isDigit(r rune) bool { return r >= '0' && r <= '9' }
func isHex(r rune) bool   { return isDigit(r) || r >= 'a' && r <= 'f' || r >= 'A' && r <= 'F' }
func isNameStart(r rune) bool {
	return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r == '_' || r >= 0x80
}
func isName(r rune) bool { return isNameStart(r) || isDigit(r) || r == '-' }
func isNonPrintable(r rune) bool {
	return r >= 0 && r <= 8 || r == 0xB || r >= 0xE && r <= 0x1F || r == 0x7F
}

func hexVal(r rune) rune {
	switch {
	case isDigit(r):
		return r - '0'
	case r >= 'a' && r <= 'f':
		return r - 'a' + 10
	}
	return r - 'A' + 10
}
