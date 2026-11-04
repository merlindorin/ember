package css

import (
	"fmt"
	"strings"
)

func word(s string) Component { return Component{Token: Token{Kind: Ident, Text: s}} }

//nolint:gochecknoglobals // derived from the properties
var initials = func() map[string][]Component {
	m := make(map[string][]Component, len(properties))
	for name, p := range properties {
		m[name] = items(componentsOf(p.initial))
	}
	return m
}()

// initialValue is a longhand's initial value as components.
func initialValue(name string) []Component { return initials[name] }

func componentsOf(src string) []Component {
	p := parser{toks: Tokenize(src)}
	var out []Component
	for p.peek().Kind != EOF {
		out = append(out, p.component())
	}
	return out
}

// validate checks a declaration's value, without var(), against its
// property's grammar.
func validate(name string, value []Component) error {
	v := items(value)
	if globalKeyword(v) != "" {
		return nil
	}
	if _, ok := shorthands[name]; ok {
		_, err := expand(name, v)
		return err
	}
	if c, ok := checks[name]; ok {
		return c(v)
	}
	return nil
}

// expand splits a shorthand's value, given without whitespace, into its
// longhands; a longhand the value leaves out gets its initial value.
func expand(name string, v []Component) (map[string][]Component, error) {
	long := shorthands[name]
	out := make(map[string][]Component, len(long))
	if k := globalKeyword(v); k != "" {
		for _, l := range long {
			out[l] = []Component{word(k)}
		}
		return out, nil
	}
	var err error
	switch name {
	case "margin", "padding", kwInset, "border-width", "border-style", propBorderColor, "border-radius":
		err = expandBox(long, v, out)
	case "gap", "overflow", "place-items", "place-content", "place-self":
		err = expandPair(long, v, out)
	case kwFlex:
		err = expandFlex(v, out)
	case "flex-flow":
		err = expandAnyOrder(long, v, out)
	case "grid-column", "grid-row":
		err = expandGridLines(long, v, out)
	case "border-top", "border-right", "border-bottom", "border-left", "outline":
		err = expandAnyOrder(long, v, out)
	case "background":
		err = expandBackground(v, out)
	case "border":
		side := shorthands["border-top"]
		one := map[string][]Component{}
		if err = expandAnyOrder(side, v, one); err == nil {
			for _, l := range side {
				if one[l] == nil {
					one[l] = initialValue(l)
				}
			}
			for i, l := range long[:12] {
				out[l] = one[side[i/4]]
			}
			for _, l := range long[12:] {
				out[l] = initialValue(l)
			}
		}
	case "border-image":
		err = expandBorderImage(v, out)
	case "transition":
		err = expandTransition(v, out)
	case "list-style":
		if len(v) != 1 || checks["list-style-type"](v) != nil {
			err = fmt.Errorf("list-style takes one list-style-type, such as none")
		} else {
			out["list-style-type"] = v
		}
	default:
		err = fmt.Errorf("%s can't be expanded", name)
	}
	if err != nil {
		return nil, err
	}
	for _, l := range long {
		if _, ok := out[l]; !ok {
			out[l] = initialValue(l)
		}
	}
	return out, nil
}

// expandBox fills four sides (or corners) from one to four values.
func expandBox(long []string, v []Component, out map[string][]Component) error {
	if len(v) < 1 || len(v) > 4 {
		return fmt.Errorf("expected 1 to 4 values, found %d", len(v))
	}
	for _, c := range v {
		if isSlash(c) {
			return fmt.Errorf("elliptical corners are not supported")
		}
		if err := checks[long[0]]([]Component{c}); err != nil {
			return err
		}
	}
	pick := [4][4]int{{0, 0, 0, 0}, {0, 1, 0, 1}, {0, 1, 2, 1}, {0, 1, 2, 3}}[len(v)-1]
	for i, l := range long {
		out[l] = []Component{v[pick[i]]}
	}
	return nil
}

