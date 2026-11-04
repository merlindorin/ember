package css

import (
	"fmt"
	"slices"
	"strings"
)

// items drops whitespace between components; commas and '/' stay. A
// value without whitespace comes back as is, so callers never write to
// what it returns.
func items(cs []Component) []Component {
	if !slices.ContainsFunc(cs, func(c Component) bool { return c.Kind == Whitespace }) {
		return cs
	}
	out := make([]Component, 0, len(cs))
	for _, c := range cs {
		if c.Kind != Whitespace {
			out = append(out, c)
		}
	}
	return out
}

func isWord(c Component, words ...string) bool {
	if c.Kind != Ident {
		return false
	}
	for _, w := range words {
		if strings.EqualFold(c.Text, w) {
			return true
		}
	}
	return false
}

func isSlash(c Component) bool { return c.Kind == Delim && c.Text == "/" }

// globalKeyword returns inherit, initial or unset when v is one alone.
func globalKeyword(v []Component) string {
	v = items(v)
	if len(v) == 1 && isWord(v[0], "inherit", "initial", kwUnset) {
		return strings.ToLower(v[0].Text)
	}
	return ""
}

func hasVar(cs []Component) bool {
	for _, c := range cs {
		if c.Kind == Function && strings.EqualFold(c.Text, "var") || hasVar(c.Args) {
			return true
		}
	}
	return false
}

// term checks one component of a value.
type term func(Component) error

func words(ws ...string) term {
	return func(c Component) error {
		if isWord(c, ws...) {
			return nil
		}
		return fmt.Errorf("expected %s, found %q", strings.Join(ws, ", "), c.String())
	}
}

func either(ts ...term) term {
	return func(c Component) error {
		var first error
		for _, t := range ts {
			err := t(c)
			if err == nil {
				return nil
			}
			if first == nil {
				first = err
			}
		}
		return first
	}
}

const (
	nonNegative = 1 << iota
	allowPercent
	allowNumber
	allowFlex
)

// dimTerm accepts a length (0 included), and what flags allow.
func dimTerm(flags int) term {
	return func(c Component) error {
		d, err := parseDim(c)
		if err != nil {
			return err
		}
		switch {
		case d.IsNumber() && flags&allowNumber == 0 && !(c.Kind == Number && c.Num == 0):
			return fmt.Errorf("%q needs a unit", c.String())
		case d.IsFlex() && flags&allowFlex == 0:
			return fmt.Errorf("fr is only for grid tracks")
		case d.HasPercent() && flags&allowPercent == 0:
			return fmt.Errorf("a percentage is not allowed here")
		case flags&nonNegative != 0 && c.Kind != Function && c.Num < 0:
			return fmt.Errorf("%q can't be negative", c.String())
		}
		return nil
	}
}

//nolint:gochecknoglobals,goconst // grammar terms, built once
var (
	length        = dimTerm(0)
	lengthPercent = dimTerm(allowPercent)
	sizeLP        = dimTerm(allowPercent | nonNegative)
	number        = dimTerm(allowNumber | nonNegative)
	intrinsic     = words("min-content", "max-content", "fit-content")
	colorTerm     = term(func(c Component) error {
		if isWord(c, "currentcolor") {
			return nil
		}
		_, err := parseColor(c)
		return err
	})
	urlTerm = term(func(c Component) error {
		if c.Kind == URL || c.Kind == Function && strings.EqualFold(c.Text, "url") {
			return nil
		}
		return fmt.Errorf("expected url(), found %q", c.String())
	})
	integer = term(func(c Component) error {
		if c.Kind == Number && c.Int {
			return nil
		}
		return fmt.Errorf("expected an integer, found %q", c.String())
	})
	borderWidth = either(dimTerm(nonNegative), words("thin", "medium", "thick"))
	borderStyle = words("none", "hidden", "solid", "dashed", "dotted", "inset", "outset")
	alignWords  = []string{"normal", "stretch", "center", "start", "end", "flex-start", "flex-end", "baseline",
		"self-start", "self-end"}
)

// check validates a whole longhand value, given without whitespace.
type check func(v []Component) error

func single(t term) check {
	return func(v []Component) error {
		if len(v) != 1 {
			return fmt.Errorf("expected one value, found %d", len(v))
		}
		return t(v[0])
	}
}

