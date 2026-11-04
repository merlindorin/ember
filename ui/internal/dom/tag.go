package dom

// Tag is an element's kind. TextTag marks a text node.
type Tag uint8

const (
	TextTag Tag = iota
	HTML
	Body
	Div
	Section
	Header
	Footer
	Nav
	Main
	Aside
	Article
	P
	H1
	H2
	H3
	H4
	Span
	Strong
	Em
	Code
	Label
	Button
	Input
	Select
	Option
	Textarea
	Details
	Summary
	Dialog
	Table
	Thead
	Tbody
	Tr
	Th
	Td
	Progress
	Meter
	Img
	Hr
	Canvas
	Kbd
	Samp
	Pre
	Caption
	Tfoot
	tagCount
)

var tagNames = [tagCount]string{ //nolint:gochecknoglobals // a lookup table by tag
	"#text", "html", "body", "div", "section", "header", "footer", "nav", "main", "aside", "article",
	"p", "h1", "h2", "h3", "h4", "span", "strong", "em", "code", "label", "button", "input",
	"select", "option", "textarea", "details", "summary", "dialog", "table", "thead", "tbody",
	"tr", "th", "td", "progress", "meter", "img", "hr", "canvas", "kbd", "samp", "pre",
	"caption", "tfoot",
}

// String is the tag's HTML name, "#text" for text nodes.
func (t Tag) String() string {
	if t >= tagCount {
		return "unknown"
	}
	return tagNames[t]
}

// TagNamed is the tag of an HTML name, false for names ui does not have.
func TagNamed(name string) (Tag, bool) {
	for i, n := range tagNames {
		if i > 0 && n == name {
			return Tag(i), true
		}
	}
	return 0, false
}

// Void reports whether the element never has children in HTML.
func (t Tag) Void() bool {
	return t == Input || t == Img || t == Hr
}
