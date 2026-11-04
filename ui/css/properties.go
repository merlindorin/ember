package css

import (
	"fmt"
	"strings"
)

// property is one longhand of the subset: whether it inherits and its
// initial value as CSS text.
type property struct {
	inherited bool
	initial   string
}

//nolint:gochecknoglobals,goconst // the properties' table
var properties = map[string]property{
	"display":    {initial: "inline"},
	"box-sizing": {initial: "content-box"},
	"width":      {initial: "auto"}, "height": {initial: "auto"},
	"min-width": {initial: "auto"}, "min-height": {initial: "auto"},
	"max-width": {initial: "none"}, "max-height": {initial: "none"},

	"margin-top": {initial: "0"}, "margin-right": {initial: "0"},
	"margin-bottom": {initial: "0"}, "margin-left": {initial: "0"},
	"padding-top": {initial: "0"}, "padding-right": {initial: "0"},
	"padding-bottom": {initial: "0"}, "padding-left": {initial: "0"},

	"border-top-width": {initial: "medium"}, "border-right-width": {initial: "medium"},
	"border-bottom-width": {initial: "medium"}, "border-left-width": {initial: "medium"},
	"border-top-style": {initial: "none"}, "border-right-style": {initial: "none"},
	"border-bottom-style": {initial: "none"}, "border-left-style": {initial: "none"},
	"border-top-color": {initial: "currentcolor"}, "border-right-color": {initial: "currentcolor"},
	"border-bottom-color": {initial: "currentcolor"}, "border-left-color": {initial: "currentcolor"},
	"border-top-left-radius": {initial: "0"}, "border-top-right-radius": {initial: "0"},
	"border-bottom-right-radius": {initial: "0"}, "border-bottom-left-radius": {initial: "0"},
	"border-image-source": {initial: "none"}, "border-image-slice": {initial: "100%"},
	"border-image-width": {initial: "1"}, "border-image-outset": {initial: "0"},
	"border-image-repeat": {initial: "stretch"},

	"row-gap": {initial: "normal"}, "column-gap": {initial: "normal"},
	"flex-grow": {initial: "0"}, "flex-shrink": {initial: "1"}, "flex-basis": {initial: "auto"},
	"flex-wrap": {initial: "nowrap"}, "flex-direction": {initial: "row"},
	"align-items": {initial: "normal"}, "align-content": {initial: "normal"},
	"align-self": {initial: "auto"}, "justify-content": {initial: "normal"},
	"justify-items": {initial: "legacy"}, "justify-self": {initial: "auto"},
	"grid-template-columns": {initial: "none"}, "grid-template-rows": {initial: "none"},
	"grid-column-start": {initial: "auto"}, "grid-column-end": {initial: "auto"},
	"grid-row-start": {initial: "auto"}, "grid-row-end": {initial: "auto"},
	"grid-auto-columns": {initial: "auto"}, "grid-auto-rows": {initial: "auto"},
	"grid-auto-flow": {initial: "row"},
	"order":          {initial: "0"},
	"aspect-ratio":   {initial: "auto"},
	"overflow-x":     {initial: "visible"}, "overflow-y": {initial: "visible"},
	"position": {initial: "static"},
	"top":      {initial: "auto"}, "right": {initial: "auto"},
	"bottom": {initial: "auto"}, "left": {initial: "auto"},
	"z-index": {initial: "auto"},

	"font-family":            {inherited: true, initial: "sans-serif"},
	"font-size":              {inherited: true, initial: "medium"},
	"font-weight":            {inherited: true, initial: "normal"},
	"font-style":             {inherited: true, initial: "normal"},
	"line-height":            {inherited: true, initial: "normal"},
	"letter-spacing":         {inherited: true, initial: "normal"},
	"border-spacing":         {inherited: true, initial: "0"},
	"caption-side":           {inherited: true, initial: "top"},
	"table-layout":           {initial: "auto"},
	"text-align":             {inherited: true, initial: "start"},
	"text-transform":         {inherited: true, initial: "none"},
	"-webkit-font-smoothing": {inherited: true, initial: "auto"},
	"white-space":            {inherited: true, initial: "normal"},
	"overflow-wrap":          {inherited: true, initial: "normal"},
	"color":                  {inherited: true, initial: "canvastext"},
	"cursor":                 {inherited: true, initial: "auto"},
	"list-style-type":        {inherited: true, initial: "disc"},
	"text-shadow":            {inherited: true, initial: "none"},
	"image-rendering":        {inherited: true, initial: "auto"},
	"accent-color":           {inherited: true, initial: "auto"},
	"caret-color":            {inherited: true, initial: "auto"},
	"visibility":             {inherited: true, initial: "visible"},
	"pointer-events":         {inherited: true, initial: "auto"},
	"text-overflow":          {initial: "clip"},
	"scrollbar-width":        {initial: "auto"},

	"background-color":    {initial: "transparent"},
	"background-image":    {initial: "none"}, // one layer: url() or linear-gradient(), no repeating or radial ones
	"background-repeat":   {initial: "repeat"},
	"background-size":     {initial: "auto"},
	"background-position": {initial: "0% 0%"},
	"object-fit":          {initial: "fill"},
	"object-position":     {initial: "50% 50%"},
	"box-shadow":          {initial: "none"},
	"outline-width":       {initial: "medium"}, "outline-style": {initial: "none"},
	"outline-color": {initial: "currentcolor"}, "outline-offset": {initial: "0"},
	"opacity":          {initial: "1"},
	"transform":        {initial: "none"},
	"transform-origin": {initial: "50% 50%"},
	"appearance":       {initial: "none"},
	"resize":           {initial: "none"},

	"transition-property": {initial: "all"}, "transition-duration": {initial: "0s"},
	"transition-timing-function": {initial: "ease"}, "transition-delay": {initial: "0s"},
}