func repeated(t term, most int) check {
	return func(v []Component) error {
		if len(v) < 1 || len(v) > most {
			return fmt.Errorf("expected 1 to %d values, found %d", most, len(v))
		}
		for _, c := range v {
			if err := t(c); err != nil {
				return err
			}
		}
		return nil
	}
}

// commaList checks each comma-separated part of v with each.
func commaList(each check) check {
	return func(v []Component) error {
		for _, part := range splitOn(v, func(c Component) bool { return c.Kind == Comma }) {
			if len(part) == 0 {
				return fmt.Errorf("empty item in a list")
			}
			if err := each(part); err != nil {
				return err
			}
		}
		return nil
	}
}

func splitOn(v []Component, sep func(Component) bool) [][]Component {
	var out [][]Component
	start := 0
	for i, c := range v {
		if sep(c) {
			out = append(out, v[start:i])
			start = i + 1
		}
	}
	return append(out, v[start:])
}

//nolint:gochecknoglobals,goconst // the properties' grammar
var checks = map[string]check{
	"display": single(words("block", "inline", "inline-block", "flex", "inline-flex", "grid", "inline-grid", "none",
		"contents", "list-item")),
	"box-sizing": single(words("content-box", "border-box")),
	"width":      single(either(sizeLP, words("auto"), intrinsic)),
	"height":     single(either(sizeLP, words("auto"), intrinsic)),
	"min-width":  single(either(sizeLP, words("auto"), intrinsic)),
	"min-height": single(either(sizeLP, words("auto"), intrinsic)),
	"max-width":  single(either(sizeLP, words("none"), intrinsic)),
	"max-height": single(either(sizeLP, words("none"), intrinsic)),

	"margin-top":     single(either(lengthPercent, words("auto"))),
	"margin-right":   single(either(lengthPercent, words("auto"))),
	"margin-bottom":  single(either(lengthPercent, words("auto"))),
	"margin-left":    single(either(lengthPercent, words("auto"))),
	"padding-top":    single(sizeLP),
	"padding-right":  single(sizeLP),
	"padding-bottom": single(sizeLP),
	"padding-left":   single(sizeLP),

	"border-top-width":           single(borderWidth),
	"border-right-width":         single(borderWidth),
	"border-bottom-width":        single(borderWidth),
	"border-left-width":          single(borderWidth),
	"border-top-style":           single(borderStyle),
	"border-right-style":         single(borderStyle),
	"border-bottom-style":        single(borderStyle),
	"border-left-style":          single(borderStyle),
	"border-top-color":           single(colorTerm),
	"border-right-color":         single(colorTerm),
	"border-bottom-color":        single(colorTerm),
	"border-left-color":          single(colorTerm),
	"border-top-left-radius":     single(sizeLP),
	"border-top-right-radius":    single(sizeLP),
	"border-bottom-right-radius": single(sizeLP),
	"border-bottom-left-radius":  single(sizeLP),
	"border-image-source":        single(either(words("none"), urlTerm)),
	"border-image-slice":         checkImageSlice,
	"border-image-width":         repeated(either(dimTerm(allowPercent|allowNumber|nonNegative), words("auto")), 4),
	"border-image-outset":        repeated(dimTerm(allowNumber|nonNegative), 4),
	"border-image-repeat":        repeated(words("stretch", "repeat", "round", "space"), 2),

	"row-gap":        single(either(sizeLP, words("normal"))),
	"column-gap":     single(either(sizeLP, words("normal"))),
	"flex-grow":      single(number),
	"flex-shrink":    single(number),
	"flex-basis":     single(either(sizeLP, words("auto", "content"), intrinsic)),
	"flex-wrap":      single(words("nowrap", "wrap", "wrap-reverse")),
	"flex-direction": single(words("row", "row-reverse", "column", "column-reverse")),
	"align-items":    single(words(alignWords...)),
	"align-self":     single(words(append([]string{"auto"}, alignWords...)...)),
	"align-content": single(words("normal", "stretch", "center", "start", "end", "flex-start", "flex-end",
		"space-between", "space-around", "space-evenly", "baseline")),
	"justify-content": single(words("normal", "stretch", "center", "start", "end", "flex-start", "flex-end", "left",
		"right", "space-between", "space-around", "space-evenly")),
	"justify-items": single(words(append([]string{"legacy", "left", "right"}, alignWords...)...)),
	"justify-self":  single(words(append([]string{"auto", "left", "right"}, alignWords...)...)),

	"grid-template-columns": checkTracks,
	"grid-template-rows":    checkTracks,
	"grid-column-start":     checkGridLine,
	"grid-column-end":       checkGridLine,
	"grid-row-start":        checkGridLine,
	"grid-row-end":          checkGridLine,
	"grid-auto-columns":     repeated(trackSize, 64),
	"grid-auto-rows":        repeated(trackSize, 64),
	"grid-auto-flow":        checkAutoFlow,
	"order":                 single(integer),
	"aspect-ratio":          checkAspectRatio,
	"overflow-x":            single(words("visible", "hidden", "clip", "auto", "scroll")),
	"overflow-y":            single(words("visible", "hidden", "clip", "auto", "scroll")),
	"position":              single(words("static", "relative", "absolute", "fixed", "sticky")),
	"top":                   single(either(lengthPercent, words("auto"))),
	"right":                 single(either(lengthPercent, words("auto"))),
	"bottom":                single(either(lengthPercent, words("auto"))),
	"left":                  single(either(lengthPercent, words("auto"))),
	"z-index":               single(either(integer, words("auto"))),

	"font-family": checkFontFamily,
	"font-size": single(either(sizeLP,
		words("xx-small", "x-small", "small", "medium", "large", "x-large", "xx-large", "smaller", "larger"))),
	"font-style":             single(words("normal", "italic", "oblique")),
	"font-weight":            single(either(fontWeightNumber, words("normal", "bold", "bolder", "lighter"))),
	"line-height":            single(either(dimTerm(allowPercent|allowNumber|nonNegative), words("normal"))),
	"letter-spacing":         single(either(length, words("normal"))),
	"border-spacing":         repeated(length, 2),
	"caption-side":           single(words("top", "bottom")),
	"table-layout":           single(words("auto", "fixed")),
	"text-align":             single(words("start", "end", "left", "right", "center", "justify")),
	"text-transform":         single(words("none", "uppercase", "lowercase", "capitalize")),
	"-webkit-font-smoothing": single(words("auto", "none", "antialiased", "subpixel-antialiased")),
	"white-space":            single(words("normal", "nowrap", "pre", "pre-wrap", "pre-line")),
	"overflow-wrap":          single(words("normal", "break-word", "anywhere")),
	"color":                  single(colorTerm),
	"cursor": single(words("auto", "default", "none", "pointer", "text", "move", "grab", "grabbing", "not-allowed",
		"crosshair", "help", "wait", "progress", "ew-resize", "ns-resize", "col-resize", "row-resize")),
	"text-overflow":   single(words("clip", "ellipsis")),
	"scrollbar-width": single(words("auto", "thin", "none")),
	"list-style-type": single(words("none", "disc", "disclosure-open", "disclosure-closed")),
	"visibility":      single(words("visible", "hidden", "collapse")),
	"pointer-events":  single(words("auto", "none")),

	"background-color":    single(colorTerm),
	"background-image":    single(either(words("none"), imageTerm)),
	"background-repeat":   single(words("repeat", "no-repeat")),
	"background-size":     repeated(either(length, words("auto")), 2),
	"background-position": checkPosition,
	"text-shadow":         checkTextShadow,
	"image-rendering":     single(words("auto", "pixelated")),
	"accent-color":        single(either(words("auto"), colorTerm)),
	"caret-color":         single(either(words("auto"), colorTerm)),
	"object-fit":          single(words("fill", "contain", "cover", "none", "scale-down")),
	"object-position":     checkPosition,
	"box-shadow":          checkBoxShadow,
	"outline-width":       single(borderWidth),
	"outline-style":       single(borderStyle),
	"outline-color":       single(colorTerm),
	"outline-offset":      single(length),
	"opacity":             single(dimTerm(allowNumber | allowPercent)),
	"transform":           checkTransform,
	"transform-origin":    checkTransformOrigin,
	"appearance":          single(words("none", "auto", "textfield")),
	"resize":              single(words("none", "both", "horizontal", "vertical")),

	"transition-property":        commaList(single(either(words("none", "all"), transitionTarget))),
	"transition-duration":        commaList(single(timeTerm(true))),
	"transition-delay":           commaList(single(timeTerm(false))),
	"transition-timing-function": commaList(single(timingFunction)),
}

