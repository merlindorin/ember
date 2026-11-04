package css

import (
	"fmt"
	"math"
	"strings"
)

// Dim is a length, percentage or number as a sum of parts, the form calc()
// resolves to before layout: calc(100% - 2rem + 4px) is Pct 100, Rem -2,
// Px 4. A plain value has one part.
type Dim struct {
	Num, Px, Pct, Em, Rem, Fr float64
	kind                      dimKind
}

type dimKind uint8

const (
	dimNumber dimKind = 1 << iota
	dimLength
	dimPercent
	dimFlex
)

// IsNumber reports whether d is a plain number.
func (d Dim) IsNumber() bool { return d.kind == dimNumber }

// HasPercent reports whether d depends on a percentage.
func (d Dim) HasPercent() bool { return d.kind&dimPercent != 0 }

// IsFlex reports whether d is a fr value.
func (d Dim) IsFlex() bool { return d.kind == dimFlex }

func (d Dim) add(o Dim, sign float64) (Dim, error) {
	if (d.kind == dimNumber) != (o.kind == dimNumber) {
		return Dim{}, fmt.Errorf("calc() can't add a number to a length")
	}
	if (d.kind|o.kind)&dimFlex != 0 {
		return Dim{}, fmt.Errorf("fr can't be used in calc()")
	}
	return Dim{
		Num: d.Num + sign*o.Num, Px: d.Px + sign*o.Px, Pct: d.Pct + sign*o.Pct,
		Em: d.Em + sign*o.Em, Rem: d.Rem + sign*o.Rem,
		kind: d.kind | o.kind,
	}, nil
}

func (d Dim) scale(f float64) Dim {
	return Dim{Num: d.Num * f, Px: d.Px * f, Pct: d.Pct * f, Em: d.Em * f, Rem: d.Rem * f, Fr: d.Fr * f, kind: d.kind}
}

// Absolute resolves em and rem against font sizes in points, leaving Px
// and Pct.
func (d Dim) Absolute(em, rem float64) Dim {
	if d.kind == dimNumber {
		return d
	}
	d.Px += d.Em*em + d.Rem*rem
	d.Em, d.Rem = 0, 0
	return d
}

// Resolve returns d in points against basis, the length 100% stands for.
// em and rem must be resolved first (Absolute).
func (d Dim) Resolve(basis float64) float64 {
	return d.Px + d.Pct*basis/100
}

// parseDim reads one component as a length, percentage, number or flex
// value, calc() included.
func parseDim(c Component) (Dim, error) {
	switch c.Kind {
	case Number:
		return Dim{Num: c.Num, kind: dimNumber}, nil
	case Percentage:
		return Dim{Pct: c.Num, kind: dimPercent}, nil
	case Dimension:
		switch strings.ToLower(c.Unit) {
		case "px":
			return Dim{Px: c.Num, kind: dimLength}, nil
		case "em":
			return Dim{Em: c.Num, kind: dimLength}, nil
		case "rem":
			return Dim{Rem: c.Num, kind: dimLength}, nil
		case "fr":
			return Dim{Fr: c.Num, kind: dimFlex}, nil
		}
		return Dim{}, fmt.Errorf("unit %q is not supported; use px, %%, em, rem or fr", c.Unit)
	case Function:
		if strings.EqualFold(c.Text, "calc") {
			return parseCalc(c.Args)
		}
		return Dim{}, fmt.Errorf("%s() is not supported here", c.Text)
	case EOF, Ident, AtKeyword, Hash, String, BadString, URL, BadURL, Delim, Whitespace, CDO, CDC, Colon, Semicolon,
		Comma, LBracket, RBracket, LParen, RParen, LBrace, RBrace:
		// nothing special for these tokens
	}
	return Dim{}, fmt.Errorf("expected a length, found %q", c.String())
}

func parseCalc(args []Component) (Dim, error) {
	cp := calcParser{cs: trim(args)}
	if len(cp.cs) == 0 {
		return Dim{}, fmt.Errorf("calc() is empty")
	}
	d, err := cp.sum()
	if err != nil {
		return Dim{}, err
	}
	if cp.i < len(cp.cs) {
		return Dim{}, fmt.Errorf("unexpected %q in calc()", cp.cs[cp.i].String())
	}
	return d, nil
}