// expandPair fills two longhands from one or two values.
func expandPair(long []string, v []Component, out map[string][]Component) error {
	if len(v) < 1 || len(v) > 2 {
		return fmt.Errorf("expected 1 or 2 values, found %d", len(v))
	}
	for i, l := range long {
		c := v[min(i, len(v)-1)]
		if err := checks[l]([]Component{c}); err != nil {
			return err
		}
		out[l] = []Component{c}
	}
	return nil
}

// expandAnyOrder gives each value to the first longhand that takes it and
// has none yet.
func expandAnyOrder(long []string, v []Component, out map[string][]Component) error {
	if len(v) == 0 || len(v) > len(long) {
		return fmt.Errorf("expected 1 to %d values, found %d", len(long), len(v))
	}
next:
	for _, c := range v {
		for _, l := range long {
			if _, taken := out[l]; !taken && checks[l]([]Component{c}) == nil {
				out[l] = []Component{c}
				continue next
			}
		}
		return fmt.Errorf("%q fits none of %s", c.String(), strings.Join(long, ", "))
	}
	return nil
}

func expandFlex(v []Component, out map[string][]Component) error {
	num := func(s string) Component {
		return Component{Token: Token{Kind: Number, Num: parseNum(s), Int: true, Text: s}}
	}
	switch {
	case len(v) == 1 && isWord(v[0], kwNone):
		out["flex-grow"], out["flex-shrink"] = []Component{num("0")}, []Component{num("0")}
		out["flex-basis"] = []Component{word("auto")}
		return nil
	case len(v) == 1 && isWord(v[0], "auto"):
		out["flex-grow"], out["flex-shrink"] = []Component{num("1")}, []Component{num("1")}
		out["flex-basis"] = []Component{word("auto")}
		return nil
	case len(v) < 1 || len(v) > 3:
		return fmt.Errorf("expected 1 to 3 values, found %d", len(v))
	}
	var factors []Component
	var basis []Component
	for _, c := range v {
		if c.Kind == Number && len(factors) < 2 && (len(factors) == 0 || basis == nil) {
			if c.Num < 0 {
				return fmt.Errorf("flex factors can't be negative")
			}
			factors = append(factors, c)
			continue
		}
		if basis != nil || checks["flex-basis"]([]Component{c}) != nil {
			return fmt.Errorf("bad flex value %q", c.String())
		}
		basis = []Component{c}
	}
	out["flex-grow"] = []Component{num("1")}
	out["flex-shrink"] = []Component{num("1")}
	out["flex-basis"] = []Component{{Token: Token{Kind: Percentage, Text: "0"}}}
	if len(factors) > 0 {
		out["flex-grow"] = factors[:1]
	}
	if len(factors) > 1 {
		out["flex-shrink"] = factors[1:2]
	}
	if basis != nil {
		out["flex-basis"] = basis
	}
	return nil
}

func parseNum(s string) float64 {
	t := Tokenize(s)[0]
	return t.Num
}

func expandGridLines(long []string, v []Component, out map[string][]Component) error {
	parts := splitOn(v, isSlash)
	if len(parts) > 2 {
		return fmt.Errorf("expected start / end")
	}
	for i, p := range parts {
		if err := checkGridLine(p); err != nil {
			return err
		}
		out[long[i]] = p
	}
	if len(parts) == 1 {
		out[long[1]] = []Component{word("auto")}
	}
	return nil
}

// expandImageSlice reads border-image's slice from v[i], then its width
// and outset after slashes, and returns where it stopped.
func expandImageSlice(v []Component, i int, out map[string][]Component) (int, error) {
	j := i
	for j < len(v) && (v[j].Kind == Number || v[j].Kind == Percentage || isWord(v[j], "fill")) {
		j++
	}
	if err := checkImageSlice(v[i:j]); err != nil {
		return i, err
	}
	out["border-image-slice"] = v[i:j]
	i = j
	for _, l := range []string{propBorderImageWidth, propBorderImageOutset} {
		if i >= len(v) || !isSlash(v[i]) {
			break
		}
		i++
		j = i
		for j < len(v) && !isSlash(v[j]) && checks[l]([]Component{v[j]}) == nil {
			j++
		}
		if j > i {
			out[l] = v[i:j]
		} else if l == propBorderImageOutset || j >= len(v) || !isSlash(v[j]) {
			return i, fmt.Errorf("expected %s after '/'", strings.TrimPrefix(l, "border-image-"))
		}
		i = j
	}
	return i, nil
}