func fontWeightNumber(c Component) error {
	if c.Kind == Number && c.Num >= 1 && c.Num <= 1000 {
		return nil
	}
	return fmt.Errorf("font-weight is 1 to 1000, found %q", c.String())
}

func timeTerm(nonNeg bool) term {
	return func(c Component) error {
		d, err := duration(c)
		if err == nil && nonNeg && d < 0 {
			return fmt.Errorf("%q can't be negative", c.String())
		}
		return err
	}
}

// animated lists what transitions run on: colours, opacity and transforms.
//
//nolint:gochecknoglobals // lookup table
var animated = map[string]bool{
	"color": true, "background-color": true, "border-color": true, "outline-color": true,
	"border-top-color": true, "border-right-color": true, "border-bottom-color": true, "border-left-color": true,
	"opacity": true, "transform": true, "box-shadow": true,
}

func transitionTarget(c Component) error {
	if c.Kind == Ident && animated[strings.ToLower(c.Text)] {
		return nil
	}
	return fmt.Errorf("transitions run only on colours, opacity, transform and box-shadow, not %q", c.String())
}

func timingFunction(c Component) error {
	if isWord(c, "ease", "linear", "ease-in", "ease-out", "ease-in-out", "step-start", "step-end") {
		return nil
	}
	if c.Kind != Function {
		return fmt.Errorf("expected a timing function, found %q", c.String())
	}
	args := splitOn(items(c.Args), func(c Component) bool { return c.Kind == Comma })
	switch strings.ToLower(c.Text) {
	case "cubic-bezier":
		if len(args) != 4 {
			return fmt.Errorf("cubic-bezier() takes four numbers")
		}
		for i, a := range args {
			if len(a) != 1 || a[0].Kind != Number || i%2 == 0 && (a[0].Num < 0 || a[0].Num > 1) {
				return fmt.Errorf("bad cubic-bezier() argument")
			}
		}
		return nil
	case "steps":
		if len(args) < 1 || len(args) > 2 || len(args[0]) != 1 || integer(args[0][0]) != nil || args[0][0].Num < 1 {
			return fmt.Errorf("steps() takes a positive integer")
		}
		if len(args) == 2 &&
			(len(args[1]) != 1 || !isWord(args[1][0], "jump-start", "jump-end", "jump-none", "jump-both", kwStart, kwEnd)) {
			return fmt.Errorf("bad steps() position")
		}
		return nil
	}
	return fmt.Errorf("%s() is not a timing function", c.Text)
}