type calcParser struct {
	cs []Component
	i  int
}

func (p *calcParser) space() bool {
	saw := false
	for p.i < len(p.cs) && p.cs[p.i].Kind == Whitespace {
		p.i++
		saw = true
	}
	return saw
}

func (p *calcParser) sum() (Dim, error) {
	d, err := p.product()
	if err != nil {
		return Dim{}, err
	}
	for {
		start := p.i
		before := p.space()
		if p.i >= len(p.cs) || p.cs[p.i].Kind != Delim || p.cs[p.i].Text != "+" && p.cs[p.i].Text != "-" {
			p.i = start
			return d, nil
		}
		op := p.cs[p.i].Text
		p.i++
		if !before || !p.space() {
			return Dim{}, fmt.Errorf("calc() needs spaces around %q", op)
		}
		var o Dim
		if o, err = p.product(); err != nil {
			return Dim{}, err
		}
		sign := 1.0
		if op == "-" {
			sign = -1
		}
		if d, err = d.add(o, sign); err != nil {
			return Dim{}, err
		}
	}
}

func (p *calcParser) product() (Dim, error) {
	d, err := p.value()
	if err != nil {
		return Dim{}, err
	}
	for {
		start := p.i
		p.space()
		if p.i >= len(p.cs) || p.cs[p.i].Kind != Delim || p.cs[p.i].Text != "*" && p.cs[p.i].Text != "/" {
			p.i = start
			return d, nil
		}
		op := p.cs[p.i].Text
		p.i++
		p.space()
		var o Dim
		if o, err = p.value(); err != nil {
			return Dim{}, err
		}
		switch {
		case op == "/" && !o.IsNumber():
			return Dim{}, fmt.Errorf("calc() can only divide by a number")
		case op == "/" && o.Num == 0:
			return Dim{}, fmt.Errorf("calc() divides by zero")
		case op == "/":
			d = d.scale(1 / o.Num)
		case o.IsNumber():
			d = d.scale(o.Num)
		case d.IsNumber():
			d = o.scale(d.Num)
		default:
			return Dim{}, fmt.Errorf("calc() can't multiply two lengths")
		}
	}
}

func (p *calcParser) value() (Dim, error) {
	if p.i >= len(p.cs) {
		return Dim{}, fmt.Errorf("calc() ends too early")
	}
	c := p.cs[p.i]
	p.i++
	if c.Kind == LParen {
		return parseCalc(c.Args)
	}
	return parseDim(c)
}

// Color is a colour in sRGB, each channel 0–1, alpha not premultiplied.
type Color struct {
	R, G, B, A float64
}

func (c Color) String() string {
	ch := func(v float64) int { return int(math.Round(v * 255)) }
	if c.A >= 1 {
		return fmt.Sprintf("rgb(%d, %d, %d)", ch(c.R), ch(c.G), ch(c.B))
	}
	return fmt.Sprintf("rgba(%d, %d, %d, %s)", ch(c.R), ch(c.G), ch(c.B), trimFloat(c.A))
}

func trimFloat(v float64) string {
	s := fmt.Sprintf("%.3f", v)
	s = strings.TrimRight(s, "0")
	return strings.TrimSuffix(s, ".")
}

// parseColor reads a colour: hex, rgb(), rgba(), oklch(), a named colour
// or transparent.
// currentcolor is the caller's to handle.
func parseColor(c Component) (Color, error) {
	switch c.Kind {
	case Hash:
		return parseHex(c.Text)
	case Ident:
		if strings.EqualFold(c.Text, "transparent") {
			return Color{}, nil
		}
		if v, ok := named[strings.ToLower(c.Text)]; ok {
			return Color{float64(v>>16) / 255, float64(v>>8&0xff) / 255, float64(v&0xff) / 255, 1}, nil
		}
		return Color{}, fmt.Errorf("unknown colour %q", c.Text)
	case Function:
		switch strings.ToLower(c.Text) {
		case "rgb", "rgba":
			return parseRGB(c.Args)
		case "oklch":
			return parseOKLCH(c.Args)
		}
		return Color{}, fmt.Errorf("%s() colours are not supported; use hex, rgb(), oklch() or a name", c.Text)
	case EOF, AtKeyword, String, BadString, URL, BadURL, Delim, Number, Percentage, Dimension, Whitespace, CDO, CDC,
		Colon, Semicolon, Comma, LBracket, RBracket, LParen, RParen, LBrace, RBrace:
		// nothing special for these tokens
	}
	return Color{}, fmt.Errorf("expected a colour, found %q", c.String())
}