// shorthands lists each shorthand's longhands, in the order CSS gives them.
//
//nolint:gochecknoglobals // the shorthands' table
var shorthands = map[string][]string{
	"margin":        {"margin-top", "margin-right", "margin-bottom", "margin-left"},
	"padding":       {"padding-top", "padding-right", "padding-bottom", "padding-left"},
	"inset":         {"top", "right", "bottom", "left"},
	"border-width":  {"border-top-width", "border-right-width", "border-bottom-width", "border-left-width"},
	"border-style":  {"border-top-style", "border-right-style", "border-bottom-style", "border-left-style"},
	"border-color":  {"border-top-color", "border-right-color", "border-bottom-color", "border-left-color"},
	"border-top":    {"border-top-width", "border-top-style", "border-top-color"},
	"border-right":  {"border-right-width", "border-right-style", "border-right-color"},
	"border-bottom": {"border-bottom-width", "border-bottom-style", "border-bottom-color"},
	"border-left":   {"border-left-width", "border-left-style", "border-left-color"},
	"border": {"border-top-width", "border-right-width", "border-bottom-width", "border-left-width", "border-top-style",
		"border-right-style", "border-bottom-style", "border-left-style", "border-top-color", "border-right-color",
		"border-bottom-color", "border-left-color", "border-image-source", "border-image-slice", "border-image-width",
		"border-image-outset", "border-image-repeat"},
	"border-radius": {"border-top-left-radius", "border-top-right-radius", "border-bottom-right-radius",
		"border-bottom-left-radius"},
	"border-image": {"border-image-source", "border-image-slice", "border-image-width", "border-image-outset",
		"border-image-repeat"},
	"gap":           {"row-gap", "column-gap"},
	"flex":          {"flex-grow", "flex-shrink", "flex-basis"},
	"flex-flow":     {"flex-direction", "flex-wrap"},
	"place-items":   {"align-items", "justify-items"},
	"place-content": {"align-content", "justify-content"},
	"place-self":    {"align-self", "justify-self"},
	"grid-column":   {"grid-column-start", "grid-column-end"},
	"grid-row":      {"grid-row-start", "grid-row-end"},
	"overflow":      {"overflow-x", "overflow-y"},
	"outline":       {"outline-color", "outline-style", "outline-width"},
	"background":    {"background-color", "background-image", "background-repeat", "background-position"},
	"list-style":    {"list-style-type"},
	"transition":    {"transition-property", "transition-duration", "transition-timing-function", "transition-delay"},
}

// outside names properties a browser knows that the subset leaves out,
// with why.
//
//nolint:gochecknoglobals,goconst // unsupported properties' advice
var outside = map[string]string{
	"float":            "floats are not supported; use flex or grid",
	"clear":            "floats are not supported; use flex or grid",
	"vertical-align":   "vertical-align is not supported; use align-items or align-self",
	"columns":          "multi-column layout is not supported; use grid",
	"column-count":     "multi-column layout is not supported; use grid",
	"column-width":     "multi-column layout is not supported; use grid",
	"border-collapse":  "tables as layout are not supported; use grid",
	"font":             "the font shorthand is not supported; use font-family, font-size, font-weight and line-height",
	"text-decoration":  "text-decoration is not in the subset",
	"word-wrap":        "word-wrap is the old name of overflow-wrap",
	"grid-gap":         "grid-gap is the old name of gap",
	"grid-row-gap":     "grid-row-gap is the old name of row-gap",
	"grid-column-gap":  "grid-column-gap is the old name of column-gap",
	"-webkit-box-flex": "old flexbox is not supported; use flex",
	"box-flex":         "old flexbox is not supported; use flex",
	"zoom":             "zoom is not supported; use transform: scale()",
	"clip":             "clip is not supported; use overflow: clip",
	"animation":        "animations are not supported yet; use transition",
}

// checkProperty returns why name can't be used, or "" when it can.
func checkProperty(name string) string {
	if strings.HasPrefix(name, "--") {
		return ""
	}
	if _, ok := properties[name]; ok {
		return ""
	}
	if _, ok := shorthands[name]; ok {
		return ""
	}
	if why, ok := outside[name]; ok {
		return why
	}
	if strings.HasPrefix(name, "-") {
		return fmt.Sprintf("vendor-prefixed property %q is not supported", name)
	}
	return fmt.Sprintf("unknown property %q", name)
}