func checkImageSlice(v []Component) error {
	n := 0
	for i, c := range v {
		if isWord(c, "fill") {
			if i != 0 && i != len(v)-1 {
				return fmt.Errorf("fill goes first or last")
			}
			continue
		}
		if c.Kind != Number && c.Kind != Percentage || c.Num < 0 {
			return fmt.Errorf("border-image-slice takes numbers or percentages, found %q", c.String())
		}
		n++
	}
	if n < 1 || n > 4 {
		return fmt.Errorf("border-image-slice takes 1 to 4 values")
	}
	return nil
}

func checkGridLine(v []Component) error {
	switch {
	case len(v) == 1 && isWord(v[0], "auto"):
		return nil
	case len(v) == 1 && integer(v[0]) == nil && v[0].Num != 0:
		return nil
	case len(v) == 2 && isWord(v[0], "span") && integer(v[1]) == nil && v[1].Num > 0,
		len(v) == 2 && isWord(v[1], "span") && integer(v[0]) == nil && v[0].Num > 0:
		return nil
	}
	return fmt.Errorf("expected auto, a line number or span n")
}

func checkAspectRatio(v []Component) error {
	switch {
	case len(v) == 1 && isWord(v[0], "auto"):
		return nil
	case len(v) == 1 && v[0].Kind == Number && v[0].Num >= 0:
		return nil
	case len(v) == 3 && v[0].Kind == Number && isSlash(v[1]) && v[2].Kind == Number && v[0].Num >= 0 && v[2].Num >= 0:
		return nil
	}
	return fmt.Errorf("expected auto, a number or a ratio w / h")
}

func checkFontFamily(v []Component) error {
	for _, part := range splitOn(v, func(c Component) bool { return c.Kind == Comma }) {
		switch {
		case len(part) == 1 && part[0].Kind == String:
		case len(part) > 0 && allIdents(part):
		default:
			return fmt.Errorf("font-family names a face, a quoted file, or a list of them")
		}
	}
	return nil
}

