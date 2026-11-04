package css

import (
	"path"
	"strings"
)

// FontFace is an @font-face rule: a family's face for a weight and style,
// read from Src, a path in the sheet's file system.
type FontFace struct {
	Family string
	Src    string
	Weight int
	Italic bool
}

// FontFaces lists the sheet's @font-face rules in order.
func (s *Sheet) FontFaces() []FontFace { return s.faces }

func (p *parser) fontFace(pos Pos) (FontFace, bool) {
	var block []Component
	for {
		t := p.peek()
		if t.Kind == EOF {
			p.warnf(pos, "@font-face has no { block }")
			return FontFace{}, false
		}
		c := p.component()
		if c.Kind == LBrace {
			block = c.Args
			break
		}
		if c.Kind != Whitespace {
			p.warnf(c.Pos, "@font-face takes no prelude")
		}
	}
	f := FontFace{Weight: 400}
	for _, part := range splitOn(block, func(c Component) bool { return c.Kind == Semicolon }) {
		part = trim(part)
		if len(part) == 0 {
			continue
		}
		name := part[0]
		v := items(part[1:])
		if name.Kind != Ident || len(v) < 2 || v[0].Kind != Colon {
			p.warnf(name.Pos, "expected a descriptor in @font-face")
			continue
		}
		v = v[1:]
		switch strings.ToLower(name.Text) {
		case propFontFamily:
			if fam := familyName(v); fam != "" {
				f.Family = fam
			} else {
				p.warnf(v[0].Pos, "font-family: expected one family name")
			}
		case "src":
			if src := imageSource(nil, v[:1]); src != "" && urlTerm(v[0]) == nil {
				f.Src = path.Join(path.Dir(p.file), src)
			} else {
				p.warnf(v[0].Pos, "src: expected url(file.ttf)")
			}
		case propFontWeight:
			switch {
			case len(v) == 1 && isWord(v[0], "normal"):
				f.Weight = 400
			case len(v) == 1 && isWord(v[0], "bold"):
				f.Weight = 700
			case len(v) == 1 && fontWeightNumber(v[0]) == nil:
				f.Weight = int(v[0].Num)
			default:
				p.warnf(v[0].Pos, "font-weight: expected normal, bold or 1 to 1000")
			}
		case propFontStyle:
			if len(v) == 1 && isWord(v[0], "normal", "italic", "oblique") {
				f.Italic = !isWord(v[0], "normal")
			} else {
				p.warnf(v[0].Pos, "font-style: expected normal or italic")
			}
		default:
			p.warnf(name.Pos, "@font-face: %q is not supported", name.Text)
		}
	}
	if f.Family == "" || f.Src == "" {
		p.warnf(pos, "@font-face needs font-family and src; dropped")
		return FontFace{}, false
	}
	return f, true
}

func familyName(v []Component) string {
	if len(v) == 1 && v[0].Kind == String {
		return v[0].Text
	}
	if len(v) == 0 || !allIdents(v) {
		return ""
	}
	var names []string
	for _, c := range v {
		names = append(names, c.Text)
	}
	return strings.Join(names, " ")
}