func expandBorderImage(v []Component, out map[string][]Component) error {
	i := 0
	for i < len(v) {
		c := v[i]
		switch {
		case isWord(c, kwNone) || urlTerm(c) == nil:
			if out["border-image-source"] != nil {
				return fmt.Errorf("border-image has two sources")
			}
			out["border-image-source"] = []Component{c}
			i++
		case isWord(c, "stretch", "repeat", "round", "space"):
			if out["border-image-repeat"] != nil {
				return fmt.Errorf("border-image has two repeats")
			}
			j := i + 1
			if j < len(v) && isWord(v[j], "stretch", "repeat", "round", "space") {
				j++
			}
			out["border-image-repeat"] = v[i:j]
			i = j
		case c.Kind == Number || c.Kind == Percentage || isWord(c, "fill"):
			if out["border-image-slice"] != nil {
				return fmt.Errorf("border-image has two slices")
			}
			var err error
			if i, err = expandImageSlice(v, i, out); err != nil {
				return err
			}
		default:
			return fmt.Errorf("bad border-image part %q", c.String())
		}
	}
	return nil
}

func expandTransition(v []Component, out map[string][]Component) error {
	long := shorthands["transition"]
	lists := make(map[string][]Component, len(long))
	for n, part := range splitOn(v, func(c Component) bool { return c.Kind == Comma }) {
		if len(part) == 0 {
			return fmt.Errorf("empty transition")
		}
		one := map[string]Component{}
		for _, c := range part {
			switch {
			case timingFunction(c) == nil && one["transition-timing-function"].Kind == EOF:
				one["transition-timing-function"] = c
			case c.Kind == Dimension && timeTerm(false)(c) == nil:
				switch {
				case one["transition-duration"].Kind == EOF:
					if timeTerm(true)(c) != nil {
						return fmt.Errorf("a duration can't be negative")
					}
					one["transition-duration"] = c
				case one["transition-delay"].Kind == EOF:
					one["transition-delay"] = c
				default:
					return fmt.Errorf("a transition has at most two times")
				}
			case c.Kind == Ident && one["transition-property"].Kind == EOF:
				if !isWord(c, "all", kwNone) {
					if err := transitionTarget(c); err != nil {
						return err
					}
				}
				one["transition-property"] = c
			default:
				return fmt.Errorf("bad transition part %q", c.String())
			}
		}
		for _, l := range long {
			c, ok := one[l]
			if !ok {
				c = initialValue(l)[0]
			}
			if n > 0 {
				lists[l] = append(lists[l], Component{Token: Token{Kind: Comma}})
			}
			lists[l] = append(lists[l], c)
		}
	}
	for l, c := range lists {
		out[l] = c
	}
	return nil
}

// expandBackground reads background's colour, image and repeat in any order, and the
// rest as its position.
func expandBackground(v []Component, out map[string][]Component) error {
	var pos []Component
	for _, c := range v {
		switch {
		case out["background-color"] == nil && colorTerm(c) == nil:
			out["background-color"] = []Component{c}
		case out["background-image"] == nil && (isWord(c, kwNone) || imageTerm(c) == nil):
			out["background-image"] = []Component{c}
		case out["background-repeat"] == nil && isWord(c, "repeat", "no-repeat"):
			out["background-repeat"] = []Component{c}
		default:
			pos = append(pos, c)
		}
	}
	if len(pos) > 0 {
		if err := checkPosition(pos); err != nil {
			return err
		}
		out["background-position"] = pos
	}
	return nil
}