func allIdents(v []Component) bool {
	for _, c := range v {
		if c.Kind != Ident {
			return false
		}
	}
	return true
}

//nolint:gochecknoglobals // a grammar term, built once
var trackBreadth = either(sizeLP, dimTerm(allowFlex|nonNegative), words("auto"), intrinsic)

func trackSize(c Component) error {
	if c.Kind == Function {
		args := splitOn(items(c.Args), func(c Component) bool { return c.Kind == Comma })
		switch strings.ToLower(c.Text) {
		case "minmax":
			if len(args) != 2 || len(args[0]) != 1 || len(args[1]) != 1 {
				return fmt.Errorf("minmax() takes two sizes")
			}
			if d, err := parseDim(args[0][0]); err == nil && d.IsFlex() {
				return fmt.Errorf("minmax()'s minimum can't be fr")
			}
			if err := trackBreadth(args[0][0]); err != nil {
				return err
			}
			return trackBreadth(args[1][0])
		case "fit-content":
			if len(args) != 1 || len(args[0]) != 1 {
				return fmt.Errorf("fit-content() takes one length")
			}
			return sizeLP(args[0][0])
		}
	}
	return trackBreadth(c)
}

func checkTracks(v []Component) error {
	if len(v) == 1 && isWord(v[0], kwNone) {
		return nil
	}
	if len(v) == 0 {
		return fmt.Errorf("expected tracks")
	}
	if err := checkAutoRepeat(v); err != nil {
		return err
	}
	for _, c := range v {
		if c.Kind == LBracket {
			return fmt.Errorf("named grid lines are not supported")
		}
		if c.Kind == Function && strings.EqualFold(c.Text, "repeat") {
			if err := checkRepeat(c); err != nil {
				return err
			}
			continue
		}
		if err := trackSize(c); err != nil {
			return err
		}
	}
	return nil
}

func isAutoRepeat(c Component) bool {
	if c.Kind != Function || !strings.EqualFold(c.Text, "repeat") {
		return false
	}
	a := items(c.Args)
	return len(a) > 0 && isWord(a[0], "auto-fill", "auto-fit")
}

// checkAutoRepeat enforces CSS's rule for a track list with
// repeat(auto-fill|auto-fit, …): one such repeat, and only fixed sizes
// everywhere in the list.
func checkAutoRepeat(v []Component) error {
	autos := 0
	for _, c := range v {
		if isAutoRepeat(c) {
			autos++
		}
	}
	if autos == 0 {
		return nil
	}
	if autos > 1 {
		return fmt.Errorf("only one repeat(auto-fill/auto-fit) per track list")
	}
	for _, c := range v {
		tracks := []Component{c}
		if c.Kind == Function && strings.EqualFold(c.Text, "repeat") {
			if a := items(c.Args); len(a) > 2 {
				tracks = a[2:]
			}
		}
		for _, t := range tracks {
			if !fixedSize(t) {
				return fmt.Errorf("with repeat(auto-fill/auto-fit), every track needs a fixed size; %q is not", t.String())
			}
		}
	}
	return nil
}

func fixedBreadth(c Component) bool { return sizeLP(c) == nil }

func fixedSize(c Component) bool {
	if c.Kind == Function && strings.EqualFold(c.Text, "minmax") {
		args := splitOn(items(c.Args), func(c Component) bool { return c.Kind == Comma })
		if len(args) != 2 || len(args[0]) != 1 || len(args[1]) != 1 {
			return false
		}
		lo, hi := args[0][0], args[1][0]
		inflexible := fixedBreadth(lo) || isWord(lo, "auto", "min-content", "max-content")
		return fixedBreadth(lo) && trackBreadth(hi) == nil || inflexible && fixedBreadth(hi)
	}
	return fixedBreadth(c)
}

func checkRepeat(c Component) error {
	args := items(c.Args)
	if len(args) < 3 || args[1].Kind != Comma {
		return fmt.Errorf("repeat() takes a count, a comma and tracks")
	}
	if !isWord(args[0], "auto-fill", "auto-fit") && (integer(args[0]) != nil || args[0].Num < 1) {
		return fmt.Errorf("repeat()'s count is a positive integer, auto-fill or auto-fit")
	}
	for _, t := range args[2:] {
		if t.Kind == Function && strings.EqualFold(t.Text, "repeat") {
			return fmt.Errorf("repeat() can't nest")
		}
		if err := trackSize(t); err != nil {
			return err
		}
	}
	return nil
}