func parseHex(s string) (Color, error) {
	v := [4]float64{3: 1}
	switch len(s) {
	case 3, 4:
		for i, r := range s {
			if !isHex(r) {
				return Color{}, fmt.Errorf("bad hex colour #%s", s)
			}
			v[i] = float64(hexVal(r)*17) / 255
		}
	case 6, 8:
		for i := 0; i < len(s); i += 2 {
			a, b := rune(s[i]), rune(s[i+1])
			if !isHex(a) || !isHex(b) {
				return Color{}, fmt.Errorf("bad hex colour #%s", s)
			}
			v[i/2] = float64(hexVal(a)*16+hexVal(b)) / 255
		}
	default:
		return Color{}, fmt.Errorf("bad hex colour #%s", s)
	}
	return Color{v[0], v[1], v[2], v[3]}, nil
}

// colorArgs splits a colour function's arguments into its channels and
// alpha, in the modern (space and slash) or legacy (comma) form.
func colorArgs(args []Component, legacyOK bool) ([3]Component, *Component, bool, error) {
	var ch [3]Component
	var alpha *Component
	legacy := false
	var buf [16]*Component
	parts := buf[:0]
	for i := range args {
		if args[i].Kind == Whitespace {
			continue
		}
		if len(parts) == len(buf) {
			return ch, nil, false, fmt.Errorf("expected three channels, found more")
		}
		parts = append(parts, &args[i])
		if args[i].Kind == Comma {
			legacy = true
		}
	}
	n := 0
	take := func(c *Component) {
		if n < len(ch) {
			ch[n] = *c
		}
		n++
	}
	switch {
	case legacy && !legacyOK:
		return ch, nil, false, fmt.Errorf("commas are not allowed here")
	case legacy:
		if !commasBetween(parts) {
			return ch, nil, false, fmt.Errorf("bad commas in colour")
		}
		for i := 0; i < len(parts); i += 2 {
			if n == 3 && i == len(parts)-1 {
				alpha = parts[i]
				break
			}
			take(parts[i])
		}
	default:
		i, err := slash(parts)
		if err != nil {
			return ch, nil, false, err
		}
		if i < len(parts) {
			alpha = parts[i+1]
		}
		for _, p := range parts[:i] {
			take(p)
		}
	}
	if n != 3 {
		return ch, nil, false, fmt.Errorf("expected three channels, found %d", n)
	}
	return ch, alpha, legacy, nil
}

// commasBetween reports whether parts alternate values and commas.
func commasBetween(parts []*Component) bool {
	for i, p := range parts {
		if (i%2 == 1) != (p.Kind == Comma) {
			return false
		}
	}
	return true
}

// slash is where the modern form's '/' stands in parts, before its one
// alpha; len(parts) without one.
func slash(parts []*Component) (int, error) {
	for i, p := range parts {
		if p.Kind == Delim && p.Text == "/" {
			if i != len(parts)-2 {
				return 0, fmt.Errorf("expected one alpha after '/'")
			}
			return i, nil
		}
	}
	return len(parts), nil
}

func channel(c Component, pct float64, numScale float64) (float64, error) {
	switch c.Kind {
	case Number:
		return c.Num * numScale, nil
	case Percentage:
		return c.Num / 100 * pct, nil
	case Ident:
		if strings.EqualFold(c.Text, kwNone) {
			return 0, nil
		}
	case EOF, Function, AtKeyword, Hash, String, BadString, URL, BadURL, Delim, Dimension, Whitespace, CDO, CDC, Colon,
		Semicolon, Comma, LBracket, RBracket, LParen, RParen, LBrace, RBrace:
		// nothing special for these tokens
	}
	return 0, fmt.Errorf("bad colour channel %q", c.String())
}

func parseAlpha(a *Component) (float64, error) {
	if a == nil {
		return 1, nil
	}
	v, err := channel(*a, 1, 1)
	return clamp01(v), err
}