func checkBoxShadow(v []Component) error {
	if len(v) == 1 && isWord(v[0], kwNone) {
		return nil
	}
	return commaList(func(part []Component) error {
		lengths, colors, insets, lastLength := 0, 0, 0, -1
		for i, c := range part {
			switch {
			case isWord(c, kwInset):
				insets++
			case length(c) == nil:
				if lastLength >= 0 && lastLength != i-1 {
					return fmt.Errorf("a shadow's lengths go together")
				}
				lastLength = i
				lengths++
			case colorTerm(c) == nil:
				colors++
			default:
				return fmt.Errorf("bad shadow part %q", c.String())
			}
		}
		if lengths < 2 || lengths > 4 || colors > 1 || insets > 1 {
			return fmt.Errorf("a shadow is inset? <x> <y> <blur>? <spread>? <colour>?")
		}
		return nil
	})(v)
}

//nolint:gochecknoglobals // lookup table
var transformArity = map[string][2]int{
	"translate": {1, 2}, "translatex": {1, 1}, "translatey": {1, 1},
	"scale": {1, 2}, "scalex": {1, 1}, "scaley": {1, 1},
	"rotate": {1, 1}, "skew": {1, 2}, "skewx": {1, 1}, "skewy": {1, 1},
	"matrix": {6, 6},
}