func parseRGB(args []Component) (Color, error) {
	ch, alpha, legacy, err := colorArgs(args, true)
	if err != nil {
		return Color{}, fmt.Errorf("rgb(): %w", err)
	}
	var v [3]float64
	for i, c := range ch {
		if legacy && (c.Kind != ch[0].Kind || c.Kind == Ident) {
			return Color{}, fmt.Errorf("rgb(): legacy channels must all be numbers or all percentages")
		}
		if v[i], err = channel(c, 1, 1.0/255); err != nil {
			return Color{}, fmt.Errorf("rgb(): %w", err)
		}
		v[i] = clamp01(v[i])
	}
	a, err := parseAlpha(alpha)
	if err != nil {
		return Color{}, fmt.Errorf("rgb(): %w", err)
	}
	return Color{v[0], v[1], v[2], a}, nil
}

func parseOKLCH(args []Component) (Color, error) {
	ch, alpha, _, err := colorArgs(args, false)
	if err != nil {
		return Color{}, fmt.Errorf("oklch(): %w", err)
	}
	l, err := channel(ch[0], 1, 1)
	if err != nil {
		return Color{}, fmt.Errorf("oklch(): %w", err)
	}
	c, err := channel(ch[1], 0.4, 1)
	if err != nil {
		return Color{}, fmt.Errorf("oklch(): %w", err)
	}
	h, err := hue(ch[2])
	if err != nil {
		return Color{}, fmt.Errorf("oklch(): %w", err)
	}
	a, err := parseAlpha(alpha)
	if err != nil {
		return Color{}, fmt.Errorf("oklch(): %w", err)
	}
	col := oklchToSRGB(clamp01(l), max(c, 0), h)
	col.A = a
	return col, nil
}

func hue(c Component) (float64, error) {
	switch c.Kind {
	case Number:
		return c.Num, nil
	case Ident:
		if strings.EqualFold(c.Text, kwNone) {
			return 0, nil
		}
	case Dimension:
		if a, err := angle(c); err == nil {
			return a, nil
		}
	case EOF, Function, AtKeyword, Hash, String, BadString, URL, BadURL, Delim, Percentage, Whitespace, CDO, CDC, Colon,
		Semicolon, Comma, LBracket, RBracket, LParen, RParen, LBrace, RBrace:
		// nothing special for these tokens
	}
	return 0, fmt.Errorf("bad hue %q", c.String())
}

// angle reads an angle in degrees.
func angle(c Component) (float64, error) {
	if c.Kind == Number && c.Num == 0 {
		return 0, nil
	}
	if c.Kind == Dimension {
		switch strings.ToLower(c.Unit) {
		case "deg":
			return c.Num, nil
		case "rad":
			return c.Num * 180 / math.Pi, nil
		case "grad":
			return c.Num * 0.9, nil
		case "turn":
			return c.Num * 360, nil
		}
	}
	return 0, fmt.Errorf("expected an angle, found %q", c.String())
}

// duration reads a time in seconds.
func duration(c Component) (float64, error) {
	if c.Kind == Dimension {
		switch strings.ToLower(c.Unit) {
		case "s":
			return c.Num, nil
		case "ms":
			return c.Num / 1000, nil
		}
	}
	return 0, fmt.Errorf("expected a time in s or ms, found %q", c.String())
}

func oklchToSRGB(l, c, h float64) Color {
	hr := h * math.Pi / 180
	a, b := c*math.Cos(hr), c*math.Sin(hr)
	lr := l + 0.3963377774*a + 0.2158037573*b
	mr := l - 0.1055613458*a - 0.0638541728*b
	sr := l - 0.0894841775*a - 1.2914855480*b
	l3, m3, s3 := lr*lr*lr, mr*mr*mr, sr*sr*sr
	r := 4.0767416621*l3 - 3.3077115913*m3 + 0.2309699292*s3
	g := -1.2684380046*l3 + 2.6097574011*m3 - 0.3413193965*s3
	bl := -0.0041960863*l3 - 0.7034186147*m3 + 1.7076147010*s3
	return Color{R: encodeSRGB(r), G: encodeSRGB(g), B: encodeSRGB(bl), A: 1}
}

func encodeSRGB(v float64) float64 {
	v = clamp01(v)
	if v <= 0.0031308 {
		return 12.92 * v
	}
	return 1.055*math.Pow(v, 1/2.4) - 0.055
}

func clamp01(v float64) float64 { return min(max(v, 0), 1) }