func checkTransform(v []Component) error {
	if len(v) == 1 && isWord(v[0], kwNone) {
		return nil
	}
	for _, c := range v {
		if c.Kind != Function {
			return fmt.Errorf("expected a transform function, found %q", c.String())
		}
		name := strings.ToLower(c.Text)
		ar, ok := transformArity[name]
		if !ok {
			return fmt.Errorf("%s() is not a 2D transform", c.Text)
		}
		args := splitOn(items(c.Args), func(c Component) bool { return c.Kind == Comma })
		if len(args) < ar[0] || len(args) > ar[1] {
			return fmt.Errorf("%s() takes %d to %d arguments", c.Text, ar[0], ar[1])
		}
		var arg term
		switch {
		case strings.HasPrefix(name, "translate"):
			arg = lengthPercent
		case strings.HasPrefix(name, "scale"):
			arg = dimTerm(allowNumber | allowPercent)
		case name == fnMatrix:
			arg = dimTerm(allowNumber)
		default:
			arg = func(c Component) error { _, err := angle(c); return err }
		}
		for _, a := range args {
			if len(a) != 1 {
				return fmt.Errorf("%s() arguments are separated by commas", c.Text)
			}
			if err := arg(a[0]); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkAutoFlow(v []Component) error {
	if len(v) < 1 || len(v) > 2 {
		return fmt.Errorf("expected row or column, and dense")
	}
	dir, dense := 0, 0
	for _, c := range v {
		switch {
		case isWord(c, "row", "column"):
			dir++
		case isWord(c, "dense"):
			dense++
		default:
			return fmt.Errorf("expected row, column or dense, found %q", c.String())
		}
	}
	if dir > 1 || dense > 1 {
		return fmt.Errorf("expected row or column, and dense")
	}
	return nil
}

func checkTransformOrigin(v []Component) error {
	if len(v) < 1 || len(v) > 2 {
		return fmt.Errorf("transform-origin takes one or two positions")
	}
	for _, c := range v {
		if err := either(lengthPercent, words(kwLeft, kwCenter, kwRight, kwTop, kwBottom))(c); err != nil {
			return err
		}
	}
	return nil
}

// position is a background-position read into its two axes: each an offset from the start
// edge, or from the end one (right, bottom) when that edge has an offset; keywords alone
// are percentages (left 0%, center 50%, right 100%), as Chrome computes them.
type position struct {
	x, y       Component
	right, bot bool
}

//nolint:gochecknoglobals // fixed values the grammar fills in
var (
	fifty      = Component{Token: Token{Kind: Percentage, Num: 50, Text: "50"}}
	zeroPct    = Component{Token: Token{Kind: Percentage, Num: 0, Text: "0"}}
	hundredPct = Component{Token: Token{Kind: Percentage, Num: 100, Text: "100"}}
)

func checkPosition(v []Component) error {
	_, err := parsePosition(v)
	return err
}

// parsePosition reads CSS's <bg-position>: one or two values (keywords or lengths), or
// three or four as edges with offsets ("right 6px center", "left 10% bottom 2px").
func parsePosition(v []Component) (position, error) {
	var p position
	var err error
	switch len(v) {
	case 1:
		p.x, p.y = fifty, fifty
		switch {
		case isYEdge(v[0]):
			err = p.set('y', v[0], nil)
		case isEdge(v[0]):
			err = p.set('x', v[0], nil)
		default:
			err = lengthPercent(v[0])
		}
	case 2:
		err = p.pair(v[0], v[1])
	case 3, 4:
		err = p.edges(v)
	default:
		err = fmt.Errorf("background-position takes 1 to 4 values")
	}
	return p, err
}

func isXEdge(c Component) bool { return isWord(c, kwLeft, kwRight) }
func isYEdge(c Component) bool { return isWord(c, kwTop, kwBottom) }
func isEdge(c Component) bool  { return isXEdge(c) || isYEdge(c) || isWord(c, kwCenter) }

// set places p on axis by the keyword kw, offset by off from that edge.
func (p *position) set(axis byte, kw Component, off *Component) error {
	end := isWord(kw, kwRight, kwBottom)
	o := zeroPct
	switch {
	case isWord(kw, kwCenter):
		if off != nil {
			return fmt.Errorf("center takes no offset")
		}
		o = fifty
	case off != nil:
		o = *off
	case end:
		o, end = hundredPct, false
	}
	if axis == 'x' {
		p.x, p.right = o, end
	} else {
		p.y, p.bot = o, end
	}
	return nil
}

// pair reads a two-value position, each a keyword or an offset.
func (p *position) pair(a, b Component) error {
	if isYEdge(a) || isXEdge(b) {
		a, b = b, a
	}
	if isYEdge(a) || isXEdge(b) {
		return fmt.Errorf("bad background-position")
	}
	for i, c := range []Component{a, b} {
		axis := byte('x' + i)
		if isEdge(c) {
			if err := p.set(axis, c, nil); err != nil {
				return err
			}
			continue
		}
		if err := lengthPercent(c); err != nil {
			return err
		}
		if axis == 'x' {
			p.x = c
		} else {
			p.y = c
		}
	}
	return nil
}

// edges reads a three- or four-value position: edges, each with an
// optional offset.
func (p *position) edges(v []Component) error {
	var groups [][2]*Component
	for i := 0; i < len(v); i++ {
		if !isEdge(v[i]) {
			return fmt.Errorf("with 3 or 4 values, background-position is edges with offsets")
		}
		g := [2]*Component{&v[i], nil}
		if i+1 < len(v) && !isEdge(v[i+1]) {
			g[1] = &v[i+1]
			if err := lengthPercent(v[i+1]); err != nil {
				return err
			}
			i++
		}
		groups = append(groups, g)
	}
	if len(groups) != 2 {
		return fmt.Errorf("bad background-position")
	}
	a, b := groups[0], groups[1]
	if isYEdge(*a[0]) || isXEdge(*b[0]) {
		a, b = b, a
	}
	if isYEdge(*a[0]) || isXEdge(*b[0]) {
		return fmt.Errorf("bad background-position")
	}
	if err := p.set('x', *a[0], a[1]); err != nil {
		return err
	}
	return p.set('y', *b[0], b[1])
}

// checkTextShadow is text-shadow: none, or shadows of two or three lengths
// and a colour.
func checkTextShadow(v []Component) error {
	if len(v) == 1 && isWord(v[0], kwNone) {
		return nil
	}
	return commaList(func(part []Component) error {
		lengths, colors, last := 0, 0, -1
		for i, c := range part {
			switch {
			case length(c) == nil:
				if last >= 0 && last != i-1 {
					return fmt.Errorf("a text shadow's lengths go together")
				}
				last = i
				lengths++
			case colorTerm(c) == nil:
				colors++
			default:
				return fmt.Errorf("bad text shadow part %q", c.String())
			}
		}
		if lengths < 2 || lengths > 3 || colors > 1 {
			return fmt.Errorf("a text shadow is <x> <y> <blur>? <colour>?")
		}
		return nil
	})(v)
}
